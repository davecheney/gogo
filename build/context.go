package build

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/davecheney/gogo/log"
	"github.com/davecheney/gogo/project"
)

type Context struct {
	project.Resolver
	goroot, goos, goarch string
	workdir, archchar    string

	targetCache

	project.Statistics

	Toolchain
	SearchPaths []string
}

type targetCache struct {
	sync.Mutex
	m map[*project.Package]Future
}

func (c *targetCache) addTargetIfMissing(pkg *project.Package, f func() Future) Future {
	c.Lock()
	if c.m == nil {
		c.m = make(map[*project.Package]Future)
	}
	target, ok := c.m[pkg]
	if !ok {
		target = f()
		c.m[pkg] = target
	}
	return target
}

// NewDefaultContext returns a Context that represents the version
// of Go that compiled gogo.
func NewDefaultContext(p *project.Project) (*Context, error) {
	return NewContext(p, "gc", runtime.GOROOT(), runtime.GOOS, runtime.GOARCH)
}

// NewContext returns a Context that can be used to build *Project
// using the specified goroot, goos, and goarch.
func NewContext(p *project.Project, toolchain, goroot, goos, goarch string) (*Context, error) {
	workdir, err := ioutil.TempDir("", "gogo")
	if err != nil {
		return nil, err
	}
	archchar, err := build.ArchChar(goarch)
	if err != nil {
		return nil, err
	}
	ctx := &Context{
		Resolver: p,
		goroot:   goroot,
		goos:     goos,
		goarch:   goarch,
		workdir:  workdir,
		archchar: archchar,
		// cgoEnabled: true,
	}
	f, ok := toolchains[toolchain]
	if !ok {
		return nil, fmt.Errorf("no toolchain %q registered", toolchain)
	}
	tc, err := f(ctx)
	if err != nil {
		return nil, err
	}
	ctx.Toolchain = tc
	ctx.SearchPaths = []string{ctx.stdlib(), workdir}
	return ctx, nil
}

// Destroy removes any temporary files associated with this Context.
func (ctx *Context) Destroy() error {
	return os.RemoveAll(ctx.workdir)
}

// Workdir returns the path to the temporary working directory for this context.
// The contents of Workdir are removed when the Destroy method is invoked.
func (ctx *Context) Workdir() string { return ctx.workdir }

// Bindir returns the path when final binary executables will be stored.
func (ctx *Context) Bindir() string {
	// TODO(dfc) hack, don't want to make Context depend on Project
	return filepath.Join(ctx.Workdir(), ctx.goos, ctx.goarch)
}

// Mkdir creates a directory named path, along with any necessary
// parents, and returns nil, or else returns an error.  If path is
// already a directory, MkdirAll does nothing and returns nil.
func (c *Context) Mkdir(path string) error {
	// TODO(dfc) insert cache
	log.Debugf("mkdir %q", path)
	return os.MkdirAll(path, 0777)
}

// Pkgdir returns the path to the temporary location where intermediary packages
// are created during build and test phases.
func (ctx *Context) Pkgdir() string {
	return filepath.Join(ctx.workdir, "pkg", ctx.Toolchain.name(), ctx.goos, ctx.goarch)
}

func (ctx *Context) stdlib() string { return filepath.Join(ctx.goroot, "pkg", ctx.goos+"_"+ctx.goarch) }
