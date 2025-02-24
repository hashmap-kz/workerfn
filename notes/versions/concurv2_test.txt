package versions

import (
	"context"
	"errors"
	"testing"
)

// Task function for testing
func mockTask2(ctx context.Context, input int) (int, error) {
	if input%2 == 0 {
		return input * 2, nil // Double even numbers
	}
	return 0, errors.New("odd number error") // Return error for odd numbers
}

func BenchmarkProcessConcurrentlyWithResultV2(b *testing.B) {
	tasks := make([]int, tasks)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessConcurrentlyWithResultV2(ctx, tasks, mockTask2)
	}
}

func BenchmarkProcessConcurrentlyWithResultAndLimitV2(b *testing.B) {
	tasks := make([]int, tasks)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessConcurrentlyWithResultAndLimitV2(ctx, workers, tasks, mockTask2)
	}
}
