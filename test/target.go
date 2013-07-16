package test

import (
	gobuild "go/build"
	"path/filepath"

	"github.com/davecheney/gogo/build"
)

// target implements a build.Future
type target struct {
	err chan error
	*gobuild.Package
	*build.Context
}

func (t *target) Result() error {
	result := <-t.err
	t.err <- result
	return result
}

func newTarget(ctx *build.Context, pkg *gobuild.Package) target {
	return target{
		err:     make(chan error, 1),
		Context: ctx,
		Package: pkg,
	}
}

func (t *target) Srcdir() string {
	return filepath.Join(t.SrcRoot, t.ImportPath)
}
