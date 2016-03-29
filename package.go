package main

import (
	"fmt"
)

// Package represents package info, needed for this tool.
type Package struct {
	Name string
	Path string
}

// PackageStat holds stats about dependencies in a given package.
type PackageStat struct {
	*Package

	DepsCount      int
	DepsCallsCount int

	LOCCum               int
	Depth, DepthInternal int
}

// NewPackage creates new Package.
func NewPackage(name, path string) Package {
	return Package{
		Name: name,
		Path: path,
	}
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
