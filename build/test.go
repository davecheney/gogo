package build

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/davecheney/gogo"
)

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
	compile := compile(pkg, deps, true)
	buildtest := buildTest(pkg, compile)
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
