package versions

import (
	"context"
	"sync"
)

// channel-based version

func ProcessConcurrentlyWithResultV2[T any, R any](
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

func ProcessConcurrentlyWithResultAndLimitV2[T any, R any](
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
