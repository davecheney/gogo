package main

import (
	"fmt"
	stdbuild "go/build"

	"github.com/davecheney/gogo"
	"github.com/davecheney/gogo/build"
	"github.com/davecheney/gogo/log"
)

func init() {
	registerCommand("test", TestCmd)
}

var TestCmd = &Command{
	Run: func(project *gogo.Project, args []string) error {
		ctx, err := gogo.NewContext(project, *goroot, *goos, *goarch)
		if err != nil {
			return err
		}
		var pkgs []*gogo.Package
		if A {
			var err error
			args, err = project.SrcPaths[0].AllPackages()
			if err != nil {
				return fmt.Errorf("could not fetch packages in srcpath %v: %v", project.SrcPaths[0], err)
			}
		}
		for _, arg := range args {
			pkg, err := ctx.ResolvePackage(arg)
			if err != nil {
				if _, ok := err.(*stdbuild.NoGoError); ok {
					log.Debugf("skipping %q", arg)
					continue
				}
				return fmt.Errorf("failed to resolve package %q: %v", arg, err)
			}
			pkgs = append(pkgs, pkg)
		}
		for _, pkg := range pkgs {
			if err := build.Test(ctx, pkg).Result(); err != nil {
				return err
			}
		}
		return nil //		return ctx.Destroy()
	},
	AddFlags: addBuildFlags,
}
