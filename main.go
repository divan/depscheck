package main

import (
	"fmt"
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

	result := w.TopWalk()

	result.PrintPretty()
}
