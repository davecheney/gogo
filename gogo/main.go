package main

import (
	"flag"
	"log"
	"os"

	"github.com/davecheney/gogo"
	"github.com/davecheney/gogo/build"
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
	project := gogo.NewProject(mustGetwd())
	first, rest := flag.Arg(0), flag.Args()[1:]
	var cmd *gogo.Command
	switch first {
	case "build":
		cmd = build.Build
	default:
		log.Fatal("unknown command %q", first)
	}
	if err := cmd.Run(project, rest); err != nil {
		log.Fatal(err)
	}
}
