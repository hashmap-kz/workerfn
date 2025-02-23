package concur

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Task function for testing
func mockTaskW(_ context.Context, input int) (int, error) {
	if input%2 == 0 {
		return input * 2, nil // Double even numbers
	}
	return 0, errors.New("odd number error") // Return error for odd numbers
}

func mockTaskSuccessW(_ context.Context, _ int) error {
	return nil // No error
}

func mockTaskFailureW(_ context.Context, input int) error {
	if input%2 == 0 {
		return errors.New("task failed")
	}
	return nil
}

// with results

// Test ProcessConcurrentlyWithResultAndLimit
func TestProcessConcurrentlyWithResultAndLimit(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5, 6} // 2, 4, 6 will succeed
	ctx := context.Background()

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, 2, tasks, mockTaskW)
	assert.Len(t, results, 3)
	assert.Len(t, errs, 3) // 1, 3, 5 should fail
}

// Test Context Cancellation
func TestProcessConcurrentlyWithResultAndLimit_Cancellation(t *testing.T) {
	tasks := []int{2, 4, 6, 8, 10} // All tasks should return valid results

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, 2, tasks, mockTaskW)

	assert.Empty(t, results) // Should return no results
	assert.Empty(t, errs)    // Should return no errors since no task runs
}

// Test Worker Limit Enforcement
func TestProcessConcurrentlyWithResultAndLimit_WorkerLimit(t *testing.T) {
	tasks := make([]int, 100)
	for i := 0; i < 100; i++ {
		tasks[i] = i
	}

	ctx := context.Background()
	start := time.Now()
	_, _ = ProcessConcurrentlyWithResultAndLimit(ctx, 5, tasks, func(ctx context.Context, i int) (int, error) {
		time.Sleep(10 * time.Millisecond) // Simulate work
		return i, nil
	})

	duration := time.Since(start)
	assert.Greater(t, duration, 200*time.Millisecond) // Should take more than 200ms (ensuring limited concurrency)
}

func TestProcessConcurrentlyWithResultAndLimit_LargeInput(t *testing.T) {
	tasks := make([]int, 10000)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}

	ctx := context.Background()
	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, 10, tasks, mockTaskW)

	assert.Greater(t, len(results), 0)        // Ensure some results are returned
	assert.LessOrEqual(t, len(results), 5000) // At most half should be filtered
	assert.Len(t, errs, 5000)                 // Half should fail
}

// benchmarks

func BenchmarkProcessConcurrentlyWithResultAndLimit(b *testing.B) {
	tasks := make([]int, 10000)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}
	ctx := context.Background()

	taskFunc := func(_ context.Context, input int) (int, error) {
		if input%2 == 0 {
			return input * 2, nil // Double even numbers
		}
		return 0, errors.New("odd number error") // Return error for odd numbers
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessConcurrentlyWithResultAndLimit(ctx, 10, tasks, taskFunc)
	}
}

func BenchmarkProcessConcurrentlyWithLimit(b *testing.B) {
	tasks := make([]int, 10000)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}
	ctx := context.Background()

	taskFunc := func(_ context.Context, input int) error {
		if input%2 == 0 {
			return nil // Double even numbers
		}
		return errors.New("odd number error") // Return error for odd numbers
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessConcurrentlyWithLimit(ctx, 10, tasks, taskFunc)
	}
}

// without results

func mockTaskWithLimit(_ context.Context, _ int, activeWorkers *atomic.Int32, maxWorkers *atomic.Int32, wg *sync.WaitGroup) error {
	defer wg.Done()
	currentWorkers := activeWorkers.Add(1)
	if currentWorkers > maxWorkers.Load() {
		maxWorkers.Store(currentWorkers)
	}
	time.Sleep(10 * time.Millisecond) // Simulate work
	activeWorkers.Add(-1)
	return nil
}

func TestProcessConcurrentlyWithLimit_Success(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5}
	ctx := context.Background()

	errs := ProcessConcurrentlyWithLimit(ctx, 3, tasks, mockTaskSuccessW)

	assert.Empty(t, errs, "No errors should be returned")
}

func TestProcessConcurrentlyWithLimit_SomeFail(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5, 6} // 2, 4, 6 should fail
	ctx := context.Background()

	errs := ProcessConcurrentlyWithLimit(ctx, 3, tasks, mockTaskFailureW)

	assert.Len(t, errs, 3, "Only even-numbered tasks should fail")
}

func TestProcessConcurrentlyWithLimit_Cancel(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5}
	ctx, cancel := context.WithCancel(context.Background())

	cancel() // Cancel immediately before tasks start

	errs := ProcessConcurrentlyWithLimit(ctx, 3, tasks, mockTaskFailureW)

	assert.Empty(t, errs, "No tasks should run after context is canceled")
}

