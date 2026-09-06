.PHONY: generate lint test

generate:
	go tool mockery

lint:
	golangci-lint run --config .golangci.yaml

test:
	go test ./...
