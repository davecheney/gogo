package gogo

import (
	"os"
	"path"
	"path/filepath"
)

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

	// SrcPaths represents the location of package sources.
	SrcPaths []SrcPath
}

// NewProject returns a *Project if root represents a valid gogo project.
func NewProject(root string) (*Project, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(root, ".gogo")); err != nil {
		return nil, err
	}
	p := &Project{
		root: root,
	}
	p.SrcPaths = []SrcPath{SrcPath{p, "src"}}
	return p, nil
}

// Root returns the top level directory representing this project.
func (p *Project) Root() string { return p.root }

// Bindir returns the top level directory representing the binary
// directory of this project.
func (p *Project) Bindir() string { return filepath.Join(p.root, "bin") }

// SrcPath represents a directory containing the source of
// some packages.
type SrcPath struct {
	*Project
	path string
}

// Srcdir returns the path to the root of this SrcPaths src.
func (s *SrcPath) Srcdir() string { return filepath.Join(s.root, s.path) }

// AllPackages returns the import paths of all the packages
// inside this SrcPath.
func (s *SrcPath) AllPackages() ([]string, error) {
	return allPackages(s.Srcdir(), "")
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
