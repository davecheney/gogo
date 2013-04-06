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

func pushImports(m map[string]*gogo.Package, root *gogo.Package) {
	for _, path := range root.Imports() {
		if stdlib[path] {
			// skip
			continue
		}
		if _, ok := m[path]; !ok {
			pkg := mustResolvePackage(root.Project(), path)
			m[pkg.Path()] = pkg
			pushImports(m, pkg)
		}
	}
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

func getTarget(targets map[*gogo.Package]gogo.Target, pkg *gogo.Package) *buildTarget {
	if pkg == nil {
		panic("nil package")
	}
	if _, ok := targets[pkg]; !ok {
		targets[pkg] = &buildTarget{Package: pkg}
	}
	return targets[pkg].(*buildTarget)
}

func main() {
	flag.Parse()
	project := gogo.NewProject(mustGetwd())
	root := mustResolvePackage(project, flag.Arg(0))
	tobuild := map[string]*gogo.Package{
		root.Path(): root,
	}
	pushImports(tobuild, root)
	targets := make(map[*gogo.Package]gogo.Target)
	for _, pkg := range tobuild {
		log.Printf("package %q imports %v", pkg.Path(), pkg.Imports())
		t := getTarget(targets, pkg)
		for _, i := range pkg.Imports() {
			if pkg, ok := tobuild[i]; ok {
				t.deps = append(t.deps, getTarget(targets, pkg))
			}
		}
	}
	executions := make(map[*buildTarget]*gogo.Execution)
	for _, t := range targets {
		log.Printf("package %q depends on %v", t.(*buildTarget).Path(), t.Deps())
		e := buildExecution(executions, t.(*buildTarget))
		go e.Execute()
	}
	result := executions[targets[root].(*buildTarget)]
	log.Println(result.Wait())
}

func buildExecution(m map[*buildTarget]*gogo.Execution, t *buildTarget) *gogo.Execution {
	var deps []*gogo.Execution
	for _, d := range t.Deps() {
		deps = append(deps, buildExecution(m, d.(*buildTarget)))
	}
	m[t] = gogo.NewExecution(t, nil, deps...)
	return m[t]
}
