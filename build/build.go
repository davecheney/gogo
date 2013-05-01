// Package gogo/build provides functions for building and testing
// Go packages.
package build

import (
	"path/filepath"
	"time"

	"github.com/davecheney/gogo"
	"github.com/davecheney/gogo/log"
)

// Build returns a Future representing the result of compiling the
// package pkg, and its dependencies.
// If pkg is a command, then the results of build include linking
// the final binary into pkg.Context.Bindir().
func Build(pkg *gogo.Package) gogo.Future {
	if pkg.Name == "main" {
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
		Compile := Compile(pkg, deps, false)
		pkg.Context.Targets[pkg] = Compile
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
	Compile := Compile(pkg, deps, false)
	ld := Ld(pkg, Compile)
	return ld
}

// Compile returns a Future representing all the steps required to build a go package.
func Compile(pkg *gogo.Package, deps []gogo.Future, includeTests bool) gogo.Future {
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

// Pack returns a Future representing the result of packing a
// set of Context specific object files into an archive.
func Pack(pkg *gogo.Package, deps []objFuture) pkgFuture {
	t := &packTarget{
		target: target{
			err: make(chan error, 1),
		},
		deps:    deps,
		Package: pkg,
	}
	go t.execute()
	return t
}

type gcTarget struct {
	target
	deps    []gogo.Future
	gofiles []string
	*gogo.Package
}

func (t *gcTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.err <- err
			return
		}
	}
	log.Debugf("gc %q: %s", t.Package.ImportPath, t.gofiles)
	t.err <- t.build()
}

// Gc returns a Future representing the result of compiling a
// set of gofiles with the Context specified gc Compiler.
func Gc(pkg *gogo.Package, deps []gogo.Future, gofiles []string) objFuture {
	t := &gcTarget{
		target: target{
			err: make(chan error, 1),
		},
		deps:    deps,
		gofiles: gofiles,
		Package: pkg,
	}
	go t.execute()
	return t
}

func (t *gcTarget) objfile() string { return filepath.Join(objdir(t.Context, t.Package), "_go_.6") }

func (t *gcTarget) build() error {
	t0 := time.Now()
	if err := t.Mkdir(objdir(t.Context, t.Package)); err != nil {
		return err
	}
	err := t.Gc(t.ImportPath, t.Srcdir, t.objfile(), t.gofiles)
	t.Record("gc", time.Since(t0))
	return err
}

type asmTarget struct {
	target
	sfile string
	*gogo.Package
}

func (t *asmTarget) execute() {
	log.Debugf("as %q: %s", t.Package.ImportPath, t.sfile)
	t.err <- t.build()
}

// Asm returns a Future representing the result of assembling
// sfile with the Context specified asssembler.
func Asm(pkg *gogo.Package, sfile string) objFuture {
	t := &asmTarget{
		target: target{
			err: make(chan error, 1),
		},
		sfile:   sfile,
		Package: pkg,
	}
	go t.execute()
	return t
}

func (t *asmTarget) objfile() string {
	return filepath.Join(objdir(t.Context, t.Package), t.sfile[:len(t.sfile)-len(".s")]+".6")
}

func (t *asmTarget) build() error {
	t0 := time.Now()
	if err := t.Mkdir(objdir(t.Context, t.Package)); err != nil {
		return err
	}
	err := t.Asm(t.Srcdir, t.objfile(), t.sfile)
	t.Record("asm", time.Since(t0))
	return err
}

type ldTarget struct {
	target
	deps []gogo.Future
	*gogo.Package
}

func (t *ldTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.err <- err
			return
		}
	}
	log.Infof("ld %q", t.Package.ImportPath)
	t.err <- t.build()
}

// Ld returns a Future representing the result of linking a
// Package into a command with the Context provided linker.
func Ld(pkg *gogo.Package, deps ...gogo.Future) gogo.Future {
	t := &ldTarget{
		target: target{
			err: make(chan error, 1),
		},
		deps:    deps,
		Package: pkg,
	}
	go t.execute()
	return t
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

// objdir returns the destination for object files compiled for this Package.
func objdir(ctx *gogo.Context, pkg *gogo.Package) string {
	return filepath.Join(ctx.Workdir(), filepath.FromSlash(pkg.ImportPath), "_obj")
}
