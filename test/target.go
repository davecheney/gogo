package test

import (
	"github.com/davecheney/gogo/build"
	"github.com/davecheney/gogo/project"
)

// target implements a build.Future
type target struct {
	err chan error
	*project.Package
	*build.Context
}

func (t *target) Result() error {
	result := <-t.err
	t.err <- result
	return result
}

func newTarget(ctx *build.Context, pkg *project.Package) target {
	return target{
		err:     make(chan error, 1),
		Context: ctx,
		Package: pkg,
	}
}
