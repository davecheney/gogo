package gogo

import (
	"fmt"
	"log"
	"sync"
)

type Context struct {
}

type Target interface {
	Execute(*Context) error
	Deps() []Target
}

type Execution struct {
	target        Target
	deps          []*Execution
	ctx           *Context
	started, done chan struct{}
	err           struct {
		sync.Mutex
		val error
	}
}

func NewExecution(target Target, ctx *Context, deps ...*Execution) *Execution {
	return &Execution{
		target:  target,
		deps:    deps,
		ctx:     ctx,
		done:    make(chan struct{}),
		started: make(chan struct{}),
	}
}

func (e *Execution) Deps() []*Execution { return e.deps }

func (e *Execution) setError(err error) {
	e.err.Lock()
	e.err.val = err
	e.err.Unlock()
}

func (e *Execution) Execute() {
	close(e.started) // will panic if executed twice
	defer close(e.done)

	for _, dep := range e.deps {
		// log.Printf("%s waiting on %s", e, dep)
		if err := dep.Wait(); err != nil {
			log.Printf("dependent %s failed: %v", dep, err)
			e.setError(err)
			return
		}
	}
	if err := e.target.Execute(e.ctx); err != nil {
		log.Printf("%s failed: %v", e, err)
		e.setError(err)
		return
	}
	log.Printf("%s successful", e)
}

func (e *Execution) Wait() error {
	<-e.done

	e.err.Lock()
	defer e.err.Unlock()
	return e.err.val
}

func (e *Execution) String() string { return fmt.Sprintf("execution %q", e.target) }
