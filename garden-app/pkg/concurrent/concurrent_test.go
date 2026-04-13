package concurrent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestRunBasic(t *testing.T) {
	tasks := []Task[int]{
		{
			Name: "task1",
			Fn: func(ctx context.Context) (int, error) {
				return 1, nil
			},
		},
		{
			Name: "task2",
			Fn: func(ctx context.Context) (int, error) {
				return 2, nil
			},
		},
		{
			Name: "task3",
			Fn: func(ctx context.Context) (int, error) {
				return 3, nil
			},
		},
	}

	results := Run(context.Background(), time.Second, tasks)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	expected := map[string]int{
		"task1": 1,
		"task2": 2,
		"task3": 3,
	}

	for _, result := range results {
		if result.Error != nil {
			t.Errorf("unexpected error for %s: %v", result.Name, result.Error)
		}
		if result.Value != expected[result.Name] {
			t.Errorf("expected %d for %s, got %d", expected[result.Name], result.Name, result.Value)
		}
	}
}

func TestRunEmptyTasks(t *testing.T) {
	results := Run[any](context.Background(), time.Second, nil)
	if results != nil {
		t.Errorf("expected nil for empty tasks, got %v", results)
	}

	results2 := Run(context.Background(), time.Second, []Task[int]{})
	if results2 != nil {
		t.Errorf("expected nil for empty tasks slice, got %v", results2)
	}
}

func TestRunWithErrors(t *testing.T) {
	testErr := errors.New("test error")
	tasks := []Task[int]{
		{
			Name: "success",
			Fn: func(ctx context.Context) (int, error) {
				return 42, nil
			},
		},
		{
			Name: "failure",
			Fn: func(ctx context.Context) (int, error) {
				return 0, testErr
			},
		},
	}

	results := Run(context.Background(), time.Second, tasks)

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	for _, result := range results {
		switch result.Name {
		case "success":
			if result.Error != nil {
				t.Errorf("unexpected error for success task: %v", result.Error)
			}
			if result.Value != 42 {
				t.Errorf("expected 42, got %d", result.Value)
			}
		case "failure":
			if result.Error != testErr {
				t.Errorf("expected test error, got %v", result.Error)
			}
			if result.Value != 0 {
				t.Errorf("expected 0 for failed task, got %d", result.Value)
			}
		}
	}
}

func TestRunTimeout(t *testing.T) {
	tasks := []Task[int]{
		{
			Name: "fast",
			Fn: func(ctx context.Context) (int, error) {
				return 1, nil
			},
		},
		{
			Name: "slow",
			Fn: func(ctx context.Context) (int, error) {
				select {
				case <-time.After(100 * time.Millisecond):
					return 2, nil
				case <-ctx.Done():
					return 0, ctx.Err()
				}
			},
		},
	}

	// Set a very short timeout
	results := Run(context.Background(), 10*time.Millisecond, tasks)

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	for _, result := range results {
		switch result.Name {
		case "fast":
			if result.Error != nil {
				t.Errorf("unexpected error for fast task: %v", result.Error)
			}
			if result.Value != 1 {
				t.Errorf("expected 1, got %d", result.Value)
			}
		case "slow":
			if result.Error != context.DeadlineExceeded {
				t.Errorf("expected deadline exceeded error, got %v", result.Error)
			}
		}
	}
}

func TestRunWithResults(t *testing.T) {
	testErr := errors.New("test error")
	tasks := []Task[int]{
		{
			Name: "success1",
			Fn: func(ctx context.Context) (int, error) {
				return 1, nil
			},
		},
		{
			Name: "failure",
			Fn: func(ctx context.Context) (int, error) {
				return 0, testErr
			},
		},
		{
			Name: "success2",
			Fn: func(ctx context.Context) (int, error) {
				return 2, nil
			},
		},
	}

	results := RunWithResults(context.Background(), time.Second, tasks)

	// Should only have successful results
	if len(results) != 2 {
		t.Errorf("expected 2 successful results, got %d", len(results))
	}

	// Check values (order may vary)
	sum := 0
	for _, v := range results {
		sum += v
	}
	if sum != 3 {
		t.Errorf("expected sum of 3, got %d", sum)
	}
}

