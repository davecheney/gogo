package gogo

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Test(pkg *Package) []Future {
	// commands are built as packages for testing.
	return testPackage(pkg)
}

func testPackage(pkg *Package) []Future {
	// build dependencies
	var deps []Future
	for _, dep := range pkg.Imports {
		deps = append(deps, Build(dep)...)
	}
	buildtest := buildTest(pkg, deps)
	runtest := runTest(pkg, buildtest)
	return []Future{runtest}
}

type buildTestTarget struct {
	future
	deps []Future
	*Package
}

func (t *buildTestTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.future.err <- err
			return
		}
	}
	t.future.err <- t.build()
}

func (t *buildTestTarget) objfile() string { return filepath.Join(t.Objdir(), "_go_.6") }
func (t *buildTestTarget) pkgfile() string { return t.Package.ImportPath() + ".a" }

func (t *buildTestTarget) build() error {
	gofiles := t.GoFiles
	gofiles = append(gofiles, t.TestGoFiles...)
	objdir := t.Objdir()
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
	if err := t.buildTestMain(objdir); err != nil {
		return err
	}
	if err := t.Gc(objdir, objdir, t.Package.Name()+".6", []string{"_testmain.go"}); err != nil {
		return err
	}
	return t.Ld(filepath.Join(objdir, t.Package.Name()+".test"), filepath.Join(objdir, t.Package.Name()+".6"))
}

func (t *buildTestTarget) buildTestMain(objdir string) error {
	return writeTestmain(filepath.Join(t.Objdir(), "_testmain.go"), t.Package)
}

func buildTest(pkg *Package, deps []Future) Future {
	t := &buildTestTarget{
		future: future{
			err: make(chan error, 1),
		},
		deps:    deps,
		Package: pkg,
	}
	go t.execute()
	return &t.future
}

type runTestTarget struct {
	future
	deps []Future
	*Package
}

func (t *runTestTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.future.err <- err
			return
		}
	}
	log.Printf("test %q", t.Package.ImportPath())
	t.future.err <- t.build()
}

func (t *runTestTarget) build() error {
	cmd := exec.Command(filepath.Join(t.Objdir(), t.Package.Name()+".test"))
	cmd.Dir = t.Package.Srcdir()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("cd %s; %s", cmd.Dir, strings.Join(cmd.Args, " "))
	return cmd.Run()
}

func runTest(pkg *Package, deps ...Future) Future {
	t := &runTestTarget{
		future: future{
			err: make(chan error, 1),
		},
		deps:    deps,
		Package: pkg,
	}
	go t.execute()
	return &t.future
}
