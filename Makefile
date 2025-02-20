.PHONY: lint fmt build test-cover test-race test-bench

lint:
	golangci-lint run ./...

fmt:
	gofumpt -w .

build:
	go build ./main.go

test-cover:
	go test -cover ./...

test-race:
	go test -race ./...

test-bench:
	go test -bench . github.com/hashmap-kz/workerfn/pkg/concur
