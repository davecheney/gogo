package main

import (
	"fmt"

	"github.com/davecheney/gogo"
)

var TestCmd = &Command{
	Run: func(project *gogo.Project, args []string) error {
		ctx, err := gogo.NewDefaultContext(project)
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
			for _, t := range gogo.Test(pkg) {
				if err := t.Result(); err != nil {
					return err
				}
			}
		}
		return ctx.Destroy()
	},
}
