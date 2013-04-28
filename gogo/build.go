package main

import (
	"flag"
	"fmt"
	stdbuild "go/build"
	"time"

	"github.com/davecheney/gogo"
	"github.com/davecheney/gogo/build"
	"github.com/davecheney/gogo/log"
)

func init() {
	registerCommand("build", BuildCmd)
}

var (
	// build flags

	// should we build all packages in this project.
	// defaults to true when build is invoked from the project root.
	A bool

	// should we perform a release build +release tag ?
	// defaults to false, +debug.
	R bool
)

func addBuildFlags(fs *flag.FlagSet) {
	fs.BoolVar(&A, "a", false, "build all packages in this project")
	fs.BoolVar(&R, "r", false, "perform a release build")
}

var BuildCmd = &Command{
	Run: func(project *gogo.Project, args []string) error {
		t0 := time.Now()
		defer func() {
			log.Infof("build duration: %v", time.Since(t0))
		}()
		ctx, err := gogo.NewContext(project, *goroot, *goos, *goarch)
		if err != nil {
			return err
		}
		defer func() {
			log.Debugf("build statistics: %v", ctx.Statistics.String())
		}()
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
			if err := build.Build(pkg).Result(); err != nil {
				return err
			}
		}
		return ctx.Destroy()
	},
	AddFlags: addBuildFlags,
}
