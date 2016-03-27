package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"

	"golang.org/x/tools/go/loader"
)

func main() {
	var conf loader.Config

	conf.CreateFromFilenames(".", os.Args[1:]...)
	p, err := conf.Load()
	if err != nil {
		fmt.Println(err)
		return
	}

	w := NewWalker(p)

	// work only with top package for now.
	// TODO: work with all recursive sub-packages (optionally?)
	topPkg := p.InitialPackages()[0]

	for _, pkg := p.InitialPackages() {
	for _, f := range topPkg.Files {
		w.Walk(f, topPkg, true)
	}
}

	// Print stats
	w.PrintPretty()
}

// pkgName returns qualified package name from SelectorExpr.
func pkgName(x *ast.SelectorExpr) string {
	n, ok := x.X.(*ast.Ident)
	if !ok {
		return ""
	}

	return n.Name
}

type Walker struct {
	P          *loader.Program
	Packages   map[string]Package
	CacheLOC   map[*ast.FuncDecl]int
	CacheNodes map[string]*ast.FuncDecl

	Stdlib bool

	SelectorsMap map[string]*Selector
	Counter      map[string]int

	Selectors []*Selector
}

func NewWalker(p *loader.Program) *Walker {
	// work only with top package for now.
	// TODO: work with all recursive sub-packages (optionally?)
	top := p.InitialPackages()[0]

	// prepare map of resolved imports
	packages := make(map[string]Package)
	for _, pkg := range top.Pkg.Imports() {
		if IsStdlib(pkg.Path()) {
			continue
		}
		packages[pkg.Name()] = Package{
			Name: pkg.Name(),
			Path: pkg.Path(),
		}
	}

	return &Walker{
		P:          p,
		Packages:   packages,
		CacheLOC:   make(map[*ast.FuncDecl]int),
		CacheNodes: make(map[string]*ast.FuncDecl),

		Stdlib: false,

		SelectorsMap: make(map[string]*Selector),
		Counter:      make(map[string]int),

		Selectors: []*Selector{},
	}
}

// Walk walks through function body block,
// looking for internal and external dependencies expressions.
func (w *Walker) Walk(topNode ast.Node, parent *loader.PackageInfo, top bool) *Selector {
	var sel *Selector
	ast.Inspect(topNode, func(n ast.Node) bool {
		if x, ok := n.(*ast.SelectorExpr); ok {
			sel = w.WalkSelectorExpr(topNode, parent, x, top)
			return false
		}

		if !top {
			if x, ok := n.(*ast.CallExpr); ok {
				sel = w.WalkCallExpr(topNode, parent, x, top)
				if sel != nil {
					sel.DepthInternal++
				}
				return false
			}
		}
		return true
	})
	return sel
}

func (w *Walker) WalkCallExpr(node ast.Node, pkg *loader.PackageInfo, expr *ast.CallExpr, top bool) *Selector {
	var name string
	switch expr := expr.Fun.(type) {
	case *ast.Ident:
		name = expr.Name
	case *ast.SelectorExpr:
		return w.WalkSelectorExpr(node, pkg, expr, top)
	}

	// lookup this object in package
	obj := w.FindObject(pkg, name)
	if obj == nil {
		return nil
	}

	return w.walkFunc(node, obj, pkg, name, true)
}

func (w *Walker) WalkSelectorExpr(node ast.Node, parent *loader.PackageInfo, expr *ast.SelectorExpr, top bool) *Selector {
	var (
		n   string // package name
		obj types.Object
	)

	// Look for Selections map first
	s, ok := parent.Selections[expr]
	if ok {
		// (pkg).pkgvar.Method()
		obj = s.Obj()
		if obj.Pkg() == nil {
			return nil
		}
		n = obj.Pkg().Name()
	} else {
		// pkg.Func()
		n = pkgName(expr)
	}

	var pkg *loader.PackageInfo
	if n == parent.Pkg.Name() {
		pkg = parent
	} else {
		pkg = w.FindImport(parent, n)
		if pkg == nil {
			return nil
		}
	}

	name := expr.Sel.Name
	internal := (n == parent.Pkg.Name())

	// lookup this object in package
	if obj == nil {
		obj = w.FindObject(pkg, name)
		if obj == nil {
			return nil
		}
	}

	sel := w.walkFunc(node, obj, pkg, name, internal)
	if top && sel != nil {
		w.Selectors = append(w.Selectors, sel)
		w.SelectorsMap[sel.String()] = sel
		w.Counter[sel.String()]++
	}
	return sel
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

			sel := w.Walk(fnDecl, pkg, false)
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
