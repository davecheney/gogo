package gogo

import (
	"go/build"
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
	workdir              string
	archchar             string
	Targets              map[*Package]Target

	// pkgs is a map of import paths to resolved Packages within
	// the scope of this context.
	pkgs map[string]*Package
	Toolchain
	SearchPaths []string
}

// NewDefaultContext returns a Context that represents the version
// of Go that compiled this package.
func NewDefaultContext(p *Project) (*Context, error) {
	return newContext(p, runtime.GOROOT(), runtime.GOOS, runtime.GOARCH)
}

func newContext(p *Project, goroot, goos, goarch string) (*Context, error) {
	workdir, err := ioutil.TempDir("", "gogo")
	if err != nil {
		return nil, err
	}
	archchar, err := build.ArchChar(goarch)
	if err != nil {
		return nil, err
	}
	ctx := &Context{
		Project:  p,
		goroot:   goroot,
		goos:     goos,
		goarch:   goarch,
		workdir:  workdir,
		archchar: archchar,
		Targets:  make(map[*Package]Target),
		pkgs:     make(map[string]*Package),
	}
	tc, err := newGcToolchain(ctx)
	if err != nil {
		return nil, err
	}
	ctx.Toolchain = tc
	ctx.SearchPaths = []string{ctx.stdlib(), workdir}
	return ctx, nil
}

// ResolvePackage resolves the import path to a Package.
func (c *Context) ResolvePackage(path string) (*Package, error) {
	if pkg, ok := c.pkgs[path]; ok {
		return pkg, nil
	}
	pkg, err := newPackage(c, path)
	if err != nil {
		return nil, err
	}
	c.pkgs[path] = pkg
	return pkg, nil
}

// Destroy removes any temporary files associated with this Context.
func (ctx *Context) Destroy() error {
	return os.RemoveAll(ctx.workdir)
}

// Objdir returns the destination for object files compiled for this Package.
func (ctx *Context) Objdir(pkg *Package) string {
	return filepath.Join(ctx.workdir, filepath.FromSlash(pkg.ImportPath()), "_obj")
}

// TestObjDir returns the destination for test object files compiled for this Package.
func (ctx *Context) TestObjdir(pkg *Package) string {
	return filepath.Join(ctx.workdir, filepath.FromSlash(pkg.ImportPath()), "_test")
}

func (ctx *Context) Workdir() string { return ctx.workdir }

func (ctx *Context) Pkgdir() string { return filepath.Join(ctx.workdir, "pkg", ctx.goos, ctx.goarch) }
func (ctx *Context) Bindir() string {
	return filepath.Join(ctx.Project.Bindir(), ctx.goos, ctx.goarch)
}
func (ctx *Context) stdlib() string { return filepath.Join(ctx.goroot, "pkg", ctx.goos+"_"+ctx.goarch) }
