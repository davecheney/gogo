package build

import (
	"fmt"
	"log"

	"github.com/davecheney/gogo"
)

var BuildCmd = &gogo.Command{
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
		for _, t := range Build(ctx, pkg) {
			if err := t.Wait(); err != nil {
				return err
			}
		}
	}
	return ctx.Destroy()
}

func Build(ctx *gogo.Context, pkg *gogo.Package) []gogo.Target {
	if pkg.Name == "main" {
		return buildCommand(ctx, pkg)
	}
	return buildPackage(ctx, pkg)
}

func buildPackage(ctx *gogo.Context, pkg *gogo.Package) []gogo.Target {
	var deps []gogo.Target
	for _, dep := range pkg.Imports {
		deps = append(deps, buildPackage(ctx, dep)...)
	}
	if _, ok := ctx.Targets[pkg]; !ok {
		gc := Gc(ctx, pkg, deps...)
		pack := Pack(ctx, pkg, gc)
		ctx.Targets[pkg] = pack
	}
	log.Printf("build package %q", pkg.ImportPath)
	return []gogo.Target{ctx.Targets[pkg]}
}

func buildCommand(ctx *gogo.Context, pkg *gogo.Package) []gogo.Target {
	var deps []gogo.Target
	for _, dep := range pkg.Imports {
		deps = append(deps, buildPackage(ctx, dep)...)
	}
	if _, ok := ctx.Targets[pkg]; !ok {
		gc := Gc(ctx, pkg, deps...)
		pack := Pack(ctx, pkg, gc)
		ld := Ld(ctx, pkg, pack)
		ctx.Targets[pkg] = ld
	}
	log.Printf("build command %q", pkg.ImportPath)
	return []gogo.Target{ctx.Targets[pkg]}
}
