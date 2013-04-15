package test

import (
	"log"
	"sync"
	"os"
	"path/filepath"

	"github.com/davecheney/gogo"
)

type target struct {
        done chan struct{}
        err  struct {
                sync.Mutex
                val error
        }
}

func (t *target) Wait() error {
        <-t.done
        t.err.Lock()
        defer t.err.Unlock()
        return t.err.val
}

func (t *target) setErr(err error) {
        t.err.Lock()
        t.err.val = err
        t.err.Unlock()
}

type buildTestTarget struct {
        target
        deps []gogo.Target
        *gogo.Package
        *gogo.Context
}

func (t *buildTestTarget) execute() {
        defer close(t.done)
        for _, dep := range t.deps {
                if err := dep.Wait(); err != nil {
                        t.setErr(err)
                        return
                }
        }
        if err := t.build(); err != nil {
                t.setErr(err)
        }
}

func (t *buildTestTarget) objdir() string  { return t.Context.TestObjdir(t.Package) }
func (t *buildTestTarget) objfile() string { return filepath.Join(t.objdir(), "_go_.6") }

func (t *buildTestTarget) build() error {
        gofiles := t.GoFiles
        objdir := t.objdir()
        if err := os.MkdirAll(objdir, 0777); err != nil {
                return err
        }
        if err := t.Gc(t.ImportPath, t.Srcdir(), t.objfile(), gofiles); err != nil {
		return err
	}
	if err := t.buildTestMain(t.objdir(), gofiles); err != nil {
		return err
	}
	if err := t.Gc(t.objdir(), t.objdir(), t.Package.Name+".6", []string{"_testmain.go"}); err != nil {
		return err
	}
	return t.Ld(filepath.Join(t.objdir(), t.Package.Name+".test"), t.Package.Name+".6")
}

func (t *buildTestTarget) buildTestMain(objdir string, gofiles []string) error {
	return nil
}


func buildTest(ctx *gogo.Context, pkg *gogo.Package, deps []gogo.Target) *buildTestTarget {
        t := &buildTestTarget{
                target: target{
                        done: make(chan struct{}),
                },
                deps:    deps,
                Package: pkg,
                Context: ctx,
        }
        go t.execute()
        return t
}

type runTestTarget struct {
        target
        deps []gogo.Target
        *gogo.Package
        *gogo.Context
}

func (t *runTestTarget) execute() {   
        defer close(t.done)
        for _, dep := range t.deps {
                if err := dep.Wait(); err != nil {
                        t.setErr(err)
                        return
                }
        }
        if err := t.build(); err != nil {
                t.setErr(err)
        }
}

func (t *runTestTarget) build() error {
	log.Printf("testing package %q", t.Package.ImportPath)
	return nil
}

func runTest(ctx *gogo.Context, pkg *gogo.Package, deps ...gogo.Target) *runTestTarget {
        t := &runTestTarget{
                target: target{
                        done: make(chan struct{}),
                },
                deps:    deps,
                Package: pkg,
                Context: ctx,
        }
        go t.execute()
        return t
}
