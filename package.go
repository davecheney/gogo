package gogo

import (
	"io"
	"os"
	"path/filepath"
)

// Package describes a Go package.
// The contents of a Package will be influenced by the Context from which
// they are resolved.
type Package struct {
	// Name returns the name of the package.
	Name string

	// ImportPath represents the import path that would is used to import this package into another.
	ImportPath string

	// Srcdir returns the path to this package.
	Srcdir string

	// Source files
	GoFiles        []string // .go source files (excluding CgoFiles, TestGoFiles, XTestGoFiles)
	CgoFiles       []string // .go source files that import "C"
	SFiles         []string // .s source files
	CFiles         []string // .c source files
	HFiles         []string // .h c header files
	IgnoredGoFiles []string // .go source files ignored for this build

	// Cgo directives
	CgoPkgConfig []string // Cgo pkg-config directives
	CgoCFLAGS    []string // Cgo CFLAGS directives
	CgoLDFLAGS   []string // Cgo LDFLAGS directives

	// Test information
	TestGoFiles  []string // _test.go files in package
	XTestGoFiles []string // _test.go files outside package

	Imports      []string
	TestImports  []string
	XTestImports []string
}

func (p *Package) openFile(name string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(p.Srcdir, name))
}
