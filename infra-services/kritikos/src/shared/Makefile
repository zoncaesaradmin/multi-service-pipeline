# Shared Module Makefile
.PHONY: help build test clean deps fmt vet lint coverage coverage-html coverage-check coverage-enforce

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Coverage parameters
COVERAGE_THRESHOLD=85
COVERAGE_OUT=shared_coverage.tmp
COVERAGE_HTML=coverage.html

# Build tags support (can be passed from parent Makefile)
BUILD_TAGS ?=
ifneq ($(BUILD_TAGS),)
	BUILD_FLAGS = -tags $(BUILD_TAGS)
else
	BUILD_FLAGS =
endif

# Default target
help:
	@echo "Shared Module Makefile"
	@echo "====================="
	@echo ""
	@echo "Available targets:"
	@echo "  build           - Build the module (verify compilation)"
	@echo "  test            - Run all tests"
	@echo "  coverage        - Run tests with coverage"
	@echo "  coverage-html   - Generate HTML coverage report"
	@echo "  coverage-check  - Check coverage meets threshold ($(COVERAGE_THRESHOLD)%)"
	@echo "  coverage-enforce- Enforce coverage threshold (fail if below)"
	@echo "  clean           - Clean build artifacts"
	@echo "  deps            - Download dependencies"
	@echo "  fmt             - Format code"
	@echo "  vet             - Vet code"
	@echo "  lint            - Lint code (if golangci-lint available)"
	@echo "  ci              - Run CI pipeline (fmt, vet, test, coverage-enforce)"
	@echo "  help            - Show this help"

# Build the module (verify compilation)
build:
	@echo "Building shared module..."
	@$(GOBUILD) $(BUILD_FLAGS) ./...
	@echo "Build completed successfully"

# Run all tests
test:
	@echo "Running tests with SERVICE_HOME set to repository root..."
	@SERVICE_HOME="$$(cd ../../ && pwd)" $(GOTEST) $(BUILD_FLAGS) -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage and SERVICE_HOME set to repository root..."
	@SERVICE_HOME="$$(cd ../../ && pwd)" $(GOTEST) $(BUILD_FLAGS) -coverprofile=$(COVERAGE_OUT) ./...
	@$(GOCMD) tool cover -func=$(COVERAGE_OUT)

# Generate HTML coverage report
coverage-html: coverage
	@echo "Generating HTML coverage report..."
	@$(GOCMD) tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# Check coverage percentage
coverage-check: coverage
	@echo "Checking coverage threshold..."
	@COVERAGE=$$($(GOCMD) tool cover -func=$(COVERAGE_OUT) | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Current coverage: $$COVERAGE%"; \
	echo "Required coverage: $(COVERAGE_THRESHOLD)%"; \
	if [ $$(echo "$$COVERAGE >= $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "✅ Coverage check passed!"; \
	else \
		echo "⚠️  Coverage below threshold"; \
	fi

# Enforce coverage threshold (fail build if below)
coverage-enforce: coverage
	@echo "Enforcing coverage threshold..."
	@COVERAGE=$$($(GOCMD) tool cover -func=$(COVERAGE_OUT) | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Current coverage: $$COVERAGE%"; \
	echo "Required coverage: $(COVERAGE_THRESHOLD)%"; \
	if [ $$(echo "$$COVERAGE >= $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "✅ Coverage enforcement passed!"; \
	else \
		echo "❌ Coverage enforcement failed! Current: $$COVERAGE%, Required: $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning shared module artifacts..."
	@$(GOCLEAN)
	@rm -f $(COVERAGE_OUT) $(COVERAGE_HTML)
	@echo "Clean completed"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	@$(GOFMT) -w .
	@$(GOCMD) fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	@$(GOVET) ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping lint check"; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# CI pipeline
ci: deps fmt vet test coverage-enforce
	@echo "✅ CI pipeline completed successfully!"

# Development workflow
dev: deps fmt vet test coverage-check
	@echo "✅ Development checks completed!"

# Package-specific coverage reports
coverage-utils:
	@echo "Running coverage for utils package..."
	@$(GOTEST) -coverprofile=utils_coverage.out ./utils
	@$(GOCMD) tool cover -func=utils_coverage.out

coverage-types:
	@echo "Running coverage for types package..."
	@$(GOTEST) -coverprofile=types_coverage.out ./types
	@$(GOCMD) tool cover -func=types_coverage.out

coverage-logging:
	@echo "Running coverage for logging package..."
	@$(GOTEST) -coverprofile=logging_coverage.out ./logging
	@$(GOCMD) tool cover -func=logging_coverage.out

# Detailed coverage by package
coverage-detailed: coverage-utils coverage-types coverage-logging
	@echo "Detailed coverage report by package completed"

# Benchmark tests
benchmark:
	@echo "Running benchmark tests..."
	@$(GOTEST) -bench=. -benchmem ./...

# Race condition detection
race:
	@echo "Running tests with race detection..."
	@$(GOTEST) -race ./...

# Full test suite (unit + race + benchmark)
test-full: test race benchmark
	@echo "Full test suite completed"
