package main

import (
	"flag"
	"log"
	"os"

	"github.com/davecheney/gogo"
	"github.com/davecheney/gogo/build"
	"github.com/davecheney/gogo/test"
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
	var cmd *gogo.Command
	switch first {
	case "build":
		cmd = build.BuildCmd
	case "test":
		cmd = test.TestCmd
	default:
		log.Fatal("unknown command %q", first)
	}
	if err := cmd.Run(project, rest); err != nil {
		log.Fatal(err)
	}
}
