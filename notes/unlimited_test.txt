package concur

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Task function for testing
func mockTaskU(_ context.Context, input int) (int, error) {
	if input%2 == 0 {
		return input * 2, nil // Double even numbers
	}
	return 0, errors.New("odd number error") // Return error for odd numbers
}

func mockTaskSuccessU(_ context.Context, _ int) error {
	return nil // No error
}

func mockTaskFailureU(_ context.Context, input int) error {
	if input%2 == 0 {
		return errors.New("task failed")
	}
	return nil
}

func mockTaskCounterU(_ context.Context, _ int, counter *atomic.Int32) error {
	counter.Add(1)
	return nil
}

// with results

// Test ProcessConcurrentlyWithResult
func TestProcessConcurrentlyWithResult(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5, 6} // 2, 4, 6 will succeed

	ctx := context.Background()
	results, errs := ProcessConcurrentlyWithResult(ctx, tasks, mockTaskU)

	assert.Len(t, results, 3)
	assert.Len(t, errs, 3) // 1, 3, 5 should fail
}

// Test Context Cancellation
func TestProcessConcurrentlyWithResult_Cancellation(t *testing.T) {
	tasks := []int{2, 4, 6, 8, 10} // All tasks should return valid results

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	results, errs := ProcessConcurrentlyWithResult(ctx, tasks, mockTaskU)

	assert.Empty(t, results) // Should return no results
	assert.Empty(t, errs)    // Should return no errors since no task runs
}

func TestProcessConcurrentlyWithResult_LargeInput(t *testing.T) {
	tasks := make([]int, 10000)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}

	ctx := context.Background()
	results, errs := ProcessConcurrentlyWithResult(ctx, tasks, mockTaskU)

	assert.Greater(t, len(results), 0)        // Ensure some results are returned
	assert.LessOrEqual(t, len(results), 5000) // At most half should be filtered
	assert.Len(t, errs, 5000)                 // Half should fail
}

// benchmarks

func BenchmarkProcessConcurrentlyWithResult(b *testing.B) {
	tasks := make([]int, 10000)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessConcurrentlyWithResult(ctx, tasks, mockTaskU)
	}
}

// without results

func TestProcessConcurrently_Success(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5}
	ctx := context.Background()

	errs := ProcessConcurrently(ctx, tasks, mockTaskSuccessU)

	assert.Empty(t, errs, "No errors should be returned")
}

func TestProcessConcurrently_SomeFail(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5, 6} // 2, 4, 6 will fail
	ctx := context.Background()

	errs := ProcessConcurrently(ctx, tasks, mockTaskFailureU)

	assert.Len(t, errs, 3, "Only even-numbered tasks should fail")
}

func TestProcessConcurrently_Cancel(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5}
	ctx, cancel := context.WithCancel(context.Background())

	cancel() // Cancel immediately before tasks start

	errs := ProcessConcurrently(ctx, tasks, mockTaskFailureU)

	assert.Empty(t, errs, "No tasks should run after context is canceled")
}

func TestProcessConcurrently_Concurrency(t *testing.T) {
	tasks := make([]int, 100)
	var counter atomic.Int32
	ctx := context.Background()

	ProcessConcurrently(ctx, tasks, func(ctx context.Context, n int) error {
		_ = mockTaskCounterU(ctx, n, &counter)
		return nil
	})

	assert.Equal(t, int32(100), counter.Load(), "All tasks should have been executed concurrently")
}
