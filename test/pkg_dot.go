package test

import (
	. "github.com/divan/depscheck/test/sample"
)

func Xtest() {
	SampleFunc()
	var foo Foo
	foo.Bar()
}
