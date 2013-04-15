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

func main() {
	flag.Parse()
	project, err := gogo.NewProject(mustGetwd())
	if err != nil {
		log.Fatalf("unable to construct project: %v", err)
	}
	first, rest := flag.Arg(0), flag.Args()[1:]
	var cmd *Command
	switch first {
	case "build":
		cmd = BuildCmd
	case "test":
		cmd = TestCmd
	default:
		log.Fatal("unknown command %q", first)
	}
	if err := cmd.Run(project, rest); err != nil {
		log.Fatal(err)
	}
}
