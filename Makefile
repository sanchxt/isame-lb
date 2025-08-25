# Makefile for Isame Load Balancer

.PHONY: build test lint run clean install-tools help

# Build variables
BINARY_NAME=isame-lb
CLI_BINARY_NAME=isame-ctl
BUILD_DIR=./bin
CMD_DIR=./cmd

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build the main load balancer binary
build: clean
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/$(BINARY_NAME)/
	@echo "Building $(CLI_BINARY_NAME)..."
	$(GOBUILD) -o $(BUILD_DIR)/$(CLI_BINARY_NAME) $(CMD_DIR)/$(CLI_BINARY_NAME)/
	@echo "Build completed successfully!"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

# Run linter
lint:
	@echo "Running linter..."
	@echo "Running go vet..."
	$(GOCMD) vet ./...
	@echo "Checking formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Code needs formatting. Run 'make fmt'" && exit 1)
	@echo "Running staticcheck..."
	@which $(shell go env GOPATH)/bin/staticcheck > /dev/null 2>&1 || (echo "Installing staticcheck..." && $(GOCMD) install honnef.co/go/tools/cmd/staticcheck@latest)
	$(shell go env GOPATH)/bin/staticcheck ./...

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Run the load balancer locally
run: build
	@echo "Starting $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Tidy go modules
tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@which golangci-lint > /dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.54.2; \
	}

# Development setup
dev-setup: install-tools tidy
	@echo "Development environment setup complete!"

# Build for multiple platforms
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)/$(BINARY_NAME)/
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)/$(BINARY_NAME)/
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)/$(BINARY_NAME)/

# Help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binaries"
	@echo "  test        - Run tests"
	@echo "  lint        - Run linter"
	@echo "  fmt         - Format code"
	@echo "  run         - Build and run the load balancer"
	@echo "  clean       - Clean build artifacts"
	@echo "  tidy        - Tidy go modules"
	@echo "  install-tools - Install development tools"
	@echo "  dev-setup   - Setup development environment"
	@echo "  build-all   - Build for multiple platforms"
	@echo "  help        - Show this help message"

# Default target
.DEFAULT_GOAL := help