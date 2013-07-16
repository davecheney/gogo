package build

import (
	"go/build"
	"path/filepath"
	"strings"
)

// cgo support functions

// cgo returns a Future representing the result of
// successful cgo pre processing and a list of GoFiles
// which would be produced from the source CgoFiles.
// These filenames are only valid of the Result of the
// cgo Future is nil.
func cgo(ctx *Context, pkg *build.Package, deps []Future) ([]ObjFuture, []string) {
	srcdir := filepath.Join(pkg.SrcRoot, pkg.ImportPath)
	objdir := objdir(ctx, pkg)

	var args = []string{"-objdir", objdir, "--", "-I", srcdir, "-I", objdir}
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
	cgo := Cgo(ctx, pkg, deps, args)

	cgodefun := Cc(ctx, pkg, cgo, "_cgo_defun.c")

	var ofiles []string
	var deps2 []Future
	for _, gccfile := range gccfiles {
		args := []string{"-fPIC", "-pthread", "-I", srcdir, "-I", objdir}
		args = append(args, pkg.CgoCFLAGS...)
		ofile := gccfile[:len(gccfile)-2] + ".o"
		ofiles = append(ofiles, ofile)
		deps2 = append(deps2, Gcc(ctx, pkg, []Future{cgodefun}, append(args, "-o", ofile, "-c", gccfile)))
	}

	args = []string{"-pthread", "-o", filepath.Join(objdir, "_cgo_.o")}
	args = append(args, ofiles...)
	args = append(args, pkg.CgoLDFLAGS...)
	gcc := Gcc(ctx, pkg, deps2, args)

	cgo = Cgo(ctx, pkg, []Future{gcc}, []string{"-dynimport", filepath.Join(objdir, "_cgo_.o"), "-dynout", filepath.Join(objdir, "_cgo_import.c")})

	cgoimport := Cc(ctx, pkg, cgo, "_cgo_import.c") // _cgo_import.c is relative to objdir

	args = []string{"-I", srcdir, "-fPIC", "-pthread", "-o", filepath.Join(objdir, "_all.o")}
	for _, ofile := range ofiles {
		// hack
		if strings.Contains(ofile, "_cgo_main") {
			continue
		}
		args = append(args, ofile)
	}

	// more hacking
	libgcc, err := ctx.Libgcc()
	if err != nil {
		panic(err)
	}

	args = append(args, "-Wl,-r", "-nostdlib", libgcc)
	all := Gcc(ctx, pkg, []Future{cgoimport}, args)

	f := &cgoFuture{
		target: newTarget(ctx, pkg),
		dep:    all,
	}
	go func() { f.err <- f.dep.Result() }()
	return []ObjFuture{f, cgoimport, cgodefun}, gofiles
}

type cgoFuture struct {
	target
	dep Future
}

func (f *cgoFuture) Objfile() string {
	return filepath.Join(objdir(f.Context, f.Package), "_all.o")
}

// nilFuture represents a future of no work which always
// returns nil immediately.
type nilFuture struct{}

func (*nilFuture) Result() error { return nil }

// Cgo returns a Future representing the result of running the
// cgo command.
func Cgo(ctx *Context, pkg *build.Package, deps []Future, args []string) Future {
	cgo := &cgoTarget{
		target: newTarget(ctx, pkg),
		deps:   deps,
		args:   args,
	}
	go cgo.execute()
	return cgo
}

// Cc returns a Future representing the result of compiling a
// .c source file with the Context specified cc compiler.
func Cc(ctx *Context, pkg *build.Package, dep Future, cfile string) ObjFuture {
	cc := &ccTarget{
		target: newTarget(ctx, pkg),
		dep:    dep,
		cfile:  cfile,
	}
	go cc.execute()
	return cc
}

// Gcc returns a Future representing the result of invoking the
// system gcc compiler.
func Gcc(ctx *Context, pkg *build.Package, deps []Future, args []string) Future {
	gcc := &gccTarget{
		target: newTarget(ctx, pkg),
		deps:   deps,
		args:   args,
	}
	go gcc.execute()
	return gcc
}
