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

					node := w.FindFnNode(dp, x.Sel.Name)
					if node != nil {
						lines := w.Lines(node)
						_, depth, linesCum, depthInt := w.WalkExternal(node, dp)
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
	var (
		lines int
		ok    bool
	)
	if lines, ok = w.CacheLOC[node]; ok {
		return lines
	}
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
			w.CacheLOC[node] = lines
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
	P          *loader.Program
	Packages   map[string]Package
	CacheLOC   map[ast.Node]int
	CacheNodes map[string]ast.Node

	Stdlib bool
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
		CacheLOC:   make(map[ast.Node]int),
		CacheNodes: make(map[string]ast.Node),

		Stdlib: false,
	}
}

// WalkExternal walks through function body block,
// looking for external dependencies expressions.
func (w *Walker) WalkExternal(topNode ast.Node, parent *loader.PackageInfo) (lines, depth, locCum, depthInt int) {
	ast.Inspect(topNode, func(n ast.Node) bool {
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
				node := w.FindFnNode(parent, name)
				// skip recursive calls
				if node == topNode {
					return false
				}

				if node != nil {
					depthInt++
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

			// skip recursive calls
			if obj == nil {
				return true
			}

			if _, ok := obj.Type().(*types.Signature); ok {

				node := w.FindFnNode(pkg, name)
				if node == topNode {
					return false
				}

				if node != nil {
					depth++
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

func (w *Walker) FindObject(pkg *loader.PackageInfo, name string) types.Object {
	scope := pkg.Pkg.Scope()
	return scope.Lookup(name)
}

func (w *Walker) FindFnNode(pkg *loader.PackageInfo, fnName string) ast.Node {
	var (
		node ast.Node
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
					node = n
					w.CacheNodes[qName] = n
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
