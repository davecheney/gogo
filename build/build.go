package build

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	return nil // t.Pack(t.Context, t.Package)
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

func (t *gcTarget) objdir() string { return t.Context.Objdir(t.Package) }

func (t *gcTarget) build() error {
	gofiles := t.GoFiles()
	if len(gofiles) < 1 {
		return nil
	}
	objdir := t.objdir()
	if err := os.MkdirAll(objdir, 0777); err != nil {
		return err
	}
	return t.Gc(t.ImportPath(), t.Srcdir(), filepath.Join(objdir, "_go_.6"), gofiles)
}

type asmTarget struct {
	baseTarget
	deps []gogo.Target
	*gogo.Package
	*gogo.Context
}

func (t *asmTarget) execute() {
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

func newAsmTarget(ctx *gogo.Context, pkg *gogo.Package, deps ...gogo.Target) *gcTarget {
	return &gcTarget{
		baseTarget: baseTarget{
			done: make(chan struct{}),
		},
		deps:    deps,
		Package: pkg,
		Context: ctx,
	}
}

func (t *asmTarget) build() error {
	return nil // t.Project.Toolchain().Asm(t.Context, t.Package)
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

func (t *ldTarget) objdir() string  { return t.Context.Objdir(t.Package) }
func (t *ldTarget) pkgfile() string { return t.Package.Pkgfile(t.Context) }

func (t *ldTarget) build() error {
	objdir := t.objdir()
	if err := os.MkdirAll(objdir, 0777); err != nil {
		return err
	}
	return t.Ld(filepath.Join(objdir, "a.out"), t.pkgfile())
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
	log.Printf("build package %q", pkg.ImportPath())
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
	log.Printf("build command %q", pkg.ImportPath())
	return []gogo.Target{ctx.Targets[pkg]}
}
