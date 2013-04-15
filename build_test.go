package gogo_test

import (
	"testing"

	"github.com/davecheney/gogo"
)

const root = "testdata"

func newProject(t *testing.T) *gogo.Project {
        p, err := gogo.NewProject(root)
        if err != nil {
                t.Fatalf("could not resolve project root %q: %v", root, err)
        }
        return p
}

var buildPackageTests = []struct {
	pkg string
}{
	{"a"},
	{"b"}, // imports a
	{"helloworld"},
//	{"stdio"}, // imports "C"
}

func TestBuildPackage(t *testing.T) {
	project := newProject(t)
	for _, tt := range buildPackageTests {
		ctx, err := gogo.NewDefaultContext(project)
		if err != nil {
			t.Fatalf("NewDefaultContext(): %v", err)
		}
		pkg, err := project.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatalf("ResolvePackage(): %v", err)
		}
		targets := gogo.BuildPackage(ctx, pkg)
		if len := len(targets); len != 1 {
			t.Fatalf("buildPackage %q: expected %d target, got %d", tt.pkg, 1, len)
		}
		if err := targets[0].Wait(); err != nil {
			t.Fatalf("buildPackage %q: %v", tt.pkg, err)
		}
	}
}

var buildCommandTests = []struct {
	pkg string
}{
	{"b"},
	{"helloworld"}, // links in a stdlib pkg
	//{"hellocgo"},	// imports a cgo pkg
}

func TestBuildCommand(t *testing.T) {
	project := newProject(t)
	for _, tt := range buildCommandTests {
		ctx, err := gogo.NewDefaultContext(project)
		if err != nil {
			t.Fatalf("NewDefaultContext(): %v", err)
		}
		pkg, err := project.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatalf("ResolvePackage(): %v", err)
		}
		targets := gogo.BuildCommand(ctx, pkg)
		if len := len(targets); len != 1 {
			t.Fatalf("buildCommand %q: expected %d target, got %d", tt.pkg, 1, len)
		}
		if err := targets[0].Wait(); err != nil {
			t.Fatalf("buildCommand %q: %v", tt.pkg, err)
		}
	}
}

var buildTests = []struct {
	pkg string
}{
	{"a"},
	{"b"},
//	{ "k" },
	{"helloworld"},
}

func TestBuild(t *testing.T) {
	project := newProject(t)
	for _, tt := range buildTests {
		ctx, err := gogo.NewDefaultContext(project)
		if err != nil {
			t.Fatalf("NewDefaultContext(): %v", err)
		}
		pkg, err := project.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatalf("ResolvePackage(): %v", err)
		}
		targets := gogo.Build(ctx, pkg)
		if len := len(targets); len != 1 {
			t.Fatalf("build %q: expected %d target, got %d", tt.pkg, 1, len)
		}
		if err := targets[0].Wait(); err != nil {
			t.Fatalf("build %q: %v", tt.pkg, err)
		}
	}
}
