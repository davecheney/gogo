package build

import (
	"github.com/davecheney/gogo"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// cgo support functions

// cgo returns a Future representing the result of
// successful cgo pre processing and a list of GoFiles
// which would be produced from the source CgoFiles.
// These filenames are only valid of the Result of the
// cgo Future is nil.
func cgo(pkg *gogo.Package, deps []gogo.Future) ([]objFuture, []string) {
	srcdir := pkg.Srcdir()
	objdir := pkg.Objdir()

	var args = []string{"-objdir", objdir, "--", "-I", pkg.Srcdir(), "-I", objdir}
	args = append(args, pkg.CgoCFLAGS...)
	var gofiles = []string{filepath.Join(objdir, "_cgo_gotypes.go")}
	var gccfiles = []string{filepath.Join(objdir, "_cgo_main.c"), filepath.Join(objdir, "_cgo_export.c")}
	for _, cgofile := range pkg.CgoFiles {
		args = append(args, cgofile)
		gofiles = append(gofiles, filepath.Join(objdir, strings.Replace(cgofile, ".go", ".cgo1.go", 1)))
		gccfiles = append(gccfiles, filepath.Join(objdir, strings.Replace(cgofile, ".go", ".cgo2.c", 1)))
	}
	for _, cfile := range pkg.CFiles {
		gccfiles = append(gccfiles, filepath.Join(srcdir, cfile))
	}
	cgo := Cgo(pkg, deps, args)

	cgodefun := Cc(pkg, cgo, "_cgo_defun.c")

	var ofiles []string
	var deps2 []gogo.Future
	for _, gccfile := range gccfiles {
		args := []string{"-fPIC", "-pthread", "-I", srcdir, "-I", objdir}
		args = append(args, pkg.CgoCFLAGS...)
		ofile := gccfile[:len(gccfile)-2] + ".o"
		ofiles = append(ofiles, ofile)
		deps2 = append(deps2, Gcc(pkg, []gogo.Future{cgodefun}, append(args, "-o", ofile, "-c", gccfile)))
	}

	args = []string{"-pthread", "-o", filepath.Join(pkg.Objdir(), "_cgo_.o")}
	args = append(args, ofiles...)
	args = append(args, pkg.CgoLDFLAGS...)
	gcc := Gcc(pkg, deps2, args)

	cgo = Cgo(pkg, []gogo.Future{gcc}, []string{"-dynimport", filepath.Join(pkg.Objdir(), "_cgo_.o"), "-dynout", filepath.Join(pkg.Objdir(), "_cgo_import.c")})

	cgoimport := Cc(pkg, cgo, "_cgo_import.c") // _cgo_import.c is relative to objdir

	args = []string{"-I", srcdir, "-fPIC", "-pthread", "-o", filepath.Join(objdir, "_all.o")}
	for _, ofile := range ofiles {
		// hack
		if strings.Contains(ofile, "_cgo_main") {
			continue
		}
		args = append(args, ofile)
	}

	// more hacking
	libgcc, err := pkg.Libgcc()
	if err != nil {
		panic(err)
	}

	args = append(args, "-Wl,-r", "-nostdlib", libgcc)
	all := Gcc(pkg, []gogo.Future{cgoimport}, args)

	f := &cgoFuture{
		future: future{
			err: make(chan error, 1),
		},
		dep:     all,
		Package: pkg,
	}
	go func() { f.future.err <- f.dep.Result() }()
	return []objFuture{f, cgoimport, cgodefun}, gofiles
}

type cgoFuture struct {
	future
	dep gogo.Future
	*gogo.Package
}

func (f *cgoFuture) objfile() string {
	return filepath.Join(f.Objdir(), "_all.o")
}

// nilFuture represents a future of no work which always
// returns nil immediately.
type nilFuture struct{}

func (*nilFuture) Result() error { return nil }

type cgoTarget struct {
	future
	deps []gogo.Future
	args []string
	*gogo.Package
}

// Cgo returns a Future representing the result of running the
// cgo command.
func Cgo(pkg *gogo.Package, deps []gogo.Future, args []string) gogo.Future {
	cgo := &cgoTarget{
		future: future{
			err: make(chan error, 1),
		},
		deps:    deps,
		args:    args,
		Package: pkg,
	}
	go cgo.execute()
	return cgo
}

func (t *cgoTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.future.err <- err
			return
		}
	}
	log.Printf("cgo %q: %s", t.Package.ImportPath(), t.args)
	t.future.err <- t.build()
}

func (t *cgoTarget) build() error {
	if err := os.MkdirAll(t.Objdir(), 0777); err != nil {
		return err
	}
	return t.Cgo(t.Srcdir(), t.args)
}

type ccTarget struct {
	future
	dep   gogo.Future
	cfile string
	*gogo.Package
}

// Cc returns a Future representing the result of compiling a
// .c source file with the Context specified cc compiler.
func Cc(pkg *gogo.Package, dep gogo.Future, cfile string) objFuture {
	cc := &ccTarget{
		future: future{
			err: make(chan error, 1),
		},
		dep:     dep,
		cfile:   cfile,
		Package: pkg,
	}
	go cc.execute()
	return cc
}

func (t *ccTarget) objfile() string {
	return filepath.Join(t.Objdir(), strings.Replace(t.cfile, ".c", ".6", 1))
}

func (t *ccTarget) execute() {
	if err := t.dep.Result(); err != nil {
		t.future.err <- err
		return
	}
	log.Printf("cc %q: %s", t.Package.ImportPath(), t.cfile)
	t.future.err <- t.Cc(t.Srcdir(), t.Objdir(), t.objfile(), filepath.Join(t.Objdir(), t.cfile))
}

type gccTarget struct {
	future
	deps []gogo.Future
	args []string
	*gogo.Package
}

// Gcc returns a Future representing the result of invoking the
// system gcc compiler.
func Gcc(pkg *gogo.Package, deps []gogo.Future, args []string) gogo.Future {
	gcc := &gccTarget{
		future: future{
			err: make(chan error, 1),
		},
		deps:    deps,
		args:    args,
		Package: pkg,
	}
	go gcc.execute()
	return gcc
}

func (t *gccTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.future.err <- err
			return
		}
	}
	log.Printf("gcc %q: %s", t.Package.ImportPath(), t.args)
	t.future.err <- t.Gcc(t.Srcdir(), t.args)
}
