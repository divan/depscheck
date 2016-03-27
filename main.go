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

	for _, f := range top.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				var (
					n   string // package name
					obj types.Object
				)

				// handle methods
				s, ok := top.Selections[x]
				if ok {
					// (pkg).pkgvar.Method()
					obj = s.Obj()
					if obj.Pkg() == nil {
						return false
					}
					n = s.Obj().Pkg().Name()
				} else {
					// pkg.Func()
					n = pkgName(x)
				}

				// if it's not a selector for external package, skip it
				pkgInfo, ok := w.Packages[n]
				if !ok {
					break
				}

				sel := Selector{
					Pkg:  pkgInfo,
					Name: x.Sel.Name,
				}

				// lookup this object in package
				pkg := p.Package(pkgInfo.Path)
				if obj == nil {
					obj = w.FindObject(pkg, x.Sel.Name)
					if obj == nil {
						return true
					}
				}
				if _, ok := obj.Type().(*types.Signature); ok {
					sel.Type = "func"

					node := w.FindFnNode(pkg, x.Sel.Name)
					if node != nil {
						lines := w.LOC(node)
						_, depth, linesCum, depthInt := w.Walk(node, pkg, true)
						sel.LOC, sel.Depth = lines, depth
						sel.LOCCum, sel.DepthInternal = linesCum, depthInt
					}
				}
				w.Selectors[sel.String()] = sel
				w.Counter[sel.String()]++
			}
			return true
		})
	}

	// Print stats
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "LOC", "Type", "Count", "Depth", "LOCCum", "DepthInt"})
	var results [][]string
	for name, count := range w.Counter {
		sel := w.Selectors[name]
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
	CacheLOC   map[*ast.FuncDecl]int
	CacheNodes map[string]*ast.FuncDecl

	Stdlib bool

	Selectors map[string]Selector
	Counter   map[string]int
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

		Selectors: make(map[string]Selector),
		Counter:   make(map[string]int),
	}
}

// Walk walks through function body block,
// looking for internal and external dependencies expressions.
func (w *Walker) Walk(topNode ast.Node, parent *loader.PackageInfo, internal bool) (lines, depth, locCum, depthInt int) {
	ast.Inspect(topNode, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if !internal {
				return false
			}
			expr, ok := x.Fun.(*ast.Ident)
			if !ok {
				break
			}
			name := expr.Name

			// lookup this object in package
			obj := w.FindObject(parent, name)
			if obj == nil {
				return false
			}

			if _, ok := obj.Type().(*types.Signature); ok {
				node := w.FindFnNode(parent, name)
				// skip recursive calls
				if node == topNode {
					return false
				}

				if node != nil {
					depthInt++
					loc := w.LOC(node)
					locCum += loc

					lines1, depth1, lines2, depth2 := w.Walk(node, parent, true)
					lines = lines1
					depth += depth1
					locCum += lines2
					depthInt += depth2
				}
			}
		case *ast.SelectorExpr:
			var (
				n   string // package name
				obj types.Object
			)

			s, ok := parent.Selections[x]
			if ok {
				// (pkg).pkgvar.Method()
				obj = s.Obj()
				if obj.Pkg() == nil {
					return false
				}
				n = obj.Pkg().Name()
			} else {
				// pkg.Func()
				n = pkgName(x)
			}

			var pkg *loader.PackageInfo
			if n == parent.Pkg.Name() {
				pkg = parent
			} else {
				pkg = w.FindImport(parent, n)
				if pkg == nil {
					return false
				}
			}

			name := x.Sel.Name

			// lookup this object in package
			if obj == nil {
				obj = w.FindObject(pkg, name)
				if obj == nil {
					return false
				}
			}

			if _, ok := obj.Type().(*types.Signature); ok {
				node := w.FindFnNode(pkg, name)

				// skip recursive calls
				if node == topNode {
					return false
				}

				if node != nil {
					depth++
					lines1, depth1, lines2, depth2 := w.Walk(node, pkg, true)
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
