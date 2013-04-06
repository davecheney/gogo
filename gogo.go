package gogo

import (
	"sync"
)

type Context struct {
}

type Target interface {
	Execute(*Context) error
	Deps() []Target
}

type Execution struct {
	target Target
	deps   []*Execution
	ctx    *Context
	count  sync.WaitGroup
	done   sync.WaitGroup
	err    struct {
		sync.Mutex
		val error
	}
}

func NewExecution(target Target, ctx *Context, deps ...*Execution) *Execution {
	e := &Execution{
		target: target,
		deps:   deps,
		ctx:    ctx,
	}
	e.count.Add(1)
	e.done.Add(1)
	return e
}

func (e *Execution) Execute() {
	e.count.Done() // will panic if executed twice
	defer e.done.Done()

	e.err.Lock()
	defer e.err.Unlock()

	for _, dep := range e.deps {
		if e.err.val = dep.Wait(); e.err.val != nil {
			return
		}
	}

	e.err.val = e.target.Execute(e.ctx)
}

func (e *Execution) Wait() error {
	e.done.Wait()

	e.err.Lock()
	defer e.err.Unlock()
	return e.err.val
}
