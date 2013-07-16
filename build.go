package main

import (
	"flag"
	"fmt"
	stdbuild "go/build"
	"path/filepath"
	"time"

	"github.com/davecheney/gogo/build"
	"github.com/davecheney/gogo/project"
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
	Run: func(proj *project.Project, args []string) error {
		t0 := time.Now()
		defer func() {
			log.Infof("build duration: %v", time.Since(t0))
		}()
		ctx, err := build.NewContext(proj, *toolchain, *goroot, *goos, *goarch)
		if err != nil {
			return err
		}
		defer func() {
			log.Debugf("build statistics: %v", ctx.Statistics.String())
		}()
		var pkgs []*project.Package
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
				if _, ok := err.(*stdbuild.NoGoError); ok {
					log.Debugf("skipping %q", arg)
					continue
				}
				return fmt.Errorf("failed to resolve package %q: %v", arg, err)
			}
			pkgs = append(pkgs, pkg)
		}
		results := make(chan build.Future, len(pkgs))
		go func() {
			defer close(results)
			for _, pkg := range pkgs {
				results <- build.Build(ctx, pkg)
			}
		}()
		for result := range results {
			if err := result.Result(); err != nil {
				return err
			}
		}
		return ctx.Destroy()
	},
	AddFlags: addBuildFlags,
}
