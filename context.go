package gogo

import (
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

// Context represents the execution of a series of Targets
// for a Project.
type Context struct {
	*Project
	goroot, goos, goarch string
	workdir              string
	archchar             string
	Targets              map[*Package]Future

	// The build and release tags specify build constraints
	// that should be considered satisfied when processing +build lines.
	// Clients creating a new context may customize BuildTags, which
	// defaults to empty, but it is usually an error to customize ReleaseTags,
	// which defaults to the list of Go releases the current release is compatible with.
	// In addition to the BuildTags and ReleaseTags, build constraints
	// consider the values of GOARCH and GOOS as satisfied tags.
	BuildTags   []string
	ReleaseTags []string

	// pkgs is a map of import paths to resolved Packages within
	// the scope of this context.
	pkgs map[string]*Package
	Toolchain
	SearchPaths []string
	cgoEnabled  bool
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
		Project:    p,
		goroot:     goroot,
		goos:       goos,
		goarch:     goarch,
		workdir:    workdir,
		archchar:   archchar,
		Targets:    make(map[*Package]Future),
		pkgs:       make(map[string]*Package),
		cgoEnabled: true,
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

func (ctx *Context) Workdir() string { return ctx.workdir }

func (ctx *Context) Pkgdir() string { return filepath.Join(ctx.workdir, "pkg", ctx.goos, ctx.goarch) }
func (ctx *Context) Bindir() string {
	return filepath.Join(ctx.Project.Bindir(), ctx.goos, ctx.goarch)
}
func (ctx *Context) stdlib() string { return filepath.Join(ctx.goroot, "pkg", ctx.goos+"_"+ctx.goarch) }

// from $GOROOT/src/pkg/go/build/build.go

// goodOSArchFile returns false if the name contains a $GOOS or $GOARCH
// suffix which does not match the current system.
// The recognized name formats are:
//
//     name_$(GOOS).*
//     name_$(GOARCH).*
//     name_$(GOOS)_$(GOARCH).*
//     name_$(GOOS)_test.*
//     name_$(GOARCH)_test.*
//     name_$(GOOS)_$(GOARCH)_test.*
//
func (ctxt *Context) goodOSArchFile(name string) bool {
	if dot := strings.Index(name, "."); dot != -1 {
		name = name[:dot]
	}
	l := strings.Split(name, "_")
	if n := len(l); n > 0 && l[n-1] == "test" {
		l = l[:n-1]
	}
	n := len(l)
	if n >= 2 && knownOS[l[n-2]] && knownArch[l[n-1]] {
		return l[n-2] == ctxt.goos && l[n-1] == ctxt.goarch
	}
	if n >= 1 && knownOS[l[n-1]] {
		return l[n-1] == ctxt.goos
	}
	if n >= 1 && knownArch[l[n-1]] {
		return l[n-1] == ctxt.goarch
	}
	return true
}

var knownOS = make(map[string]bool)
var knownArch = make(map[string]bool)

func init() {
	for _, v := range strings.Fields(goosList) {
		knownOS[v] = true
	}
	for _, v := range strings.Fields(goarchList) {
		knownArch[v] = true
	}
}

// match returns true if the name is one of:
//
//      $GOOS
//      $GOARCH
//      cgo (if cgo is enabled)
//      !cgo (if cgo is disabled)
//      ctxt.Compiler
//      !ctxt.Compiler
//      tag (if tag is listed in ctxt.BuildTags or ctxt.ReleaseTags)
//      !tag (if tag is not listed in ctxt.BuildTags or ctxt.ReleaseTags)
//      a comma-separated list of any of these
//
func (c *Context) match(name string) bool {
	if name == "" {
		return false
	}
	if i := strings.Index(name, ","); i >= 0 {
		// comma-separated list
		return c.match(name[:i]) && c.match(name[i+1:])
	}
	if strings.HasPrefix(name, "!!") { // bad syntax, reject always
		return false
	}
	if strings.HasPrefix(name, "!") { // negation
		return len(name) > 1 && !c.match(name[1:])
	}

	// Tags must be letters, digits, underscores or dots.
	// Unlike in Go identifiers, all digits are fine (e.g., "386").
	for _, c := range name {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '.' {
			return false
		}
	}

	// special tags
	if c.cgoEnabled && name == "cgo" {
		return true
	}
	if name == c.goos || name == c.goarch || name == c.Toolchain.name() {
		return true
	}

	// other tags
	for _, tag := range c.BuildTags {
		if tag == name {
			return true
		}
	}
	for _, tag := range c.ReleaseTags {
		if tag == name {
			return true
		}
	}

	return false
}
