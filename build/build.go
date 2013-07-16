// Package gogo/build provides functions for building Go packages.
package build

import (
	"go/build"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/davecheney/gogo/log"
)

// A Future represents the result of a build operation.
type Future interface {
	// Result returns the result of the work as an error, or nil if the work
	// was performed successfully.
	// Implementers must observe these invariants
	// 1. There may be multiple concurrent callers to Result, or Result may
	//    be called many times in sequence, it must always return the same
	// 2. Result blocks until the work has been performed.
	Result() error
}

type errFuture struct{ error }

func (e errFuture) Result() error { return e.error }

// Build returns a Future representing the result of compiling the package pkg
// and its dependencies. If pkg is a command, then the results of build include
// linking the final binary into pkg.Context.Bindir().
func Build(ctx *Context, pkg *build.Package) Future {
	if pkg.Name == "main" {
		return buildCommand(ctx, pkg)
	}
	return buildPackage(ctx, pkg)
}

// buildPackage returns a Future repesenting the results of compiling
// pkg and its dependencies.
func buildPackage(ctx *Context, pkg *build.Package) Future {
	var deps []Future
	for _, dep := range pkg.Imports {
		// TODO(dfc) use project.Spec
		pkg, err := ctx.ResolvePackage(runtime.GOOS, runtime.GOARCH, dep).Result()
		if err != nil {
			return &errFuture{err}
		}
		deps = append(deps, buildPackage(ctx, pkg))
	}
	return ctx.addTargetIfMissing(pkg, func() Future { return Compile(ctx, pkg, deps) })
}

// buildCommand returns a Future repesenting the results of compiling
// pkg as a command and linking the result into pkg.Context.Bindir().
func buildCommand(ctx *Context, pkg *build.Package) Future {
	var deps []Future
	for _, dep := range pkg.Imports {
		// TODO(dfc) use project.Spec
		pkg, err := ctx.ResolvePackage(runtime.GOOS, runtime.GOARCH, dep).Result()
		if err != nil {
			return errFuture{err}
		}
		deps = append(deps, buildPackage(ctx, pkg))
	}
	compile := Compile(ctx, pkg, deps)
	ld := Ld(ctx, pkg, compile)
	return ld
}

// Compile returns a Future representing all the steps required to build a go package.
func Compile(ctx *Context, pkg *build.Package, deps []Future) PkgFuture {
	var gofiles []string
	gofiles = append(gofiles, pkg.GoFiles...)
	var objs []ObjFuture
	if len(pkg.CgoFiles) > 0 {
		cgo, cgofiles := cgo(ctx, pkg, deps)
		deps = append(deps, cgo[0])
		objs = append(objs, cgo...)
		gofiles = append(gofiles, cgofiles...)
	}
	objs = append(objs, Gc(ctx, pkg, deps, gofiles))
	for _, sfile := range pkg.SFiles {
		objs = append(objs, Asm(ctx, pkg, sfile))
	}
	return Pack(ctx, pkg, objs)
}

// ObjFuture represents a Future that produces an Object file.
type ObjFuture interface {
	Future

	// Objfile returns the name of the file that is
	// produced by the Target if successful.
	Objfile() string
}

// PkgFuture represents a Future that produces a pkg (.a) file.
type PkgFuture interface {
	Future

	// pkgfile returns the name of the file that is
	// produced by the Target if successful.
	pkgfile() string
}

// Pack returns a Future representing the result of packing a
// set of Context specific object files into an archive.
func Pack(ctx *Context, pkg *build.Package, deps []ObjFuture) PkgFuture {
	t := &packTarget{
		target: newTarget(ctx, pkg),
		deps:   deps,
	}
	go t.execute()
	return t
}

// Gc returns a Future representing the result of compiling a
// set of gofiles with the Context specified gc Compiler.
func Gc(ctx *Context, pkg *build.Package, deps []Future, gofiles []string) ObjFuture {
	t := &gcTarget{
		target:  newTarget(ctx, pkg),
		deps:    deps,
		gofiles: gofiles,
	}
	go t.execute()
	return t
}

// Asm returns a Future representing the result of assembling
// sfile with the Context specified asssembler.
func Asm(ctx *Context, pkg *build.Package, sfile string) ObjFuture {
	t := &asmTarget{
		target: newTarget(ctx, pkg),
		sfile:  sfile,
	}
	go t.execute()
	return t
}

// Ld returns a Future representing the result of linking a
// Package into a command with the Context provided linker.
func Ld(ctx *Context, pkg *build.Package, afile PkgFuture) Future {
	t := &ldTarget{
		target: newTarget(ctx, pkg),
		afile:  afile,
	}
	go t.execute()
	return t
}

// objdir returns the destination for object files compiled for this Package.
func objdir(ctx *Context, pkg *build.Package) string {
	return filepath.Join(ctx.Workdir(), filepath.FromSlash(pkg.ImportPath), "_obj")
}

// Toolchain represents a standardised set of command line tools
// used to build and test Go programs.
type Toolchain interface {
	Gc(importpath, srcdir, outfile string, files []string) error
	Asm(srcdir, ofile, sfile string) error
	Pack(string, ...string) error
	Ld(string, string) error
	Cc(srcdir, objdir, ofile, cfile string) error

	Cgo(string, []string) error
	Gcc(string, []string) error
	Libgcc() (string, error)

	name() string
}

type toolchain struct {
	cgo string
	gcc string
	*Context
}

func (t *toolchain) Cgo(cwd string, args []string) error {
	return run(cwd, t.cgo, args...)
}

func (t *toolchain) Gcc(cwd string, args []string) error {
	return run(cwd, t.gcc, args...)
}

func (t *toolchain) Libgcc() (string, error) {
	libgcc, err := runOut(".", t.gcc, "-print-libgcc-file-name")
	return strings.Trim(string(libgcc), "\r\n"), err
}

func run(dir, command string, args ...string) error {
	_, err := runOut(dir, command, args...)
	return err
}

func runOut(dir, command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	log.Debugf("cd %s; %s %s", dir, command, strings.Join(args, " "))
	if err != nil {
		log.Errorf("%s", output)
	}
	return output, err
}
