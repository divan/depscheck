package main

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"os"
	"sort"
)

type ByName []*Selector

func (b ByName) Len() int      { return len(b) }
func (b ByName) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByName) Less(i, j int) bool {
	return b[i].String() < b[j].String()
}

func (r *Result) PrintPretty() {
	if len(r.Counter) == 0 {
		fmt.Println("No external dependencies found in this package")
		return
	}

	selectors := r.Selectors
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
		count := fmt.Sprintf("%d", r.Counter[*sel])
		results = append(results, []string{pkg, sel.Recv, sel.Name, sel.Type, count, loc, locCum, depth, depthInt})
	}
	for _, v := range results {
		table.Append(v)
	}
	table.Render() // Send output
}
