package main

import (
	"flag"
	"log"
	"os"

	"github.com/davecheney/gogo"
)

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

func main() {
	flag.Parse()
	project, err := gogo.NewProject(mustGetwd())
	if err != nil {
		log.Fatalf("unable to construct project: %v", err)
	}
	if flag.NArg() < 1 {
		log.Fatalf("no command supplied")
	}
	first, rest := flag.Arg(0), flag.Args()[1:]
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
