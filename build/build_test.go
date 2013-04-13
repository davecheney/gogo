package build

import (
	"testing"

	"github.com/davecheney/gogo"
)

var buildPackageTests = []struct {
	pkg string
}{
	{"a"},
	{"b"}, // imports a
}

func newProject() *gogo.Project {
	return gogo.NewProject("testdata")
}

func TestBuildPackage(t *testing.T) {
	project := newProject()
	for _, tt := range buildPackageTests {
		ctx, err := gogo.NewDefaultContext(project)
		if err != nil {
			t.Fatalf("NewDefaultContext(): %v", err)
		}
		pkg, err := project.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatalf("ResolvePackage(): %v", err)
		}
		targets := buildPackage(ctx, pkg)
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
	//	{"k"}, // uses cgo
}

func TestBuildCommand(t *testing.T) {
	project := newProject()
	for _, tt := range buildCommandTests {
		ctx, err := gogo.NewDefaultContext(project)
		if err != nil {
			t.Fatalf("NewDefaultContext(): %v", err)
		}
		pkg, err := project.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatalf("ResolvePackage(): %v", err)
		}
		targets := buildCommand(ctx, pkg)
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
	// 	{ "k" },
}

func TestBuild(t *testing.T) {
	project := newProject()
	for _, tt := range buildTests {
		ctx, err := gogo.NewDefaultContext(project)
		if err != nil {
			t.Fatalf("NewDefaultContext(): %v", err)
		}
		pkg, err := project.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatalf("ResolvePackage(): %v", err)
		}
		targets := build(ctx, pkg)
		if len := len(targets); len != 1 {
			t.Fatalf("build %q: expected %d target, got %d", tt.pkg, 1, len)
		}
		if err := targets[0].Wait(); err != nil {
			t.Fatalf("build %q: %v", tt.pkg, err)
		}
	}
}
