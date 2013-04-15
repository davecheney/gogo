package gogo

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Test(ctx *Context, pkg *Package) []Target {
	// commands are built as packages for testing.
	return testPackage(ctx, pkg)
}

func testPackage(ctx *Context, pkg *Package) []Target {
	// build dependencies
	var deps []Target
	for _, dep := range pkg.Imports {
		deps = append(deps, Build(ctx, dep)...)
	}
	buildtest := buildTest(ctx, pkg, deps)
	runtest := runTest(ctx, pkg, buildtest)
	return []Target{runtest}
}

type buildTestTarget struct {
	target
	deps []Target
	*Package
	*Context
}

func (t *buildTestTarget) execute() {
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

func (t *buildTestTarget) objdir() string  { return t.Context.TestObjdir(t.Package) }
func (t *buildTestTarget) objfile() string { return filepath.Join(t.objdir(), "_go_.6") }
func (t *buildTestTarget) pkgfile() string { return t.Package.ImportPath() + ".a" }

func (t *buildTestTarget) build() error {
	gofiles := t.GoFiles
	gofiles = append(gofiles, t.TestGoFiles...)
	objdir := t.objdir()
	if err := os.MkdirAll(objdir, 0777); err != nil {
		return err
	}
	if err := t.Gc(t.ImportPath(), t.Srcdir(), t.objfile(), gofiles); err != nil {
		return err
	}
	pkgdir := filepath.Dir(filepath.Join(t.Pkgdir(), t.pkgfile()))
	if err := os.MkdirAll(pkgdir, 0777); err != nil {
		return err
	}
	if err := t.Pack(t.pkgfile(), t.Pkgdir(), t.objfile()); err != nil {
		return err
	}
	if err := t.buildTestMain(t.objdir()); err != nil {
		return err
	}
	if err := t.Gc(t.objdir(), t.objdir(), t.Package.Name()+".6", []string{"_testmain.go"}); err != nil {
		return err
	}
	return t.Ld(filepath.Join(t.objdir(), t.Package.Name()+".test"), filepath.Join(t.objdir(), t.Package.Name()+".6"))
}

func (t *buildTestTarget) buildTestMain(objdir string) error {
	return writeTestmain(filepath.Join(t.objdir(), "_testmain.go"), t.Package)
}

func buildTest(ctx *Context, pkg *Package, deps []Target) *buildTestTarget {
	t := &buildTestTarget{
		target: target{
			done: make(chan struct{}),
		},
		deps:    deps,
		Package: pkg,
		Context: ctx,
	}
	go t.execute()
	return t
}

type runTestTarget struct {
	target
	deps []Target
	*Package
	*Context
}

func (t *runTestTarget) objdir() string { return t.Context.TestObjdir(t.Package) }

func (t *runTestTarget) execute() {
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

func (t *runTestTarget) build() error {
	cmd := exec.Command(filepath.Join(t.objdir(), t.Package.Name()+".test"))
	cmd.Dir = t.Package.Srcdir()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("cd %s; %s", cmd.Dir, strings.Join(cmd.Args, " "))
	return cmd.Run()
}

func runTest(ctx *Context, pkg *Package, deps ...Target) *runTestTarget {
	t := &runTestTarget{
		target: target{
			done: make(chan struct{}),
		},
		deps:    deps,
		Package: pkg,
		Context: ctx,
	}
	go t.execute()
	return t
}
