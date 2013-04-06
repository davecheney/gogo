package main

import (
	"flag"
	"fmt"
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

type buildTarget struct {
	*gogo.Package
	deps []gogo.Target
}

func (t *buildTarget) Deps() []gogo.Target                   { return t.deps }
func (t *buildTarget) AddDependantTarget(target gogo.Target) { t.deps = append(t.deps, target) }
func (t *buildTarget) Execute(*gogo.Context) error {
	log.Printf("building package %q", t.Path())
	return nil
}

func (t *buildTarget) String() string { return t.Path() }

type bridgeTarget struct {
	deps []gogo.Target
}

func (t *bridgeTarget) Deps() []gogo.Target                   { return t.deps }
func (t *bridgeTarget) AddDependantTarget(target gogo.Target) { t.deps = append(t.deps, target) }
func (t *bridgeTarget) Execute(*gogo.Context) error {
	log.Printf("bridge %s", t)
	return nil
}

func (t *bridgeTarget) String() string { return fmt.Sprintf("%v", t.deps) }

func getTarget(targets map[*gogo.Package]gogo.Target, pkg *gogo.Package) gogo.Target {
	if _, ok := targets[pkg]; !ok {
		targets[pkg] = &buildTarget{Package: pkg}
	}
	return targets[pkg]
}

func pushPackages(pkgs map[*gogo.Package][]*gogo.Package, root *gogo.Package) error {
	var deps []*gogo.Package
	for _, dep := range root.Imports() {
		if stdlib[dep] {
			// skip
			continue
		}
		pkg, err := root.Project().ResolvePackage(dep)
		if err != nil {
			return err
		}
		if err := pushPackages(pkgs, pkg); err != nil {
			return err
		}
		deps = append(deps, pkg)
	}
	pkgs[root] = deps
	return nil
}

func main() {
	flag.Parse()
	project := gogo.NewProject(mustGetwd())
	root := mustResolvePackage(project, flag.Arg(0))
	pkgs := make(map[*gogo.Package][]*gogo.Package)
	if err := pushPackages(pkgs, root); err != nil {
		log.Fatal(err)
	}
	targets := make(map[*gogo.Package]gogo.Target)
	for pkg, deps := range pkgs {
		log.Printf("%s imports %v", pkg, deps)
		t := getTarget(targets, pkg)
		for _, dep := range deps {
			t.AddDependantTarget(getTarget(targets, dep))
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
