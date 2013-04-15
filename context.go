package gogo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

// Context represents the execution of a series of Targets
// for a Project.
type Context struct {
	*Project
	goroot, goos, goarch string
	basedir              string
	Targets              map[*Package]Target
	Toolchain
	SearchPaths []string
}

// NewDefaultContext returns a Context that represents the version
// of Go that compiled this package.
func NewDefaultContext(p *Project) (*Context, error) {
	return newContext(p, runtime.GOROOT(), runtime.GOOS, runtime.GOARCH)
}

func newContext(p *Project, goroot, goos, goarch string) (*Context, error) {
	basedir, err := ioutil.TempDir("", "gogo")
	if err != nil {
		return nil, err
	}
	ctx := &Context{
		Project: p,
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
	ctx.SearchPaths = []string{ctx.stdlib(), ctx.Pkgdir()}
	return ctx, nil
}

// Destroy removes any temporary files associated with this Context.
func (ctx *Context) Destroy() error {
	return os.RemoveAll(ctx.basedir)
}

// Objdir returns the destination for object files compiled for this Package.
func (ctx *Context) Objdir(pkg *Package) string {
	return filepath.Join(ctx.basedir, filepath.FromSlash(pkg.ImportPath), "_obj")
}

// TestObjDir returns the destination for test object files compiled for this Package.
func (ctx *Context) TestObjdir(pkg *Package) string {
	return filepath.Join(ctx.basedir, filepath.FromSlash(pkg.ImportPath), "_test")
}

func (ctx *Context) Pkgdir() string { return filepath.Join(ctx.basedir, "pkg", ctx.goos, ctx.goarch) }
func (ctx *Context) Bindir() string {
	return filepath.Join(ctx.Project.root, "bin", ctx.goos, ctx.goarch)
}
func (ctx *Context) stdlib() string { return filepath.Join(ctx.goroot, "pkg", ctx.goos+"_"+ctx.goarch) }
