package versions

import (
	"context"
	"sync"
)

// pack channels in struct

func ProcessConcurrentlyWithResultAndLimitV4[T any, R any](
	ctx context.Context,
	workerLimit int,
	tasks []T,
	taskFunc func(context.Context, T) (R, error),
) ([]R, []error) {
	type outcome struct {
		result R
		err    error
	}
	outcomeCh := make(chan outcome, len(tasks))
	taskChan := make(chan T, len(tasks))
	var wg sync.WaitGroup

	// Start a fixed number of worker goroutines.
	for i := 0; i < workerLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				// Non-blocking check before processing each task.
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Execute the task.
				result, err := taskFunc(ctx, task)
				// Optionally, drop results if the context was canceled during execution.
				if ctx.Err() != nil {
					return
				}

				// Send the outcome to the channel.
				outcomeCh <- outcome{result: result, err: err}
			}
		}()
	}

	// Distribute tasks to the workers.
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			break
		case taskChan <- task:
		}
	}
	close(taskChan) // Signal that no more tasks will be sent.

	// Wait for all workers to finish processing.
	wg.Wait()
	close(outcomeCh)

	// Collect outcomes from the channel.
	var results []R
	var errors []error
	for o := range outcomeCh {
		if o.err != nil {
			errors = append(errors, o.err)
		} else {
			results = append(results, o.result)
		}
	}
	return results, errors
}
