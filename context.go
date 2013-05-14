package gogo

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
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

	Toolchain
	cgoEnabled bool

	Statistics

	resolver
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
		cgoEnabled: true,
		resolver: resolver{
			pkgs: make(map[string]PkgFuture),
		},
	}
	tc, err := newGcToolchain(ctx)
	if err != nil {
		return nil, err
	}
	ctx.Toolchain = tc
	ctx.resolver.SearchPaths = []string{ctx.stdlib(), workdir}
	return ctx, nil
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
func (ctx *Context) Pkgdir() string {
	return filepath.Join(ctx.workdir, "pkg", ctx.Toolchain.name(), ctx.goos, ctx.goarch)
}

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

// ResolvePackage returns a Package representing the first occurence
// of an import path.
func (c *Context) ResolvePackage(importpath string) (*Package, error) {
	r := c.resolvePackage(c, importpath).Result()
	return r.Package, r.error
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

// scanFiles scans the Package recording all source files relevant to the
// current Context.
func (c *Context) scanFiles(pkg *Package) error {
	t0 := time.Now()
	defer func() {
		c.Record("scanFiles", time.Since(t0))
	}()
	files, err := ioutil.ReadDir(pkg.Srcdir)
	if err != nil {
		return err
	}
	imports := make(map[string]struct{})
	testimports := make(map[string]struct{})
	xtestimports := make(map[string]struct{})
	fset := token.NewFileSet()
	var firstFile string
	for _, file := range files {
		if file.IsDir() {
			// skip
			continue
		}
		filename := file.Name()
		if strings.HasPrefix(filename, "_") || strings.HasPrefix(filename, ".") {
			continue
		}

		ext := filepath.Ext(filename)

		if !c.goodOSArchFile(filename) {
			if ext == ".go" {
				pkg.IgnoredGoFiles = append(pkg.IgnoredGoFiles, filename)
			}
			continue
		}

		switch ext {
		case ".go", ".c", ".s", ".h", ".S", ".swig", ".swigcxx":
			// tentatively okay - read to make sure
		default:
			// skip
			continue
		}

		r, err := pkg.openFile(filename)
		if err != nil {
			return err
		}
		var data []byte
		if strings.HasSuffix(filename, ".go") {
			data, err = readImports(r, false)
		} else {
			data, err = readComments(r)
		}
		r.Close()
		if err != nil {
			return err
		}

		// Look for +build comments to accept or reject the file.
		if !c.shouldBuild(data) {
			if ext == ".go" {
				pkg.IgnoredGoFiles = append(pkg.IgnoredGoFiles, filename)
			}
			continue
		}

		switch ext {
		case ".s":
			pkg.SFiles = append(pkg.SFiles, filename)
			continue
		case ".c":
			pkg.CFiles = append(pkg.CFiles, filename)
			continue
		case ".h":
			pkg.HFiles = append(pkg.HFiles, filename)
			continue
		}

		pf, err := parser.ParseFile(fset, filename, data, parser.ImportsOnly|parser.ParseComments)
		if err != nil {
			return err
		}

		n := pf.Name.Name
		if n == "documentation" {
			pkg.IgnoredGoFiles = append(pkg.IgnoredGoFiles, filename)
			continue
		}

		isTest := strings.HasSuffix(filename, "_test.go")
		var isXTest bool
		if isTest && strings.HasSuffix(n, "_test") {
			isXTest = true
			n = n[:len(n)-len("_test")]
		}
		if pkg.Name == "" {
			pkg.Name = n
			firstFile = filename
		} else if n != pkg.Name {
			return fmt.Errorf("found packages %s (%s) and %s (%s) in %s", pkg.Name, firstFile, n, filename, pkg.ImportPath)
		}
		var isCgo bool
		for _, decl := range pf.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.ImportSpec:
						quoted := spec.Path.Value
						path, err := strconv.Unquote(quoted)
						if err != nil {
							return err
						}
						switch path {
						case "":
							return fmt.Errorf("package %q imported blank path: %v", pkg.Name, spec.Pos())
						case "C":
							if isTest {
								return fmt.Errorf("use of cgo in test %s not supported", filename)
							}
							cg := spec.Doc
							if cg == nil && len(decl.Specs) == 1 {
								cg = decl.Doc
							}
							if cg != nil {
								if err := c.saveCgo(pkg, filename, cg); err != nil {
									return err
								}
							}
							isCgo = true
						default:
							if isXTest {
								xtestimports[path] = struct{}{}
							} else if isTest {
								testimports[path] = struct{}{}
							} else {
								imports[path] = struct{}{}
							}
						}
					default:
						// skip
					}
				}
			default:
				// skip
			}
		}
		if isCgo {
			if c.cgoEnabled {
				pkg.CgoFiles = append(pkg.CgoFiles, filename)
			}
		} else if isXTest {
			pkg.XTestGoFiles = append(pkg.XTestGoFiles, filename)
		} else if isTest {
			pkg.TestGoFiles = append(pkg.TestGoFiles, filename)
		} else {
			pkg.GoFiles = append(pkg.GoFiles, filename)
		}
	}
	if pkg.Name == "" {
		return &build.NoGoError{pkg.ImportPath}
	}
	for i := range imports {
		if stdlib[i] {
			continue
		}
		pkg.Imports = append(pkg.Imports, i)
	}

	for i := range testimports {
		if stdlib[i] {
			continue
		}
		pkg.TestImports = append(pkg.TestImports, i)
	}

	for i := range xtestimports {
		if stdlib[i] {
			continue
		}
		pkg.XTestImports = append(pkg.XTestImports, i)
	}
	return nil
}

