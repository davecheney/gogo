package build

import (
	"testing"

	"github.com/davecheney/gogo"
)

const root = "../testdata"

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
	{"stdlib/bytes"}, // uses build tags, has .s files
	//	{"stdio"}, // imports "C"
}

func TestBuildPackage(t *testing.T) {
	project := newProject(t)
	for _, tt := range buildPackageTests {
		ctx, err := gogo.NewDefaultContext(project)
		if err != nil {
			t.Fatalf("NewDefaultContext(): %v", err)
		}
		defer ctx.Destroy()
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatalf("ResolvePackage(): %v", err)
		}
		if err := buildPackage(pkg).Result(); err != nil {
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
		defer ctx.Destroy()
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatalf("ResolvePackage(): %v", err)
		}
		if err := buildCommand(pkg).Result(); err != nil {
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
		defer ctx.Destroy()
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatalf("ResolvePackage(): %v", err)
		}
		if err := Build(pkg).Result(); err != nil {
			t.Fatalf("build %q: %v", tt.pkg, err)
		}
	}
}