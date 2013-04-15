package gogo

import (
	"path/filepath"
)

type Project struct {
	root string
	pkgs map[string]*Package
}

func NewProject(root string) (*Project, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Project{
		root: root,
		pkgs: make(map[string]*Package),
	}, nil
}

// Root returns the top level directory representing this project.
func (p *Project) Root() string { return p.root }

func (p *Project) ResolvePackage(path string) (*Package, error) {
	if pkg, ok := p.pkgs[path]; ok {
		return pkg, nil
	}
	pkg := &Package{
		Project:    p,
		name:       filepath.Base(path),
		importPath: path,
	}
	if err := pkg.readFiles(); err != nil {
		return nil, err
	}

	if err := pkg.readImports(); err != nil {
		return nil, err
	}
	p.pkgs[path] = pkg
	return pkg, nil
}

func (p *Project) srcdir() string { return filepath.Join(p.root, "src") }
