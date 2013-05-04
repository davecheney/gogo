package gogo

import (
	"path/filepath"
	"sync"
)

// PkgFuture represents an attempt to resolve
// a package import path into a *gogo.Package
type PkgFuture interface {
	Result() struct {
		*Package
		error
	}
}

// pkgFuture implemenst PkgFuture
type pkgFuture struct {
	result chan struct {
		*Package
		error
	}
}

func (t *pkgFuture) Result() struct {
	*Package
	error
} {
	result := <-t.result
	t.result <- result
	return result
}

// resolver resolves packages.
type resolver struct {
	sync.Mutex
	SearchPaths []string
	pkgs        map[string]PkgFuture
}

// ResolvePackage resolves the import path to a Package.
func (r *resolver) resolvePackage(ctx *Context, path string) PkgFuture {
	r.Lock()
	defer r.Unlock()
	if f, ok := r.pkgs[path]; ok {
		return f
	}
	pkg := &Package{
		ImportPath: path,
		Srcdir:     filepath.Join(ctx.Root(), "src", path),
	}
	f := &pkgFuture{
		result: make(chan struct {
			*Package
			error
		}, 1),
	}
	go func() {
		err := ctx.scanFiles(pkg)
		f.result <- struct {
			*Package
			error
		}{pkg, err}
	}()
	r.pkgs[path] = f
	return f
}
