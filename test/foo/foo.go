package foo

import "github.com/divan/depscheck/test/bar"

func Foo(x int) {
	if x == 2 {
		bar.Bar(3)
	}
}
