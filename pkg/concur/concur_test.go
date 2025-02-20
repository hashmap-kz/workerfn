package concur

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Task function for testing
func mockTask(ctx context.Context, input int) (int, error) {
	if input%2 == 0 {
		return input * 2, nil // Double even numbers
	}
	return 0, errors.New("odd number error") // Return error for odd numbers
}

// Filter function for testing
func mockFilter(result int) bool {
	return result > 4 // Only include results greater than 4
}

// Test ProcessConcurrentlyWithResult
func TestProcessConcurrentlyWithResult(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5, 6} // 2, 4, 6 will succeed

	ctx := context.Background()
	results, errs := ProcessConcurrentlyWithResult(ctx, tasks, mockTask, mockFilter)

	assert.ElementsMatch(t, []int{8, 12}, results) // Filtered: 2*2=4 (excluded), 4*2=8, 6*2=12
	assert.Len(t, errs, 3)                         // 1, 3, 5 should fail
}

// Test ProcessConcurrentlyWithResultAndLimit
func TestProcessConcurrentlyWithResultAndLimit(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5, 6} // 2, 4, 6 will succeed
	ctx := context.Background()

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, 2, tasks, mockTask, mockFilter)

	assert.ElementsMatch(t, []int{8, 12}, results) // Should filter correctly
	assert.Len(t, errs, 3)                         // 1, 3, 5 should fail
}

func TestProcessConcurrentlyWithResult_Cancellation(t *testing.T) {
	tasks := []int{2, 4, 6, 8, 10} // All tasks should return valid results

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	results, errs := ProcessConcurrentlyWithResult(ctx, tasks, mockTask, mockFilter)

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
	}, func(i int) bool { return true })

	duration := time.Since(start)
	assert.Greater(t, duration, 200*time.Millisecond) // Should take more than 200ms (ensuring limited concurrency)
}

func TestProcessConcurrentlyWithResult_LargeInput(t *testing.T) {
	tasks := make([]int, 10000)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}

	ctx := context.Background()
	results, errs := ProcessConcurrentlyWithResult(ctx, tasks, mockTask, mockFilter)

	assert.Greater(t, len(results), 0)        // Ensure some results are returned
	assert.LessOrEqual(t, len(results), 5000) // At most half should be filtered
	assert.Len(t, errs, 5000)                 // Half should fail
}

func TestProcessConcurrentlyWithResultAndLimit_LargeInput(t *testing.T) {
	tasks := make([]int, 10000)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}

	ctx := context.Background()
	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, 10, tasks, mockTask, mockFilter)

	assert.Greater(t, len(results), 0)        // Ensure some results are returned
	assert.LessOrEqual(t, len(results), 5000) // At most half should be filtered
	assert.Len(t, errs, 5000)                 // Half should fail
}

// bench

func BenchmarkProcessConcurrentlyWithResult(b *testing.B) {
	tasks := make([]int, 10000)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessConcurrentlyWithResult(ctx, tasks, mockTask, mockFilter)
	}
}

func BenchmarkProcessConcurrentlyWithResultAndLimit(b *testing.B) {
	tasks := make([]int, 10000)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessConcurrentlyWithResultAndLimit(ctx, 10, tasks, mockTask, mockFilter)
	}
}

// no results funcs

func mockTaskSuccess(ctx context.Context, input int) error {
	return nil // No error
}

func mockTaskFailure(ctx context.Context, input int) error {
	if input%2 == 0 {
		return errors.New("task failed")
	}
	return nil
}

func mockTaskCounter(ctx context.Context, _ int, counter *atomic.Int32) error {
	counter.Add(1)
	return nil
}

func TestProcessConcurrently_Success(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5}
	ctx := context.Background()

	errs := ProcessConcurrently(ctx, tasks, mockTaskSuccess)

	assert.Empty(t, errs, "No errors should be returned")
}

func TestProcessConcurrently_SomeFail(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5, 6} // 2, 4, 6 will fail
	ctx := context.Background()

	errs := ProcessConcurrently(ctx, tasks, mockTaskFailure)

	assert.Len(t, errs, 3, "Only even-numbered tasks should fail")
}

