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
	filterFunc func(R) bool, // Function to determine if a result should be included
) ([]R, []error) {
	var wg sync.WaitGroup
	results := make([]R, len(tasks))    // Preallocated slice for results
	errors := make([]error, len(tasks)) // Preallocated slice for errors

	for i, task := range tasks {
		wg.Add(1)
		go func(index int, task T) {
			defer wg.Done()

			// Run the task and get the result or error
			result, err := taskFunc(ctx, task)
			if err != nil {
				errors[index] = err
			} else {
				results[index] = result
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
			errors[index] = taskFunc(ctx, task)
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
			for index := range taskChan {
				// Run the task and get the result or error
				result, err := taskFunc(ctx, tasks[index])
				if err != nil {
					errors[index] = err
				} else {
					results[index] = result
				}
			}
		}()
	}

	// Send tasks to the channel
	for i := range tasks {
		taskChan <- i
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

// ProcessConcurrentlyWithLimit is similar to ProcessConcurrentlyWithResultAndLimit, but it does not collect results
func ProcessConcurrentlyWithLimit[T any](
	ctx context.Context,
	workerLimit int,
	tasks []T,
	taskFunc func(context.Context, T) error,
) []error {
	errors := make([]error, len(tasks)) // Preallocated slice for errors

	taskChan := make(chan int, len(tasks)) // Channel to distribute tasks
	var wg sync.WaitGroup

	// Start a fixed number of worker goroutines
	for i := 0; i < workerLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range taskChan {
				errors[index] = taskFunc(ctx, tasks[index])
			}
		}()
	}

	// Send tasks to the channel
	for i := range tasks {
		taskChan <- i
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
