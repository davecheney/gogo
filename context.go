package gogo

import (
	"io/ioutil"
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
		tt := buildPackage(ctx, pkg)
		for _, t := range tt {
			if err := t.Wait(); err != nil {
				return err
			}
		}
	}
	return nil
}
