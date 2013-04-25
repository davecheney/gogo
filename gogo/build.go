package main

import (
	"flag"
	"fmt"

	"github.com/davecheney/gogo"
	"github.com/davecheney/gogo/build"
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
		ctx, err := gogo.NewContext(project, *goroot, *goos, *goarch)
		if err != nil {
			return err
		}
		var pkgs []*gogo.Package
		for _, arg := range args {
			pkg, err := ctx.ResolvePackage(arg)
			if err != nil {
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
