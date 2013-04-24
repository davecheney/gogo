// gogo is an alternative build tool for the Go programming language.
//
// Package gogo provides the basic types and interfaces to describe Projects and
// Packages.
//
// The gogo command can be installed with go get
//
//	go get github.com/davecheney/gogo
package gogo

// A Future represents some work to be performed.
type Future interface {
	// Result returns the result of the work as an error, or nil if the work
	// was performed successfully.
	// Implementers must observe these invariants
	// 1. There may be multiple concurrent callers to Result, or Result may
	//     be called many times in sequence, it must always return the same
	//     value.
	// 2. Result blocks until the work has been performed.
	Result() error
}
