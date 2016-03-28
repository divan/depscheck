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
	}
}

// Walk walks through function body block (node),
// looking for internal and external dependencies expressions.
func (w *Walker) Walk(node ast.Node, pkg *loader.PackageInfo) *Selector {
	var sel *Selector
	ast.Inspect(node, func(n ast.Node) bool {
		if x, ok := n.(*ast.SelectorExpr); ok {
			sel = w.WalkSelectorExpr(node, pkg, x)
			return sel != nil
		}

		if x, ok := n.(*ast.CallExpr); ok {
			sel = w.WalkCallExpr(node, pkg, x)
			if sel != nil {
				sel.DepthInternal++
			}
			return sel != nil
		}
		return true
	})
	return sel
}

// WalkCallExpr walks down through CallExpr AST-node. It may represent both
// local and external dependency call, so handle both.
func (w *Walker) WalkCallExpr(node ast.Node, pkg *loader.PackageInfo, expr *ast.CallExpr) *Selector {
	var name string
	switch expr := expr.Fun.(type) {
	case *ast.Ident:
		name = expr.Name
	case *ast.SelectorExpr:
		return w.WalkSelectorExpr(node, pkg, expr)
	}

	// lookup this object in package
	obj := w.FindObject(pkg, name)
	if obj == nil {
		return nil
	}

	return w.walkFunc(node, obj, pkg, name, true)
}

// WalkSelectorExpr walks throug SelecorExpr node.
func (w *Walker) WalkSelectorExpr(node ast.Node, pkg *loader.PackageInfo, expr *ast.SelectorExpr) *Selector {
	var (
		pkgName string
		obj     types.Object
	)

	// Look for Selections map first
	s, ok := pkg.Selections[expr]
	if ok {
		// (pkg).pkgvar.Method()
		obj = s.Obj()
		if obj.Pkg() == nil {
			return nil
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
			return nil
		}
	}

	name := expr.Sel.Name

	// lookup this object in package
	if obj == nil {
		obj = w.FindObject(pkg, name)
		if obj == nil {
			return nil
		}
	}

	return w.walkFunc(node, obj, pkg, name, internal)
}

func (w *Walker) walkFunc(node ast.Node, obj types.Object, pkg *loader.PackageInfo, name string, internal bool) *Selector {
	if _, ok := obj.Type().(*types.Signature); ok {
		fnDecl := w.FindFnNode(pkg, name)
		if fnDecl != nil {
			// skip recursive calls
			if fnDecl == node {
				return nil
			}

			loc := w.LOC(fnDecl)

			s := NewSelector(pkg.Pkg.Name(), pkg.Pkg.Path(), name, loc)

			sel := w.Walk(fnDecl, pkg)
			if sel != nil {
				if !internal {
					s.DepthInternal += sel.DepthInternal
				} else {
					s.Depth += sel.Depth
				}
				s.LOCCum += sel.LOCCum
			}
			return s
		}
	}
	return nil
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
