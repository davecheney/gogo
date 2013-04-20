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
		gc := Gc(pkg, deps...)
		pack := Pack(pkg, gc)
		pkg.Context.Targets[pkg] = pack
	}
	log.Printf("build package %q", pkg.ImportPath())
	return []Future{pkg.Context.Targets[pkg]}
}

func buildCommand(pkg *Package) []Future {
	var deps []Future
	for _, dep := range pkg.Imports {
		deps = append(deps, buildPackage(dep)...)
	}
	if _, ok := pkg.Context.Targets[pkg]; !ok {
		gc := Gc(pkg, deps...)
		pack := Pack(pkg, gc)
		ld := Ld(pkg, pack)
		pkg.Context.Targets[pkg] = ld
	}
	log.Printf("build command %q", pkg.ImportPath())
	return []Future{pkg.Context.Targets[pkg]}
}

type packTarget struct {
	target
	deps []Future
	*Package
}

func Pack(pkg *Package, deps ...Future) *packTarget {
	t := &packTarget{
		target: target{
			done: make(chan struct{}),
		},
		deps:    deps,
		Package: pkg,
	}
	go t.execute()
	return t
}

func (t *packTarget) execute() {
	defer close(t.done)
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.setErr(err)
			return
		}
	}
	if err := t.build(); err != nil {
		t.setErr(err)
	}
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
	target
	deps []Future
	*Package
}

func (t *gcTarget) execute() {
	defer close(t.done)
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.setErr(err)
			return
		}
	}
	if err := t.build(); err != nil {
		t.setErr(err)
	}
}

func Gc(pkg *Package, deps ...Future) *gcTarget {
	t := &gcTarget{
		target: target{
			done: make(chan struct{}),
		},
		deps:    deps,
		Package: pkg,
	}
	go t.execute()
	return t
}

func (t *gcTarget) objfile() string { return filepath.Join(t.Objdir(), "_go_.6") }

func (t *gcTarget) build() error {
	gofiles := t.GoFiles
	if err := os.MkdirAll(t.Objdir(), 0777); err != nil {
		return err
	}
	return t.Gc(t.ImportPath(), t.Srcdir(), t.objfile(), gofiles)
}

type asmTarget struct {
	target
	deps []Future
	*Package
}

func (t *asmTarget) execute() {
	defer close(t.done)
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.setErr(err)
			return
		}
	}
	if err := t.build(); err != nil {
		t.setErr(err)
	}
}

func newAsmTarget(pkg *Package, deps ...Future) *gcTarget {
	t := &gcTarget{
		target: target{
			done: make(chan struct{}),
		},
		deps:    deps,
		Package: pkg,
	}
	go t.execute()
	return t
}

func (t *asmTarget) build() error {
	return nil // t.Project.Toolchain().Asm(t.Context, t.Package)
}

type ldTarget struct {
	target
	deps []Future
	*Package
}

func (t *ldTarget) execute() {
	defer close(t.done)
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.setErr(err)
			return
		}
	}
	if err := t.build(); err != nil {
		t.setErr(err)
	}
}

func Ld(pkg *Package, deps ...Future) *ldTarget {
	t := &ldTarget{
		target: target{
			done: make(chan struct{}),
		},
		deps:    deps,
		Package: pkg,
	}
	go t.execute()
	return t
}

func (t *ldTarget) pkgfile() string { return filepath.Join(t.Workdir(), t.Package.ImportPath()+".a") }

func (t *ldTarget) build() error {
	bindir := t.Package.Context.Bindir()
	if err := os.MkdirAll(bindir, 0777); err != nil {
		return err
	}
	return t.Ld(filepath.Join(bindir, filepath.Base(t.Package.ImportPath())), t.pkgfile())
}
