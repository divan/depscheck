package main

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"io"
	"os"
	"sort"
)

// Result holds final result of this tool.
type Result struct {
	Selectors map[string]*Selector
	Counter   map[string]int
}

// NewResult inits new Result.
func NewResult() *Result {
	return &Result{
		Selectors: make(map[string]*Selector),
		Counter:   make(map[string]int),
	}
}

// Add adds new selector to the result.
func (r *Result) Add(sel *Selector) {
	key := sel.ID()
	if _, ok := r.Selectors[key]; !ok {
		r.Selectors[key] = sel
	}
	r.Counter[key]++
}

// PrintPretty prints results to stdout in a pretty table form.
func (r *Result) PrintPretty() {
	if len(r.Counter) == 0 {
		fmt.Println("No external dependencies found in this package")
		return
	}
	selectors := r.All()
	sort.Sort(ByName(selectors))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Pkg", "Recv", "Name", "Type", "Count", "LOC", "LOCCum", "Depth", "DepthInt"})

	var results [][]string
	var lastPkg string
	for _, sel := range selectors {
		pkg := ""
		if lastPkg != sel.Pkg.Name {
			lastPkg = sel.Pkg.Name
			pkg = sel.Pkg.Name
		}
		var loc, locCum, depth, depthInt string
		if sel.Type == "func" || sel.Type == "method" {
			loc = fmt.Sprintf("%d", sel.LOC)
			locCum = fmt.Sprintf("%d", sel.LOCCum())
			depth = fmt.Sprintf("%d", sel.Depth())
			depthInt = fmt.Sprintf("%d", sel.DepthInternal())
		}
		count := fmt.Sprintf("%d", r.Counter[sel.ID()])
		results = append(results, []string{pkg, sel.Recv, sel.Name, sel.Type, count, loc, locCum, depth, depthInt})
	}
	for _, v := range results {
		table.Append(v)
	}
	table.Render() // Send output
}

// All returns all known selectors in result.
func (r *Result) All() []*Selector {
	var ret []*Selector
	for _, sel := range r.Selectors {
		ret = append(ret, sel)
	}
	return ret
}

// PackageStats returns stats by packages in all selectors.
func (r *Result) PackagesStats() []*PackageStat {
	pkgs := make(map[Package]*PackageStat)
	for _, sel := range r.Selectors {
		if _, ok := pkgs[sel.Pkg]; !ok {
			pkgs[sel.Pkg] = NewPackageStat(sel.Pkg)
		}
		pkgs[sel.Pkg].DepsCount++
		pkgs[sel.Pkg].DepsCallsCount += r.Counter[sel.ID()]
		pkgs[sel.Pkg].LOCCum += sel.LOCCum()
		pkgs[sel.Pkg].Depth += sel.Depth()
		pkgs[sel.Pkg].DepthInternal += sel.DepthInternal()

	}

	var ret []*PackageStat
	for _, stat := range pkgs {
		ret = append(ret, stat)
	}
	sort.Sort(ByPackageName(ret))
	return ret
}

func (sel *Selector) LOCCum() int {
	if !sel.IsFunc() {
		return 0
	}

	ret := sel.LOC
	for _, dep := range sel.Deps {
		ret += dep.LOCCum()
	}

	return ret
}

func (sel *Selector) Depth() int {
	if !sel.IsFunc() {
		return 0
	}

	ret := 0
	for _, dep := range sel.Deps {
		fmt.Println(dep)
		if dep.Pkg != sel.Pkg {
			ret++
			ret += dep.Depth()
		}
	}

	return ret
}

func (sel *Selector) DepthInternal() int {
	if !sel.IsFunc() {
		return 0
	}

	ret := 0
	for _, dep := range sel.Deps {
		if dep.Pkg == sel.Pkg {
			ret++
			ret += dep.DepthInternal()
		}
	}

	return ret
}

func (sel *Selector) IsFunc() bool {
	return sel.Type == "func" || sel.Type == "method"
}

type ByName []*Selector

func (b ByName) Len() int      { return len(b) }
func (b ByName) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByName) Less(i, j int) bool {
	return b[i].ID() < b[j].ID()
}

type ByPackageName []*PackageStat

func (b ByPackageName) Len() int      { return len(b) }
func (b ByPackageName) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByPackageName) Less(i, j int) bool {
	return b[i].Name < b[j].Name
}

// LinterOutput analyzes results and print linter output.
//
// Linter output means suggestions which dependencies may be
// copied to your source, because of its size.
func (r *Result) LinterOutput(w io.Writer) {
	if len(r.Counter) == 0 {
		return
	}

	for _, p := range r.PackagesStats() {
		if p.CanBeAvoided() {
			fmt.Fprintf(w, "Package %s (%s) is a good candidate for removing from dependencies.\n", p.Name, p.Path)
			fmt.Fprintf(w, "  Only %d LOC used, in %d calls, with %d level of nesting\n", p.LOCCum, p.DepsCount, p.DepthInternal)
		}
	}
}
