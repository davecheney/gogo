package gogo

import (
	"path/filepath"
	"testing"
)

var packageImportTests = []struct {
	path    string
	imports []string
}{
	{"a", nil},
	{"a/b", []string{"a"}},
	{"c", []string{"b", "fmt"}},
}

func TestPackageImports(t *testing.T) {
	proj := newProject(t)
	for _, tt := range packageImportTests {
		pkg, err := proj.ResolvePackage(tt.path)
		if err != nil {
			t.Fatalf("Project.ResolvePackage(): %v", err)
		}
		for i, im := range pkg.Imports {
			if im.Name() != tt.imports[i] {
				t.Fatalf("Package %q: expecting import %q, got %q", pkg.ImportPath(), im.Name, tt.imports[i])
			}
		}
	}
}

var newPackageTests = []struct {
	importpath string
	expected   map[string]struct{ name, srcdir string }
}{
	{
		"a",
		map[string]struct{ name, srcdir string }{
			"a": {"a", "src/a"},
		},
	},
	{
		"a/b",
		map[string]struct{ name, srcdir string }{
			"a":   {"a", "src/a"},
			"a/b": {"b", "src/a/b"},
		},
	},
}

func TestNewPackage(t *testing.T) {
	proj := newProject(t)
	for _, tt := range newPackageTests {
		_, err := proj.ResolvePackage(tt.importpath)
		if err != nil {
			t.Fatal(err)
		}
		for importpath, pkg := range proj.pkgs {
			if expected, ok := tt.expected[importpath]; ok {
				if pkg.Name() != expected.name {
					t.Fatalf("pkg.Name(): expected %q, got %q", expected.name, pkg.Name())
				}
				if expected := abs(t, filepath.Join(proj.Root(), expected.srcdir)); expected != pkg.Srcdir() {
					t.Fatalf("pkg.Srcdir(): expected %q, got %q", expected, pkg.Srcdir())
				}
			} else {
				t.Fatalf("pkg cache was missing %q", importpath)
			}
		}
	}
}
