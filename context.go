package gogo

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"
)

type Context struct {
	goos, goarch string
	basedir      string
	targets      map[*Package]Target
}

func newContext() (*Context, error) {
	basedir, err := ioutil.TempDir("", "gogo")
	if err != nil {
		return nil, err
	}
	return &Context{
		goos:    runtime.GOOS,
		goarch:  runtime.GOARCH,
		basedir: basedir,
		targets: make(map[*Package]Target),
	}, nil
}

func BuildPackages(pkgs ...*Package) error {
	ctx, err := newContext()
	if err != nil {
		return err
	}
	return ctx.BuildPackages(pkgs...)
}

func (ctx *Context) BuildPackages(pkgs ...*Package) error {
	for _, pkg := range pkgs {
		var tt []Target
		log.Printf("building: %v", pkg.path)
		if pkg.name == "main" {
			tt = buildCommand(ctx, pkg)
		} else {
			tt = buildPackage(ctx, pkg)
		}
		for _, t := range tt {
			if err := t.Wait(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ctx *Context) objdir(pkg *Package) string { return filepath.Join(ctx.basedir, pkg.path, "_obj") }
func (ctx *Context) tooldir() string {
	return filepath.Join(runtime.GOROOT(), "pkg", "tool", ctx.goos+"_"+ctx.goarch)
}
func (ctx *Context) stdlib() string {
	return filepath.Join(runtime.GOROOT(), "pkg", ctx.goos+"_"+ctx.goarch)
}
