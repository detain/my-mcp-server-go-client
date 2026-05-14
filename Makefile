.PHONY: build build-all clean test lint tidy

BINARY_NAME=mcp-proxy-client
BIN_DIR=bin
RELEASES_DIR=releases

# Default build
build:
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/server

# Cross-compilation targets
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/server

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/server

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/server

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/server

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/server

# Build all platforms
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows

# Clean
clean:
	rm -rf $(BIN_DIR) $(RELEASES_DIR)

# Test
test:
	go test -v -race ./...

# Lint
lint:
	golangci-lint run

# Tidy dependencies
tidy:
	go mod tidy

# Run locally
run: build
	./$(BIN_DIR)/$(BINARY_NAME)
