package build

import (
	"testing"

	"github.com/davecheney/gogo"
)

var testPackageTests = []struct {
	pkg string
}{
	{"a"},
	// 	{"stdlib/bytes"}, // includes asm files, disabled needs go 1.1 features
	// 	{"extdata"}, external tests are not supported
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
		if err := testPackage(pkg).Result(); err != nil {
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
		if err := Test(pkg).Result(); err != nil {
			t.Fatalf("testPackage %q: %v", tt.pkg, err)
		}
	}
}
