package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/tools/go/loader"
)

var (
	stdlib = flag.Bool("stdlib", false, "Thread stdlib packages as external dependencies")
)

func main() {
	flag.Usage = Usage
	flag.Parse()

	var conf loader.Config

	conf.FromArgs(flag.Args(), false)
	p, err := conf.Load()
	if err != nil {
		fmt.Println(err)
		return
	}

	w := NewWalker(p, *stdlib)

	result := w.TopWalk()

	result.PrintPretty()
}

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] <args>\n\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s\n", loader.FromArgsUsage)
}
