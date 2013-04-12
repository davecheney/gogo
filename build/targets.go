package build

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/davecheney/gogo"
)

type target struct {
	done chan struct{}
	err  struct {
		sync.Mutex
		val error
	}
}

func (t *target) Wait() error {
	<-t.done
	t.err.Lock()
	defer t.err.Unlock()
	return t.err.val
}

func (t *target) setErr(err error) {
	t.err.Lock()
	t.err.val = err
	t.err.Unlock()
}

type packTarget struct {
	target
	deps []gogo.Target
	*gogo.Package
	*gogo.Context
}

func newPackTarget(ctx *gogo.Context, pkg *gogo.Package, deps ...gogo.Target) *packTarget {
	return &packTarget{
		target: target{
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
			t.setErr(err)
			return
		}
	}
	if err := t.build(); err != nil {
		t.setErr(err)
	}
}

func (t *packTarget) objdir() string  { return t.Context.Objdir(t.Package) }
func (t *packTarget) objfile() string { return filepath.Join(t.objdir(), "_go_.6") }
func (t *packTarget) pkgfile() string { return t.Package.ImportPath()+".a" }

func (t *packTarget) build() error {
	ofile := t.pkgfile()
	pkgdir := filepath.Dir(filepath.Join(t.Pkgdir(), ofile))
	if err := os.MkdirAll(pkgdir, 0777); err != nil {
		return err
	}
	return t.Pack(ofile, t.Pkgdir(), t.objfile())
}

type gcTarget struct {
	target
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
		target: target{
			done: make(chan struct{}),
		},
		deps:    deps,
		Package: pkg,
		Context: ctx,
	}
}

func (t *gcTarget) objdir() string  { return t.Context.Objdir(t.Package) }
func (t *gcTarget) objfile() string { return filepath.Join(t.objdir(), "_go_.6") }

func (t *gcTarget) build() error {
	gofiles := t.GoFiles
	objdir := t.objdir()
	if err := os.MkdirAll(objdir, 0777); err != nil {
		return err
	}
	return t.Gc(t.ImportPath(), t.Srcdir(), t.objfile(), gofiles)
}

type asmTarget struct {
	target
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
		target: target{
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
	target
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
		target: target{
			done: make(chan struct{}),
		},
		deps:    deps,
		Package: pkg,
		Context: ctx,
	}
}

func (t *ldTarget) objdir() string  { return t.Context.Objdir(t.Package) }
func (t *ldTarget) pkgfile() string { return filepath.Join(t.Pkgdir(), t.Package.ImportPath()+".a") }

func (t *ldTarget) build() error {
	objdir := t.objdir()
	if err := os.MkdirAll(objdir, 0777); err != nil {
		return err
	}
	return t.Ld(filepath.Join(objdir, "a.out"), t.pkgfile())
}
