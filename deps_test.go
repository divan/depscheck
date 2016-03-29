package main

import (
	"golang.org/x/tools/go/loader"
	"testing"
)

func TestExportedFuncs(t *testing.T) {
	var result *Result

	result = getResult(t, "test", "test/test_exported.go")
	checkCount(t, result, 1)
	checkSelector(t, result, "xsample.func.SampleFunc", 1, 6, 14, 0, 2)

	result = getResult(t, "test", "test/test_exported2.go")
	checkCount(t, result, 2)
	checkSelector(t, result, "xsample.func.SampleFunc", 1, 6, 14, 0, 2)
	checkSelector(t, result, "xsample.(Foo).method.Bar", 1, 3, 3, 0, 0)
}

func getResult(t *testing.T, name string, sources ...string) *Result {
	var conf loader.Config
	conf.CreateFromFilenames(name, sources...)
	p, err := conf.Load()
	if err != nil {
		t.Fatal(err)
	}

	w := NewWalker(p)
	return w.TopWalk()
}

func checkCount(t *testing.T, r *Result, want int) {
	if have := len(r.Counter); have != want {
		t.Fatalf("Expected to have %d selectors, but have %d", want, have)
	}

}

func checkSelector(t *testing.T, r *Result, fn string, count, loc, loccum, depth, depthint int) {
	sel, ok := r.Selectors[fn]
	if !ok {
		t.Fatalf("Expected to see func '%s' in result, but could not", fn)
	}
	if r.Counter[fn] != count {
		t.Fatalf("Expected to func '%s' to have Count %d , but got %d", fn, count, r.Counter[fn])
	}
	if sel.LOC != loc {
		t.Fatalf("Expected to func '%s' to have %d LOC, but got %d", fn, loc, sel.LOC)
	}
	if sel.LOCCum != loccum {
		t.Fatalf("Expected to func '%s' to have %d Cumulative LOC, but got %d", fn, loccum, sel.LOCCum)
	}
	if sel.Depth != depth {
		t.Fatalf("Expected to func '%s' to have Depth %d, but got %d", fn, depth, sel.Depth)
	}
	if sel.DepthInternal != depthint {
		t.Fatalf("Expected to func '%s' to have %d Depth Internal, but got %d", fn, depthint, sel.DepthInternal)
	}
}
