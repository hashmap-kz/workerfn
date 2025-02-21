package versions

import (
	"context"
	"sync"
)

// mutex-based approach

func ProcessConcurrentlyWithResultV3[T any, R any](
	ctx context.Context,
	tasks []T,
	taskFunc func(context.Context, T) (R, error),
) ([]R, []error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]R, 0, len(tasks))
	errors := make([]error, 0, len(tasks))

	for _, task := range tasks {
		wg.Add(1)
		go func(task T) {
			defer wg.Done()

			// Check if context is already canceled before running
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Execute the task
			result, err := taskFunc(ctx, task)

			// Check again if context is canceled before appending results
			if ctx.Err() != nil {
				return
			}

			mu.Lock()
			if err != nil {
				errors = append(errors, err)
			} else {
				results = append(results, result)
			}
			mu.Unlock()
		}(task)
	}

	wg.Wait() // Ensure all tasks are completed
	return results, errors
}

func ProcessConcurrentlyWithResultAndLimitV3[T any, R any](
	ctx context.Context,
	workerLimit int,
	tasks []T,
	taskFunc func(context.Context, T) (R, error),
) ([]R, []error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]R, 0, len(tasks))
	errors := make([]error, 0, len(tasks))
	taskChan := make(chan T, len(tasks)) // Buffered channel to distribute tasks

	// Start worker goroutines
	for i := 0; i < workerLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				// Check if context is already canceled before executing the task
				if ctx.Err() != nil {
					return
				}

				// Execute task
				result, err := taskFunc(ctx, task)

				// Check again if context is canceled before modifying results
				if ctx.Err() != nil {
					return
				}

				// Safely append results and errors
				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					results = append(results, result)
				}
				mu.Unlock()
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
	close(taskChan) // Signal workers that all tasks are sent

	// Wait for workers to finish
	wg.Wait()

	return results, errors
}
