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

func TestResolvePackage(t *testing.T) {
	proj := NewProject(root)
	pkg, err := proj.ResolvePackage("a")
	if err != nil {
		t.Fatalf("Project.ResolvePackage(): %v", err)
	}
	if pkg.name != "a" {
		t.Fatalf("Package.name: expected %q, got %q", "a", pkg.name)
	}
	expected := filepath.Join(root, "src", "a")
	if pkg.path != expected {
		t.Fatalf("Package.path: expected %q, got %q", expected, pkg.path)
	}
}

func TestProjectSrcDir(t *testing.T) {
	proj := NewProject(root)
	expected := filepath.Join(root, "src")
	if proj.srcdir() != expected {
		t.Fatalf("Project.srcdir(): expected %q, got %q", expected, proj.srcdir())
	}
}
		

