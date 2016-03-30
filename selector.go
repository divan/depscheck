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

func (sel *Selector) LOCCum() int {
	if !sel.IsFunc() {
		return 0
	}

	ret := sel.LOC
	for _, dep := range sel.Deps {
		ret += dep.LOCCum()
	}

	return ret
}

func (sel *Selector) Depth() int {
	if !sel.IsFunc() {
		return 0
	}

	ret := 0
	for _, dep := range sel.Deps {
		if dep.Pkg != sel.Pkg {
			ret++
			ret += dep.Depth()
		}
	}

	return ret
}

func (sel *Selector) DepthInternal() int {
	if !sel.IsFunc() {
		return 0
	}

	ret := 0
	for _, dep := range sel.Deps {
		if dep.Pkg == sel.Pkg {
			ret++
			ret += dep.DepthInternal()
		}
	}

	return ret
}

func (sel *Selector) IsFunc() bool {
	return sel.Type == "func" || sel.Type == "method"
}

func (sel *Selector) PrintDeps() {
	sel.printDeps(0)
}

func (sel *Selector) printDeps(depth int) {
	fmt.Println(strings.Repeat("  ", depth), sel.Pkg.Name+"."+sel.Name)
	for _, dep := range sel.Deps {
		dep.printDeps(depth + 2)
	}
}

type ByID []*Selector

func (b ByID) Len() int      { return len(b) }
func (b ByID) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByID) Less(i, j int) bool {
	return b[i].ID() < b[j].ID()
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
