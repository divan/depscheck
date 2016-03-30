package xsample

type Foo struct{}

var Sample = 123

func SampleFunc() int {
	x := 12
	x++
	y := 5
	Xfunc()
	return y + x
}

func Xfunc() {
	y := 2
	y += 22
	y++
	YFunc()
}

func YFunc() {
	x := 12
	_ = x
}

func (s Foo) Bar() {
	x := 42
	_ = x
}
