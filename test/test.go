package test

import (
	"fmt"

	"github.com/davecheney/gogo"
	"github.com/davecheney/gogo/build"
)

var TestCmd = &gogo.Command{
	Run: run,
}

func run(project *gogo.Project, args []string) error {
	var pkgs []*gogo.Package
	for _, arg := range args {
		pkg, err := project.ResolvePackage(arg)
		if err != nil {
			return fmt.Errorf("failed to resolve package %q: %v", arg, err)
		}
		pkgs = append(pkgs, pkg)
	}
	ctx, err := gogo.NewDefaultContext(project)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		for _, t := range test(ctx, pkg) {
			if err := t.Wait(); err != nil {
				return err
			}
		}
	}
	return ctx.Destroy()
}

func test(ctx *gogo.Context, pkg *gogo.Package) []gogo.Target {
	// commands are built as packages for testing.
	return testPackage(ctx, pkg)
}

func testPackage(ctx *gogo.Context, pkg *gogo.Package) []gogo.Target {
	// build dependencies
	var deps []gogo.Target
	for _, dep := range pkg.Imports {
		deps = append(deps, build.Build(ctx, dep)...)
	}
	buildtest := buildTest(ctx, pkg, deps)
	runtest := runTest(ctx, pkg, buildtest)
	return []gogo.Target{runtest}
}
