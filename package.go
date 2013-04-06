package gogo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Package struct {
	project    *Project
	name, path string
	goFiles    []string
	cFiles     []string
	hFiles     []string
	sFiles     []string

	imports []string

	testGoFiles []string
}

func (p *Package) Path() string      { return p.path }
func (p *Package) Project() *Project { return p.project }
func (p *Package) Imports() []string { return p.imports }
func (p *Package) String() string    { return fmt.Sprintf("package %q", p.Path()) }

func (p *Package) srcdir() string {
	return filepath.Join(p.project.root, "src", p.path)
}

// readFiles populates the various package file lists
func (p *Package) readFiles() error {
	files, err := ioutil.ReadDir(p.srcdir())
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			// skip
			continue
		}
		name := file.Name()
		if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
			continue
		}
		switch ext := filepath.Ext(name); ext {
		case ".go":
			if strings.HasSuffix(name, "_test.go") {
				p.testGoFiles = append(p.testGoFiles, name)
				continue
			}
			p.goFiles = append(p.goFiles, name)
		case ".c":
			p.cFiles = append(p.cFiles, name)
		case ".h":
			p.hFiles = append(p.hFiles, name)
		case ".s":
			p.sFiles = append(p.sFiles, name)
		default:
			log.Printf("skipping unknown extension %q", ext)
		}

	}
	return nil
}

func (p *Package) openFile(name string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(p.srcdir(), name))
}

// readImports populates the import paths of this package
func (p *Package) readImports() error {
	fset := token.NewFileSet()
	imports := make(map[string]struct{})
	for _, file := range p.goFiles {
		r, err := p.openFile(file)
		if err != nil {
			return err
		}
		defer r.Close()
		pf, err := parser.ParseFile(fset, file, r, parser.ImportsOnly)
		if err != nil {
			return err
		}
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
						if path == "C" {
							// skip
							continue
						}
						if path == "" {
							return fmt.Errorf("package %q imported blank path: %s", spec.Pos())
						}
						if path == "." {
							return fmt.Errorf("package %q imported dot path: %s", spec.Pos())
						}
						imports[path] = struct{}{}
					default:
						// skip
					}
				}
			default:
				// skip

			}
		}
	}
	for i, _ := range imports {
		p.imports = append(p.imports, i)
	}
	return nil
}
