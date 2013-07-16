package main

import (
	"flag"
	"os"
	"runtime"

	"github.com/davecheney/gogo/log"
	"github.com/davecheney/gogo/project"
)

var (
	fs        = flag.NewFlagSet("gogo", flag.ExitOnError)
	goos      = fs.String("goos", runtime.GOOS, "override GOOS")
	goarch    = fs.String("goarch", runtime.GOARCH, "override GOARCH")
	goroot    = fs.String("goroot", runtime.GOROOT(), "override GOROOT")
	toolchain = fs.String("toolchain", "gc", "choose go compiler toolchain")
)

func init() {
	// setup logging variables
	fs.BoolVar(&log.Quiet, "q", log.Quiet, "suppress log messages below ERROR level")
	fs.BoolVar(&log.Verbose, "v", log.Verbose, "enable log levels below INFO level")
}

type Command struct {
	Run      func(*project.Project, []string) error
	AddFlags func(*flag.FlagSet)
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("unable to determine current working directory: %v", err)
	}
	return wd
}

var commands = make(map[string]*Command)

// registerCommand registers a command for main.
// registerCommand should only be called from init().
func registerCommand(name string, command *Command) {
	commands[name] = command
}

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Fatalf("no command supplied")
	}
}
