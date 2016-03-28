package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"golang.org/x/tools/go/loader"
)

type Walker struct {
	P          *loader.Program
	Packages   map[string]Package
	CacheLOC   map[*ast.FuncDecl]int
	CacheNodes map[string]*ast.FuncDecl

	Stdlib bool

	Selectors    []*Selector
	SelectorsMap map[string]*Selector
	Counter      map[Selector]int

	Visited map[*ast.FuncDecl]bool
}

// NewWalker inits new AST walker.
func NewWalker(p *loader.Program) *Walker {
	packages := make(map[string]Package)
	for _, pkg := range p.InitialPackages() {
		// prepare map of resolved imports
		for _, i := range pkg.Pkg.Imports() {
			if IsStdlib(i.Path()) {
				continue
			}
			packages[i.Name()] = NewPackage(i.Name(), i.Path())
		}
	}

	return &Walker{
		P:          p,
		Packages:   packages,
		CacheLOC:   make(map[*ast.FuncDecl]int),
		CacheNodes: make(map[string]*ast.FuncDecl),

		Stdlib: false,

		Selectors:    []*Selector{},
		SelectorsMap: make(map[string]*Selector),
		Counter:      make(map[Selector]int),

		Visited: make(map[*ast.FuncDecl]bool),
	}
}

// WalkFnBody walks through function body block (node),
// looking for internal and external dependencies expressions.
func (w *Walker) WalkFnBody(sel *Selector, node *ast.FuncDecl, pkg *loader.PackageInfo) *Selector {
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			sel = w.WalkSelectorExpr(sel, node, pkg, x)
			return false
		case *ast.CallExpr:
			sel = w.WalkCallExpr(sel, node, pkg, x)
			return false
		}
		return true
	})
	return sel
}

// WalkCallExpr walks down through CallExpr AST-node. It may represent both
// local and external dependency call, so handle both.
func (w *Walker) WalkCallExpr(sel *Selector, node ast.Node, pkg *loader.PackageInfo, expr *ast.CallExpr) *Selector {
	var name string
	switch expr := expr.Fun.(type) {
	case *ast.Ident:
		name = expr.Name
	case *ast.SelectorExpr:
		return w.WalkSelectorExpr(sel, node, pkg, expr)
	}

	// lookup this object in package
	obj := w.FindObject(pkg, name)
	if obj == nil {
		return sel
	}

	if _, ok := obj.Type().(*types.Signature); ok {
		fnDecl := w.FindFnNode(pkg, name)
		// skip recursive calls
		if w.Visited[fnDecl] {
			return sel
		}
		if fnDecl != nil {
			if sel == nil {
				sel = NewSelector(pkg, name)
				sel.LOC = w.LOC(fnDecl)
			}
			w.processFunc(sel, fnDecl, pkg, name)
			sel.DepthInternal++
		}
	}
	return sel
}

// WalkSelectorExpr walks throug SelecorExpr node.
func (w *Walker) WalkSelectorExpr(sel *Selector, node ast.Node, pkg *loader.PackageInfo, expr *ast.SelectorExpr) *Selector {
	var (
		pkgName string
		obj     types.Object
		recv    string
	)

	// Look for Selections map first
	s, ok := pkg.Selections[expr]
	if ok {
		// (pkg).pkgvar.Method()
		obj = s.Obj()
		if obj.Pkg() == nil {
			return sel
		}
		if s.Kind() == types.MethodVal {
			recv = recvFromSelector(s.Recv())
		}
		pkgName = obj.Pkg().Name()
	} else {
		// pkg.Func()
		pkgName = packageName(expr)
	}

	internal := (pkgName == pkg.Pkg.Name())
	if !internal {
		pkg = w.FindImport(pkg, pkgName)

		if pkg == nil {
			return sel
		}
	}

	name := expr.Sel.Name

	// lookup this object in package
	if obj == nil {
		obj = w.FindObject(pkg, name)
		if obj == nil {
			return sel
		}
	}

	if isFunc(obj) {
		fnDecl := w.FindFnNode(pkg, name)
		if fnDecl == nil {
			return sel
		}

		// skip recursive calls
		if w.Visited[fnDecl] {
			return sel
		}

		if sel == nil {
			sel = NewSelector(pkg, name)
			sel.LOC = w.LOC(fnDecl)
			sel.Recv = recv
		} else {
			if internal {
				sel.DepthInternal++
			} else {
				sel.Depth++
			}
		}

		if sel.Recv == "" {
			sel.Type = "func"
		} else {
			sel.Type = "method"
		}
		w.processFunc(sel, fnDecl, pkg, name)
	}
	return sel
}

func (w *Walker) processFunc(sel *Selector, fnDecl *ast.FuncDecl, pkg *loader.PackageInfo, name string) *Selector {
	sel.LOCCum += w.LOC(fnDecl)
	w.Visited[fnDecl] = true
	w.WalkFnBody(sel, fnDecl, pkg)
	return sel
}

func (w *Walker) FindObject(pkg *loader.PackageInfo, name string) types.Object {
	scope := pkg.Pkg.Scope()
	return scope.Lookup(name)
}

func (w *Walker) FindFnNode(pkg *loader.PackageInfo, fnName string) *ast.FuncDecl {
	var (
		node *ast.FuncDecl
		ok   bool
	)
	qName := fmt.Sprintf("%s.%s", pkg.Pkg.Path(), fnName)
	node, ok = w.CacheNodes[qName]
	if ok {
		return node
	}
	for _, f := range pkg.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if x.Name.Name == fnName {
					if x.Body == nil {
						return false
						w.CacheNodes[qName] = nil
					}
					node = x
					w.CacheNodes[qName] = x
					return false
				}
			}
			return true
		})
		if node != nil {
			return node
		}
	}
	return nil
}

func (w *Walker) FindImport(parent *loader.PackageInfo, name string) *loader.PackageInfo {
	for _, p := range parent.Pkg.Imports() {
		if p.Name() == name {
			if !w.Stdlib {
				std := IsStdlib(p.Path())
				if std {
					return nil
				}
			}
			return w.P.Package(p.Path())
		}
	}
	return nil
}

// LOC calculates readl Lines Of Code for the given function node.
// node must be ast.FuncDecl, panics otherwise.
func (w *Walker) LOC(node *ast.FuncDecl) int {
	if lines, ok := w.CacheLOC[node]; ok {
		return lines
	}

	body := node.Body
	if body == nil {
		return 0
		w.CacheLOC[node] = 0
	}

	start := w.P.Fset.Position(body.Lbrace)
	end := w.P.Fset.Position(body.Rbrace)
	lines := end.Line - start.Line

	// for cases line 'func foo() { bar() }'
	// TODO: figure out how to calculate it smarter
	if lines == 0 {
		lines = 1
	}

	w.CacheLOC[node] = lines

	return lines
}

// packageName returns qualified package name from SelectorExpr.
func packageName(x *ast.SelectorExpr) string {
	n, ok := x.X.(*ast.Ident)
	if !ok {
		return ""
	}

	return n.Name
}

func isFunc(obj types.Object) bool {
	_, ok := obj.Type().(*types.Signature)
	return ok
}

func recvFromSelector(s types.Type) string {
	if recv, ok := s.(*types.Named); ok {
		return recv.Obj().Name()
	}
	if recv, ok := s.(*types.Pointer); ok {
		named, ok := recv.Elem().(*types.Named)
		if !ok {
			return ""
		}
		return fmt.Sprintf("*%s", named.Obj().Name())
	}
	return ""
}
