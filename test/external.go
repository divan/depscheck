package main

import (
	"fmt"
	"github.com/divan/depscheck/test/foo"
	"golang.org/x/tools/go/loader"
)

func main() {
	x := foo.FooConst
	var l loader.Config
	fmt.Println(x, l)
}
