package gogo

import (
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

type packTarget struct {
	baseTarget
	deps []Target
	*Package
	*Context
}

func newPackTarget(ctx *Context, pkg *Package, deps ...Target) *packTarget {
         return &packTarget{   
                        baseTarget: baseTarget{
                                done: make(chan struct{}),
                        },
                        deps:    deps,
                        Package: pkg,
                        Context: ctx,
                }
}

func (t *packTarget) execute() {
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

func (t *packTarget) build() error {
	return t.Project.Toolchain().pack(t.Context, t.Package)
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

func newGcTarget(ctx *Context, pkg *Package, deps ...Target) *gcTarget {
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

type ldTarget struct {
        baseTarget
        deps []Target
        *Package
        *Context
}

func (t *ldTarget) execute() {
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

func newLdTarget(ctx *Context, pkg *Package, deps ...Target) *ldTarget {
        return &ldTarget{
                baseTarget: baseTarget{
                        done: make(chan struct{}),
                },
                deps:    deps,
                Package: pkg,
                Context: ctx,
        }
}

func (t *ldTarget) build() error {
        return t.Project.Toolchain().ld(t.Context, t.Package)
}

func buildPackage(ctx *Context, pkg *Package) []Target {
	var deps []Target
	for _, dep := range pkg.Imports() {
		deps = append(deps, buildPackage(ctx, dep)...)
	}
	if _, ok := ctx.targets[pkg]; !ok {
		// gc target
		gc := newGcTarget(ctx, pkg, deps...)
		go gc.execute()
		pack := newPackTarget(ctx, pkg, gc)
		go pack.execute()
		ctx.targets[pkg] = pack
	}
	return []Target{ctx.targets[pkg]}
}

func buildCommand(ctx *Context, pkg *Package) []Target {
	var deps []Target
	for _, dep := range pkg.Imports() {
		deps = append(deps, buildPackage(ctx, dep)...)
	}
	if _, ok := ctx.targets[pkg]; !ok {
		// gc target
		gc := newGcTarget(ctx, pkg, deps...)
		go gc.execute()
		pack := newPackTarget(ctx, pkg, gc)
		go pack.execute()
		ld := newLdTarget(ctx, pkg, pack)
		go ld.execute()
		ctx.targets[pkg] = ld
	}
	return []Target{ctx.targets[pkg]}
}
