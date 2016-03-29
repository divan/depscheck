package test

import (
	sss "github.com/divan/depscheck/test/sample"
)

func Xtest() {
	sss.SampleFunc()
	var foo sss.Foo
	foo.Bar()
}
