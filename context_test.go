package gogo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewDefaultContext(t *testing.T) {
	proj := newProject(t)
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

func newTestContext(t *testing.T) *Context {
	proj := newProject(t)
	ctx, err := NewDefaultContext(proj)
	if err != nil {
		t.Fatalf("unable to create context: %v", err)
	}
	return ctx
}

func TestContextPkgdir(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Destroy()
	if pkgdir := ctx.Pkgdir(); pkgdir != filepath.Join(ctx.Workdir(), "pkg", ctx.goos, ctx.goarch) {
		t.Fatalf("ctx.Objdir(): expected %q, got %q", filepath.Join(ctx.Workdir(), "pkg", ctx.goos, ctx.goarch), pkgdir)
	}
}

func TestContextBindir(t *testing.T) {
	ctx := newTestContext(t)
	defer ctx.Destroy()
	if bindir := ctx.Bindir(); bindir != filepath.Join(ctx.Project.Root(), "bin", ctx.goos, ctx.goarch) {
		t.Fatalf("ctx.Bindir(): expected %q, got %q", filepath.Join(ctx.Project.Root(), "bin", ctx.goos, ctx.goarch), bindir)
	}
}

func TestContextDestroy(t *testing.T) {
	ctx := newTestContext(t)
	if _, err := os.Stat(ctx.Workdir()); err != nil {
		t.Fatal(err)
	}
	if err := ctx.Destroy(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ctx.Workdir()); !os.IsNotExist(err) {
		t.Fatalf("context did not destroy basedir")
	}
}
