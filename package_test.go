package gogo

import "testing"

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
