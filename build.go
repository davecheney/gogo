package gogo

import (
	"log"
	"sync"
)

type Target interface {
	Wait() error
}

type baseTarget struct {
	done chan struct{}
	err  struct {
		sync.Mutex
		val error
	}
}

func (t *baseTarget) Wait() error {
	<-t.done
	t.err.Lock()
	defer t.err.Unlock()
	return t.err.val
}

type buildPackageTarget struct {
	baseTarget
	deps []Target
	*Package
	*Context
}

func (t *buildPackageTarget) execute() {
	defer close(t.done)
	for _, dep := range t.deps {
		if err := dep.Wait(); err != nil {
			t.err.Lock()
			t.err.val = err
			t.err.Unlock()
			return
		}
	}
	if err := t.build(); err != nil {
		t.err.Lock()
		t.err.val = err
		t.err.Unlock()
	}
}

func (t *buildPackageTarget) build() error {
	log.Printf("%T %q", t, t.Package.Path())
	return nil
}

type gcTarget struct {
	baseTarget
	deps []Target
	*Package
	*Context
}

func (t *gcTarget) execute() {
	defer close(t.done)
	for _, dep := range t.deps {
		if err := dep.Wait(); err != nil {
			t.err.Lock()
			t.err.val = err
			t.err.Unlock()
			return
		}
	}
	if err := t.build(); err != nil {
		t.err.Lock()
		t.err.val = err
		t.err.Unlock()
	}
}

func newGcTarget(ctx *Context, pkg *Package, deps []Target) *gcTarget {
	return &gcTarget{
		baseTarget: baseTarget{
			done: make(chan struct{}),
		},
		deps:    deps,
		Package: pkg,
		Context: ctx,
	}
}

func (t *gcTarget) build() error {
	return t.Project.Toolchain().gc(t.Context, t.Package)
}

func buildPackage(ctx *Context, pkg *Package) []Target {
	var deps []Target
	for _, dep := range pkg.Imports() {
		deps = append(deps, buildPackage(ctx, dep)...)
	}
	if _, ok := ctx.targets[pkg]; !ok {
		// gc target
		gc := newGcTarget(ctx, pkg, deps)
		go gc.execute()
		t := &buildPackageTarget{
			baseTarget: baseTarget{
				done: make(chan struct{}),
			},
			deps:    []Target{gc},
			Package: pkg,
			Context: ctx,
		}
		go t.execute()
		ctx.targets[pkg] = t
	}
	return []Target{ctx.targets[pkg]}
}
