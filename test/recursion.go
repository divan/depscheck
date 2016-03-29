package main

import (
	"github.com/divan/depscheck/test/bar"
	"github.com/divan/depscheck/test/foo"
)

func Foo(x int) {
	if x == 2 {
		bar.Bar(3)
	}
}

func Bar(x int) {
	if x == 2 {
		foo.Foo(3)
	}
}
