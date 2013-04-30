package gogo

import (
	"bytes"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"github.com/davecheney/gogo/log"
)

// Context represents a view over a set of Packages for a Project.
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

	Statistics
}

// NewDefaultContext returns a Context that represents the version
// of Go that compiled gogo.
func NewDefaultContext(p *Project) (*Context, error) {
	return NewContext(p, runtime.GOROOT(), runtime.GOOS, runtime.GOARCH)
}

// NewContext returns a Context that can be used to build *Project
// using the specified goroot, goos, and goarch.
func NewContext(p *Project, goroot, goos, goarch string) (*Context, error) {
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
	pkg, err := newPackage(c, filepath.Join(c.SrcPaths[0].Srcdir(), path), path)
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

// Workdir returns the path to the temporary working directory for this context.
// The contents of Workdir are removed when the Destroy method is invoked.
func (ctx *Context) Workdir() string { return ctx.workdir }

// Pkgdir returns the path to the temporary location where intermediary packages
// are created during build and test phases.
func (ctx *Context) Pkgdir() string { return filepath.Join(ctx.workdir, "pkg", ctx.goos, ctx.goarch) }

// Bindir returns the path when final binary executables will be stored.
func (ctx *Context) Bindir() string {
	return filepath.Join(ctx.Project.Bindir(), ctx.goos, ctx.goarch)
}

// stdlib returns the path to the standard library packages.
func (ctx *Context) stdlib() string { return filepath.Join(ctx.goroot, "pkg", ctx.goos+"_"+ctx.goarch) }

// Mkdir creates a directory named path, along with any necessary
// parents, and returns nil, or else returns an error.  If path is
// already a directory, MkdirAll does nothing and returns nil.
func (c *Context) Mkdir(path string) error {
	// TODO(dfc) insert cache
	log.Debugf("mkdir %q", path)
	return os.MkdirAll(path, 0777)
}

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

var slashslash = []byte("//")

// shouldBuild reports whether it is okay to use this file,
// The rule is that in the file's leading run of // comments
// and blank lines, which must be followed by a blank line
// (to avoid including a Go package clause doc comment),
// lines beginning with '// +build' are taken as build directives.
//
// The file is accepted only if each such line lists something
// matching the file.  For example:
//
//      // +build windows linux
//
// marks the file as applicable only on Windows and Linux.
//
func (ctxt *Context) shouldBuild(content []byte) bool {
	// Pass 1. Identify leading run of // comments and blank lines,
	// which must be followed by a blank line.
	end := 0
	p := content
	for len(p) > 0 {
		line := p
		if i := bytes.IndexByte(line, '\n'); i >= 0 {
			line, p = line[:i], p[i+1:]
		} else {
			p = p[len(p):]
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 { // Blank line
			end = len(content) - len(p)
			continue
		}
		if !bytes.HasPrefix(line, slashslash) { // Not comment line
			break
		}
	}
	content = content[:end]

	// Pass 2.  Process each line in the run.
	p = content
	for len(p) > 0 {
		line := p
		if i := bytes.IndexByte(line, '\n'); i >= 0 {
			line, p = line[:i], p[i+1:]
		} else {
			p = p[len(p):]
		}
		line = bytes.TrimSpace(line)
		if bytes.HasPrefix(line, slashslash) {
			line = bytes.TrimSpace(line[len(slashslash):])
			if len(line) > 0 && line[0] == '+' {
				// Looks like a comment +line.
				f := strings.Fields(string(line))
				if f[0] == "+build" {
					ok := false
					for _, tok := range f[1:] {
						if ctxt.match(tok) {
							ok = true
							break
						}
					}
					if !ok {
						return false // this one doesn't match
					}
				}
			}
		}
	}
	return true // everything matches
}
