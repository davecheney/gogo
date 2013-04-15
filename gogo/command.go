package main

import "github.com/davecheney/gogo"

type Command struct {
	Run func(project *gogo.Project, args []string) error
}
