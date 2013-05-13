package gogo

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestNewProject(t *testing.T) {
	p := newProject(t)
	cwd := getwd(t)
	if expected := abs(t, filepath.Join(cwd, root)); expected != p.Root() {
		t.Fatalf("Project.Root(): expected %q, got %q", expected, p.Root())
	}
}

// disabled to enable better GOPATH support
func testProjectError(t *testing.T) {
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

var resolvePackageNameTests = []struct {
	path, name string
}{
	{"a", "a"},
	{"a/b", "b"},
	{"a/a", "a"},
}

func TestResolvePackageName(t *testing.T) {
	ctx := newTestContext(t)
	for _, tt := range resolvePackageNameTests {
		pkg, err := ctx.ResolvePackage(tt.path)
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

func TestSrcPathAllPackages(t *testing.T) {
	p := newProject(t)
	pkgs, err := p.SrcPaths[0].AllPackages()
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"a", "a/a", "a/b", "b", "blankimport", "c",
		"cgotest", "d", "d/e", "d/f", "doublepkg", "empty", "empty2", "empty2/empty3", "extdata", "hellocgo",
		"helloworld", "k", "scanfiles", "stdio", "stdlib", "stdlib/bytes"}
	sort.StringSlice(pkgs).Sort()
	sort.StringSlice(expected).Sort()
	if !reflect.DeepEqual(pkgs, expected) {
		t.Fatalf("AllPackages: expected %s, got %s", expected, pkgs)
	}
}
