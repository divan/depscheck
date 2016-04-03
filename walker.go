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

	Stdlib   bool
	Internal bool

	Visited map[*ast.FuncDecl]*Selector
}

// NewWalker inits new AST walker.
func NewWalker(p *loader.Program, stdlib, internal bool) *Walker {
	packages := make(map[string]Package)
	for _, pkg := range p.InitialPackages() {
		// prepare map of resolved imports
		for _, i := range pkg.Pkg.Imports() {

			if !stdlib && IsStdlib(i.Path()) {
				continue
			}
			if !internal && IsInternal(pkg.Pkg.Path(), i.Path()) {
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

		Stdlib:   stdlib,
		Internal: internal,

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

// WalkPackage looks for dependencies used in a given package and saves
// selectors to result.
//
// It should be called for the top-level package only.
// Only external dependencies are added to result.
func (w *Walker) WalkPackage(pkg *loader.PackageInfo, result *Result) {
	for _, obj := range pkg.Uses {
		if obj.Pkg() == nil || obj.Pkg() == pkg.Pkg {
			continue
		}

		// Omit the internal modules
		if !w.Internal && IsInternal(pkg.Pkg.Path(), obj.Pkg().Path()) {
			continue
		}

		if !obj.Exported() {
			continue
		}

		depPkg := w.P.Package(obj.Pkg().Path())

		if sel := w.WalkObject(depPkg, obj); sel != nil {
			result.Add(sel)
		}
	}
}

// WalkObject builds Selector from the given pkg and object.
//
// It recursively goes into nested functions/calls adding it as Deps.
func (w *Walker) WalkObject(pkg *loader.PackageInfo, obj types.Object) *Selector {
	if obj == nil {
		return nil
	}

	if !w.Stdlib && IsStdlib(pkg.Pkg.Path()) {
		return nil
	}

	decl, def := w.FindDefDecl(pkg, obj)
	if def == nil || decl == nil {
		return nil
	}

	var typ, recv string

	switch d := def.(type) {
	case *types.Const:
		typ = "const"
	case *types.Var:
		if d.IsField() {
			return nil
		}
		typ = "var"
	case *types.Func:
		typ = "func"
		if r := d.Type().(*types.Signature).Recv(); r != nil {
			typ = "method"
			recv = printType(r.Type())
		}
	case *types.TypeName:
		typ = "type"
		if _, ok := d.Type().Underlying().(*types.Interface); ok {
			typ = "interface"
		}
	}

	fnDecl := w.FnDecl(pkg, decl)
	if fnDecl == nil {
		return NewSelector(pkg.Pkg, obj.Name(), recv, typ, 0)
	}

	if sel, ok := w.Visited[fnDecl]; ok {
		return sel
	}

	loc := w.LOC(fnDecl)
	sel := NewSelector(pkg.Pkg, fnDecl.Name.Name, recv, typ, loc)

	w.Visited[fnDecl] = sel
	deps := w.WalkFuncBody(pkg, fnDecl)

	if !deps.HasRecursion(sel) {
		sel.Deps = append(sel.Deps, deps...)
		// update visited Selector with deps
		w.Visited[fnDecl] = sel
	}

	return sel
}

// WalkFuncBody searches for all internal or external selectors, used in a given
// function. It recursively goes into it, building Deps slice.
func (w *Walker) WalkFuncBody(pkg *loader.PackageInfo, node *ast.FuncDecl) Deps {
	var deps Deps
	ast.Inspect(node, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.CallExpr:
			switch expr := expr.Fun.(type) {
			case *ast.Ident:
				obj := w.LookupObject(pkg, expr)
				s := w.WalkObject(pkg, obj)
				if s != nil {
					deps.Append(s)
				}
				return false
			case *ast.SelectorExpr:
				obj, ok := pkg.Uses[expr.Sel]
				if !ok || obj.Pkg() == nil {
					return false
				}

				depPkg := w.P.Package(obj.Pkg().Path())
				s := w.WalkObject(depPkg, obj)
				if s != nil {
					deps.Append(s)
				}
				return false
			}
			return false
		}
		return true
	})
	return deps
}

// FindDefDecl searches for declaration and definition for the given object.
func (w *Walker) FindDefDecl(pkg *loader.PackageInfo, obj types.Object) (*ast.Ident, types.Object) {
	for decl, def := range pkg.Defs {
		if def == nil || obj == nil {
			continue
		}
		if def == obj {
			return decl, def
		}
	}

	return nil, nil
}

// FnDecl searches for the FuncDecl based on ast.Ident node.
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
		w.CacheLOC[node] = 0
		return 0
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

// LookupObject searches for the object in current package by ast.Ident node.
func (w *Walker) LookupObject(pkg *loader.PackageInfo, expr *ast.Ident) types.Object {
	for decl, def := range pkg.Defs {
		if decl.Obj != nil && decl.Obj == expr.Obj {
			return def
		}
	}

	return nil
}

func printType(t types.Type) string {
	switch t := t.(type) {
	case *types.Pointer:
		return fmt.Sprintf("*%s", printType(t.Elem()))
	case *types.Named:
		return t.Obj().Name()
	}
	return t.String()
}