func TestProcessConcurrently_Cancel(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5}
	ctx, cancel := context.WithCancel(context.Background())

	cancel() // Cancel immediately before tasks start

	errs := ProcessConcurrently(ctx, tasks, mockTaskFailure)

	assert.Empty(t, errs, "No tasks should run after context is canceled")
}

func TestProcessConcurrently_Concurrency(t *testing.T) {
	tasks := make([]int, 100)
	var counter atomic.Int32
	ctx := context.Background()

	ProcessConcurrently(ctx, tasks, func(ctx context.Context, n int) error {
		mockTaskCounter(ctx, n, &counter)
		return nil
	})

	assert.Equal(t, int32(100), counter.Load(), "All tasks should have been executed concurrently")
}

// no results, with limits

func mockTaskWithLimit(ctx context.Context, _ int, activeWorkers *atomic.Int32, maxWorkers *atomic.Int32, wg *sync.WaitGroup) error {
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

	errs := ProcessConcurrentlyWithLimit(ctx, 3, tasks, mockTaskSuccess)

	assert.Empty(t, errs, "No errors should be returned")
}

func TestProcessConcurrentlyWithLimit_SomeFail(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5, 6} // 2, 4, 6 should fail
	ctx := context.Background()

	errs := ProcessConcurrentlyWithLimit(ctx, 3, tasks, mockTaskFailure)

	assert.Len(t, errs, 3, "Only even-numbered tasks should fail")
}

func TestProcessConcurrentlyWithLimit_Cancel(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5}
	ctx, cancel := context.WithCancel(context.Background())

	cancel() // Cancel immediately before tasks start

	errs := ProcessConcurrentlyWithLimit(ctx, 3, tasks, mockTaskFailure)

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
	errs := ProcessConcurrentlyWithLimit(ctx, 10, tasks, mockTaskFailure)

	assert.LessOrEqual(t, len(errs), 5000, "At most half the tasks should fail")
}

// testing cancellation

func mockTask1(ctx context.Context, input int) error {
	time.Sleep(100 * time.Millisecond) // Simulate task execution
	if input%2 == 0 {
		return errors.New("task failed") // Simulate failure for even numbers
	}
	return nil
}

func mockTaskWithResult1(ctx context.Context, input int) (int, error) {
	time.Sleep(100 * time.Millisecond) // Simulate task execution
	if input%2 == 0 {
		return 0, errors.New("task failed") // Simulate failure for even numbers
	}
	return input * 2, nil
}

func TestProcessConcurrently_Cancel1(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5}
	ctx, cancel := context.WithCancel(context.Background())

	cancel() // Cancel immediately before tasks start

	errs := ProcessConcurrently(ctx, tasks, mockTask1)

	assert.Empty(t, errs, "No tasks should run after context is canceled")
}

func TestProcessConcurrentlyWithLimit_Cancel1(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5}
	ctx, cancel := context.WithCancel(context.Background())

	cancel() // Cancel immediately before tasks start

	errs := ProcessConcurrentlyWithLimit(ctx, 3, tasks, mockTask1)

	assert.Empty(t, errs, "No tasks should run after context is canceled")
}

// TODO: since slices for results/errors are preallocated, they are not empty at cancellation

func TestProcessConcurrentlyWithResult_Cancel1(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5}
	ctx, cancel := context.WithCancel(context.Background())

	cancel() // Cancel immediately before tasks start

	results, errs := ProcessConcurrentlyWithResult(ctx, tasks, mockTaskWithResult1, func(r int) bool { return r != 0 })

	assert.Empty(t, results, "No results should be returned when context is canceled")
	assert.Empty(t, errs, "No errors should be returned when context is canceled")
}

func TestProcessConcurrentlyWithResultAndLimit_Cancel1(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5}
	ctx, cancel := context.WithCancel(context.Background())

	cancel() // Cancel immediately before tasks start

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, 3, tasks, mockTaskWithResult1, func(r int) bool { return r != 0 })

	assert.Empty(t, results, "No results should be returned when context is canceled")
	assert.Empty(t, errs, "No errors should be returned when context is canceled")
}
