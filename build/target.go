package build

import (
	"path/filepath"
	"strings"
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
	*gogo.Context
}

func (t *gcTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.err <- err
			return
		}
	}
	log.Debugf("gc %q: %s", t.ImportPath, t.gofiles)
	t.err <- t.build()
}

func (t *gcTarget) Objfile() string { return filepath.Join(objdir(t.Context, t.Package), "_go_.6") }

func (t *gcTarget) build() error {
	t0 := time.Now()
	if err := t.Mkdir(objdir(t.Context, t.Package)); err != nil {
		return err
	}
	err := t.Gc(t.ImportPath, t.Srcdir, t.Objfile(), t.gofiles)
	t.Record("gc", time.Since(t0))
	return err
}

// ccTarget implements a gogo.Future that represents
// compiling a .c file.
type ccTarget struct {
	target
	dep   gogo.Future
	cfile string
	*gogo.Package
}

func (t *ccTarget) Objfile() string {
	return filepath.Join(objdir(t.Context, t.Package), strings.Replace(t.cfile, ".c", ".6", 1))
}

func (t *ccTarget) execute() {
	t0 := time.Now()
	if err := t.dep.Result(); err != nil {
		t.err <- err
		return
	}
	log.Debugf("cc %q: %s", t.Package.ImportPath, t.cfile)
	err := t.Cc(t.Srcdir, objdir(t.Context, t.Package), t.Objfile(), filepath.Join(objdir(t.Context, t.Package), t.cfile))
	t.Record("cc", time.Since(t0))
	t.err <- err
}

// ccTarget implements a gogo.Future that represents the result of
// invoking the system gcc compiler.
type gccTarget struct {
	target
	deps []gogo.Future
	args []string
	*gogo.Package
}

func (t *gccTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.err <- err
			return
		}
	}
	t0 := time.Now()
	log.Debugf("gcc %q: %s", t.Package.ImportPath, t.args)
	err := t.Gcc(t.Srcdir, t.args)
	t.Record("gcc", time.Since(t0))
	t.err <- err
}

// asmTarget implements a gogo.Future that represents
// assembling a .s file.
type asmTarget struct {
	target
	sfile string
	*gogo.Package
	*gogo.Context
}

func (t *asmTarget) execute() {
	log.Debugf("as %q: %s", t.ImportPath, t.sfile)
	t.err <- t.build()
}

func (t *asmTarget) Objfile() string {
	return filepath.Join(objdir(t.Context, t.Package), t.sfile[:len(t.sfile)-len(".s")]+".6")
}

func (t *asmTarget) build() error {
	t0 := time.Now()
	if err := t.Mkdir(objdir(t.Context, t.Package)); err != nil {
		return err
	}
	err := t.Asm(t.Srcdir, t.Objfile(), t.sfile)
	t.Record("asm", time.Since(t0))
	return err
}

// cgoTarget implements a gogo.Future that represents
// invoking the cgo command.
type cgoTarget struct {
	target
	deps []gogo.Future
	args []string
	*gogo.Package
}

func (t *cgoTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.err <- err
			return
		}
	}
	log.Debugf("cgo %q: %s", t.Package.ImportPath, t.args)
	t.err <- t.build()
}

func (t *cgoTarget) build() error {
	t0 := time.Now()
	if err := t.Mkdir(objdir(t.Context, t.Package)); err != nil {
		return err
	}
	err := t.Cgo(t.Srcdir, t.args)
	t.Record("cgo", time.Since(t0))
	return err
}

// packTarget implements a gogo.Future that represents
// packing Go object files into a .a archive.
type packTarget struct {
	target
	deps     []ObjFuture
	objfiles []string
	*gogo.Package
	*gogo.Context
}

func (t *packTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.err <- err
			return
		}
		// collect successful objfiles for packing
		t.objfiles = append(t.objfiles, dep.Objfile())
	}
	log.Infof("pack %q: %s", t.ImportPath, t.objfiles)
	t.err <- t.build()
}

func (t *packTarget) pkgfile() string { return t.ImportPath + ".a" }

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

// ldTarget implements a gogo.Future that represents
// linking a set .a file into a command.
type ldTarget struct {
	target
	deps []gogo.Future
	*gogo.Package
	*gogo.Context
}

func (t *ldTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.err <- err
			return
		}
	}
	log.Infof("ld %q", t.ImportPath)
	t.err <- t.build()
}

func (t *ldTarget) pkgfile() string { return filepath.Join(t.Workdir(), t.ImportPath+".a") }

func (t *ldTarget) build() error {
	t0 := time.Now()
	bindir := t.Context.Bindir()
	if err := t.Mkdir(bindir); err != nil {
		return err
	}
	err := t.Ld(filepath.Join(bindir, filepath.Base(t.ImportPath)), t.pkgfile())
	t.Record("ld", time.Since(t0))
	return err
}
