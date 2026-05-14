.PHONY: build clean test lint tidy

BINARY_NAME=mcp-proxy-client
BIN_DIR=bin

build:
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/server

clean:
	rm -rf $(BIN_DIR)

test:
	go test -v ./...

lint:
	golangci-lint run

tidy:
	go mod tidy

run: build
	./$(BIN_DIR)/$(BINARY_NAME)
