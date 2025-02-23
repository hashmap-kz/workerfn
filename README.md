# Concurrent Task Processing in Go (worker-pool small functions)

## Overview
`concur` is a Go package that provides functions for executing tasks concurrently. It supports:

- **Unlimited concurrency** (all tasks run in parallel)
- **Worker-limited concurrency** (control over the number of workers)
- **Result filtering** (custom filtering of results)
- **Error handling** (collects and returns encountered errors)
- **Context support** (cancellation and timeout handling)

## Installation
```sh
go get "github.com/hashmap-kz/workerfn/"
```

## Usage

### Import the Package
```go
import (
    "context"
    "fmt"
    "github.com/hashmap-kz/workerfn/pkg/concur"
)
```

## Functions

---

### `ProcessConcurrentlyWithResultAndLimit`
Executes tasks concurrently with a limited number of workers.

```go
func ProcessConcurrentlyWithResultAndLimit[T any, R any](
    ctx context.Context,
    workerLimit int,
    tasks []T,
    taskFunc func(context.Context, T) (R, error),
) ([]R, []error)
```

#### Parameters
- `ctx` – Context for cancellation and timeout handling.
- `workerLimit` – The maximum number of worker goroutines.
- `tasks` – A slice of tasks to be processed.
- `taskFunc` – A function that processes a task and returns a result and an error.

#### Returns
- A slice of filtered results.
- A slice of non-nil errors encountered during task execution.

#### Example Usage
```go
type uploadTask struct {
	storage   uploader.Uploader
	localPath string
	localDir  string
	remoteDir string
}

func uploadListOfFilesOnRemote(storageType uploader.UploaderType, l *slog.Logger, tasks []uploadTask, cfg config.UploadConfig) error {
	workerLimit := 8

	uploaded, errors := concur.ProcessConcurrentlyWithResultAndLimit(
		context.Background(),
		workerLimit,
		tasks,
		uploadWorker,
  )

	if len(errors) != 0 {
		for _, e := range errors {
			slog.Error("upload.failed.details",
				slog.String("type", string(storageType)),
				slog.String("err", e.Error()),
			)
		}
	}

	for _, e := range uploaded {
		l.LogAttrs(context.Background(), logger.LevelTrace, "upload.success",
			slog.String("type", string(storageType)),
			slog.String("remote", e),
		)
	}

	if len(errors) != 0 {
		return fmt.Errorf("upload.failed: %s", storageType)
	}
	return nil
}

func uploadWorker(ctx context.Context, uploadTask uploadTask) (string, error) {

	relativePath := uploadTask.localPath[len(uploadTask.localDir):]
	remotePath := filepath.ToSlash(filepath.Join(uploadTask.remoteDir, relativePath))

	if err := uploadTask.storage.Upload(uploadTask.localPath, remotePath); err != nil {
		return "", err
	}

	return remotePath, nil
}
```

---

## License
This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Contributing
Feel free to open issues or submit pull requests.

## Contact
For issues or feature requests, open a GitHub issue.

---

### 🚀 Happy Coding!

