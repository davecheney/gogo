package build

import (
	"path/filepath"
	"time"

	"github.com/davecheney/gogo"
	"github.com/davecheney/gogo/log"
)

// target implements a gogo.Future
type target struct {
	err chan error
}

func (t *target) Result() error {
	result := <-t.err
	t.err <- result
	return result
}

// gcTarget implements a gogo.Future that represents
// compiling a set of Go files.
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

// asmTarget implements a gogo.Future that represents
// assembling a .s file.
type asmTarget struct {
	target
	sfile string
	*gogo.Package
}

func (t *asmTarget) execute() {
	log.Debugf("as %q: %s", t.Package.ImportPath, t.sfile)
	t.err <- t.build()
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

// packTarget implements a gogo.Future that represents
// packing Go object files into a .a archive.
type packTarget struct {
	target
	deps     []objFuture
	objfiles []string
	*gogo.Package
}

func (t *packTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.err <- err
			return
		}
		// collect successful objfiles for packing
		t.objfiles = append(t.objfiles, dep.objfile())
	}
	log.Infof("pack %q: %s", t.Package.ImportPath, t.objfiles)
	t.err <- t.build()
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