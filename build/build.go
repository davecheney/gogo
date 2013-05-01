// Package gogo/build provides functions for building and testing
// Go packages.
package build

import (
	"path/filepath"

	"github.com/davecheney/gogo"
)

// Build returns a Future representing the result of compiling the
// package pkg, and its dependencies.
// If pkg is a command, then the results of build include linking
// the final binary into pkg.Context.Bindir().
func Build(pkg *gogo.Package) gogo.Future {
	ctx := pkg.Context
	if pkg.Name == "main" {
		return buildCommand(ctx, pkg)
	}
	return buildPackage(ctx, pkg)
}

// buildPackage returns a Future repesenting the results of compiling
// pkg and its dependencies.
func buildPackage(ctx *gogo.Context, pkg *gogo.Package) gogo.Future {
	var deps []gogo.Future
	for _, dep := range pkg.Imports {
		deps = append(deps, buildPackage(ctx, dep))
	}
	if _, ok := ctx.Targets[pkg]; !ok {
		Compile := Compile(pkg, deps, false)
		ctx.Targets[pkg] = Compile
	}
	return ctx.Targets[pkg]
}

// buildCommand returns a Future repesenting the results of compiling
// pkg as a command and linking the result into pkg.Context.Bindir().
func buildCommand(ctx *gogo.Context, pkg *gogo.Package) gogo.Future {
	var deps []gogo.Future
	for _, dep := range pkg.Imports {
		deps = append(deps, buildPackage(ctx, dep))
	}
	Compile := Compile(pkg, deps, false)
	ld := Ld(pkg, Compile)
	return ld
}

// Compile returns a Future representing all the steps required to build a go package.
func Compile(pkg *gogo.Package, deps []gogo.Future, includeTests bool) gogo.Future {
	var gofiles []string
	gofiles = append(gofiles, pkg.GoFiles...)
	var objs []ObjFuture
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

// ObjFuture represents a Future that produces an Object file.
type ObjFuture interface {
	gogo.Future

	// Objfile returns the name of the file that is
	// produced by the Future if successful.
	Objfile() string
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
func Pack(pkg *gogo.Package, deps []ObjFuture) pkgFuture {
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

// Gc returns a Future representing the result of compiling a
// set of gofiles with the Context specified gc Compiler.
func Gc(pkg *gogo.Package, deps []gogo.Future, gofiles []string) ObjFuture {
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

// Asm returns a Future representing the result of assembling
// sfile with the Context specified asssembler.
func Asm(pkg *gogo.Package, sfile string) ObjFuture {
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

// objdir returns the destination for object files compiled for this Package.
func objdir(ctx *gogo.Context, pkg *gogo.Package) string {
	return filepath.Join(ctx.Workdir(), filepath.FromSlash(pkg.ImportPath), "_obj")
}
