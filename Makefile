.PHONY: generate lint test test-unit test-integration

generate:
	go tool mockery

lint:
	golangci-lint run --config .golangci.yaml

test: test-unit test-integration

test-unit:
	go test ./...

test-integration:
	go test -tags=integration ./internal/adapter/postgres/...
