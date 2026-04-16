package concur

import (
	"context"
	"sync"
)

// TaskOutcome returns an original task, its index, the result of task execution and optional error
type TaskOutcome[T any, R any] struct {
	Index  int // original index in tasks slice
	Input  T   // original task input
	Result R
	Err    error
	// since we're using preallocated slice for store both results and errors -
	// we have to filter elements -> whether an element was set by index, or it's just an
	// empty preallocated value, that we don't want to include in results lists.
	HasContent bool // whether the task actually ran & finished
}

// ProcessConcurrentlyWithResultAndLimit executes a list of tasks concurrently with a limited number of workers,
// collects their results and errors.
//
// Parameters:
//   - ctx: Context for cancellation and timeout handling.
//   - tasks: A slice of input tasks of type T to be processed.
//   - taskFunc: A function that processes a task of type T and returns a result of type R and an error.
//     taskFunc must synchronize any goroutines it spawns before returning.
//     Goroutines that outlive taskFunc may cause data races on outcomes.
//   - workerLimit: The maximum number of worker goroutines to execute tasks concurrently.
//
// Returns:
// - A slice results.
// - A slice of non-nil errors encountered during task execution.
func ProcessConcurrentlyWithResultAndLimitV2[T any, R any](
	ctx context.Context,
	workerLimit int,
	tasks []T,
	taskFunc func(context.Context, T) (R, error),
) []TaskOutcome[T, R] {
	if len(tasks) == 0 {
		return []TaskOutcome[T, R]{}
	}

	if workerLimit < 1 {
		workerLimit = 1
	}

	if workerLimit > len(tasks) {
		workerLimit = len(tasks)
	}

	outcomes := make([]TaskOutcome[T, R], len(tasks)) // Preallocated slice for results and errors
	taskChan := make(chan int, workerLimit+2)         // Channel to distribute tasks
	var wg sync.WaitGroup

	// Pre-fill inputs + index so caller always sees them
	for i, t := range tasks {
		outcomes[i].Index = i
		outcomes[i].Input = t
	}

	for i := 0; i < workerLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				// Stop processing if context is canceled
				case <-ctx.Done():
					return
				case idx, ok := <-taskChan:
					// Exit if channel is closed
					if !ok {
						return
					}

					// Check if context is already canceled before executing the task
					if ctx.Err() != nil {
						return
					}

					res, err := taskFunc(ctx, tasks[idx])

					// If ctx was canceled mid-task, you can still store result/err
					// or bail out here. I'll store it, because it's safer for callers.
					outcomes[idx].Result = res
					outcomes[idx].Err = err
					outcomes[idx].HasContent = true
				}
			}
		}()
	}

	flush := func() {
		close(taskChan)
		wg.Wait()
	}

	for i := range tasks {
		select {
		case <-ctx.Done():
			flush()
			return outcomes
		case taskChan <- i:
		}
	}

	flush()
	return outcomes
}
