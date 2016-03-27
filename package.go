package main

// Package represents package info, needed for this tool.
type Package struct {
	Name string
	Path string
}

// NewPackage creates new Package.
func NewPackage(name, path string) Package {
	return Package{
		Name: name,
		Path: path,
	}
}
