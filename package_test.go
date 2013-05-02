package gogo

import (
	"path/filepath"
	"reflect"
	"runtime"
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
	ctx := newTestContext(t)
	defer ctx.Destroy()
	for _, tt := range packageImportTests {
		pkg, err := ctx.ResolvePackage(tt.path)
		if err != nil {
			t.Fatalf("Project.ResolvePackage(): %v", err)
		}
		for i, im := range pkg.Imports {
			if im.Name != tt.imports[i] {
				t.Fatalf("Package %q: expecting import %q, got %q", pkg.ImportPath, im.Name, tt.imports[i])
			}
		}
	}
}

var resolvePackageTests = []struct {
	importpath  string
	imports     map[string]struct{ name, srcdir string }
	testimports map[string]struct{ name, srcdir string }
}{
	{
		"a",
		map[string]struct{ name, srcdir string }{
			"a": {"a", "src/a"},
		},
		nil,
	},
	{
		"a/b",
		map[string]struct{ name, srcdir string }{
			"a":   {"a", "src/a"},
			"a/b": {"b", "src/a/b"},
		},
		nil,
	},
	{
		// d imports d/e, d should not import d/f as that is a test dep
		"d",
		map[string]struct{ name, srcdir string }{
			"a":   {"a", "src/a"},
			"a/b": {"b", "src/a/b"},
			"d":   {"d", "src/d"},
			"d/e": {"e", "src/d/e"},
		},
		map[string]struct{ name, srcdir string }{
			"d/f": {"f", "src/d/f"},
		},
	},
}

func TestResolvePackage(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Destroy()
	for _, tt := range resolvePackageTests {
		if _, err := ctx.ResolvePackage(tt.importpath); err != nil {
			t.Fatal(err)
		}
		for importpath, pkg := range ctx.pkgs {
			if expected, ok := tt.imports[importpath]; ok {
				if pkg.Name != expected.name {
					t.Fatalf("pkg.Name: expected %q, got %q", expected.name, pkg.Name)
				}
				if expected := abs(t, filepath.Join(ctx.Project.Root(), expected.srcdir)); expected != pkg.Srcdir {
					t.Fatalf("pkg.Srcdir(): expected %q, got %q", expected, pkg.Srcdir)
				}
			} else if expected, ok := tt.testimports[importpath]; ok {
				if pkg.Name != expected.name {
					t.Fatalf("pkg.Name: expected %q, got %q", expected.name, pkg.Name)
				}
				if expected := abs(t, filepath.Join(ctx.Project.Root(), expected.srcdir)); expected != pkg.Srcdir {
					t.Fatalf("pkg.Srcdir(): expected %q, got %q", expected, pkg.Srcdir)
				}
			} else { // if _, ok := stdlib[importpath]; !ok {
				t.Fatalf("pkg cache was missing testimports %q", importpath)
			}
		}
	}
}

var scanFilesTests = []struct {
	path           string
	gofiles        []string
	cgofiles       []string
	sfiles         []string
	testgofiles    []string
	xtestgofiles   []string
	ignoredgofiles []string
}{
	{path: "scanfiles",
		gofiles:        []string{"go1.go"},
		cgofiles:       []string{"cgo.go"},
		testgofiles:    []string{"scanfiles_test.go"},
		xtestgofiles:   []string{"scanfiles_external_test.go"},
		ignoredgofiles: []string{"doc.go"},
	},
	{path: "stdlib/bytes",
		gofiles:     []string{"buffer.go", "bytes.go", "bytes_decl.go", "reader.go"},
		sfiles:      []string{"asm_" + runtime.GOARCH + ".s"},
		testgofiles: []string{"buffer_test.go", "bytes_test.go", "example_test.go", "reader_test.go"},
	},
}

func TestPackageScanFiles(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Destroy()
	for _, tt := range scanFilesTests {
		p, err := ctx.ResolvePackage(tt.path)
		if err != nil {
			t.Fatalf("resolvepackage: %v", err)
		}
		if !reflect.DeepEqual(tt.gofiles, p.GoFiles) {
			t.Fatalf("pkg.GoFiles: expected %q, got %q", tt.gofiles, p.GoFiles)
		}
		if !reflect.DeepEqual(tt.sfiles, p.SFiles) {
			t.Fatalf("pkg.SFiles: expected %q, got %q", tt.sfiles, p.SFiles)
		}
		if !reflect.DeepEqual(tt.cgofiles, p.CgoFiles) {
			t.Fatalf("pkg.CgoFiles: expected %q, got %q", tt.cgofiles, p.CgoFiles)
		}
		if !reflect.DeepEqual(tt.testgofiles, p.TestGoFiles) {
			t.Fatalf("pkg.TestGoFiles: expected %q, got %q", tt.testgofiles, p.TestGoFiles)
		}
		if !reflect.DeepEqual(tt.ignoredgofiles, p.IgnoredGoFiles) {
			t.Fatalf("pkg.IgnoredGoFiles: expected %q, got %q", tt.ignoredgofiles, p.IgnoredGoFiles)
		}
	}
}

var resolvePackageErrorTests = []struct {
	path string
	err  string
}{
	{"cgotest", "use of cgo in test cgo_test.go not supported"},
	{"doublepkg", "found packages a (a.go) and b (b.go) in doublepkg"},
	{"blankimport", `blank.go:3:8: invalid import path: ""`},
	{"empty", "no Go source files in empty"},
	{"empty2", "no Go source files in empty2/empty3"},
}

func TestResolvePackageError(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Destroy()
	for _, tt := range resolvePackageErrorTests {
		if _, err := ctx.ResolvePackage(tt.path); err.Error() != tt.err {
			t.Fatalf("resolve package: expected %q, got %q", tt.err, err.Error())
		}
	}
}
