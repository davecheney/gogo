package gogo

import (
	"path"
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
		project: p,
		name:    path.Base(pp),
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