// from $GOROOT/src/pkg/go/build/build.go

// saveCgo saves the information from the #cgo lines in the import "C" comment.
// These lines set CFLAGS and LDFLAGS and pkg-config directives that affect
// the way cgo's C code is built.
//
// TODO(rsc): This duplicates code in cgo.
// Once the dust settles, remove this code from cgo.
func (ctx *Context) saveCgo(pkg *Package, filename string, cg *ast.CommentGroup) error {
	text := cg.Text()
	for _, line := range strings.Split(text, "\n") {
		orig := line

		// Line is
		//      #cgo [GOOS/GOARCH...] LDFLAGS: stuff
		//
		line = strings.TrimSpace(line)
		if len(line) < 5 || line[:4] != "#cgo" || (line[4] != ' ' && line[4] != '\t') {
			continue
		}

		// Split at colon.
		line = strings.TrimSpace(line[4:])
		i := strings.Index(line, ":")
		if i < 0 {
			return fmt.Errorf("%s: invalid #cgo line: %s", filename, orig)
		}
		line, argstr := line[:i], line[i+1:]

		// Parse GOOS/GOARCH stuff.
		f := strings.Fields(line)
		if len(f) < 1 {
			return fmt.Errorf("%s: invalid #cgo line: %s", filename, orig)
		}

		cond, verb := f[:len(f)-1], f[len(f)-1]
		if len(cond) > 0 {
			ok := false
			for _, c := range cond {
				if ctx.match(c) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}

		args, err := splitQuoted(argstr)
		if err != nil {
			return fmt.Errorf("%s: invalid #cgo line: %s", filename, orig)
		}
		for _, arg := range args {
			if !safeName(arg) {
				return fmt.Errorf("%s: malformed #cgo argument: %s", filename, arg)
			}
		}

		switch verb {
		case "CFLAGS":
			pkg.CgoCFLAGS = append(pkg.CgoCFLAGS, args...)
		case "LDFLAGS":
			pkg.CgoLDFLAGS = append(pkg.CgoLDFLAGS, args...)
		case "pkg-config":
			pkg.CgoPkgConfig = append(pkg.CgoPkgConfig, args...)
		default:
			return fmt.Errorf("%s: invalid #cgo verb: %s", filename, orig)
		}
	}
	return nil
}
