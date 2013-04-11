package gogo

type Target interface {
	Wait() error
}
