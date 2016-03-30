package test

import (
	"fmt"
	"github.com/divan/depscheck/test/sample"
	"math"
	"strings"
)

type Test struct {
	X string
	Y int
	Z bool
}

func Xtest() {
	t := &Test{
		Y: xsample.Sample + xsample.SampleFunc(),
	}
	_ = math.Pi
	if strings.HasPrefix("test", "t") {
		fmt.Println("OK")
	}
	_ = t.X
	fmt.Println(math.Max(1, 2))
	go func() {
		_ = math.Min(1, 2)
	}()
}
