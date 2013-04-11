package gogo

import (
	"io/ioutil"
	"path/filepath"
)

type Context struct {
	goroot, goos, goarch string
	basedir              string
	Targets              map[*Package]Target
	Toolchain
	SearchPaths []string
}

func newContext(goroot, goos, goarch string) (*Context, error) {
	basedir, err := ioutil.TempDir("", "gogo")
	if err != nil {
		return nil, err
	}
	ctx := &Context{
		goroot:  goroot,
		goos:    goos,
		goarch:  goarch,
		basedir: basedir,
		Targets: make(map[*Package]Target),
	}
	tc, err := newGcToolchain(ctx)
	if err != nil {
		return nil, err
	}
	ctx.Toolchain = tc
	ctx.SearchPaths = []string{ctx.stdlib(), ctx.basedir}
	return ctx, nil
}

func (ctx *Context) Objdir(pkg *Package) string { return filepath.Join(ctx.basedir, pkg.path, "_obj") }
func (ctx *Context) stdlib() string {
	return filepath.Join(ctx.goroot, "pkg", ctx.goos+"_"+ctx.goarch)
}
