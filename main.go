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

	LOC int
}

func main() {
	var conf loader.Config

	conf.CreateFromFilenames(".", os.Args[1:]...)
	p, err := conf.Load()
	if err != nil {
		fmt.Println(err)
		return
	}

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

	// find all selectors ('f.x') for imported packages
	selectors := make(map[string]Selector)
	counter := make(map[string]int)
	for _, f := range top.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				n, ok := x.X.(*ast.Ident)
				if !ok {
					break
				}
				pkgName := n.Name

				// it's not a selector for external package, skip it
				pkg, ok := packages[pkgName]
				if !ok {
					break
				}

				sel := Selector{
					Pkg:  packages[pkgName],
					Name: x.Sel.Name,
				}

				// lookup this object in package
				dp := p.Package(pkg.Path)
				scope := dp.Pkg.Scope()
				obj := scope.Lookup(x.Sel.Name)

				if obj != nil {
					if _, ok := obj.Type().(*types.Signature); ok {
						sel.Type = "func"

						lines := Lines(p, dp.Pkg, x.Sel.Name)
						sel.LOC = lines
					}
				}

				name := fmt.Sprintf("%s.%s", pkgName, x.Sel.Name)

				selectors[name] = sel
				counter[name]++
			}
			return true
		})
	}

	// Print stats
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "LOC", "Type", "Count"})
	var results [][]string
	for name, count := range counter {
		sel := selectors[name]
		loc := fmt.Sprintf("%d", sel.LOC)
		count := fmt.Sprintf("%d", count)
		results = append(results, []string{name, loc, sel.Type, count})
	}
	sort.Sort(ByName(results))
	for _, v := range results {
		table.Append(v)
	}
	table.Render() // Send output
}

func Lines(p *loader.Program, pkg *types.Package, name string) int {
	var lines int
	for k, _ := range p.AllPackages[pkg].Scopes {
		// skip non-file scopes
		if _, ok := k.(*ast.File); !ok {
			continue
		}
		// inspect package top-level node to find func decls
		ast.Inspect(k, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if x.Name.Name == name {
					if x.Body == nil {
						break
					}
					start := p.Fset.Position(x.Body.Lbrace)
					end := p.Fset.Position(x.Body.Rbrace)
					lines = end.Line - start.Line
					if lines == 0 {
						lines = 1
					}
					return false
				}
			}
			return true
		})
		if lines != 0 {
			return lines
		}
	}
	return 0
}

type ByName [][]string

func (b ByName) Len() int           { return len(b) }
func (b ByName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByName) Less(i, j int) bool { return b[i][0] < b[j][0] }
