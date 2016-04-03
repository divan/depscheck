package main

import (
	"testing"
)

func TestPackageChecks(t *testing.T) {
	var pkg, subpkg string

	checkResult := func(pkg, subpkg string, want bool) {
		got := IsInternal(pkg, subpkg)
		if got != want {
			t.Fatalf("Expecting IsInternal to return %v in this case: (%s, %s)", want, pkg, subpkg)
		}
	}

	pkg, subpkg = "github.com/divan/depscheck", "github.com/divan/depscheck/foo"
	checkResult(pkg, subpkg, true)
	pkg, subpkg = "github.com/divan/depscheck/bar", "github.com/divan/depscheck/foo"
	checkResult(pkg, subpkg, true)
	pkg, subpkg = "github.com/divan/package1", "github.com/divan/package2"
	checkResult(pkg, subpkg, false)
}
