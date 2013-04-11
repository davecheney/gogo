package build

import (
	"fmt"
	"log"
	"sync"

	"github.com/davecheney/gogo"
)

var Build = &gogo.Command{
	Run: run,
}

func run(project *gogo.Project, args []string) error {
	var pkgs []*gogo.Package
	for _, arg := range args {
		pkg, err := project.ResolvePackage(arg)
		if err != nil {
			return fmt.Errorf("failed to resolve package %q: %v", arg, err)
		}
		pkgs = append(pkgs, pkg)
	}
        ctx, err := project.NewContext()
        if err != nil {
                return err
        }
        for _, pkg := range pkgs {
                var tt []gogo.Target
                log.Printf("building: %v", pkg.ImportPath())
                if pkg.Name() == "main" {
                        tt = buildCommand(ctx, pkg)
                } else {
                        tt = buildPackage(ctx, pkg)
                }
                for _, t := range tt {
                        if err := t.Wait(); err != nil {
                                return err
                        }
                }
        }
        return nil
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
	deps []gogo.Target
	*gogo.Package
	*gogo.Context
}

func newPackTarget(ctx *gogo.Context, pkg *gogo.Package, deps ...gogo.Target) *packTarget {
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
	return t.Project.Toolchain().Pack(t.Context, t.Package)
}

type gcTarget struct {
	baseTarget
	deps []gogo.Target
	*gogo.Package
	*gogo.Context
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

func newGcTarget(ctx *gogo.Context, pkg *gogo.Package, deps ...gogo.Target) *gcTarget {
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
	return t.Project.Toolchain().Gc(t.Context, t.Package)
}

type ldTarget struct {
	baseTarget
	deps []gogo.Target
	*gogo.Package
	*gogo.Context
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

func newLdTarget(ctx *gogo.Context, pkg *gogo.Package, deps ...gogo.Target) *ldTarget {
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
	return t.Project.Toolchain().Ld(t.Context, t.Package)
}

func buildPackage(ctx *gogo.Context, pkg *gogo.Package) []gogo.Target {
	var deps []gogo.Target
	for _, dep := range pkg.Imports() {
		deps = append(deps, buildPackage(ctx, dep)...)
	}
	if _, ok := ctx.Targets[pkg]; !ok {
		// gc target
		gc := newGcTarget(ctx, pkg, deps...)
		go gc.execute()
		pack := newPackTarget(ctx, pkg, gc)
		go pack.execute()
		ctx.Targets[pkg] = pack
	}
	return []gogo.Target{ctx.Targets[pkg]}
}

func buildCommand(ctx *gogo.Context, pkg *gogo.Package) []gogo.Target {
	var deps []gogo.Target
	for _, dep := range pkg.Imports() {
		deps = append(deps, buildPackage(ctx, dep)...)
	}
	if _, ok := ctx.Targets[pkg]; !ok {
		// gc target
		gc := newGcTarget(ctx, pkg, deps...)
		go gc.execute()
		pack := newPackTarget(ctx, pkg, gc)
		go pack.execute()
		ld := newLdTarget(ctx, pkg, pack)
		go ld.execute()
		ctx.Targets[pkg] = ld
	}
	return []gogo.Target{ctx.Targets[pkg]}
}
