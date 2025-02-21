package versions

import (
	"context"
	"sync"
)

// preallocated slices based version

func ProcessConcurrentlyWithResultV1[T any, R any](
	ctx context.Context,
	tasks []T,
	taskFunc func(context.Context, T) (R, error),
	filterFunc func(R) bool, // Function to determine if a result should be included
) ([]R, []error) {
	var wg sync.WaitGroup
	results := make([]R, len(tasks))    // Preallocated slice for results
	errors := make([]error, len(tasks)) // Preallocated slice for errors

	for i, task := range tasks {
		wg.Add(1)
		go func(index int, task T) {
			defer wg.Done()

			// Stop execution if the context is canceled
			if ctx.Err() != nil {
				return
			}

			select {
			case <-ctx.Done(): // Check if context is canceled before running the task
				return
			default:
				result, err := taskFunc(ctx, task)
				if err != nil {
					errors[index] = err
				} else {
					results[index] = result
				}
			}
		}(i, task)
	}

	wg.Wait() // Wait for all tasks to complete

	// Apply the filter function to results
	var filteredResults []R
	for _, result := range results {
		if filterFunc(result) { // Include only results that pass the filter
			filteredResults = append(filteredResults, result)
		}
	}

	// Filter nil errors for cleaner return
	var filteredErrors []error
	for _, err := range errors {
		if err != nil {
			filteredErrors = append(filteredErrors, err)
		}
	}

	return filteredResults, filteredErrors
}

func ProcessConcurrentlyWithResultAndLimitV1[T any, R any](
	ctx context.Context,
	workerLimit int,
	tasks []T,
	taskFunc func(context.Context, T) (R, error),
	filterFunc func(R) bool, // Function to determine if a result should be included
) ([]R, []error) {
	results := make([]R, len(tasks))    // Preallocated slice for results
	errors := make([]error, len(tasks)) // Preallocated slice for errors

	taskChan := make(chan int, len(tasks)) // Channel to distribute tasks
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
					if ctx.Err() != nil {
						return // Double-check if context is already canceled
					}

					// Execute task and store result or error
					result, err := taskFunc(ctx, tasks[index])
					if err != nil {
						errors[index] = err
					} else {
						results[index] = result
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

	// Apply the filter function to results
	var filteredResults []R
	for _, result := range results {
		if filterFunc(result) { // Include only results that pass the filter
			filteredResults = append(filteredResults, result)
		}
	}

	// Filter nil errors for cleaner return
	var filteredErrors []error
	for _, err := range errors {
		if err != nil {
			filteredErrors = append(filteredErrors, err)
		}
	}

	return filteredResults, filteredErrors
}
