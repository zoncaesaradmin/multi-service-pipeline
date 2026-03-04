# Test Runner Makefile
.PHONY: build build-local test clean deep-clean run deps tidy fmt vet run-integration-tests run-performance-tests help

# Variables
ROOT ?= $(shell git rev-parse --show-toplevel)
PROTO_DIR=${ROOT}/proto
BINARY_NAME=testrunner.bin
BINARY_DIR=./../../bin
BINARY_PATH=$(BINARY_DIR)/$(BINARY_NAME)
MAIN_PATH=./cmd/testmain.go
GO_FILES=$(shell find . -name "*.go" -type f)

# Default target
all: clean deps fmt vet test build

ensureproto:
	@protoc --go_out=${ROOT}/client/golang/src/telemetry/utils/alert --proto_path=$(PROTO_DIR) $(PROTO_DIR)/alert.proto

ensurebindir:
	@mkdir -p $(BINARY_DIR)

copydepfiles:
	@cp -rf features $(BINARY_DIR)
	@cp -rf testdata $(BINARY_DIR)

# Build the binary
build: ensurebindir ensureproto
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_PATH) $(MAIN_PATH)
	@$(MAKE) copydepfiles
	@echo "Build completed: $(BINARY_PATH)"

# Build the binary with local tags
build-local: ensurebindir ensureproto
	@echo "Building $(BINARY_NAME) with local tags ..."
	go build -tags local -o $(BINARY_PATH) $(MAIN_PATH)
	@$(MAKE) copydepfiles
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
	#@rm -rf ../../../../client/golang/src/telemetry/utils/alert/alert.pb.go
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
	@echo "Cratos Test Runner Makefile"
	@echo "========================="
	@echo ""
	@echo "🔧 Build & Run:"
	@echo "  build            - Build test runner (production mode)"
	@echo "  build-local      - Build with local message bus support"
	@echo "  run              - Run all test suites"
	@echo ""
	@echo "🧪 Test Modes:"
	@echo "  run-integration-tests - Run integration test suite"
	@echo "  run-performance-tests - Run performance benchmarks"
	@echo ""
	@echo "🛠️  Development:"
	@echo "  deps             - Download Go dependencies"
	@echo "  tidy             - Clean up go.mod and go.sum"
	@echo "  fmt              - Format Go code"
	@echo "  vet              - Run Go vet checks"
	@echo ""
	@echo "🧹 Cleanup:"
	@echo "  clean            - Remove build artifacts"
	@echo "  deep-clean       - Full cleanup including vendor/"
	@echo ""
	@echo "Environment Setup:"
	@echo "  SERVICE_HOME     - Repository root path (required)"
	@echo "  BUILD_TAGS       - Build mode configuration"
	@echo ""
	@echo "Test Topics (override with env vars):"
	@echo "  PROCESSING_RULES_TOPIC   - Rules topic (default: cisco_nir-alertRules)"
	@echo "  PROCESSING_INPUT_TOPIC   - Input topic (default: cisco_nir-anomalies)"
	@echo "  PROCESSING_OUTPUT_TOPIC  - Output topic (default: cisco_nir-prealerts)"
	@echo "  RULE_TASKS_TOPIC        - Rule tasks (default: cisco_nir-ruletasks)"
