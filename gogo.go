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
	AddDependantTarget(Target)
}

type execution struct {
	target        Target
	deps          []*execution
	ctx           *Context
	started, done chan struct{}
	err           struct {
		sync.Mutex
		val error
	}
}

func newExecution(target Target, ctx *Context, deps ...*execution) *execution {
	return &execution{
		target:  target,
		deps:    deps,
		ctx:     ctx,
		done:    make(chan struct{}),
		started: make(chan struct{}),
	}
}

func (e *execution) Deps() []*execution { return e.deps }

func (e *execution) setError(err error) {
	e.err.Lock()
	e.err.val = err
	e.err.Unlock()
}

func (e *execution) Execute() {
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
	log.Printf("%d %s successful", len(e.deps), e)
}

func (e *execution) Wait() error {
	<-e.done

	e.err.Lock()
	defer e.err.Unlock()
	return e.err.val
}

func (e *execution) String() string { return fmt.Sprintf("execution %q", e.target) }

func ExecuteTargets(targets []Target) error {
        executions := make(map[Target]*execution)
        for _, t := range targets {
                e := buildExecution(executions, t)
                go e.Execute()
        }
	var err error 
	for _, e := range executions {
		if err1 := e.Wait(); err1 != nil && err != nil {
			err = err1
		}	
	}
	return err
}

func buildExecution(m map[Target]*execution, t Target) *execution {
        var deps []*execution
        for _, d := range t.Deps() {
                deps = append(deps, buildExecution(m, d))
        }
        if _, ok := m[t]; !ok {
                m[t] = newExecution(t, nil, deps...)
        }
        return m[t]
}

