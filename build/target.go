package build

import (
	"path/filepath"
	"time"

	"github.com/davecheney/gogo"
	"github.com/davecheney/gogo/log"
)

// target implements a gogo.Future
type target struct {
	err chan error
}

func (t *target) Result() error {
	result := <-t.err
	t.err <- result
	return result
}

// packTarget implements a gogo.Future that represents
// packing Go object files into a .a archive.
type packTarget struct {
	target
	deps     []objFuture
	objfiles []string
	*gogo.Package
}

func (t *packTarget) execute() {
	for _, dep := range t.deps {
		if err := dep.Result(); err != nil {
			t.err <- err
			return
		}
		// collect successful objfiles for packing
		t.objfiles = append(t.objfiles, dep.objfile())
	}
	log.Infof("pack %q: %s", t.Package.ImportPath, t.objfiles)
	t.err <- t.build()
}

func (t *packTarget) pkgfile() string { return t.Package.ImportPath + ".a" }

func (t *packTarget) build() error {
	t0 := time.Now()
	ofile := t.pkgfile()
	pkgdir := filepath.Dir(filepath.Join(t.Pkgdir(), ofile))
	if err := t.Mkdir(pkgdir); err != nil {
		return err
	}
	err := t.Pack(ofile, t.Pkgdir(), t.objfiles...)
	t.Record("pack", time.Since(t0))
	return err
}
