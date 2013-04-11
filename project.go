package gogo

import (
	"io/ioutil"
	"log"
	"path/filepath"
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

func (p *Project) AllPackages() ([]*Package, error) {
	dirs, err := findAllDirs(p.srcdir())
	if err != nil {
		return nil, err
	}
	var pkgs []*Package
	for _, dir := range dirs {
		pkg, err := p.ResolvePackage(dir)
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

func findAllDirs(dir string) ([]string, error) {
	log.Printf("scanning directory %v", dir)
	ents, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var dirs = []string{dir}
	for _, ent := range ents {
		if !ent.IsDir() {
			continue
		}
		if ent.Name()[0] == '.' {
			continue
		}
		d, err := findAllDirs(filepath.Join(dir, ent.Name()))
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, d...)
	}
	return dirs, nil
}

func (p *Project) Toolchain() Toolchain { return new(gcToolchain) }

func (p *Project) srcdir() string { return filepath.Join(p.root, "src") }
func (p *Project) pkgdir(ctx *Context) string {
	return filepath.Join(p.root, "pkg", ctx.goos, ctx.goarch)
}
