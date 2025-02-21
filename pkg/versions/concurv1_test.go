package versions

import (
	"context"
	"errors"
	"testing"
)

// Task function for testing
func mockTask1(ctx context.Context, input int) (int, error) {
	if input%2 == 0 {
		return input * 2, nil // Double even numbers
	}
	return 0, errors.New("odd number error") // Return error for odd numbers
}

// Filter function for testing
func mockFilter1(result int) bool {
	return result > 4 // Only include results greater than 4
}

func BenchmarkProcessConcurrentlyWithResultV1(b *testing.B) {
	tasks := make([]int, tasks)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessConcurrentlyWithResultV1(ctx, tasks, mockTask1, mockFilter1)
	}
}

func BenchmarkProcessConcurrentlyWithResultAndLimitV1(b *testing.B) {
	tasks := make([]int, tasks)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessConcurrentlyWithResultAndLimitV1(ctx, workers, tasks, mockTask1, mockFilter1)
	}
}
