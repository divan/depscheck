package main

import (
	"strings"
)

func IsInternal(pkg, subpkg string) bool {
//	fmt.Printf("\n\n%s\n%s\n-----------\n", pkg, subpkg)
	//Skip if any is stdlib
	if IsStdlib(pkg) || IsStdlib(subpkg) {
		return false
	}

	// Or it is submodule
	if strings.HasPrefix(subpkg, pkg) {
		return true
	}

	// Or it is on same nesting level
	if i := strings.LastIndex(pkg, "/"); i > 0 {
//		fmt.Printf("%s\n==================\n\n", pkg[0:i])
		if strings.HasPrefix(subpkg, pkg[0:i]) {
			return true
		}
	}
	return false
}
