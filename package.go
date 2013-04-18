package gogo

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Package describes a Go package.
type Package struct {
	// The Context that resolved this package.
	*Context

	// The name of the package
	name string

	// The import path of the package.
	importPath string

	// The path to the source of this package relative to Project.Root()
	srcdir string

	// Source files
	GoFiles        []string // .go source files (excluding CgoFiles, TestGoFiles, XTestGoFiles)
	CgoFiles       []string // .go source files that import "C"
	IgnoredGoFiles []string // .go source files ignored for this build

	// Cgo directives
	CgoPkgConfig []string // Cgo pkg-config directives
	CgoCFLAGS    []string // Cgo CFLAGS directives
	CgoLDFLAGS   []string // Cgo LDFLAGS directives

	// Test information
	TestGoFiles  []string // _test.go files in package
	XTestGoFiles []string // _test.go files outside package

	Imports []*Package
}

// newPackage constructs a new Package for the Context context.
func newPackage(context *Context, path string) (*Package, error) {
	pkg := &Package{
		Context:    context,
		importPath: path,
		srcdir:     filepath.Join("src", path),
	}
	files, err := ioutil.ReadDir(pkg.Srcdir())
	if err != nil {
		return nil, err
	}
	return pkg, pkg.scanFiles(files)
}

// Name returns the name of the package.
func (p *Package) Name() string { return p.name }

// ImportPath returns the import path that would is used to import this package into another.
func (p *Package) ImportPath() string { return p.importPath }

// Srcdir returns the path to this package.
func (p *Package) Srcdir() string { return filepath.Join(p.Project.Root(), p.srcdir) }

func (p *Package) openFile(name string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(p.Srcdir(), name))
}

// scanFiles scans the Package recording all source files relevant to the
// current Context.
func (p *Package) scanFiles(files []os.FileInfo) error {
	imports := make(map[string]struct{})
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
		switch ext {
		case ".go", ".c", ".s", ".h", ".S", ".swig", ".swigcxx":
			// tentatively okay - read to make sure
		default:
			// skip
			continue
		}
		// This file is part of the part of the build Context.
		switch ext {
		// add non go files here
		}
		r, err := p.openFile(filename)
		if err != nil {
			return err
		}
		defer r.Close()
		pf, err := parser.ParseFile(fset, filename, r, parser.ImportsOnly)
		if err != nil {
			return err
		}
		pkg := pf.Name.Name
		if pkg == "documentation" {
			p.IgnoredGoFiles = append(p.IgnoredGoFiles, filename)
			continue
		}

		isTest := strings.HasSuffix(filename, "_test.go")
		var isXTest bool
		if isTest && strings.HasSuffix(pkg, "_test") {
			isXTest = true
			pkg = pkg[:len(pkg)-len("_test")]
		}
		if p.name == "" {
			p.name = pkg
			firstFile = filename
		} else if pkg != p.name {
			return fmt.Errorf("found packages %s (%s) and %s (%s) in %s", p.name, firstFile, pkg, filename, p.importPath)
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
							return fmt.Errorf("package %q imported blank path: %s", spec.Pos())
						case ".":
							return fmt.Errorf("package %q imported dot path: %s", spec.Pos())
						case "C":
							if isTest {
								return fmt.Errorf("use of cgo in test %s not supported", filename)
							}
							cg := spec.Doc
							if cg == nil && len(decl.Specs) == 1 {
								cg = decl.Doc
							}
							if cg != nil {
								if err := p.saveCgo(filename, cg); err != nil {
									return err
								}
							}
							isCgo = true
						default:
							imports[path] = struct{}{}
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
			if p.cgoEnabled {
				p.CgoFiles = append(p.CgoFiles, filename)
			}
		} else if isXTest {
			p.XTestGoFiles = append(p.XTestGoFiles, filename)
		} else if isTest {
			p.TestGoFiles = append(p.TestGoFiles, filename)
		} else {
			p.GoFiles = append(p.GoFiles, filename)
		}
	}
	if p.name == "" {
		return &build.NoGoError{p.importPath}
	}
	for i, _ := range imports {
		if stdlib[i] {
			// skip
			continue
		}
		pkg, err := p.ResolvePackage(i)
		if err != nil {
			return err
		}
		p.Imports = append(p.Imports, pkg)
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
func (p *Package) saveCgo(filename string, cg *ast.CommentGroup) error {
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
				if p.Context.match(c) {
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
			p.CgoCFLAGS = append(p.CgoCFLAGS, args...)
		case "LDFLAGS":
			p.CgoLDFLAGS = append(p.CgoLDFLAGS, args...)
		case "pkg-config":
			p.CgoPkgConfig = append(p.CgoPkgConfig, args...)
		default:
			return fmt.Errorf("%s: invalid #cgo verb: %s", filename, orig)
		}
	}
	return nil
}

// Objdir returns the destination for object files compiled for this Package.
func (p *Package) Objdir() string {
	return filepath.Join(p.Context.workdir, filepath.FromSlash(p.ImportPath()), "_obj")
}

// TestObjDir returns the destination for test object files compiled for this Package.
func (p *Package) TestObjdir() string {
	return filepath.Join(p.Context.workdir, filepath.FromSlash(p.ImportPath()), "_test")
}
