package main

import (
	"fmt"
	"go/types"
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
	LOC int // actual Lines Of Code

	Deps []*Selector
}

// String implements Stringer interface for Selector.
func (s *Selector) String() string {
	var out string
	if s.Recv != "" {
		out = fmt.Sprintf("%s.(%s).%s.%s", s.Pkg.Name, s.Recv, s.Type, s.Name)
	}
	out = fmt.Sprintf("%s.%s.%s", s.Pkg.Name, s.Type, s.Name)

	if s.Type == "func" || s.Type == "method" {
		out = fmt.Sprintf("%s LOC: %d, %d, Depth: %d,%d", out, s.LOC, s.LOCCum(), s.Depth(), s.DepthInternal())
	}

	return out
}

// ID generates uniqie string ID for this selector.
func (s *Selector) ID() string {
	if s.Recv != "" {
		return fmt.Sprintf("%s.(%s).%s.%s", s.Pkg.Name, s.Recv, s.Type, s.Name)
	}
	return fmt.Sprintf("%s.%s.%s", s.Pkg.Name, s.Type, s.Name)
}

// NewSelector creates new Selector.
func NewSelector(pkg *types.Package, name, recv, typ string, loc int) *Selector {
	return &Selector{
		Pkg: Package{
			Name: pkg.Name(),
			Path: pkg.Path(),
		},
		Name: name,

		Recv: recv,
		Type: typ,

		LOC: loc,
	}
}

func (sel *Selector) Append(s *Selector) {
	for _, d := range sel.Deps {
		if d.ID() == s.ID() {
			return
		}
	}
	sel.Deps = append(sel.Deps, s)
}