func TestRunWithResultsEmpty(t *testing.T) {
	results := RunWithResults[int](context.Background(), time.Second, nil)
	if results != nil {
		t.Errorf("expected nil for empty tasks, got %v", results)
	}
}

func TestRunWithResultsAllFailures(t *testing.T) {
	testErr := errors.New("test error")
	tasks := []Task[int]{
		{
			Name: "failure1",
			Fn: func(ctx context.Context) (int, error) {
				return 0, testErr
			},
		},
		{
			Name: "failure2",
			Fn: func(ctx context.Context) (int, error) {
				return 0, testErr
			},
		},
	}

	results := RunWithResults(context.Background(), time.Second, tasks)

	if len(results) != 0 {
		t.Errorf("expected 0 results (all failed), got %d", len(results))
	}
}

func TestRunFuncsBasic(t *testing.T) {
	var executed []string
	var mu sync.Mutex
	tasks := []TaskFunc{
		{
			Name: "task1",
			Fn: func(ctx context.Context) error {
				mu.Lock()
				executed = append(executed, "task1")
				mu.Unlock()
				return nil
			},
		},
		{
			Name: "task2",
			Fn: func(ctx context.Context) error {
				mu.Lock()
				executed = append(executed, "task2")
				mu.Unlock()
				return nil
			},
		},
	}

	errors := RunFuncs(context.Background(), time.Second, tasks)

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %v", errors)
	}

	mu.Lock()
	if len(executed) != 2 {
		t.Errorf("expected 2 tasks to execute, got %d", len(executed))
	}
	mu.Unlock()
}

func TestRunFuncsEmpty(t *testing.T) {
	errors := RunFuncs(context.Background(), time.Second, nil)
	if errors != nil {
		t.Errorf("expected nil for empty tasks, got %v", errors)
	}

	errors = RunFuncs(context.Background(), time.Second, []TaskFunc{})
	if errors != nil {
		t.Errorf("expected nil for empty tasks slice, got %v", errors)
	}
}

func TestRunFuncsWithErrors(t *testing.T) {
	testErr1 := errors.New("error1")
	testErr2 := errors.New("error2")

	tasks := []TaskFunc{
		{
			Name: "success",
			Fn: func(ctx context.Context) error {
				return nil
			},
		},
		{
			Name: "failure1",
			Fn: func(ctx context.Context) error {
				return testErr1
			},
		},
		{
			Name: "failure2",
			Fn: func(ctx context.Context) error {
				return testErr2
			},
		},
	}

	errs := RunFuncs(context.Background(), time.Second, tasks)

	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errs))
	}

	if errs["failure1"] != testErr1 {
		t.Errorf("expected error1 for failure1, got %v", errs["failure1"])
	}

	if errs["failure2"] != testErr2 {
		t.Errorf("expected error2 for failure2, got %v", errs["failure2"])
	}

	if _, hasSuccess := errs["success"]; hasSuccess {
		t.Error("should not have error for success task")
	}
}

func TestRunFuncsTimeout(t *testing.T) {
	tasks := []TaskFunc{
		{
			Name: "fast",
			Fn: func(ctx context.Context) error {
				return nil
			},
		},
		{
			Name: "slow",
			Fn: func(ctx context.Context) error {
				select {
				case <-time.After(100 * time.Millisecond):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		},
	}

	// Set a very short timeout
	errs := RunFuncs(context.Background(), 10*time.Millisecond, tasks)

	if len(errs) != 1 {
		t.Errorf("expected 1 error (slow task timeout), got %d", len(errs))
	}

	if errs["slow"] != context.DeadlineExceeded {
		t.Errorf("expected deadline exceeded for slow task, got %v", errs["slow"])
	}

	if _, hasFast := errs["fast"]; hasFast {
		t.Error("fast task should not have error")
	}
}

func TestRunContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	tasks := []Task[int]{
		{
			Name: "task1",
			Fn: func(taskCtx context.Context) (int, error) {
				// Wait for context cancellation
				<-taskCtx.Done()
				return 0, taskCtx.Err()
			},
		},
	}

	// Cancel context after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	results := Run(ctx, time.Second, tasks)

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if results[0].Error != context.Canceled {
		t.Errorf("expected context canceled error, got %v", results[0].Error)
	}
}
