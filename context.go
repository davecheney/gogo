package gogo

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
)

type Context struct {
	goos, goarch string
	basedir      string
	Targets      map[*Package]Target
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
		Targets: make(map[*Package]Target),
	}, nil
}

func (ctx *Context) objdir(pkg *Package) string { return filepath.Join(ctx.basedir, pkg.path, "_obj") }
func (ctx *Context) tooldir() string {
	return filepath.Join(runtime.GOROOT(), "pkg", "tool", ctx.goos+"_"+ctx.goarch)
}
func (ctx *Context) stdlib() string {
	return filepath.Join(runtime.GOROOT(), "pkg", ctx.goos+"_"+ctx.goarch)
}
