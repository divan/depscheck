package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/tools/go/loader"
)

var (
	stdlib   = flag.Bool("stdlib", false, "Treat stdlib packages as external dependencies")
	tests    = flag.Bool("tests", false, "Include tests for deps analysis")
	verbose  = flag.Bool("v", false, "Be verbose and print whole deps info table")
	totals   = flag.Bool("totalonly", false, "Print only totals stats")
	internal = flag.Bool("internal", false, "Include intertanl packages analysis")
)

func main() {
	flag.Usage = Usage
	flag.Parse()

	var conf loader.Config

	conf.FromArgs(flag.Args(), *tests)
	p, err := conf.Load()
	if err != nil {
		fmt.Println(err)
		return
	}

	w := NewWalker(p, *stdlib, *internal)

	result := w.TopWalk()

	// Output results
	topPackage := p.InitialPackages()[0].Pkg.Path()
	fmt.Println(result.Totals(topPackage))
	if *totals {
		return
	}
	if len(result.Counter) == 0 {
		fmt.Println("No external dependencies found in this package")
		return
	}
	if *verbose {
		result.PrintStats()
		result.PrintPackagesStats()
	}

	// Do not report suggestions in stdlib mode.
	// Stlib is smarter than this tool.
	if !*stdlib {
		result.Suggestions()
	}

	if !*verbose {
		fmt.Println("Run with -v option to see detailed stats for dependencies.")
	}
}

// Usage prints usage information for this program.
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] <args>\n\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s\n", loader.FromArgsUsage)
}
