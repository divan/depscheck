package main

import (
	"fmt"
	"go/types"
	"strings"
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

// LOCCum returns cumulative LOC count for Selector and all it's dependencies.
func (s *Selector) LOCCum() int {
	if !s.IsFunc() {
		return 0
	}

	ret := s.LOC
	for _, dep := range s.Deps {
		ret += dep.LOCCum()
	}

	return ret
}

// Depth returns Depth for Selector and all it's external dependencies.
func (s *Selector) Depth() int {
	if !s.IsFunc() {
		return 0
	}

	ret := 0
	for _, dep := range s.Deps {
		if dep.Pkg != s.Pkg {
			ret++
			ret += dep.Depth()
		}
	}

	return ret
}

// DepthInternal returns Depth for Selector and all it's internal dependencies.
func (s *Selector) DepthInternal() int {
	if !s.IsFunc() {
		return 0
	}

	ret := 0
	for _, dep := range s.Deps {
		if dep.Pkg == s.Pkg {
			ret++
			ret += dep.DepthInternal()
		}
	}

	return ret
}

// IsFunc returns true if Selector is either a function or a method.
func (s *Selector) IsFunc() bool {
	return s.Type == "func" || s.Type == "method"
}

// PrintDeps recursively prints deps for selector.
func (s *Selector) PrintDeps() {
	s.printDeps(0)
}

func (s *Selector) printDeps(depth int) {
	fmt.Println(strings.Repeat("  ", depth), fmt.Sprintf("%s.%s", s.Pkg.Name, s.Name))
	for _, dep := range s.Deps {
		dep.printDeps(depth + 1)
	}
}

// ByID is helper type for sorting selectors by ID.
type ByID []*Selector

func (b ByID) Len() int      { return len(b) }
func (b ByID) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByID) Less(i, j int) bool {
	return b[i].ID() < b[j].ID()
}

// Deps is a shorthand for Dependencies - a slice of Selectors.
type Deps []*Selector

// Append adds new Selector to Deps.
func (deps *Deps) Append(s *Selector) {
	for _, d := range *deps {
		if d.ID() == s.ID() {
			return
		}
	}
	*deps = append(*deps, s)
}

// HasRecursion attempts to find selector in nested dependencies
// to avoid recursion.
func (deps Deps) HasRecursion(s *Selector) bool {
	for _, dep := range deps {
		if dep.ID() == s.ID() {
			return true
		}

		if dep.Deps != nil {
			has := dep.Deps.HasRecursion(s)
			if has {
				return true
			}
		}
	}
	return false
}
