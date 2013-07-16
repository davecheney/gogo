package main

import (
	"fmt"
	gobuild "go/build"
	"path/filepath"

	"github.com/davecheney/gogo/build"
	"github.com/davecheney/gogo/log"
	"github.com/davecheney/gogo/project"
	"github.com/davecheney/gogo/test"
)

func init() {
	registerCommand("test", TestCmd)
}

var TestCmd = &Command{
	Run: func(proj *project.Project, args []string) error {
		ctx, err := build.NewContext(proj, *toolchain, *goroot, *goos, *goarch)
		if err != nil {
			return err
		}
		var pkgs []*gobuild.Package
		if A {
			var err error
			args, err = proj.SrcDirs[0].FindAll()
			if err != nil {
				return fmt.Errorf("could not fetch packages in srcpath %v: %v", proj.SrcDirs[0], err)
			}
		}
		for _, arg := range args {
			if arg == "." {
				var err error
				arg, err = filepath.Rel(proj.SrcDirs[0].SrcDir(), mustGetwd())
				if err != nil {
					return err
				}
			}
			pkg, err := ctx.ResolvePackage("linux", "amd64", arg).Result()
			if err != nil {
				if _, ok := err.(*gobuild.NoGoError); ok {
					log.Debugf("skipping %q", arg)
					continue
				}
				return fmt.Errorf("failed to resolve package %q: %v", arg, err)
			}
			pkgs = append(pkgs, pkg)
		}
		for _, pkg := range pkgs {
			if err := test.Test(ctx, pkg).Result(); err != nil {
				return err
			}
		}
		return ctx.Destroy()
	},
	AddFlags: addBuildFlags,
}
