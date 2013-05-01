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
func Test(ctx *gogo.Context, pkg *gogo.Package) gogo.Future {
	// commands are built as packages for testing.
	return testPackage(ctx, pkg)
}

func testPackage(ctx *gogo.Context, pkg *gogo.Package) gogo.Future {
	// build dependencies
	var deps []gogo.Future
	for _, dep := range pkg.Imports {
		deps = append(deps, Build(ctx, dep))
	}
	compile := Compile(ctx, pkg, deps, true)
	buildtest := buildTest(ctx, pkg, compile)
	runtest := runTest(ctx, pkg, buildtest)
	return runtest
}

type buildTestTarget struct {
	target
	deps []gogo.Future
}

func (t *buildTestTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.err <- err
			return
		}
	}
	t.err <- t.build()
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

func buildTest(ctx *gogo.Context, pkg *gogo.Package, deps ...gogo.Future) gogo.Future {
	t := &buildTestTarget{
		target: newTarget(ctx, pkg),
		deps:   deps,
	}
	go t.execute()
	return t
}

type runTestTarget struct {
	target
	deps []gogo.Future
}

func (t *runTestTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.err <- err
			return
		}
	}
	log.Infof("test %q", t.Package.ImportPath)
	t.err <- t.build()
}

func (t *runTestTarget) build() error {
	cmd := exec.Command(filepath.Join(objdir(t.Context, t.Package), t.Package.Name+".test"))
	cmd.Dir = t.Package.Srcdir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Infof("cd %s; %s", cmd.Dir, strings.Join(cmd.Args, " "))
	return cmd.Run()
}

func runTest(ctx *gogo.Context, pkg *gogo.Package, deps ...gogo.Future) gogo.Future {
	t := &runTestTarget{
		target: newTarget(ctx, pkg),
		deps:   deps,
	}
	go t.execute()
	return t
}

// testobjdir returns the destination for test object files compiled for this Package.
func testobjdir(ctx *gogo.Context, pkg *gogo.Package) string {
	return filepath.Join(ctx.Workdir(), filepath.FromSlash(pkg.ImportPath), "_test")
}
