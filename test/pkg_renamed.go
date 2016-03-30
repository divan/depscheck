package test

import (
	x "github.com/divan/depscheck/test/sample"
)

func Xtest() {
	x.SampleFunc()
	var foo x.Foo
	foo.Bar()
}
