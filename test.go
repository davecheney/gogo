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
	gofiles := pkg.GoFiles
	gofiles = append(gofiles, pkg.TestGoFiles...)
	gc := Gc(pkg, gofiles, deps...)
	pack := Pack(pkg, gc)
	buildtest := buildTest(pkg, pack)
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
	objdir := t.Objdir()
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

func buildTest(pkg *Package, deps ...Future) Future {
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
