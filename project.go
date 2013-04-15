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

// Bindir returns the top level directory representing the binary
// directory of this project.
func (p *Project) Bindir() string { return filepath.Join(p.root, "bin") }

func (p *Project) ResolvePackage(path string) (*Package, error) {
	if pkg, ok := p.pkgs[path]; ok {
		return pkg, nil
	}
	pkg, err := newPackage(p, path)
	if err != nil {
		return nil, err
	}
	p.pkgs[path] = pkg
	return pkg, nil
}
