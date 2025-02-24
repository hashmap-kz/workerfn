.PHONY: lint fmt build test-cover test-race test-bench

COV_REPORT := coverage.txt

lint:
	golangci-lint run ./...

fmt:
	gofumpt -w .

build:
	go build ./main.go

test-cover:
	go test -coverprofile=$(COV_REPORT) ./...
	go tool cover -html=$(COV_REPORT)

test-race:
	go test -race ./...

test-bench:
	go test -bench . github.com/hashmap-kz/workerfn/pkg/concur -benchmem
