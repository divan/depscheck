package main

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"os"
	"sort"
)

type ByName [][]string

func (b ByName) Len() int           { return len(b) }
func (b ByName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByName) Less(i, j int) bool { return b[i][0] < b[j][0] }

func (w *Walker) PrintPretty() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Type", "Count", "LOC", "LOCCum", "Depth", "DepthInt"})
	var results [][]string
	for name, count := range w.Counter {
		sel := w.SelectorsMap[name]
		loc := fmt.Sprintf("%d", sel.LOC)
		locCum := fmt.Sprintf("%d", sel.LOCCum)
		depth := fmt.Sprintf("%d", sel.Depth-1)
		depthInt := fmt.Sprintf("%d", sel.DepthInternal-1)
		count := fmt.Sprintf("%d", count)
		results = append(results, []string{name, sel.Type, count, loc, locCum, depth, depthInt})
	}
	sort.Sort(ByName(results))
	for _, v := range results {
		table.Append(v)
	}
	table.Render() // Send output
}
