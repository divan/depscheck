package main

import "github.com/divan/depscheck/test/foo"

func Foo(foo foo.Fooer) {
	foo.Foo(42)
}
