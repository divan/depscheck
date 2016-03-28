package main

import (
	"fmt"
	"go/ast"
	"os"

	"golang.org/x/tools/go/loader"
)

func main() {
	var conf loader.Config

	conf.FromArgs(os.Args[1:], false)
	p, err := conf.Load()
	if err != nil {
		fmt.Println(err)
		return
	}

	w := NewWalker(p)

	for _, pkg := range p.InitialPackages() {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				if x, ok := n.(*ast.SelectorExpr); ok {
					sel := w.WalkSelectorExpr(file, pkg, x)
					if sel != nil && sel.Pkg.Path != pkg.Pkg.Path() {
						fmt.Println("TOP", sel.Name, sel.Depth, sel.DepthInternal)
						if _, ok := w.SelectorsMap[sel.String()]; !ok {
							w.Selectors = append(w.Selectors, sel)
							w.SelectorsMap[sel.String()] = sel
						}
						w.Counter[*sel]++
					}
					return true
				}
				return true
			})
		}
	}

	w.PrintPretty()
}