func TestProcessConcurrentlyWithLimit_WorkerLimit(t *testing.T) {
	tasks := make([]int, 100)
	ctx := context.Background()

	var activeWorkers atomic.Int32
	var maxWorkers atomic.Int32
	var wg sync.WaitGroup

	wg.Add(len(tasks))
	ProcessConcurrentlyWithLimit(ctx, 5, tasks, func(ctx context.Context, n int) error {
		return mockTaskWithLimit(ctx, n, &activeWorkers, &maxWorkers, &wg)
	})
	wg.Wait()

	assert.LessOrEqual(t, maxWorkers.Load(), int32(5), "No more than 5 workers should run concurrently")
}

func TestProcessConcurrentlyWithLimit_LargeInput(t *testing.T) {
	tasks := make([]int, 10000)
	for i := range tasks {
		tasks[i] = i
	}

	ctx := context.Background()
	errs := ProcessConcurrentlyWithLimit(ctx, 10, tasks, mockTaskFailureW)

	assert.LessOrEqual(t, len(errs), 5000, "At most half the tasks should fail")
}

// v2

func TestProcessConcurrentlyWithResultAndLimit_Success(t *testing.T) {
	t.Parallel()

	tasks := []int{1, 2, 3, 4, 5}
	workerLimit := 3
	taskFunc := func(ctx context.Context, task int) (int, error) {
		return task * 2, nil
	}
	ctx := context.Background()

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, workerLimit, tasks, taskFunc)
	assert.Empty(t, errs, "expected no errors")

	// Since each task's result is stored by its index,
	// the order of results should match the order of tasks.
	expected := []int{2, 4, 6, 8, 10}
	assert.Equal(t, expected, results, "results should match expected values")
}

func TestProcessConcurrentlyWithResultAndLimit_Error(t *testing.T) {
	t.Parallel()

	tasks := []int{1, 2, 3}
	workerLimit := 2
	taskFunc := func(ctx context.Context, task int) (int, error) {
		return 0, errors.New(fmt.Sprintf("error on task %d", task))
	}
	ctx := context.Background()

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, workerLimit, tasks, taskFunc)
	assert.Empty(t, results, "expected no results")
	assert.Len(t, errs, len(tasks), "expected an error per task")

	// Verify error messages.
	for i, err := range errs {
		expectedErrMsg := fmt.Sprintf("error on task %d", tasks[i])
		assert.EqualError(t, err, expectedErrMsg)
	}
}

func TestProcessConcurrentlyWithResultAndLimit_Mixed(t *testing.T) {
	t.Parallel()

	tasks := []int{1, 2, 3, 4}
	workerLimit := 2
	taskFunc := func(ctx context.Context, task int) (int, error) {
		// Return an error for even tasks.
		if task%2 == 0 {
			return 0, errors.New(fmt.Sprintf("error on task %d", task))
		}
		return task * 10, nil
	}
	ctx := context.Background()

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, workerLimit, tasks, taskFunc)
	// Expect results for tasks 1 and 3; errors for tasks 2 and 4.
	assert.Len(t, results, 2, "expected two results")
	assert.Len(t, errs, 2, "expected two errors")

	// Verify the results.
	expectedResults := []int{10, 30}
	// The order is preserved by the index.
	assert.Equal(t, expectedResults, results, "results should match expected values")
}

func TestProcessConcurrentlyWithResultAndLimit_ContextCancellation(t *testing.T) {
	t.Parallel()

	tasks := []int{1, 2, 3, 4, 5}
	workerLimit := 2
	var mu sync.Mutex
	executedTasks := make([]int, 0)
	taskFunc := func(ctx context.Context, task int) (int, error) {
		// Simulate work.
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		executedTasks = append(executedTasks, task)
		mu.Unlock()
		return task * 3, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel context after 150ms.
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, workerLimit, tasks, taskFunc)
	// We cannot guarantee how many tasks complete due to cancellation.
	totalOutcomes := len(results) + len(errs)
	assert.Less(t, totalOutcomes, len(tasks)+1, "not all tasks should complete after cancellation")
	assert.NotNil(t, ctx.Err(), "expected context to be canceled")
}

func TestProcessConcurrentlyWithResultAndLimit_ConcurrencySafety(t *testing.T) {
	t.Parallel()

	numTasks := 1000
	tasks := make([]int, numTasks)
	for i := 0; i < numTasks; i++ {
		tasks[i] = i
	}
	workerLimit := 50
	taskFunc := func(ctx context.Context, task int) (int, error) {
		return task * 2, nil
	}
	ctx := context.Background()

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, workerLimit, tasks, taskFunc)
	assert.Empty(t, errs, "expected no errors")
	assert.Len(t, results, numTasks, "expected all tasks to be processed")
	// Verify each result.
	for i, task := range tasks {
		expected := task * 2
		assert.Equal(t, expected, results[i], "result for task %d should be %d", task, expected)
	}
}
