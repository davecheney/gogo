package gogo

import (
	"path/filepath"
	"testing"
)

func TestNewProject(t *testing.T) {
	p := newProject(t)
	cwd := getwd(t)
	if expected := abs(t, filepath.Join(cwd, root)); expected != p.Root() {
		t.Fatalf("Project.Root(): expected %q, got %q", expected, p.Root())
	}
}

func TestProjectError(t *testing.T) {
	cwd := getwd(t)
	// assumes $CWD/missing is missing
	if _, err := NewProject(filepath.Join(cwd, "missing")); err == nil {
		t.Fatalf("Opening project on non existant directory, expected error, got %v", err)
	}
}

func TestProjectBindir(t *testing.T) {
	p := newProject(t)
	cwd := getwd(t)
	if expected := abs(t, filepath.Join(cwd, root, "bin")); expected != p.Bindir() {
		t.Fatalf("Project.Bindir(): expected %q, got %q", expected, p.Bindir())
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
	ctx := newTestContext(t)
	for _, tt := range resolvePackageTests {
		pkg, err := ctx.ResolvePackage(tt.path)
		if err != nil {
			t.Fatalf("Project.ResolvePackage(): %v", err)
		}
		if pkg.Name() != tt.name {
			t.Fatalf("Package.name: expected %q, got %q", tt.name, pkg.Name())
		}
		if pkg.ImportPath() != tt.path {
			t.Fatalf("Package.path: expected %q, got %q", tt.path, pkg.ImportPath())
		}
	}
}
