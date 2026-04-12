// Package concurrent provides utilities for executing tasks concurrently with timeout support.
package concurrent

import (
	"context"
	"sync"
	"time"
)

// Task represents a function that can be executed concurrently.
// T is the type of the result returned by the task.
type Task[T any] struct {
	Name string
	Fn   func(context.Context) (T, error)
}

// Result holds the result of a task execution.
type Result[T any] struct {
	Name  string
	Value T
	Error error
}

// Run executes tasks concurrently with individual timeouts.
// Each task is executed in its own goroutine with the specified timeout.
// If a task times out or errors, it returns a zero value but doesn't fail other tasks.
// The function waits for all tasks to complete before returning.
func Run[T any](ctx context.Context, timeout time.Duration, tasks []Task[T]) []Result[T] {
	if len(tasks) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	results := make([]Result[T], len(tasks))

	for i, task := range tasks {
		wg.Add(1)
		go func(index int, t Task[T]) {
			defer wg.Done()

			taskCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			val, err := t.Fn(taskCtx)
			results[index] = Result[T]{
				Name:  t.Name,
				Value: val,
				Error: err,
			}
		}(i, task)
	}

	wg.Wait()
	return results
}

// RunWithResults is a convenience function that executes tasks and returns only successful results.
// Tasks that error or timeout are filtered out from the returned slice.
func RunWithResults[T any](ctx context.Context, timeout time.Duration, tasks []Task[T]) []T {
	results := Run(ctx, timeout, tasks)

	var successful []T
	for _, result := range results {
		if result.Error == nil {
			successful = append(successful, result.Value)
		}
	}
	return successful
}

// TaskFunc is a simpler version of Task that doesn't return a named result.
// Useful when you only care about executing functions concurrently without collecting results.
type TaskFunc struct {
	Name string
	Fn   func(context.Context) error
}

// RunFuncs executes function tasks concurrently with timeout.
// Returns a map of task names to errors for tasks that failed.
func RunFuncs(ctx context.Context, timeout time.Duration, tasks []TaskFunc) map[string]error {
	if len(tasks) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errors := make(map[string]error)
	var mu sync.Mutex

	for _, task := range tasks {
		wg.Add(1)
		go func(t TaskFunc) {
			defer wg.Done()

			taskCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			if err := t.Fn(taskCtx); err != nil {
				mu.Lock()
				errors[t.Name] = err
				mu.Unlock()
			}
		}(task)
	}

	wg.Wait()
	return errors
}
