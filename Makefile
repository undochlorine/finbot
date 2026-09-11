.PHONY: generate lint test test-unit test-integration up down

generate:
	go tool mockery

lint:
	golangci-lint run --config .golangci.yaml

test: test-unit test-integration

test-unit:
	go test ./...

test-integration:
	go test -tags=integration ./internal/adapter/postgres/... ./internal/adapter/rediscache/...

up:
	docker compose up --build -d

down:
	docker compose down
