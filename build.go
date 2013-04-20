package gogo

import (
	"log"
	"os"
	"path/filepath"
)

func Build(pkg *Package) []Future {
	if pkg.Name() == "main" {
		return buildCommand(pkg)
	}
	return buildPackage(pkg)
}

func buildPackage(pkg *Package) []Future {
	var deps []Future
	for _, dep := range pkg.Imports {
		deps = append(deps, buildPackage(dep)...)
	}
	if _, ok := pkg.Context.Targets[pkg]; !ok {
		compile := compile(pkg, deps, false)
		pkg.Context.Targets[pkg] = compile
	}
	return []Future{pkg.Context.Targets[pkg]}
}

func buildCommand(pkg *Package) []Future {
	var deps []Future
	for _, dep := range pkg.Imports {
		deps = append(deps, buildPackage(dep)...)
	}
	if _, ok := pkg.Context.Targets[pkg]; !ok {
		compile := compile(pkg, deps, false)
		ld := Ld(pkg, compile)
		pkg.Context.Targets[pkg] = ld
	}
	return []Future{pkg.Context.Targets[pkg]}
}

// compile is a helper which combines all the steps required
// to build a go package
func compile(pkg *Package, deps []Future, includeTests bool) Future {
	gofiles := pkg.GoFiles
	if includeTests {
		gofiles = append(gofiles, pkg.TestGoFiles...)
	}
	targets := []Future{Gc(pkg, deps, gofiles)}
	for _, sfile := range pkg.SFiles {
		targets = append(targets, Asm(pkg, sfile))
	}
	pack := Pack(pkg, targets...)
	return pack
}

type packTarget struct {
	future
	deps []Future
	*Package
}

// Pack returns a Future representing the result of packing a
// set of Context specific object files into an archive.
func Pack(pkg *Package, deps ...Future) Future {
	t := &packTarget{
		future: future{
			err: make(chan error, 1),
		},
		deps:    deps,
		Package: pkg,
	}
	go t.execute()
	return &t.future
}

func (t *packTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.future.err <- err
			return
		}
	}
	log.Printf("pack %q", t.Package.ImportPath())
	t.future.err <- t.build()
}

func (t *packTarget) objfile() string { return filepath.Join(t.Objdir(), "_go_.6") }
func (t *packTarget) pkgfile() string { return t.Package.ImportPath() + ".a" }

func (t *packTarget) build() error {
	ofile := t.pkgfile()
	pkgdir := filepath.Dir(filepath.Join(t.Pkgdir(), ofile))
	if err := os.MkdirAll(pkgdir, 0777); err != nil {
		return err
	}
	return t.Pack(ofile, t.Pkgdir(), t.objfile())
}

type gcTarget struct {
	future
	deps    []Future
	gofiles []string
	*Package
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
func Gc(pkg *Package, deps []Future, gofiles []string) Future {
	t := &gcTarget{
		future: future{
			err: make(chan error, 1),
		},
		deps:    deps,
		gofiles: gofiles,
		Package: pkg,
	}
	go t.execute()
	return &t.future
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
	*Package
}

func (t *asmTarget) execute() {
	log.Printf("as %q: %s", t.Package.ImportPath(), t.sfile)
	t.future.err <- t.build()
}

// Asm returns a Future representing the result of assembling
// sfile with the Context specified asssembler.
func Asm(pkg *Package, sfile string) Future {
	t := &asmTarget{
		future: future{
			err: make(chan error, 1),
		},
		sfile:   sfile,
		Package: pkg,
	}
	go t.execute()
	return &t.future
}

func (t *asmTarget) build() error {
	if err := os.MkdirAll(t.Objdir(), 0777); err != nil {
		return err
	}
	return t.Asm(t.ImportPath(), t.Srcdir(), t.sfile)
}

type ldTarget struct {
	future
	deps []Future
	*Package
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
func Ld(pkg *Package, deps ...Future) Future {
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
