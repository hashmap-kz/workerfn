package concur

import (
	"context"
	"errors"
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

// TODO:
//// Test Context Cancellation
//func TestProcessConcurrentlyWithResult_Cancellation(t *testing.T) {
//	tasks := []int{2, 4, 6, 8, 10} // All tasks should return valid results
//
//	ctx, cancel := context.WithCancel(context.Background())
//	cancel() // Cancel immediately
//
//	results, errs := ProcessConcurrentlyWithResult(ctx, tasks, mockTask, mockFilter)
//
//	assert.Empty(t, results) // Should return no results
//	assert.Empty(t, errs)    // Should return no errors since no task runs
//}

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
