package build

import (
	"testing"

	"github.com/davecheney/gogo/project"
)

const root = "../testdata"

func newProject(t *testing.T) *project.Project {
	p, err := project.NewProject(root)
	if err != nil {
		t.Fatalf("could not resolve project root %q: %v", root, err)
	}
	return p
}
