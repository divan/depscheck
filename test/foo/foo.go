package foo

import "github.com/divan/depscheck/test/bar"

const FooConst = 42

var FooVar = "42"

type Fooer interface {
	Foo(int)
}

func Foo(x int) {
	if x == 2 {
		bar.Bar(3)
	}
}
