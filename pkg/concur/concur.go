package concur

import (
	"context"
	"sync"
)

// ProcessConcurrentlyWithResult executes a list of tasks concurrently without limiting the number of workers,
// collects their results and errors, and allows filtering of the results based on a user-defined filter function.
//
// Parameters:
//   - ctx: Context for cancellation and timeout handling.
//   - tasks: A slice of input tasks of type T to be processed.
//   - taskFunc: A function that processes a task of type T and returns a result of type R and an error.
//   - filterFunc: A function that determines whether a result of type R should be included in the final output.
//
// Returns:
//   - A slice of filtered results (based on the provided filter function).
//   - A slice of non-nil errors encountered during task execution.
func ProcessConcurrentlyWithResult[T any, R any](
	ctx context.Context,
	tasks []T,
	taskFunc func(context.Context, T) (R, error),
) ([]R, []error) {
	var wg sync.WaitGroup
	resultChan := make(chan R, len(tasks))  // Buffered channel for results
	errChan := make(chan error, len(tasks)) // Buffered channel for errors

	for _, task := range tasks {
		wg.Add(1)
		go func(task T) {
			defer wg.Done()

			select {
			case <-ctx.Done(): // Check if context is canceled before running the task
				return
			default:
				// Execute task and store result or error
				result, err := taskFunc(ctx, task)
				select {
				case <-ctx.Done(): // Stop sending results/errors if context is canceled
					return
				case resultChan <- result:
				}
				if err != nil {
					select {
					case errChan <- err:
					case <-ctx.Done():
						return
					}
				}
			}
		}(task)
	}

	// Close result & error channels once all goroutines finish
	go func() {
		wg.Wait() // Wait for all tasks to complete
		close(resultChan)
		close(errChan)
	}()

	// Collect results

	var results []R
	for result := range resultChan {
		results = append(results, result)
	}

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	return results, errors
}

// ProcessConcurrently is similar to ProcessConcurrentlyWithResult, but it does not collect results
func ProcessConcurrently[T any](
	ctx context.Context,
	tasks []T,
	taskFunc func(context.Context, T) error,
) []error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(tasks)) // Buffered channel for errors

	for _, task := range tasks {
		wg.Add(1)
		go func(task T) {
			defer wg.Done()

			select {
			case <-ctx.Done(): // Check if context is canceled before running the task
				return
			default:
				// Execute task and store error (if any)
				if err := taskFunc(ctx, task); err != nil {
					select {
					case errChan <- err: // Send error safely
					case <-ctx.Done(): // Stop sending if context is canceled
					}
				}
			}
		}(task)
	}

	// Close errChan safely after all goroutines finish
	go func() {
		wg.Wait() // Wait for all tasks to complete
		close(errChan)
	}()

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	return errors
}

// ProcessConcurrentlyWithResultAndLimit executes a list of tasks concurrently with a limited number of workers,
// collects their results and errors, and allows filtering of the results based on a user-defined filter function.
//
// Parameters:
//   - ctx: Context for cancellation and timeout handling.
//   - tasks: A slice of input tasks of type T to be processed.
//   - taskFunc: A function that processes a task of type T and returns a result of type R and an error.
//   - workerLimit: The maximum number of worker goroutines to execute tasks concurrently.
//   - filterFunc: A function that determines whether a result of type R should be included in the final output.
//     This allows for custom filtering of results, such as excluding zero values, nil pointers, or empty collections.
//
// Returns:
// - A slice of filtered results (based on the provided filter function).
// - A slice of non-nil errors encountered during task execution.
func ProcessConcurrentlyWithResultAndLimit[T any, R any](
	ctx context.Context,
	workerLimit int,
	tasks []T,
	taskFunc func(context.Context, T) (R, error),
) ([]R, []error) {
	resultChan := make(chan R, len(tasks))  // Buffered channel for results
	errChan := make(chan error, len(tasks)) // Buffered channel for errors
	taskChan := make(chan T, len(tasks))    // Channel to distribute tasks

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
				case task, ok := <-taskChan:
					if !ok {
						return // Exit if channel is closed
					}

					if ctx.Err() != nil {
						return // Double-check if context is already canceled
					}

					// Execute task and store result or error
					result, err := taskFunc(ctx, task)
					select {
					case <-ctx.Done(): // Stop sending results/errors if context is canceled
						return
					case resultChan <- result:
					}
					if err != nil {
						select {
						case errChan <- err:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}()
	}

	// Send tasks to the channel
	for _, task := range tasks {
		select {
		case <-ctx.Done(): // Stop sending tasks if context is canceled
			break
		case taskChan <- task:
		}
	}
	close(taskChan) // Close the channel to signal workers to stop

	// Close resultChan and errChan safely after all workers finish
	go func() {
		wg.Wait() // Wait for all tasks to complete
		close(resultChan)
		close(errChan)
	}()

	// Collect results

	var results []R
	for result := range resultChan {
		results = append(results, result)
	}

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	return results, errors
}

// ProcessConcurrentlyWithLimit is similar to ProcessConcurrentlyWithResultAndLimit, but it does not collect results
func ProcessConcurrentlyWithLimit[T any](
	ctx context.Context,
	workerLimit int,
	tasks []T,
	taskFunc func(context.Context, T) error,
) []error {
	errChan := make(chan error, len(tasks)) // Buffered channel for errors
	taskChan := make(chan T, len(tasks))    // Channel to distribute tasks

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
				case task, ok := <-taskChan:
					if !ok {
						return // Exit if channel is closed
					}

					if ctx.Err() != nil {
						return // Double-check if context is already canceled
					}

					// Execute task and store error (if any)
					if err := taskFunc(ctx, task); err != nil {
						select {
						case errChan <- err: // Send error safely
						case <-ctx.Done(): // Stop sending if context is canceled
							return
						}
					}
				}
			}
		}()
	}

	// Send tasks to the channel
	for _, task := range tasks {
		select {
		case <-ctx.Done(): // Stop sending tasks if context is canceled
			break
		case taskChan <- task:
		}
	}
	close(taskChan) // Close the channel to signal workers to stop

	// Close errChan safely after all workers finish
	go func() {
		wg.Wait() // Wait for all workers to finish
		close(errChan)
	}()

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	return errors
}
