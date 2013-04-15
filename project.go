package gogo

import (
	"path/filepath"
)

type Project struct {
	root string
}

func NewProject(root string) (*Project, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Project{
		root: root,
	}, nil
}

// Root returns the top level directory representing this project.
func (p *Project) Root() string { return p.root }

// Bindir returns the top level directory representing the binary
// directory of this project.
func (p *Project) Bindir() string { return filepath.Join(p.root, "bin") }
