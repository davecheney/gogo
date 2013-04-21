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

// from $GOROOT/src/pkg/go/build/syslist.go

const goosList = "darwin freebsd linux netbsd openbsd plan9 windows "
const goarchList = "386 amd64 arm "
