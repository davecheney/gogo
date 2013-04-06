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

func mustDependantPackages(packages *gogo.Package) map[string]*gogo.Package {
	deps, err := packages.DependantPackages()
	if err != nil {
		log.Fatalf("failed to resolve dependant packages: %v", err)
	}
	return deps
}

type buildTarget struct {
	*gogo.Package
	deps []gogo.Target
}

func (t *buildTarget) Deps() []gogo.Target { return t.deps }
func (t *buildTarget) Execute(*gogo.Context) error {
	log.Printf("building package %q", t.Path())
	return nil
}

func (t *buildTarget) String() string { return t.Path() }

func getTarget(targets map[*gogo.Package]gogo.Target, pkg *gogo.Package) gogo.Target {
	if pkg == nil {
		panic("nil package")
	}
	if _, ok := targets[pkg]; !ok {
		targets[pkg] = &buildTarget{Package: pkg}
	}
	return targets[pkg]
}

func main() {
	flag.Parse()
	project := gogo.NewProject(mustGetwd())
	root := mustResolvePackage(project, flag.Arg(0))
	deps := mustDependantPackages(root)
	targets := make(map[*gogo.Package]gogo.Target)
	for _, pkg := range deps {
		log.Printf("%s imports %v", pkg, pkg.Imports())
		t := getTarget(targets, pkg).(*buildTarget)
		for _, i := range pkg.Imports() {
			if pkg, ok := deps[i]; ok {
				t.deps = append(t.deps, getTarget(targets, pkg))
			}
		}
	}
	if err := gogo.ExecuteTargets(toTargets(targets)); err != nil {
		log.Fatalf("%v", err)
	}
}

func toTargets(m map[*gogo.Package]gogo.Target) []gogo.Target {
	var targets []gogo.Target
	for _, t := range m {
		targets = append(targets, t)
	}
	return targets
}
