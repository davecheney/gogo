package build

import (
	"fmt"
	"log"

	"github.com/davecheney/gogo"
)

var Build = &gogo.Command{
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
		var tt []gogo.Target
		if pkg.Name() == "main" {
			tt = buildCommand(ctx, pkg)
		} else {
			tt = buildPackage(ctx, pkg)
		}
		for _, t := range tt {
			if err := t.Wait(); err != nil {
				return err
			}
		}
	}
	return nil
}

func buildPackage(ctx *gogo.Context, pkg *gogo.Package) []gogo.Target {
	var deps []gogo.Target
	for _, dep := range pkg.Imports() {
		deps = append(deps, buildPackage(ctx, dep)...)
	}
	if _, ok := ctx.Targets[pkg]; !ok {
		// gc target
		gc := newGcTarget(ctx, pkg, deps...)
		go gc.execute()
		pack := newPackTarget(ctx, pkg, gc)
		go pack.execute()
		ctx.Targets[pkg] = pack
	}
	log.Printf("build package %q", pkg.ImportPath())
	return []gogo.Target{ctx.Targets[pkg]}
}

func buildCommand(ctx *gogo.Context, pkg *gogo.Package) []gogo.Target {
	var deps []gogo.Target
	for _, dep := range pkg.Imports() {
		deps = append(deps, buildPackage(ctx, dep)...)
	}
	if _, ok := ctx.Targets[pkg]; !ok {
		// gc target
		gc := newGcTarget(ctx, pkg, deps...)
		go gc.execute()
		pack := newPackTarget(ctx, pkg, gc)
		go pack.execute()
		ld := newLdTarget(ctx, pkg, pack)
		go ld.execute()
		ctx.Targets[pkg] = ld
	}
	log.Printf("build command %q", pkg.ImportPath())
	return []gogo.Target{ctx.Targets[pkg]}
}
