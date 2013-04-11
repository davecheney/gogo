package gogo

type Command struct {
	Run func(project *Project, args []string) error
}
