package concur

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Helper: extract number of outcomes with HasContent=true
func countHasContent[T any, R any](outcomes []TaskOutcome[T, R]) int {
	n := 0
	for _, o := range outcomes {
		if o.HasContent {
			n++
		}
	}
	return n
}

// Basic happy-path: all tasks processed, no errors, correct Index/Input/Result.
func TestProcessConcurrently_AllTasksProcessed_NoError(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5}
	ctx := context.Background()

	taskFunc := func(ctx context.Context, x int) (int, error) {
		return x * x, nil
	}

	outcomes := ProcessConcurrentlyWithResultAndLimitV2(ctx, 3, tasks, taskFunc)
	assert.Len(t, outcomes, len(tasks))

	for i, o := range outcomes {
		assert.Equal(t, i, o.Index, "Index should match original position")
		assert.Equal(t, tasks[i], o.Input, "Input should be pre-filled with original task")
		assert.True(t, o.HasContent, "All tasks should have been processed")
		assert.NoError(t, o.Err, "No error expected for task %d", i)
		assert.Equal(t, tasks[i]*tasks[i], o.Result, "Result should be square of input")
	}
}

// Verify that errors are propagated and mixed success/failure is handled correctly.
func TestProcessConcurrently_ErrorsAndResults(t *testing.T) {
	tasks := []int{1, 2, 3, 4}
	ctx := context.Background()

	someErr := errors.New("even number error")

	taskFunc := func(ctx context.Context, x int) (int, error) {
		if x%2 == 0 {
			return 0, someErr
		}
		return x * 10, nil
	}

	outcomes := ProcessConcurrentlyWithResultAndLimitV2(ctx, 2, tasks, taskFunc)
	assert.Len(t, outcomes, len(tasks))

	for i, o := range outcomes {
		assert.Equal(t, i, o.Index)
		assert.Equal(t, tasks[i], o.Input)
		assert.True(t, o.HasContent, "Task %d should have been processed", i)

		if o.Input%2 == 0 {
			// Even -> error
			assert.Error(t, o.Err)
			assert.Same(t, someErr, o.Err, "Error should be the same instance")
		} else {
			// Odd -> success
			assert.NoError(t, o.Err)
			assert.Equal(t, o.Input*10, o.Result)
		}
	}
}

// workerLimit < 1 should behave like workerLimit == 1 (no panic, all tasks processed).
func TestProcessConcurrently_WorkerLimitLessThanOne(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5, 6}
	ctx := context.Background()

	var maxConcurrent int32
	var current int32

	taskFunc := func(ctx context.Context, x int) (int, error) {
		// Simple concurrency tracking: not bulletproof, but good enough sanity check
		cur := atomic.AddInt32(&current, 1)
		if cur > atomic.LoadInt32(&maxConcurrent) {
			atomic.StoreInt32(&maxConcurrent, cur)
		}
		defer atomic.AddInt32(&current, -1)

		time.Sleep(5 * time.Millisecond)
		return x + 1, nil
	}

	outcomes := ProcessConcurrentlyWithResultAndLimitV2(ctx, 0, tasks, taskFunc)
	assert.Len(t, outcomes, len(tasks))

	for i, o := range outcomes {
		assert.True(t, o.HasContent)
		assert.NoError(t, o.Err)
		assert.Equal(t, tasks[i]+1, o.Result)
	}

	// We don't assert exact concurrency level here, just that it didn't explode
	assert.GreaterOrEqual(t, maxConcurrent, int32(1))
}

// Empty tasks slice should return an empty outcomes slice without panicking.
func TestProcessConcurrently_EmptyTasks(t *testing.T) {
	var tasks []int
	ctx := context.Background()

	taskFunc := func(ctx context.Context, x int) (int, error) {
		return x * 2, nil
	}

	outcomes := ProcessConcurrentlyWithResultAndLimitV2(ctx, 4, tasks, taskFunc)
	assert.Len(t, outcomes, 0)
}

// If the context is already canceled before we start, no taskFunc calls should happen.
// HasContent should remain false, but Index/Input must still be pre-filled.
func TestProcessConcurrently_ContextAlreadyCanceled(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling

	var calls int32
	taskFunc := func(ctx context.Context, x int) (int, error) {
		atomic.AddInt32(&calls, 1)
		return x * 2, nil
	}

	outcomes := ProcessConcurrentlyWithResultAndLimitV2(ctx, 3, tasks, taskFunc)
	assert.Len(t, outcomes, len(tasks))
	assert.Equal(t, int32(0), atomic.LoadInt32(&calls), "taskFunc should never be called when ctx is pre-canceled")

	for i, o := range outcomes {
		assert.Equal(t, i, o.Index)
		assert.Equal(t, tasks[i], o.Input)
		assert.False(t, o.HasContent, "No tasks should be processed when ctx is pre-canceled")
		assert.Nil(t, o.Err)
	}
}

// Cancellation during processing: some tasks should be processed, some not.
// HasContent should correctly reflect which tasks actually ran.
func TestProcessConcurrently_RespectsCancellationDuringProcessing(t *testing.T) {
	tasks := make([]int, 50)
	for i := range tasks {
		tasks[i] = i + 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var startedCount int32

	taskFunc := func(ctx context.Context, x int) (int, error) {
		// Count how many tasks actually started
		cur := atomic.AddInt32(&startedCount, 1)

		// After a few tasks, trigger cancellation
		if cur == 5 {
			cancel()
		}

		// Small delay to simulate work
		time.Sleep(10 * time.Millisecond)
		return x * 3, nil
	}

	outcomes := ProcessConcurrentlyWithResultAndLimitV2(ctx, 5, tasks, taskFunc)
	assert.Len(t, outcomes, len(tasks))

	processed := countHasContent(outcomes)
	started := int(atomic.LoadInt32(&startedCount))

	// At least some tasks must have been processed.
	assert.Greater(t, processed, 0)
	assert.Greater(t, started, 0)

	// Not all tasks should be processed due to cancellation.
	assert.Less(t, processed, len(tasks))

	// For tasks that were processed, Result and Err should be consistent.
	for i, o := range outcomes {
		assert.Equal(t, i, o.Index)
		assert.Equal(t, tasks[i], o.Input)

		if o.HasContent {
			assert.NoError(t, o.Err)
			assert.Equal(t, o.Input*3, o.Result)
		} else {
			// Unprocessed tasks: zero-value Result, nil error
			assert.Nil(t, o.Err)
		}
	}
}

// Ensure outcomes are sorted by Index (even if processing happens out-of-order).
func TestProcessConcurrently_ResultsSortedByIndex(t *testing.T) {
	tasks := []int{10, 20, 30, 40, 50}
	ctx := context.Background()

	taskFunc := func(ctx context.Context, x int) (int, error) {
		// Introduce small jitter based on value to encourage out-of-order completion
		time.Sleep(time.Duration(60-x/2) * time.Millisecond)
		return x / 10, nil
	}

	outcomes := ProcessConcurrentlyWithResultAndLimitV2(ctx, 3, tasks, taskFunc)
	assert.Len(t, outcomes, len(tasks))

	for i, o := range outcomes {
		// Because the function sorts by Index, outcomes slice should be in index order.
		assert.Equal(t, i, o.Index, "outcome slice must be sorted by Index")
		assert.Equal(t, tasks[i], o.Input)
		assert.True(t, o.HasContent)
		assert.NoError(t, o.Err)
		assert.Equal(t, tasks[i]/10, o.Result)
	}
}
