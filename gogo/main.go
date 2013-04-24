package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/davecheney/gogo"
)

const projectdir = ".gogo"

type Command struct {
	Run func(project *gogo.Project, args []string) error
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("unable to determine current working directory: %v", err)
	}
	return wd
}

// findProjectRoot works upwards from path seaching for the
// .gogo directory which identifies the project root.
func findProjectRoot(path string) (string, error) {
	start := path
	for path != "/" {
		root := filepath.Join(path, projectdir)
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				path = filepath.Dir(path)
				continue
			}
			return "", err
		}
		return path, nil
	}
	return "", fmt.Errorf("could not find project root in %q or its parents", start)
}

var (
	fs     = flag.NewFlagSet("gogo", flag.ContinueOnError)
	goos   = fs.String("goos", runtime.GOOS, "override GOOS")
	goarch = fs.String("goarch", runtime.GOARCH, "override GOARCH")
	goroot = fs.String("goroot", runtime.GOROOT(), "override GOROOT")
)

func main() {
	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalf("could not parse flags: %v", err)
	}

	root, err := findProjectRoot(mustGetwd())
	if err != nil {
		log.Fatalf("could not locate project root: %v", err)
	}

	log.Printf("project root %q", root)

	project, err := gogo.NewProject(root)
	if err != nil {
		log.Fatalf("unable to construct project: %v", err)
	}
	if fs.NArg() < 1 {
		log.Fatalf("no command supplied")
	}
	first, rest := fs.Arg(0), fs.Args()[1:]
	var cmd *Command
	switch first {
	case "build":
		cmd = BuildCmd
	case "test":
		cmd = TestCmd
	default:
		log.Fatalf("unknown command %q", first)
	}
	if err := cmd.Run(project, rest); err != nil {
		log.Fatal(err)
	}
}
