package main

import (
	"context"
	"io"
)

// Task is a function that receives a resource and returns an error.
type Task[R io.Closer] func(R) error

// Pool manages a pool of workers, each with its own resource.
type Pool[R io.Closer] struct {
	tasks  chan Task[R]
	cancel context.CancelFunc
}

// New creates a new Pool with n workers. creator creates a resource for each worker.
func New[R io.Closer](ctx context.Context, n int, creator func(context.Context) (R, error)) *Pool[R] {
	tasks := make(chan Task[R])
	ctx, cancel := context.WithCancel(ctx)
	p := &Pool[R]{tasks: tasks, cancel: cancel}
	for i := 0; i < n; i++ {
		go func() {
			resource, err := creator(ctx)
			if err != nil {
				return
			}
			defer resource.Close()
			for {
				select {
				case <-ctx.Done():
					return
				case task, ok := <-tasks:
					if !ok {
						return
					}
					_ = task(resource)
				}
			}
		}()
	}
	return p
}

// Submit adds a task to the pool.
func (p *Pool[R]) Submit(task Task[R]) {
	p.tasks <- task
}

// Close stops the pool and closes the task channel.
func (p *Pool[R]) Close() {
	p.cancel()
	close(p.tasks)
}
