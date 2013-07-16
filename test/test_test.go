package test

import (
	"path/filepath"
	"testing"

	"github.com/davecheney/gogo/build"
	"github.com/davecheney/gogo/project"
)

var testPackageTests = []struct {
	pkg string
}{
	{"a"},
	// 	{"stdlib/bytes"}, // includes asm files, disabled needs go 1.1 features
	{"extdata"}, // external tests are not supported
	{"stdio"},   // imports "C"
}

func TestTestPackage(t *testing.T) {
	project := newProject(t)
	for _, tt := range testPackageTests {
		ctx, err := build.NewDefaultContext(project)
		if err != nil {
			t.Fatalf("NewDefaultContext(): %v", err)
		}
		defer ctx.Destroy()
		pkg, err := ctx.ResolvePackage("linux", "amd64", tt.pkg).Result()
		if err != nil {
			t.Fatalf("ResolvePackage(): %v", err)
		}
		if err := testPackage(ctx, pkg).Result(); err != nil {
			t.Fatalf("testPackage %q: %v", tt.pkg, err)
		}
	}
}

func TestTest(t *testing.T) {
	project := newProject(t)
	for _, tt := range testPackageTests {
		ctx, err := gogo.NewDefaultContext(project)
		if err != nil {
			t.Fatalf("NewDefaultContext(): %v", err)
		}
		defer ctx.Destroy()
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatalf("ResolvePackage(): %v", err)
		}
		if err := Test(ctx, pkg).Result(); err != nil {
			t.Fatalf("testPackage %q: %v", tt.pkg, err)
		}
	}
}

func TestTestObjdir(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Destroy()
	pkg, err := ctx.ResolvePackage("a")
	if err != nil {
		t.Fatalf("project.ResolvePackage(): %v", err)
	}
	if testdir := testobjdir(ctx, pkg); testdir != filepath.Join(ctx.Workdir(), pkg.ImportPath, "_test") {
		t.Fatalf("pkg.Objdir(): expected %q, got %q", filepath.Join(ctx.Workdir(), pkg.ImportPath, "_test"), testdir)
	}
}
