.PHONY: fmt tidy test build run lint
fmt:
	go fmt ./...
tidy:
	go mod tidy
test:
	go test ./...
build:
	go build -o bin/reproq-tui ./cmd/reproq-tui
run:
	go run ./cmd/reproq-tui
lint:
	golangci-lint run ./...
