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

var commands = map[string]*gogo.Command{
	"build": build.Build,
}

func main() {
	flag.Parse()
	project := gogo.NewProject(mustGetwd())
	first, rest := flag.Arg(0), flag.Args()[1:]
	cmd, ok := commands[first]
	if !ok {
		log.Fatal("unknown command %q", first)
	}
	if err := cmd.Run(project, rest); err != nil {
		log.Fatal(err)
	}
}
