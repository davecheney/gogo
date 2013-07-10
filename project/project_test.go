package project

import (
	"path/filepath"
	"testing"
        "os"
	"reflect"
	"sort"
)

const root = "../testdata"

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

func TestSrcDirFindAll(t *testing.T) {
	p := newProject(t)
	s := &SrcDir{p, "src"}
	expected := []string{"empty","a","a/a","a/b","hellocgo","k","scanfiles","b","cgotest","doublepkg","stdlib","stdlib/bytes","blankimport","c","empty2","empty2/empty3","helloworld","d","d/f","d/e","extdata","stdio"}
	actual, err := s.FindAll();
	if err != nil { t.Fatal(err) }
	sort.StringSlice(actual).Sort()
	sort.StringSlice(expected).Sort()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("SrcDir.FindAll: expected %q, got %q", expected, actual)
	}
}
	
