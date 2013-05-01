package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/davecheney/gogo"
	"github.com/davecheney/gogo/log"
)

// Test returns a Future representing the result of compiling the
// package pkg, and its dependencies, and linking it with the
// test runner.
func Test(pkg *gogo.Package) gogo.Future {
	// commands are built as packages for testing.
	return testPackage(pkg)
}

func testPackage(pkg *gogo.Package) gogo.Future {
	// build dependencies
	var deps []gogo.Future
	for _, dep := range pkg.Imports {
		deps = append(deps, Build(dep))
	}
	Compile := Compile(pkg, deps, true)
	buildtest := buildTest(pkg, Compile)
	runtest := runTest(pkg, buildtest)
	return runtest
}

type buildTestTarget struct {
	future
	deps []gogo.Future
	*gogo.Package
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

func (t *buildTestTarget) build() error {
	objdir := objdir(t.Context, t.Package)
	if err := t.buildTestMain(objdir); err != nil {
		return err
	}
	if err := t.Gc(objdir, objdir, t.Package.Name+".6", []string{"_testmain.go"}); err != nil {
		return err
	}
	return t.Ld(filepath.Join(objdir, t.Package.Name+".test"), filepath.Join(objdir, t.Package.Name+".6"))
}

func (t *buildTestTarget) buildTestMain(_ string) error {
	return writeTestmain(filepath.Join(objdir(t.Context, t.Package), "_testmain.go"), t.Package)
}

func buildTest(pkg *gogo.Package, deps ...gogo.Future) gogo.Future {
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
	deps []gogo.Future
	*gogo.Package
}

func (t *runTestTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.future.err <- err
			return
		}
	}
	log.Infof("test %q", t.Package.ImportPath)
	t.future.err <- t.build()
}

func (t *runTestTarget) build() error {
	cmd := exec.Command(filepath.Join(objdir(t.Context, t.Package), t.Package.Name+".test"))
	cmd.Dir = t.Package.Srcdir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Infof("cd %s; %s", cmd.Dir, strings.Join(cmd.Args, " "))
	return cmd.Run()
}

func runTest(pkg *gogo.Package, deps ...gogo.Future) gogo.Future {
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

// testobjdir returns the destination for test object files compiled for this Package.
func testobjdir(ctx *gogo.Context, pkg *gogo.Package) string {
	return filepath.Join(ctx.Workdir(), filepath.FromSlash(pkg.ImportPath), "_test")
}
