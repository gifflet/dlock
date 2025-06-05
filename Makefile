.PHONY: build install clean test example help

# Build settings
BINARY_NAME=dlock
MAIN_PATH=./cmd/dlock/main.go
PKG_PATH=./pkg/dlock
EXAMPLE_PATH=./examples

# Go settings
GO=go
GOFLAGS=-ldflags="-s -w"

# Default target
help:
	@echo "Available targets:"
	@echo "  build       - Build the CLI binary"
	@echo "  install     - Install the CLI globally"
	@echo "  clean       - Remove built binaries"
	@echo "  test        - Run tests"
	@echo "  example     - Run the usage example"
	@echo "  help        - Show this help"

# Build the CLI binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary built: $(BINARY_NAME)"

# Install the CLI globally
install:
	@echo "Installing $(BINARY_NAME) globally..."
	$(GO) install $(GOFLAGS) $(MAIN_PATH)
	@echo "$(BINARY_NAME) installed globally"

# Clean built binaries
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	$(GO) clean
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v $(PKG_PATH)/...
	@echo "Tests complete"

# Run the usage example
example:
	@echo "Running usage example..."
	$(GO) run $(EXAMPLE_PATH)/usage_example.go

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "Multi-platform build complete"

# Development targets
dev: build
	@echo "Development build complete"

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

run-help: build
	@echo "Running $(BINARY_NAME) -help..."
	./$(BINARY_NAME) -help

# Format and lint
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

vet:
	@echo "Running go vet..."
	$(GO) vet ./...

mod-tidy:
	@echo "Tidying go modules..."
	$(GO) mod tidy 