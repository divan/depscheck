package bar

func Bar(x int) {
	if x == 3 {
		Foo(2)
	}
}

func Foo(x int) {
	if x == 3 {
		Bar(4)
	}
}
