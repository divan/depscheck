package main

import (
	"golang.org/x/tools/go/loader"
	"testing"
)

func TestExportedFuncs(t *testing.T) {
	var result *Result
	var src string

	src = "test/exported.go"
	result = getResult(t, "test", src)
	checkCount(src, t, result, 2)
	checkSelector(src, t, result, "xsample.var.Sample", 1, 0, 0, 0, 0)
	checkSelector(src, t, result, "xsample.func.SampleFunc", 1, 6, 14, 0, 2)

	src = "test/exported2.go"
	result = getResult(t, "test", src)
	checkCount(src, t, result, 3)
	checkSelector(src, t, result, "xsample.func.SampleFunc", 1, 6, 14, 0, 2)
	checkSelector(src, t, result, "xsample.(Foo).method.Bar", 1, 3, 3, 0, 0)
	checkSelector(src, t, result, "xsample.type.Foo", 1, 0, 0, 0, 0)

	src = "test/pkg_renamed.go"
	result = getResult(t, "test", src)
	checkCount(src, t, result, 3)
	checkSelector(src, t, result, "xsample.func.SampleFunc", 1, 6, 14, 0, 2)
	checkSelector(src, t, result, "xsample.(Foo).method.Bar", 1, 3, 3, 0, 0)
	checkSelector(src, t, result, "xsample.type.Foo", 1, 0, 0, 0, 0)

	src = "test/pkg_dot.go"
	result = getResult(t, "test", src)
	checkCount(src, t, result, 3)
	checkSelector(src, t, result, "xsample.func.SampleFunc", 1, 6, 14, 0, 2)
	checkSelector(src, t, result, "xsample.(Foo).method.Bar", 1, 3, 3, 0, 0)
	checkSelector(src, t, result, "xsample.type.Foo", 1, 0, 0, 0, 0)

	src = "test/recursion.go"
	result = getResult(t, "test", src)
	checkCount(src, t, result, 2)
	checkSelector(src, t, result, "bar.func.Bar", 1, 4, 4, 0, 0)
	checkSelector(src, t, result, "foo.func.Foo", 1, 4, 8, 1, 0)
}

func TestConsts(t *testing.T) {
	var result *Result
	var src string

	src = "test/const.go"
	result = getResult(t, "test", src)
	checkCount(src, t, result, 1)
	checkSelector(src, t, result, "foo.const.FooConst", 1, 0, 0, 0, 0)
}

func TestVars(t *testing.T) {
	var result *Result
	var src string

	src = "test/var.go"
	result = getResult(t, "test", src)
	checkCount(src, t, result, 1)
	checkSelector(src, t, result, "foo.var.FooVar", 1, 0, 0, 0, 0)
}

func TestInterface(t *testing.T) {
	var result *Result
	var src string

	src = "test/interface.go"
	result = getResult(t, "test", src)
	checkCount(src, t, result, 2)
	checkSelector(src, t, result, "foo.(Fooer).method.Foo", 1, 0, 0, 0, 0)
	checkSelector(src, t, result, "foo.interface.Fooer", 1, 0, 0, 0, 0)
}

func getResult(t *testing.T, name string, sources ...string) *Result {
	var conf loader.Config
	conf.CreateFromFilenames(name, sources...)
	p, err := conf.Load()
	if err != nil {
		t.Fatal(err)
	}

	w := NewWalker(p, false)
	return w.TopWalk()
}

func checkCount(src string, t *testing.T, r *Result, want int) {
	if have := len(r.Counter); have != want {
		t.Fatalf("%s: expected to have %d selectors, but have %d", src, want, have)
	}

}

func checkSelector(src string, t *testing.T, r *Result, fn string, count, loc, loccum, depth, depthint int) {
	sel, ok := r.Selectors[fn]
	if !ok {
		t.Fatalf("%s: expected to see func '%s' in result, but could not", src, fn)
	}
	if r.Counter[fn] != count {
		t.Fatalf("%s: expected to func '%s' to have Count %d , but got %d", src, fn, count, r.Counter[fn])
	}
	if sel.LOC != loc {
		t.Fatalf("%s: expected to func '%s' to have %d LOC, but got %d", src, fn, loc, sel.LOC)
	}
	if sel.LOCCum() != loccum {
		t.Fatalf("%s: expected to func '%s' to have %d Cumulative LOC, but got %d", src, fn, loccum, sel.LOCCum())
	}
	if sel.Depth() != depth {
		t.Fatalf("%s: expected to func '%s' to have Depth %d, but got %d", src, fn, depth, sel.Depth())
	}
	if sel.DepthInternal() != depthint {
		t.Fatalf("%s: expected to func '%s' to have %d Depth Internal, but got %d", src, fn, depthint, sel.DepthInternal())
	}
}
