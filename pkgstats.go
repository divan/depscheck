package main

import (
	"fmt"
	"sort"
)

// PackageStat holds stats about dependencies in a given package.
type PackageStat struct {
	*Package

	DepsCount      int
	DepsCallsCount int

	LOCCum               int
	Depth, DepthInternal int
}

// NewPackageStat creates new PackageStat.
func NewPackageStat(pkg Package) *PackageStat {
	return &PackageStat{
		Package: &pkg,
	}
}

// String implements Stringer for PackageStat.
func (p *PackageStat) String() string {
	return fmt.Sprintf("%s: (%d, %d) [LOC: %d] Depth [%d, %d]\n", p.Path, p.DepsCount, p.DepsCallsCount, p.LOCCum, p.Depth, p.DepthInternal)
}

// PackagesStats returns stats by packages in all selectors.
func (r *Result) PackagesStats() []*PackageStat {
	pkgs := make(map[Package]*PackageStat)
	for _, sel := range r.All() {
		if _, ok := pkgs[sel.Pkg]; !ok {
			pkgs[sel.Pkg] = NewPackageStat(sel.Pkg)
		}
		pkgs[sel.Pkg].DepsCount++
		pkgs[sel.Pkg].DepsCallsCount += r.Counter[sel.ID()]
		pkgs[sel.Pkg].LOCCum += sel.LOCCum()
		pkgs[sel.Pkg].Depth += sel.Depth()
		pkgs[sel.Pkg].DepthInternal += sel.DepthInternal()

	}

	var ret []*PackageStat
	for _, stat := range pkgs {
		ret = append(ret, stat)
	}
	sort.Sort(ByPackageName(ret))
	return ret
}

// CanBeAvoided attempts to classify if package usage is small enough
// to suggest user to avoid this package as a dependency and
// instead copy/embed it's code into own project (if license permits).
func (p *PackageStat) CanBeAvoided() bool {
	// If this dependency is using another dependencies,
	// it's almost for sure - no. For internal dependency, let's
	// allow just two level of nesting.
	if p.Depth > 0 {
		return false
	}
	if p.DepthInternal > 2 {
		return false
	}

	if p.DepsCount > 3 {
		return false
	}

	// Because 42
	if p.LOCCum > 42 {
		return false
	}

	return true
}

// ByPackageName is a helper type for sorting PackageStats by Name.
type ByPackageName []*PackageStat

func (b ByPackageName) Len() int      { return len(b) }
func (b ByPackageName) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByPackageName) Less(i, j int) bool {
	return b[i].Name < b[j].Name
}
