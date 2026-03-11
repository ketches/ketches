package concurrency

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
)

// Run applies fn to each item in items using a bounded worker pool.
//
// Behavior:
//   - workerCount <= 0: auto-select based on CPU count.
//   - workerCount > len(items): capped to len(items).
//   - items empty: returns nil.
//   - collects all processing errors and returns them as a single error.
//
// Note: fn is executed concurrently, so callers must ensure it is thread-safe.
func Run[T any](items []T, workerCount int, fn func(T) error) error {
	return RunWithConfig(items, Config[T]{
		WorkerCount: workerCount,
		Fn:          fn,
	})
}

// Config defines worker-pool settings for RunWithConfig.
type Config[T any] struct {
	WorkerCount int
	Fn          func(T) error
}

// RunWithConfig applies cfg.Fn to each item in items concurrently.
func RunWithConfig[T any](items []T, cfg Config[T]) error {
	if len(items) == 0 {
		return nil
	}
	if cfg.Fn == nil {
		return fmt.Errorf("worker pool: processing function is nil")
	}

	workers := normalizeWorkerCount(cfg.WorkerCount, len(items))

	jobs := make(chan T, len(items))
	errCh := make(chan error, len(items))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				if err := cfg.Fn(item); err != nil {
					errCh <- err
				}
			}
		}()
	}

	for _, item := range items {
		jobs <- item
	}
	close(jobs)

	wg.Wait()
	close(errCh)

	return collectErrors(errCh)
}

// normalizeWorkerCount returns a safe worker count:
//   - at least 1
//   - at most itemCount
func normalizeWorkerCount(workerCount, itemCount int) int {
	if itemCount <= 0 {
		return 1
	}
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > itemCount {
		workerCount = itemCount
	}
	return workerCount
}

// collectErrors joins all errors from errCh into a single error.
// Returns nil when no errors occurred.
func collectErrors(errCh <-chan error) error {
	errs := make([]string, 0)
	for err := range errCh {
		if err == nil {
			continue
		}
		errs = append(errs, err.Error())
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}
