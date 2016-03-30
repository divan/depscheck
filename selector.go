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

	Deps Deps
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

type Deps []*Selector

func (deps *Deps) Append(s *Selector) {
	for _, d := range *deps {
		if d.ID() == s.ID() {
			return
		}
	}
	*deps = append(*deps, s)
}

func (deps Deps) HasRecursion(sel *Selector) bool {
	for _, dep := range deps {
		if dep.ID() == sel.ID() {
			return true
		}

		if dep.Deps != nil {
			return dep.Deps.HasRecursion(sel)
		}
	}
	return false
}
