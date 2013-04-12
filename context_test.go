package gogo

import (
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
	if objdir := ctx.Objdir(pkg); objdir != filepath.Join(ctx.basedir, pkg.path) {
		t.Fatalf("ctx.Objdir(): expected %q, got %q", filepath.Join(ctx.basedir, pkg.path), objdir)
	}
}

func TestConextPkgdir(t *testing.T) {
	proj := NewProject(root)
	ctx, err := NewDefaultContext(proj)
	if err != nil {
		t.Fatalf("NewDefaultContext(): %v", err)
	}
	if pkgdir := ctx.Pkgdir(); pkgdir != filepath.Join(ctx.basedir, "pkg", ctx.goos, ctx.goarch) {
		t.Fatalf("ctx.Objdir(): expected %q, got %q", filepath.Join(ctx.basedir, "pkg", ctx.goos, ctx.goarch), pkgdir)
	}
}
