package main

import (
	"fmt"
)

// Package represents package info, needed for this tool.
type Package struct {
	Name string
	Path string
}

// Selector represents Go language selector (x.f),
// which may be:
// - method of variable of external package
// - function from the external package
// - variable/const from ext. package
type Selector struct {
	Pkg  Package
	Name string
	Type string

	// Applies for functions
	LOC           int // actual Lines Of Code
	LOCCum        int // cumulative Lines Of Code
	Depth         int // depth of nested external functions calls
	DepthInternal int // depth of nested internal functions calls
}

// String implements Stringer interface for Selector.
func (s *Selector) String() string {
	return fmt.Sprintf("%s.%s", s.Pkg.Name, s.Name)
}
