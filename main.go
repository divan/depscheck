package main

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"go/ast"
	"go/types"
	"os"
	"sort"

	"golang.org/x/tools/go/loader"
)

type Package struct {
	Name string
	Path string
}

type Selector struct {
	Pkg  Package
	Name string
	Type string

	LOC, LOCCum          int
	Depth, DepthInternal int
}

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
	top := p.InitialPackages()[0]

	// find all selectors ('f.x') for imported packages
	selectors := make(map[string]Selector)
	counter := make(map[string]int)
	for _, f := range top.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				n := pkgName(x)

				// if it's not a selector for external package, skip it
				pkg, ok := w.Packages[n]
				if !ok {
					break
				}

				name := fmt.Sprintf("%s.%s", n, x.Sel.Name)

				sel := Selector{
					Pkg:  w.Packages[n],
					Name: x.Sel.Name,
				}

				// lookup this object in package
				dp := p.Package(pkg.Path)
				scope := dp.Pkg.Scope()
				obj := scope.Lookup(x.Sel.Name)

				if obj == nil {
					return true
				}
				if _, ok := obj.Type().(*types.Signature); ok {
					sel.Type = "func"

					node := w.FindFnNode(dp.Pkg, x.Sel.Name)
					if node != nil {
						lines := w.Lines(node)
						_, depth, linesCum, depthInt := w.WalkExternal(node, dp.Pkg)
						sel.LOC, sel.Depth = lines, depth
						sel.LOCCum, sel.DepthInternal = linesCum, depthInt
					}
				}
				selectors[name] = sel
				counter[name]++
			}
			return true
		})
	}

	// Print stats
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "LOC", "Type", "Count", "Depth", "LOCCum", "DepthInt"})
	var results [][]string
	for name, count := range counter {
		sel := selectors[name]
		loc := fmt.Sprintf("%d", sel.LOC)
		depth := fmt.Sprintf("%d", sel.Depth)
		locCum := fmt.Sprintf("%d", sel.LOCCum)
		depthInt := fmt.Sprintf("%d", sel.DepthInternal)
		count := fmt.Sprintf("%d", count)
		results = append(results, []string{name, loc, sel.Type, count, depth, locCum, depthInt})
	}
	sort.Sort(ByName(results))
	for _, v := range results {
		table.Append(v)
	}
	table.Render() // Send output
}

func (w *Walker) Lines(node ast.Node) int {
	var lines int
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Body == nil {
				break
			}
			start := w.P.Fset.Position(x.Body.Lbrace)
			end := w.P.Fset.Position(x.Body.Rbrace)
			lines = end.Line - start.Line
			if lines == 0 {
				lines = 1
			}
			return false
		}
		return true
	})
	if lines != 0 {
		return lines
	}
	return 0
}

type ByName [][]string

func (b ByName) Len() int           { return len(b) }
func (b ByName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByName) Less(i, j int) bool { return b[i][0] < b[j][0] }

// pkgName returns qualified package name from SelectorExpr.
func pkgName(x *ast.SelectorExpr) string {
	n, ok := x.X.(*ast.Ident)
	if !ok {
		return ""
	}

	return n.Name
}

type Walker struct {
	P        *loader.Program
	Packages map[string]Package
}

func NewWalker(p *loader.Program) *Walker {
	// work only with top package for now.
	// TODO: work with all recursive sub-packages (optionally?)
	top := p.InitialPackages()[0]

	// prepare map of resolved imports
	packages := make(map[string]Package)
	for _, pkg := range top.Pkg.Imports() {
		packages[pkg.Name()] = Package{
			Name: pkg.Name(),
			Path: pkg.Path(),
		}
	}

	return &Walker{
		P:        p,
		Packages: packages,
	}
}

// WalkExternal walks through function body block,
// looking for external dependencies expressions.
func (w *Walker) WalkExternal(node ast.Node, parent *types.Package) (lines, depth, locCum, depthInt int) {
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			expr, ok := x.Fun.(*ast.Ident)
			if !ok {
				break
			}
			name := expr.Name

			// lookup this object in package
			obj := w.FindObject(parent, name)
			if obj == nil {
				return true
			}

			if _, ok := obj.Type().(*types.Signature); ok {
				depthInt++

				node := w.FindFnNode(parent, name)

				if node != nil {
					loc := w.Lines(node)
					locCum += loc

					lines1, depth1, lines2, depth2 := w.WalkExternal(node, parent)
					lines = lines1
					depth += depth1
					locCum += lines2
					depthInt += depth2
				}
			}
		case *ast.SelectorExpr:
			n := pkgName(x)
			pkg := w.FindImport(parent, n)
			if pkg == nil {
				return true
			}

			name := x.Sel.Name

			// lookup this object in package
			obj := w.FindObject(pkg, name)
			if obj == nil {
				return true
			}

			if _, ok := obj.Type().(*types.Signature); ok {
				depth++

				node := w.FindFnNode(pkg, name)

				if node != nil {
					lines1, depth1, lines2, depth2 := w.WalkExternal(node, pkg)
					lines = lines1
					depth += depth1
					locCum += lines2
					depthInt += depth2
				}
			}
		}
		return true
	})
	return
}

// WalkInternal walks through function body block,
// looking for internal functions calls.
func (w *Walker) WalkInternal(node ast.Node, parent *types.Package) (lines, depth int) {
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			fmt.Println("DD", x)
		}
		return true
	})
	return 0, 0
}

func (w *Walker) FindObject(pkg *types.Package, name string) types.Object {
	scope := pkg.Scope()
	return scope.Lookup(name)
}

func (w *Walker) FindFnNode(pkg *types.Package, fnName string) ast.Node {
	var node ast.Node
	for k := range w.P.AllPackages[pkg].Scopes {
		// skip non-file scopes
		if _, ok := k.(*ast.File); !ok {
			continue
		}

		// inspect package top-level node to find func decls
		ast.Inspect(k, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if x.Name.Name == fnName {
					if x.Body == nil {
						return false
					}
					node = n
					return false
				}
			}
			return true
		})
		return node
	}
	return nil
}

func (w *Walker) FindImport(parent *types.Package, name string) *types.Package {
	for _, p := range parent.Imports() {
		if p.Name() == name {
			return p
		}
	}
	return nil
}
