package build

import (
	"go/build"
	"path/filepath"
	"strings"
	"time"

	"github.com/davecheney/gogo/log"
)

// target implements a Future
type target struct {
	err chan error
	*build.Package
	*Context
}

func (t *target) Result() error {
	result := <-t.err
	t.err <- result
	return result
}

func (t *target) Srcdir() string {
	return filepath.Join(t.SrcRoot, t.ImportPath)
}

func newTarget(ctx *Context, pkg *build.Package) target {
	return target{
		err:     make(chan error, 1),
		Context: ctx,
		Package: pkg,
	}
}

// gcTarget implements a Future that represents compiling a set of Go files.
type gcTarget struct {
	target
	deps    []Future
	gofiles []string
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
	err := t.Gc(t.ImportPath, t.Srcdir(), t.Objfile(), t.gofiles)
	t.Record("gc", time.Since(t0))
	return err
}

// ccTarget implements a Future that represents compiling a .c file.
type ccTarget struct {
	target
	dep   Future
	cfile string
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
	err := t.Cc(t.Srcdir(), objdir(t.Context, t.Package), t.Objfile(), filepath.Join(objdir(t.Context, t.Package), t.cfile))
	t.Record("cc", time.Since(t0))
	t.err <- err
}

// ccTarget implements a gogo.Future that represents the result of
// invoking the system gcc compiler.
type gccTarget struct {
	target
	deps []Future
	args []string
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
	err := t.Gcc(t.Srcdir(), t.args)
	t.Record("gcc", time.Since(t0))
	t.err <- err
}

// asmTarget implements a Future that represents assembling a .s file.
type asmTarget struct {
	target
	sfile string
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
	err := t.Asm(t.Srcdir(), t.Objfile(), t.sfile)
	t.Record("asm", time.Since(t0))
	return err
}

// cgoTarget implements a Future that represents invoking the cgo command.
type cgoTarget struct {
	target
	deps []Future
	args []string
}

func (t *cgoTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.err <- err
			return
		}
	}
	log.Debugf("cgo %q: %s", t.ImportPath, t.args)
	t.err <- t.build()
}

func (t *cgoTarget) build() error {
	t0 := time.Now()
	if err := t.Mkdir(objdir(t.Context, t.Package)); err != nil {
		return err
	}
	err := t.Cgo(t.Srcdir(), t.args)
	t.Record("cgo", time.Since(t0))
	return err
}

// packTarget implements a Future that represents packing Go object files into a .a archive.
type packTarget struct {
	target
	deps     []ObjFuture
	objfiles []string
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

func (t *packTarget) pkgfile() string {
	return filepath.Join(t.Workdir(), filepath.FromSlash(t.ImportPath+".a"))
}

func (t *packTarget) build() error {
	t0 := time.Now()
	afile := t.pkgfile()
	pkgdir := filepath.Dir(afile)
	if err := t.Mkdir(pkgdir); err != nil {
		return err
	}
	err := t.Pack(afile, t.objfiles...)
	t.Record("pack", time.Since(t0))
	return err
}

// ldTarget implements a Future that represents
// linking a set .a file into a command.
type ldTarget struct {
	target
	afile PkgFuture
}

func (t *ldTarget) execute() {
	if err := t.afile.Result(); err != nil {
		t.err <- err
		return
	}
	log.Infof("ld %q: %v", t.ImportPath, t.afile.pkgfile())
	t.err <- t.build()
}

func (t *ldTarget) build() error {
	t0 := time.Now()
	bindir := t.Context.Bindir()
	if err := t.Mkdir(bindir); err != nil {
		return err
	}
	err := t.Ld(filepath.Join(bindir, filepath.Base(t.ImportPath)), t.afile.pkgfile())
	t.Record("ld", time.Since(t0))
	return err
}
