package versions

import (
	"context"
	"errors"
	"testing"
)

// Task function for testing
func mockTask4(ctx context.Context, input int) (int, error) {
	if input%2 == 0 {
		return input * 2, nil // Double even numbers
	}
	return 0, errors.New("odd number error") // Return error for odd numbers
}

func BenchmarkProcessConcurrentlyWithResultV4(b *testing.B) {
	tasks := make([]int, tasks)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessConcurrentlyWithResultV4(ctx, tasks, mockTask4)
	}
}

func BenchmarkProcessConcurrentlyWithResultAndLimitV4(b *testing.B) {
	tasks := make([]int, tasks)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessConcurrentlyWithResultAndLimitV4(ctx, workers, tasks, mockTask4)
	}
}
