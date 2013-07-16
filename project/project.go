package project

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type result struct {
	*build.Package
	err error
}

// pkgFuture represents an attempt to resolve
// a package import path into a *Package
type pkgFuture struct {
	result chan result
}

func (t *pkgFuture) Result() (*build.Package, error) {
	result := <-t.result
	t.result <- result
	return result.Package, result.err
}

// Resolver resolves package import paths to Packages
type Resolver interface {
	// Resolve returns a *Package representing the source for package path
	// filtered by GOOS and GOARCH.
	ResolvePackage(goos, goarch, path string) *pkgFuture
}

// Project represents a gogo project.
// A gogo project has a simlar layout to a $GOPATH workspace.
// Each gogo project has a standard directory layout starting
// at the project root, which we'll refer too as $PROJECT.
//
// 	$PROJECT/			- the project root
// 	$PROJECT/.gogo/			- used internally by gogo and identifies
//					  the root of the project.
// 	$PROJECT/src/			- base directory for the source of packages
// 	$PROJECT/bin/			- base directory for the compiled binaries
type Project struct {
	root string

	// SrcDirs represents the location of package sources.
	SrcDirs []SrcDir

	sync.Mutex // protects pkgs
	pkgs       map[string]*pkgFuture
}

// NewProject returns a *Project if root represents a valid gogo project.
func NewProject(root string) (*Project, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(root, ".gogo")); err != nil {
		// temporarily disabled to enable easier GOPATH integration
		// return nil, err
	}

	p := &Project{
		root: root,
		pkgs: make(map[string]*pkgFuture),
	}
	p.SrcDirs = []SrcDir{{p, "src"}}
	return p, nil
}

// Root returns the top level directory representing this project.
func (p *Project) Root() string { return p.root }

// Bindir returns the top level directory representing the binary
// directory of this project.
func (p *Project) Bindir() string { return filepath.Join(p.root, "bin") }

// SrcDir represents a directory containing some Go source packages.
type SrcDir struct {
	project *Project
	path    string
}

func (s *SrcDir) SrcDir() string { return filepath.Join(s.project.root, s.path) }

// Find resolves an import path to a source directory
func (s *SrcDir) Find(path string) (string, error) {
	dir := filepath.Join(s.SrcDir(), path)
	_, err := os.Stat(dir)
	return dir, err
}

// FindAdd returns the import paths of all the packages inside this SrcPath.
func (s *SrcDir) FindAll() ([]string, error) {
	return allPackages(s.SrcDir(), "")
}

func allPackages(dir, prefix string) ([]string, error) {
	var pkgs []string
	d, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	files, err := d.Readdir(-1)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		name := f.Name()
		if name[0] == '.' {
			continue
		}
		if f.IsDir() {
			pkgs = append(pkgs, path.Join(prefix, name))
			pp, err := allPackages(filepath.Join(dir, name), path.Join(prefix, name))
			if err != nil {
				return nil, err
			}
			pkgs = append(pkgs, pp...)
		}
	}
	return pkgs, nil
}

// ResolvePackage resolves the import path to a Package.
func (p *Project) ResolvePackage(goos, goarch, path string) *pkgFuture {
	p.Lock()
	defer p.Unlock()
	if f, ok := p.pkgs[path]; ok {
		return f
	}
	pkg := &build.Package{
		ImportPath: path,
		SrcRoot:    filepath.Join(p.Root(), "src"),
	}
	f := &pkgFuture{
		result: make(chan result, 1),
	}
	go func() {
		err := scanFiles(DefaultSpec(), pkg)
		f.result <- result{pkg, err}
	}()
	p.pkgs[path] = f
	return f
}

// scanFiles scans the Package recording all source files relevant to the
// current Spec.
func scanFiles(spec Spec, pkg *build.Package) error {
	//	t0 := time.Now()
	//	defer func() {
	//		c.Record("scanFiles", time.Since(t0))
	//	}()
	files, err := ioutil.ReadDir(filepath.Join(pkg.SrcRoot, pkg.ImportPath))
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

		if !spec.goodOSArchFile(filename) {
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

		r, err := openFile(pkg, filename)
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
		if !spec.shouldBuild(data) {
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
				for _, sp := range decl.Specs {
					switch sp := sp.(type) {
					case *ast.ImportSpec:
						quoted := sp.Path.Value
						path, err := strconv.Unquote(quoted)
						if err != nil {
							return err
						}
						switch path {
						case "":
							return fmt.Errorf("package %q imported blank path: %v", pkg.Name, sp.Pos())
						case "C":
							if isTest {
								return fmt.Errorf("use of cgo in test %s not supported", filename)
							}
							cg := sp.Doc
							if cg == nil && len(decl.Specs) == 1 {
								cg = decl.Doc
							}
							if cg != nil {
								if err := spec.saveCgo(pkg, filename, cg); err != nil {
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
			if spec.cgoEnabled {
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

func openFile(pkg *build.Package, name string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(pkg.SrcRoot, pkg.ImportPath, name))
}
