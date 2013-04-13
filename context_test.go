package gogo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewDefaultContext(t *testing.T) {
	proj := NewProject(root)
	c, err := NewDefaultContext(proj)
	if err != nil {
		t.Fatalf("NewDefaultContext: %v", err)
	}
	if c.goroot != runtime.GOROOT() {
		t.Fatalf("Context.goroot: expected %q, got %v", runtime.GOROOT(), c.goroot)
	}
	if c.goos != runtime.GOOS {
		t.Fatalf("Context.goos: expected %q, got %v", runtime.GOOS, c.goos)
	}
	if c.goarch != runtime.GOARCH {
		t.Fatalf("Context.goarch: expected %q, got %v", runtime.GOARCH, c.goarch)
	}
}

func TestContextObjdir(t *testing.T) {
	proj := NewProject(root)
	ctx, err := NewDefaultContext(proj)
	if err != nil {
		t.Fatalf("NewDefaultContext(): %v", err)
	}
	pkg, err := proj.ResolvePackage("a")
	if err != nil {
		t.Fatalf("project.ResolvePackage(): %v", err)
	}
	if objdir := ctx.Objdir(pkg); objdir != filepath.Join(ctx.basedir, pkg.ImportPath, "_obj") {
		t.Fatalf("ctx.Objdir(): expected %q, got %q", filepath.Join(ctx.basedir, pkg.ImportPath, "_obj"), objdir)
	}
}

func TestContextPkgdir(t *testing.T) {
	proj := NewProject(root)
	ctx, err := NewDefaultContext(proj)
	if err != nil {
		t.Fatalf("NewDefaultContext(): %v", err)
	}
	if pkgdir := ctx.Pkgdir(); pkgdir != filepath.Join(ctx.basedir, "pkg", ctx.goos, ctx.goarch) {
		t.Fatalf("ctx.Objdir(): expected %q, got %q", filepath.Join(ctx.basedir, "pkg", ctx.goos, ctx.goarch), pkgdir)
	}
}

func TestContextBindir(t *testing.T) {
	proj := NewProject(root)
	ctx, err := NewDefaultContext(proj)
	if err != nil {
		t.Fatalf("NewDefaultContext(): %v", err)
	}
	if bindir := ctx.Bindir(); bindir != filepath.Join(ctx.Project.root, "bin", ctx.goos, ctx.goarch) {
		t.Fatalf("ctx.Bindir(): expected %q, got %q", filepath.Join(ctx.Project.root, "bin", ctx.goos, ctx.goarch), bindir)
	}
}

func TestContextDestroy(t *testing.T) {
	proj := NewProject(root)
	ctx, err := NewDefaultContext(proj)
	if err != nil {
		t.Fatalf("NewDefaultContext(): %v", err)
	}
	if _, err := os.Stat(ctx.basedir); err != nil {
		t.Fatal(err)
	}
	if err := ctx.Destroy(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ctx.basedir); !os.IsNotExist(err) {
		t.Fatalf("context did not destroy basedir")
	}
}
