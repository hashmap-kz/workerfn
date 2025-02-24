package concur

import (
	"context"
	"sync"
)

// ProcessConcurrentlyWithResultAndLimit executes a list of tasks concurrently with a limited number of workers,
// collects their results and errors.
//
// Parameters:
//   - ctx: Context for cancellation and timeout handling.
//   - tasks: A slice of input tasks of type T to be processed.
//   - taskFunc: A function that processes a task of type T and returns a result of type R and an error.
//   - workerLimit: The maximum number of worker goroutines to execute tasks concurrently.
//
// Returns:
// - A slice results.
// - A slice of non-nil errors encountered during task execution.
func ProcessConcurrentlyWithResultAndLimit[T any, R any](
	ctx context.Context,
	workerLimit int,
	tasks []T,
	taskFunc func(context.Context, T) (R, error),
) ([]R, []error) {
	if workerLimit < 1 {
		workerLimit = 1
	}

	type outcome struct {
		// since we're using preallocated slice for store both results and errors -
		// we have to filter elements -> whether an element was set by index, or it's just an
		// empty preallocated value, that we don't want to include in results lists.
		hasContent bool
		result     R
		err        error
	}

	outcomes := make([]outcome, len(tasks))   // Preallocated slice for results and errors
	taskChan := make(chan int, workerLimit+2) // Channel to distribute tasks
	var wg sync.WaitGroup

	// Start a fixed number of worker goroutines
	for i := 0; i < workerLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done(): // Stop processing if context is canceled
					return
				case index, ok := <-taskChan:
					if !ok {
						return // Exit if channel is closed
					}

					// Check if context is already canceled before executing the task
					if ctx.Err() != nil {
						return
					}

					// Execute task and store result or error
					result, err := taskFunc(ctx, tasks[index])

					// Check again if context is canceled (during execution of taskFunc), before modifying results
					if ctx.Err() != nil {
						return
					}

					outcomes[index] = outcome{hasContent: true, result: result, err: err}
				}
			}
		}()
	}

	// Send tasks to the channel
	for i := range tasks {
		select {
		case <-ctx.Done(): // Stop sending tasks if context is canceled
			break
		case taskChan <- i:
		}
	}
	close(taskChan) // Close the channel to signal workers to stop

	wg.Wait() // Wait for all workers to finish

	// Collect results and errors
	var filteredErrors []error
	var filteredResults []R
	for _, o := range outcomes {
		if !o.hasContent {
			continue
		}
		if o.err != nil {
			filteredErrors = append(filteredErrors, o.err)
		} else {
			filteredResults = append(filteredResults, o.result)
		}
	}
	return filteredResults, filteredErrors
}

// ProcessConcurrentlyWithLimit is similar to ProcessConcurrentlyWithResultAndLimit, but it does not collect results
func ProcessConcurrentlyWithLimit[T any](
	ctx context.Context,
	workerLimit int,
	tasks []T,
	taskFunc func(context.Context, T) error,
) []error {
	if workerLimit < 1 {
		workerLimit = 1
	}

	errors := make([]error, len(tasks))       // Preallocated slice for errors
	taskChan := make(chan int, workerLimit+2) // Channel to distribute tasks
	var wg sync.WaitGroup

	// Start a fixed number of worker goroutines
	for i := 0; i < workerLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done(): // Stop processing if context is canceled
					return
				case index, ok := <-taskChan:
					if !ok {
						return // Exit if channel is closed
					}

					// Check if context is already canceled before executing the task
					if ctx.Err() != nil {
						return
					}

					err := taskFunc(ctx, tasks[index])

					// Check again if context is canceled (during execution of taskFunc), before modifying results
					if ctx.Err() != nil {
						return
					}

					if err != nil {
						errors[index] = err
					}
				}
			}
		}()
	}

	// Send tasks to the channel
	for i := range tasks {
		select {
		case <-ctx.Done(): // Stop sending tasks if context is canceled
			break
		case taskChan <- i:
		}
	}
	close(taskChan) // Close the channel to signal workers to stop

	wg.Wait() // Wait for all workers to finish

	// Filter nil errors for cleaner return
	var filteredErrors []error
	for _, err := range errors {
		if err != nil {
			filteredErrors = append(filteredErrors, err)
		}
	}

	return filteredErrors
}
