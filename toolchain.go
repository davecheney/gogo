package gogo

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Toolchain interface {
	Gc(*Context, *Package) error
	Pack(*Context, *Package) error
	Ld(*Context, *Package) error
}

type gcToolchain struct {
}

func (tc *gcToolchain) Gc(ctx *Context, pkg *Package) error {
	objdir := ctx.objdir(pkg)
	if err := os.MkdirAll(objdir, 0777); err != nil {
		return err
	}
	args := []string{"-p", pkg.path, "-I", pkg.Project.pkgdir(ctx), "-o", filepath.Join(objdir, "_go_.6")}
	for _, f := range pkg.GoFiles() {
		args = append(args, f)
	}
	tooldir := ctx.tooldir()
	return run(pkg.srcdir(), filepath.Join(tooldir, "6g"), args...)
}

func (tc *gcToolchain) Pack(ctx *Context, pkg *Package) error {
	objdir := ctx.objdir(pkg)
	tooldir := ctx.tooldir()
	pkgfile := pkg.pkgfile(ctx)
	pkgdir := filepath.Dir(pkgfile)
	if err := os.MkdirAll(pkgdir, 0777); err != nil {
		return err
	}
	args := []string{"grcP", ctx.basedir, pkgfile, filepath.Join(objdir, "_go_.6")}
	return run(pkgdir, filepath.Join(tooldir, "pack"), args...)
}

func (tc *gcToolchain) Ld(ctx *Context, pkg *Package) error {
	objdir := ctx.objdir(pkg)
	tooldir := ctx.tooldir()
	args := []string{"-o", filepath.Join(objdir, "a.out"), "-L", pkg.Project.pkgdir(ctx), "-L", ctx.stdlib(), pkg.pkgfile(ctx)}
	return run(objdir, filepath.Join(tooldir, "6l"), args...)
}

func run(dir, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("[%s] %s %s", dir, command, strings.Join(args, " "))
	return cmd.Run()
}
