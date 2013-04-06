package gogo

import (
	"path"
)

type Project struct {
	root string
}

func NewProject(root string) *Project {
	return &Project{
		root: root,
	}
}

func (p *Project) ResolvePackage(pp string) (*Package, error) {
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

	return pkg, nil
}
