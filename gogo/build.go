package main

import (
	"fmt"

	"github.com/davecheney/gogo"
)

var BuildCmd = &Command{
	Run: func(project *gogo.Project, args []string) error {
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
			for _, t := range gogo.Build(ctx, pkg) {
				if err := t.Wait(); err != nil {
					return err
				}
			}
		}
		return ctx.Destroy()
	},
}
