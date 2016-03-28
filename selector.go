package main

import (
	"fmt"
	"golang.org/x/tools/go/loader"
)

// Selector represents Go language selector (x.f),
// which may be:
// - method of variable of external package
// - function from the external package
// - variable/const from ext. package
type Selector struct {
	Pkg  Package
	Name string
	Type string
	Recv string

	// Applies for functions
	LOC           int // actual Lines Of Code
	LOCCum        int // cumulative Lines Of Code
	Depth         int // depth of nested external functions calls
	DepthInternal int // depth of nested internal functions calls
}

// String implements Stringer interface for Selector.
func (s *Selector) String() string {
	if s.Recv != "" {
		return fmt.Sprintf("%s.(%s).%s", s.Pkg.Name, s.Recv, s.Name)
	}
	return fmt.Sprintf("%s.%s", s.Pkg.Name, s.Name)
}

// NewSelector creates new Selector.
func NewSelector(pkg *loader.PackageInfo, name string) *Selector {
	return &Selector{
		Pkg: Package{
			Name: pkg.Pkg.Name(),
			Path: pkg.Pkg.Path(),
		},
		Name: name,
	}
}
