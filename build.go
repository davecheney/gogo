package gogo

import (
	"log"
	"sync"
)

type Target interface {
	Wait() error
}

type buildPackageTarget struct {
	deps []Target
	done chan struct{}
	err  struct {
		sync.Mutex
		val error
	}
	*Package
}

func (t *buildPackageTarget) execute() {
	defer close(t.done)
	for _, dep := range t.deps {
		if err := dep.Wait(); err != nil {
			t.err.Lock()
			t.err.val = err
			t.err.Unlock()
			return
		}
	}
	if err := t.build(); err != nil {
		t.err.Lock()
		t.err.val = err
		t.err.Unlock()
	}
}

func (t *buildPackageTarget) Wait() error {
	<-t.done
	t.err.Lock()
	defer t.err.Unlock()
	return t.err.val
}

func (t *buildPackageTarget) build() error {
	log.Printf("%T %q", t, t.Package.Path())
	return nil
}

func BuildPackages(pkgs ...*Package) error {
	targets := make(map[*Package]Target)
	for _, pkg := range pkgs {
		tt := buildPackage(targets, pkg)
		for _, t := range tt {
			if err := t.Wait(); err != nil {
				return err
			}
		}
	}
	return nil
}

func buildPackage(targets map[*Package]Target, pkg *Package) []Target {
	var tt []Target
	for _, dep := range pkg.Imports() {
		tt = append(tt, buildPackage(targets, dep)...)
	}
	if _, ok := targets[pkg]; !ok {
		t := &buildPackageTarget{
			deps:    tt,
			done:    make(chan struct{}),
			Package: pkg,
		}
		targets[pkg] = t
		go t.execute()
	}
	return []Target{targets[pkg]}
}
