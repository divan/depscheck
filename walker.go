package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"golang.org/x/tools/go/loader"
)

// Walker holds all information needed during walking
// and analyzing AST source tree.
type Walker struct {
	P          *loader.Program
	Packages   map[string]Package
	CacheLOC   map[*ast.FuncDecl]int
	CacheNodes map[*ast.Ident]*ast.FuncDecl

	Stdlib bool

	Visited map[*ast.FuncDecl]*Selector
}

// NewWalker inits new AST walker.
func NewWalker(p *loader.Program, stdlib bool) *Walker {
	packages := make(map[string]Package)
	for _, pkg := range p.InitialPackages() {
		// prepare map of resolved imports
		for _, i := range pkg.Pkg.Imports() {
			if !stdlib && IsStdlib(i.Path()) {
				continue
			}
			packages[i.Name()] = NewPackage(i.Name(), i.Path())
		}
	}

	return &Walker{
		P:          p,
		Packages:   packages,
		CacheLOC:   make(map[*ast.FuncDecl]int),
		CacheNodes: make(map[*ast.Ident]*ast.FuncDecl),

		Stdlib: stdlib,

		Visited: make(map[*ast.FuncDecl]*Selector),
	}
}

// TopWalk walks the initial package, looking only for selectors from imported
// packages.
func (w *Walker) TopWalk() *Result {
	result := NewResult()
	for _, pkg := range w.P.InitialPackages() {
		w.WalkPackage(pkg, result)
	}
	return result
}

func (w *Walker) WalkPackage(pkg *loader.PackageInfo, result *Result) {
	for _, obj := range pkg.Uses {
		if obj.Pkg() == nil || obj.Pkg() == pkg.Pkg {
			continue
		}

		/*
			if !obj.Exported() {
				continue
			}
		*/

		switch obj.Type().(type) {
		case *types.Signature: // func or method
			depPkg := w.P.Package(obj.Pkg().Path())
			pkg := depPkg.Pkg
			if !w.Stdlib && IsStdlib(pkg.Path()) {
				fmt.Println("Skipping", pkg.Path())
				continue
			}
			if sel := w.WalkFunc(nil, depPkg, obj.Name()); sel != nil {
				result.Add(sel)
			}
		default:
			pkg := obj.Pkg()
			if !w.Stdlib && IsStdlib(pkg.Path()) {
				fmt.Println("Skipping", pkg.Path())
				continue
			}

			sel := NewSelector(obj.Pkg(), obj.Name(), "", "var", 0)
			result.Add(sel)
		}
	}
}

func (w *Walker) WalkFunc(sel *Selector, pkg *loader.PackageInfo, name string) *Selector {
	if !w.Stdlib && IsStdlib(pkg.Pkg.Path()) {
		fmt.Println("Skipping", pkg.Pkg.Path())
		return sel
	}
	decl, def := w.FindDefDecl(pkg, name)
	if def == nil || decl == nil {
		return sel
	}

	fnDecl := w.FnDecl(pkg, decl)
	if fnDecl == nil {
		return sel
	}

	if sel, ok := w.Visited[fnDecl]; ok {
		return sel
	}

	loc := w.LOC(fnDecl)

	typ := "func"
	if fnDecl.Recv != nil {
		typ = "method"
	}

	s := NewSelector(pkg.Pkg, fnDecl.Name.Name, "", typ, loc)
	if sel != nil {
		sel.Append(s)
	} else {
		sel = s
	}

	w.Visited[fnDecl] = sel
	return w.WalkFuncBody(sel, pkg, fnDecl)
}

func (w *Walker) WalkFuncBody(sel *Selector, pkg *loader.PackageInfo, node *ast.FuncDecl) *Selector {
	ast.Inspect(node, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.CallExpr:
			switch expr := expr.Fun.(type) {
			case *ast.Ident:
				name := expr.Name
				sel = w.WalkFunc(sel, pkg, name)
				return false
			case *ast.SelectorExpr:
				sel = w.WalkFunc(sel, pkg, expr.Sel.Name)
				return false
			}
			return false
		}
		return true
	})
	return sel
}

func (w *Walker) FindDefDecl(pkg *loader.PackageInfo, name string) (*ast.Ident, types.Object) {
	for decl, def := range pkg.Defs {
		if def == nil || name == "" {
			continue
		}
		if def.Name() == name {
			return decl, def
		}
	}

	return nil, nil
}

func (w *Walker) FnDecl(pkg *loader.PackageInfo, decl *ast.Ident) *ast.FuncDecl {
	if fn, ok := w.CacheNodes[decl]; ok {
		return fn
	}
	for _, f := range pkg.Files {
		for _, d := range f.Decls {
			if fnDecl, ok := d.(*ast.FuncDecl); ok {
				if decl == fnDecl.Name {
					w.CacheNodes[decl] = fnDecl
					return fnDecl
				}
			}
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
