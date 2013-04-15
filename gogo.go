package gogo

import (
	"sync"
)

type Target interface {
	Wait() error
}

type target struct {
	done chan struct{}
	err  struct {
		sync.Mutex
		val error
	}
}

func (t *target) Wait() error {
	<-t.done
	t.err.Lock()
	defer t.err.Unlock()
	return t.err.val
}

func (t *target) setErr(err error) {
	t.err.Lock()
	t.err.val = err
	t.err.Unlock()
}
