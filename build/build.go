package build

import (
	"log"
	"os"
	"path/filepath"

	"github.com/davecheney/gogo"
)

// future implements a gogo.Future
type future struct {
	err chan error
}

func (f *future) Result() error {
	result := <-f.err
	f.err <- result
	return result
}

func Build(pkg *gogo.Package) gogo.Future {
	if pkg.Name() == "main" {
		return buildCommand(pkg)
	}
	return buildPackage(pkg)
}

func buildPackage(pkg *gogo.Package) gogo.Future {
	var deps []gogo.Future
	for _, dep := range pkg.Imports {
		deps = append(deps, buildPackage(dep))
	}
	if _, ok := pkg.Context.Targets[pkg]; !ok {
		compile := compile(pkg, deps, false)
		pkg.Context.Targets[pkg] = compile
	}
	return pkg.Context.Targets[pkg]
}

func buildCommand(pkg *gogo.Package) gogo.Future {
	var deps []gogo.Future
	for _, dep := range pkg.Imports {
		deps = append(deps, buildPackage(dep))
	}
	compile := compile(pkg, deps, false)
	ld := Ld(pkg, compile)
	return ld
}

// compile is a helper which combines all the steps required
// to build a go package
func compile(pkg *gogo.Package, deps []gogo.Future, includeTests bool) gogo.Future {
	gofiles := pkg.GoFiles
	if includeTests {
		gofiles = append(gofiles, pkg.TestGoFiles...)
	}
	objs := []objFuture{Gc(pkg, deps, gofiles)}
	for _, sfile := range pkg.SFiles {
		objs = append(objs, Asm(pkg, sfile))
	}
	pack := Pack(pkg, objs)
	return pack
}

// objFuture represents a Future that produces an object file.
type objFuture interface {
	gogo.Future

	// objfile returns the name of the file that is
	// produced by the Future if successful.
	objfile() string
}

// pkgFuture represents a Future that produces a pkg (.a) file.
type pkgFuture interface {
	gogo.Future

	// pkgfile returns the name of the file that is
	// produced by the Future if successful.
	pkgfile() string
}

type packTarget struct {
	future
	deps     []objFuture
	objfiles []string
	*gogo.Package
}

// Pack returns a Future representing the result of packing a
// set of Context specific object files into an archive.
func Pack(pkg *gogo.Package, deps []objFuture) pkgFuture {
	t := &packTarget{
		future: future{
			err: make(chan error, 1),
		},
		deps:    deps,
		Package: pkg,
	}
	go t.execute()
	return t
}

func (t *packTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.future.err <- err
			return
		}
		// collect successful objfiles for packing
		t.objfiles = append(t.objfiles, dep.objfile())
	}
	log.Printf("pack %q: %s", t.Package.ImportPath(), t.objfiles)
	t.future.err <- t.build()
}

func (t *packTarget) pkgfile() string { return t.Package.ImportPath() + ".a" }

func (t *packTarget) build() error {
	ofile := t.pkgfile()
	pkgdir := filepath.Dir(filepath.Join(t.Pkgdir(), ofile))
	if err := os.MkdirAll(pkgdir, 0777); err != nil {
		return err
	}
	return t.Pack(ofile, t.Pkgdir(), t.objfiles...)
}

type gcTarget struct {
	future
	deps    []gogo.Future
	gofiles []string
	*gogo.Package
}

func (t *gcTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.future.err <- err
			return
		}
	}
	log.Printf("gc %q: %s", t.Package.ImportPath(), t.gofiles)
	t.future.err <- t.build()
}

// Gc returns a Future representing the result of compiling a
// set of gofiles with the Context specified gc compiler.
func Gc(pkg *gogo.Package, deps []gogo.Future, gofiles []string) objFuture {
	t := &gcTarget{
		future: future{
			err: make(chan error, 1),
		},
		deps:    deps,
		gofiles: gofiles,
		Package: pkg,
	}
	go t.execute()
	return t
}

func (t *gcTarget) objfile() string { return filepath.Join(t.Objdir(), "_go_.6") }

func (t *gcTarget) build() error {
	if err := os.MkdirAll(t.Objdir(), 0777); err != nil {
		return err
	}
	return t.Gc(t.ImportPath(), t.Srcdir(), t.objfile(), t.gofiles)
}

type asmTarget struct {
	future
	sfile string
	*gogo.Package
}

func (t *asmTarget) execute() {
	log.Printf("as %q: %s", t.Package.ImportPath(), t.sfile)
	t.future.err <- t.build()
}

// Asm returns a Future representing the result of assembling
// sfile with the Context specified asssembler.
func Asm(pkg *gogo.Package, sfile string) objFuture {
	t := &asmTarget{
		future: future{
			err: make(chan error, 1),
		},
		sfile:   sfile,
		Package: pkg,
	}
	go t.execute()
	return t
}

func (t *asmTarget) objfile() string {
	return filepath.Join(t.Objdir(), t.sfile[:len(t.sfile)-len(".s")]+".6")
}

func (t *asmTarget) build() error {
	if err := os.MkdirAll(t.Objdir(), 0777); err != nil {
		return err
	}
	return t.Asm(t.Srcdir(), t.objfile(), t.sfile)
}

type ldTarget struct {
	future
	deps []gogo.Future
	*gogo.Package
}

func (t *ldTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.future.err <- err
			return
		}
	}
	log.Printf("ld %q", t.Package.ImportPath())
	t.future.err <- t.build()
}

// Ld returns a Future representing the result of linking a
// Package into a command with the Context provided linker.
func Ld(pkg *gogo.Package, deps ...gogo.Future) gogo.Future {
	t := &ldTarget{
		future: future{
			err: make(chan error, 1),
		},
		deps:    deps,
		Package: pkg,
	}
	go t.execute()
	return &t.future
}

func (t *ldTarget) pkgfile() string { return filepath.Join(t.Workdir(), t.Package.ImportPath()+".a") }

func (t *ldTarget) build() error {
	bindir := t.Package.Context.Bindir()
	if err := os.MkdirAll(bindir, 0777); err != nil {
		return err
	}
	return t.Ld(filepath.Join(bindir, filepath.Base(t.Package.ImportPath())), t.pkgfile())
}
