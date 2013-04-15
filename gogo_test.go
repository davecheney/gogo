package gogo

import (
	"os"
	"path/filepath"
	"testing"
)

const root = "testdata"

func newProject(t *testing.T) *Project {
	p, err := NewProject(root)
	if err != nil {
		t.Fatalf("could not resolve project root %q: %v", root, err)
	}
	return p
}

func getwd(t *testing.T) string {
	cwd, err := os.Getwd() // assumes that tests run in the directory they were built from
	if err != nil {
		t.Fatalf("could not determine current working directory: %v", err)
	}
	return cwd
}

func abs(t *testing.T, path string) string {
	p, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("could not resolve absolute path of %q: %v", path, err)
	}
	return p
}
