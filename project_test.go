package gogo

import "testing"
import "path/filepath"

const root = "testdata"

func TestNewProject(t *testing.T) {
	p := NewProject(root)
	if p.root != root {
		t.Fatalf("Project.root: expected %q, got %q", root, p.root)
	}
	if p.pkgs == nil {
		t.Fatalf("Project.pkgs: map must be initalised")
	}
}

var resolvePackageTests = []struct {
	path, name string
}{
	{"a", "a"},
	{"a/b", "b"},
	{"a/a", "a"},
}

func TestResolvePackage(t *testing.T) {
	proj := NewProject(root)
	for _, tt := range resolvePackageTests {
		pkg, err := proj.ResolvePackage(tt.path)
		if err != nil {
			t.Fatalf("Project.ResolvePackage(): %v", err)
		}
		if pkg.Name != tt.name {
			t.Fatalf("Package.name: expected %q, got %q", tt.name, pkg.Name)
		}
		if pkg.ImportPath != tt.path {
			t.Fatalf("Package.path: expected %q, got %q", tt.path, pkg.ImportPath)
		}
	}
}

func TestProjectSrcDir(t *testing.T) {
	proj := NewProject(root)
	expected := filepath.Join(root, "src")
	if proj.srcdir() != expected {
		t.Fatalf("Project.srcdir(): expected %q, got %q", expected, proj.srcdir())
	}
}
