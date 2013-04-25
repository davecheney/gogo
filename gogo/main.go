package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/davecheney/gogo"
	"github.com/davecheney/gogo/log"
)

const projectdir = ".gogo"

type Command struct {
	Run      func(project *gogo.Project, args []string) error
	AddFlags func(fs *flag.FlagSet)
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
	fs     = flag.NewFlagSet("gogo", flag.ExitOnError)
	goos   = fs.String("goos", runtime.GOOS, "override GOOS")
	goarch = fs.String("goarch", runtime.GOARCH, "override GOARCH")
	goroot = fs.String("goroot", runtime.GOROOT(), "override GOROOT")
)

func init() {
	fs.BoolVar(&log.Quiet, "q", log.Quiet, "suppress log messages below ERROR level")
	fs.BoolVar(&log.Verbose, "v", log.Verbose, "enable log levels below INFO level")
}

var commands = make(map[string]*Command)

// registerCommand registers a command for main.
// registerCommand should only be called from init().
func registerCommand(name string, command *Command) {
	commands[name] = command
}

func main() {
	root, err := findProjectRoot(mustGetwd())
	if err != nil {
		log.Fatalf("could not locate project root: %v", err)
	}

	project, err := gogo.NewProject(root)
	if err != nil {
		log.Fatalf("unable to construct project: %v", err)
	}

	args := os.Args
	if len(args) < 2 {
		log.Fatalf("no command supplied")
	}
	cmd, ok := commands[args[1]]
	if !ok {
		log.Errorf("unknown command %q", args[1])
		fs.PrintDefaults()
		os.Exit(1)
	}
	cmd.AddFlags(fs)
	if err := fs.Parse(args[2:]); err != nil {
		log.Fatalf("could not parse flags: %v", err)
	}

	// must be below fs.Parse because the -q and -v flags will log.Infof
	log.Infof("project root %q", root)
	if err := cmd.Run(project, fs.Args()); err != nil {
		log.Fatalf("command %q failed: %v", args[1], err)
	}
}
