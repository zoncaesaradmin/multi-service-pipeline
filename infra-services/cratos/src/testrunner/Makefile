# Test Runner Makefile
.PHONY: build build-local test clean deep-clean run deps tidy fmt vet run-integration-tests run-performance-tests help

# Variables
BINARY_NAME=testrunner.bin
BINARY_DIR=./../../bin
BINARY_PATH=$(BINARY_DIR)/$(BINARY_NAME)
MAIN_PATH=./cmd/testmain.go
GO_FILES=$(shell find . -name "*.go" -type f)

# Default target
all: clean deps fmt vet test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_PATH) $(MAIN_PATH)
	@cp -rf features $(BINARY_DIR)
	@cp -rf testdata $(BINARY_DIR)
	@echo "Build completed: $(BINARY_PATH)"

# Build the binary
build-local:
	@echo "Building $(BINARY_NAME) with local tags ..."
	@mkdir -p $(BINARY_DIR)
	go build -tags local -o $(BINARY_PATH) $(MAIN_PATH)
	@cp -rf features $(BINARY_DIR)
	@cp -rf testdata $(BINARY_DIR)
	@echo "Local testrunner Build completed: $(BINARY_PATH)"

# Run the test runner
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BINARY_PATH)

# Run unit tests
test:
	@echo "Running unit tests..."
	go test -v ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...

# Clean build artifacts and generated files
clean:
	@echo "Cleaning build artifacts and generated files..."
	@rm -rf $(BINARY_DIR)
	@rm -f *.log
	@rm -rf tmp/ temp/
	@find . -name "*.test" -type f -delete
	@find . -name "*.out" -type f -delete
	@echo "Clean completed"

# Deep clean - removes all generated files including vendor and module cache
deep-clean: clean
	@echo "Deep cleaning..."
	@rm -rf vendor/
	@go clean -modcache
	@echo "Deep clean completed"

# Run integration tests against service
run-integration-tests:
	@echo "Running integration tests..."
	$(BINARY_PATH) -mode=integration

# Run performance tests
run-performance-tests:
	@echo "Running performance tests..."
	$(BINARY_PATH) -mode=performance

# Help
help:
	@echo "Available targets:"
	@echo "  build                    - Build the binary"
	@echo "  run                      - Build and run the test runner"
	@echo "  test                     - Run unit tests"
	@echo "  deps                     - Download dependencies"
	@echo "  tidy                     - Tidy dependencies"
	@echo "  fmt                      - Format code"
	@echo "  vet                      - Vet code"
	@echo "  clean                    - Clean build artifacts and generated files"
	@echo "  deep-clean               - Deep clean including vendor and module cache"
	@echo "  run-integration-tests    - Run integration tests"
	@echo "  run-performance-tests    - Run performance tests"
	@echo "  help                     - Show this help"
