package main

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
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

// PrintStats prints results to stdout in a pretty table form.
func (r *Result) PrintStats() {
	if len(r.Counter) == 0 {
		return
	}
	selectors := r.All()
	sort.Sort(ByID(selectors))

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

// PrintPackagesStats prints package stats to stdout in a pretty table form.
func (r *Result) PrintPackagesStats() {
	stats := r.PackagesStats()
	if len(stats) == 0 {
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Pkg", "Path", "Count", "Calls", "LOCCum", "Depth", "DepthInt"})

	var results [][]string
	for _, stat := range stats {
		count := fmt.Sprintf("%d", stat.DepsCount)
		callsCount := fmt.Sprintf("%d", stat.DepsCallsCount)
		loc := fmt.Sprintf("%d", stat.LOCCum)
		depth := fmt.Sprintf("%d", stat.Depth)
		depthInt := fmt.Sprintf("%d", stat.DepthInternal)
		results = append(results, []string{stat.Name, stat.Path, count, callsCount, loc, depth, depthInt})
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

// PrintDeps recursively print deps for all selectors found.
func (r *Result) PrintDeps() {
	for _, s := range r.All() {
		s.PrintDeps()
	}
}

// Suggestions analyzes results and print suggestions on deps.
//
// It attempts to suggest which dependencies could be
// copied to your source because of its small size.
func (r *Result) Suggestions() {
	if len(r.Counter) == 0 {
		return
	}

	var hasCandidates bool
	for _, p := range r.PackagesStats() {
		if p.CanBeAvoided() {
			fmt.Printf(" - Package %s (%s) is a good candidate for removing from dependencies.\n", p.Name, p.Path)
			fmt.Printf("   Only %d LOC used, in %d calls, with %d level of nesting\n", p.LOCCum, p.DepsCount, p.DepthInternal)
			hasCandidates = true
		}
	}

	if !hasCandidates {
		fmt.Println("Cool, looks like your dependencies are sane.")
	}
}

// Totals represnts total stats for all packages.
type Totals struct {
	Package string

	Packages      int
	LOC           int
	Calls         int
	Depth         int
	DepthInternal int
}

// Totals computes Totals for Result.
func (r *Result) Totals(pkg string) *Totals {
	t := &Totals{
		Package: pkg,
	}
	for _, stat := range r.PackagesStats() {
		t.Packages++
		t.LOC += stat.LOCCum
		t.Calls += stat.DepsCallsCount
		t.Depth += stat.Depth
		t.DepthInternal += stat.DepthInternal
	}
	return t
}

// String implements Stringer for Totals type.
func (t Totals) String() string {
	return fmt.Sprintf("%s: %d packages, %d LOC, %d calls, %d depth, %d depth int.",
		t.Package, t.Packages, t.LOC, t.Calls, t.Depth, t.DepthInternal)
}
