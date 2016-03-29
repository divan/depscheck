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

		Visited: make(map[*ast.FuncDecl]bool),
	}
}

// TopWalk walks the initial package, looking only for selectors from imported
// packages.
func (w *Walker) TopWalk() *Result {
	result := NewResult()
	for _, pkg := range w.P.InitialPackages() {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				if x, ok := n.(*ast.SelectorExpr); ok {
					pkgName := pkgNameFromExpr(x, pkg)

					// skip funcs/methods of current package
					if pkgName == pkg.Pkg.Name() {
						return true
					}

					sel := w.WalkSelectorExpr(nil, file, pkg, x)
					if sel != nil && sel.Pkg.Path != pkg.Pkg.Path() {
						result.Add(sel)
					}
					return true
				}
				return true
			})
		}
	}
	return result
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
				loc := w.LOC(fnDecl)
				sel = NewSelector(pkg, name, "", "", loc)
			}
			w.processFunc(sel, fnDecl, pkg, name)
			sel.DepthInternal++
		}
	}
	return sel
}

// WalkSelectorExpr walks throug SelecorExpr node.
func (w *Walker) WalkSelectorExpr(sel *Selector, node ast.Node, parent *loader.PackageInfo, expr *ast.SelectorExpr) *Selector {
	pkgName := pkgNameFromExpr(expr, parent)
	if pkgName == "" {
		return nil
	}

	pkg := w.resolvePkg(parent, pkgName)
	if pkg == nil {
		return sel
	}
	name := expr.Sel.Name

	// lookup this object in package
	obj := w.getObject(expr, parent, pkg, name)
	if obj == nil {
		return sel
	}

	// if name of current package(parent) and pkg of selector are equal, it's internal func/method call
	internal := (pkgName == parent.Pkg.Name())

	if isField(obj) {
		_, recv := recvAndType(expr, parent)
		typ := "var"
		sel = NewSelector(pkg, name, recv, typ, 0)
	} else if isFunc(obj) {
		fnDecl := w.FindFnNode(pkg, name)
		if fnDecl == nil {
			return sel
		}

		// skip recursive calls
		if w.Visited[fnDecl] {
			return sel
		}

		if sel == nil {
			loc := w.LOC(fnDecl)
			typ, recv := recvAndType(expr, parent)
			sel = NewSelector(pkg, name, recv, typ, loc)
		} else {
			sel.IncDepth(internal)
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

func (w *Walker) FindImport(pkg *loader.PackageInfo, name string) *loader.PackageInfo {
	for _, p := range pkg.Pkg.Imports() {
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

func isField(obj types.Object) bool {
	// TODO: add pointers support
	_, ok := obj.Type().(*types.Named)
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

func pkgNameFromExpr(expr *ast.SelectorExpr, parent *loader.PackageInfo) string {
	s, ok := parent.Selections[expr]
	if ok {
		if s.Obj().Pkg() == nil {
			return ""
		}
		return s.Obj().Pkg().Name()
	}
	return packageName(expr)
}

func recvAndType(expr *ast.SelectorExpr, parent *loader.PackageInfo) (string, string) {
	typ := "func"
	recv := ""

	s, ok := parent.Selections[expr]
	if ok {
		switch s.Kind() {
		case types.MethodVal:
			recv = recvFromSelector(s.Recv())
			typ = "func"
			if recv != "" {
				typ = "method"
			}
		case types.FieldVal:
			typ = "field"
		case types.MethodExpr:
			typ = "func"
		}
	}
	return typ, recv
}

func (w *Walker) resolvePkg(parent *loader.PackageInfo, pkgName string) *loader.PackageInfo {
	if pkgName != parent.Pkg.Name() {
		return w.FindImport(parent, pkgName)
	}
	return parent
}

func (w *Walker) getObject(expr *ast.SelectorExpr, parent, pkg *loader.PackageInfo, name string) types.Object {
	if s, ok := parent.Selections[expr]; ok {
		return s.Obj()
	}
	return w.FindObject(pkg, name)
}
