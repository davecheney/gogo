package main

import (
	"flag"
	"log"
	"os"

	"github.com/davecheney/gogo"
)

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("unable to determine current working directory: %v", err)
	}
	return wd
}

func mustResolvePackage(project *gogo.Project, path string) *gogo.Package {
	pkg, err := project.ResolvePackage(path)
	if err != nil {
		log.Fatalf("failed to resolve package %q: %v", path, err)
	}
	return pkg
}

func main() {
	flag.Parse()
	project := gogo.NewProject(mustGetwd())
	pkg := mustResolvePackage(project, flag.Arg(0))
	if err := gogo.BuildPackages(pkg); err != nil {
		log.Fatal(err)
	}
}
