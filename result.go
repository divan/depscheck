package main

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"os"
	"sort"
)

// Result holds final result of this tool.
type Result struct {
	SelectorsMap map[string]*Selector
	Counter      map[string]int
}

// NewResult inits new Result.
func NewResult() *Result {
	return &Result{
		SelectorsMap: make(map[string]*Selector),
		Counter:      make(map[string]int),
	}
}

// Add adds new selector to the result.
func (r *Result) Add(sel *Selector) {
	if _, ok := r.SelectorsMap[sel.ID()]; !ok {
		r.SelectorsMap[sel.ID()] = sel
	}
	r.Counter[sel.ID()]++
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
			locCum = fmt.Sprintf("%d", sel.LOCCum)
			depth = fmt.Sprintf("%d", sel.Depth)
			depthInt = fmt.Sprintf("%d", sel.DepthInternal)
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
	for _, sel := range r.SelectorsMap {
		ret = append(ret, sel)
	}
	return ret
}

type ByName []*Selector

func (b ByName) Len() int      { return len(b) }
func (b ByName) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByName) Less(i, j int) bool {
	return b[i].ID() < b[j].ID()
}
