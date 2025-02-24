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

func ProcessConcurrentlyWithResultAndLimitV1[T any, R any](
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
