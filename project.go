package gogo

import (
	"path/filepath"
	"runtime"
)

type Project struct {
	root string
	pkgs map[string]*Package
}

func NewProject(root string) *Project {
	return &Project{
		root: root,
		pkgs: make(map[string]*Package),
	}
}

func (p *Project) ResolvePackage(pp string) (*Package, error) {
	if pkg, ok := p.pkgs[pp]; ok {
		return pkg, nil
	}
	pkg := &Package{
		Project: p,
		path:    pp,
	}
	if err := pkg.readFiles(); err != nil {
		return nil, err
	}

	if err := pkg.readImports(); err != nil {
		return nil, err
	}
	p.pkgs[pp] = pkg
	return pkg, nil
}

func (p *Project) NewContext() (*Context, error) {
	return newContext(p, runtime.GOROOT(), runtime.GOOS, runtime.GOARCH)
}

func (p *Project) srcdir() string { return filepath.Join(p.root, "src") }
