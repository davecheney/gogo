package gogo_test

import (
	"testing"

	"github.com/davecheney/gogo"
)

var testPackageTests = []struct {
	pkg string
}{
	{"a"},
}

func TestTestPackage(t *testing.T) {
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
		targets := gogo.TestPackage(pkg)
		if len := len(targets); len != 1 {
			t.Fatalf("testPackage %q: expected %d target, got %d", tt.pkg, 1, len)
		}
		if err := targets[0].Wait(); err != nil {
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
		targets := gogo.Test(pkg)
		if len := len(targets); len != 1 {
			t.Fatalf("testPackage %q: expected %d target, got %d", tt.pkg, 1, len)
		}
		if err := targets[0].Wait(); err != nil {
			t.Fatalf("testPackage %q: %v", tt.pkg, err)
		}
	}
}
