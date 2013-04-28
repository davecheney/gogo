// Package gogo/build provides functions for building and testing
// Go packages.
package build

import (
	"path/filepath"
	"time"

	"github.com/davecheney/gogo"
	"github.com/davecheney/gogo/log"
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

// Build returns a Future representing the result of compiling the
// package pkg, and its dependencies.
// If pkg is a command, then the results of build include linking
// the final binary into pkg.Context.Bindir().
func Build(pkg *gogo.Package) gogo.Future {
	if pkg.Name() == "main" {
		return buildCommand(pkg)
	}
	return buildPackage(pkg)
}

// buildPackage returns a Future repesenting the results of compiling
// pkg and its dependencies.
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

// buildCommand returns a Future repesenting the results of compiling
// pkg as a command and linking the result into pkg.Context.Bindir().
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
	var gofiles []string
	gofiles = append(gofiles, pkg.GoFiles...)
	var objs []objFuture
	if len(pkg.CgoFiles) > 0 {
		cgo, cgofiles := cgo(pkg, deps)
		deps = append(deps, cgo[0])
		objs = append(objs, cgo...)
		gofiles = append(gofiles, cgofiles...)
	}
	if includeTests {
		gofiles = append(gofiles, pkg.TestGoFiles...)
	}
	objs = append(objs, Gc(pkg, deps, gofiles))
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
	log.Infof("pack %q: %s", t.Package.ImportPath, t.objfiles)
	t.future.err <- t.build()
}

func (t *packTarget) pkgfile() string { return t.Package.ImportPath + ".a" }

func (t *packTarget) build() error {
	t0 := time.Now()
	ofile := t.pkgfile()
	pkgdir := filepath.Dir(filepath.Join(t.Pkgdir(), ofile))
	if err := t.Mkdir(pkgdir); err != nil {
		return err
	}
	err := t.Pack(ofile, t.Pkgdir(), t.objfiles...)
	t.Record("pack", time.Since(t0))
	return err
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
	log.Debugf("gc %q: %s", t.Package.ImportPath, t.gofiles)
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
	t0 := time.Now()
	if err := t.Mkdir(t.Objdir()); err != nil {
		return err
	}
	err := t.Gc(t.ImportPath, t.Srcdir(), t.objfile(), t.gofiles)
	t.Record("gc", time.Since(t0))
	return err
}

type asmTarget struct {
	future
	sfile string
	*gogo.Package
}

func (t *asmTarget) execute() {
	log.Debugf("as %q: %s", t.Package.ImportPath, t.sfile)
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
	t0 := time.Now()
	if err := t.Mkdir(t.Objdir()); err != nil {
		return err
	}
	err := t.Asm(t.Srcdir(), t.objfile(), t.sfile)
	t.Record("asm", time.Since(t0))
	return err
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
	log.Infof("ld %q", t.Package.ImportPath)
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

func (t *ldTarget) pkgfile() string { return filepath.Join(t.Workdir(), t.Package.ImportPath+".a") }

func (t *ldTarget) build() error {
	t0 := time.Now()
	bindir := t.Package.Context.Bindir()
	if err := t.Mkdir(bindir); err != nil {
		return err
	}
	err := t.Ld(filepath.Join(bindir, filepath.Base(t.Package.ImportPath)), t.pkgfile())
	t.Record("ld", time.Since(t0))
	return err
}
