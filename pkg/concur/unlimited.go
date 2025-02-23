package concur

import (
	"context"
	"sync"
)

// ProcessConcurrentlyWithResult executes a list of tasks concurrently without limiting the number of workers,
// collects their results and errors.
//
// Parameters:
//   - ctx: Context for cancellation and timeout handling.
//   - tasks: A slice of input tasks of type T to be processed.
//   - taskFunc: A function that processes a task of type T and returns a result of type R and an error.
//
// Returns:
//   - A slice of filtered results (based on the provided filter function).
//   - A slice of non-nil errors encountered during task execution.
func ProcessConcurrentlyWithResult[T any, R any](
	ctx context.Context,
	tasks []T,
	taskFunc func(context.Context, T) (R, error),
) ([]R, []error) {
	type outcome struct {
		result     R
		hasContent bool
		err        error
	}
	outcomes := make([]outcome, len(tasks)) // Preallocated slice for results and errors

	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		go func(index int, task T) {
			defer wg.Done()

			// Stop execution if the context is canceled
			if ctx.Err() != nil {
				return
			}

			result, err := taskFunc(ctx, task)

			// Check again if context is canceled (during execution of taskFunc), before modifying results
			if ctx.Err() != nil {
				return
			}

			outcomes[index] = outcome{result: result, hasContent: true, err: err}
		}(i, task)
	}

	wg.Wait() // Wait for all tasks to complete

	// Collect results and errors
	var filteredErrors []error
	var filteredResults []R
	for _, o := range outcomes {
		if o.err != nil {
			filteredErrors = append(filteredErrors, o.err)
		} else {
			if o.hasContent {
				filteredResults = append(filteredResults, o.result)
			}
		}
	}
	return filteredResults, filteredErrors
}

// ProcessConcurrently is similar to ProcessConcurrentlyWithResult, but it does not collect results
func ProcessConcurrently[T any](
	ctx context.Context,
	tasks []T,
	taskFunc func(context.Context, T) error,
) []error {
	var wg sync.WaitGroup
	errors := make([]error, len(tasks)) // Preallocated slice for errors

	for i, task := range tasks {
		wg.Add(1)
		go func(index int, task T) {
			defer wg.Done()

			// Stop execution if the context is canceled
			if ctx.Err() != nil {
				return
			}

			err := taskFunc(ctx, task)

			// Check again if context is canceled (during execution of taskFunc), before modifying results
			if ctx.Err() != nil {
				return
			}

			if err != nil {
				errors[index] = err
			}
		}(i, task)
	}

	wg.Wait() // Wait for all tasks to complete

	// Filter nil errors for cleaner return
	var filteredErrors []error
	for _, err := range errors {
		if err != nil {
			filteredErrors = append(filteredErrors, err)
		}
	}

	return filteredErrors
}
