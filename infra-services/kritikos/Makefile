LANG=python
IMAGE_NAME=kritikos

include ../Makefile.include

# Simplified Cratos Project Makefile
.PHONY: help devenv dev dev-run dev-clean test-coverage buildall clean package package-local pre_build post_build

# Default ENTRYPOINT_MODE if not set
ENTRYPOINT_MODE ?= start

# Build tags (empty by default, but can be set from environment/command line)
BUILD_TAGS ?=

# Helper function to safely append a tag if not already present
define append_tag
$(if $(filter $(1),$(BUILD_TAGS)),$(BUILD_TAGS),$(if $(BUILD_TAGS),$(BUILD_TAGS) $(1),$(1)))
endef

# Local build tags - safely append 'local' to existing BUILD_TAGS
LOCAL_BUILD_TAGS = $(call append_tag,local)

# Default target
help:
	@echo "Cratos Project - Simplified Makefile"
	@echo "====================================="
	@echo ""
	@echo "🚀 LOCAL DEVELOPMENT (Mac/PC):"
	@echo "  dev          - Full development workflow:"
	@echo "                 • Clean environment and processes"
	@echo "                 • Build service and testrunner"
	@echo "                 • Set up LocalBus topics"
	@echo "                 • Run integration tests"
	@echo "                 • Generate coverage reports"
	@echo "                 • Collect logs and test results"
	@echo "                 • Auto cleanup on completion"
	@echo ""
	@echo "  dev-run      - Quick development iteration:"
	@echo "                 • Use existing binaries"
	@echo "                 • Run integration tests only"
	@echo "                 • Skip build steps"
	@echo "                 • Ideal for rapid testing"
	@echo ""
	@echo "  dev-clean    - Development cleanup:"
	@echo "                 • Kill service and testrunner processes"
	@echo "                 • Clean test artifacts and logs"
	@echo "                 • Remove LocalBus data"
	@echo "                 • Reset environment completely"
	@echo ""
	@echo "  test-coverage- Generate coverage report:"
	@echo "                 • Process raw coverage data"
	@echo "                 • Generate HTML report"
	@echo "                 • Show function coverage"
	@echo "                 • Display coverage summary"
	@echo ""
	@echo "  devenv      - Setup development environment:"
	@echo "                 • Create .env from template"
	@echo "                 • Configure SERVICE_HOME"
	@echo "                 • Set up local environment"
	@echo ""
	@echo "  testrun     - Run testrunner directly:"
	@echo "                 • Execute testrunner binary"
	@echo "                 • Useful for debugging"
	@echo ""
	@echo "🏭 PRODUCTION BUILD:"
	@echo "  buildall     - Production build with coverage:"
	@echo "                 • Enforce coverage requirements"
	@echo "                 • Build service with production tags"
	@echo "                 • Build testrunner"
	@echo "                 • Ready for deployment"
	@echo ""
	@echo "📦 CI/CD & BUILD SYSTEM:"
	@echo "  package      - Production mode build:"
	@echo "                 • No local development tags"
	@echo "                 • Suitable for deployment"
	@echo ""
	@echo "  package-local- Local development build:"
	@echo "                 • Includes 'local' build tag"
	@echo "                 • Enables local-only features"
	@echo ""
	@echo "  pre_build   - CI/CD pre-build steps"
	@echo "  post_build  - CI/CD post-build cleanup"
	@echo ""
	@echo "🧹 CLEANUP:"
	@echo "  clean       - Clean all artifacts:"
	@echo "                 • Remove build outputs"
	@echo "                 • Clean test results"
	@echo "                 • Remove temporary files"
	@echo ""
	@echo "Parameters:"
	@echo "  ENTRYPOINT_MODE - Build/run mode (default: start)"
	@echo "  BUILD_TAGS      - Go build tags:"
	@echo "                    • Production: empty by default"
	@echo "                    • Local dev: 'local' tag added"
	@echo ""
	@echo "Examples:"
	@echo "  make devenv                        # Set up development environment"
	@echo "  make dev                           # Full development cycle"
	@echo "  make dev-run                       # Quick test iteration"
	@echo "  make test-coverage                 # Process coverage data"
	@echo "  make buildall BUILD_TAGS=feature   # Build with custom tag"
	@echo "  make package-local                 # Local development build"
	@echo "  ENTRYPOINT_MODE=test make package  # Run tests in CI/CD"

# ==============================================================================
# LOCAL DEVELOPMENT WORKFLOW
# ==============================================================================

# Full development workflow with automatic cleanup
dev: dev-clean
	@echo "🚀 Starting full development workflow..."
	@echo ""
	@echo "📋 Workflow Steps:"
	@echo "  1. Clean environment and setup directories"
	@echo "  2. Build service and testrunner binaries"
	@echo "  3. Setup LocalBus topics for testing"
	@echo "  4. Start service with coverage tracking"
	@echo "  5. Run integration test suite"
	@echo "  6. Collect logs and generate reports"
	@echo "  7. Process coverage data"
	@echo "  8. Clean up processes and temporary files"
	@echo ""
	@cd test && ./local_test_run.sh build
	@echo ""
	@echo "✅ Development workflow completed!"
	@echo "📁 Results available in:"
	@echo "   • Coverage Report: test/results/coverage.html"
	@echo "   • Test Results:   test/results/report.txt"
	@echo "   • Logs:          test/results/logs/"

# Quick development iteration - skips rebuild, uses existing binaries
dev-run:
	@echo "⚡ Quick development run (using existing binaries)..."
	@echo "📋 Steps:"
	@echo "  1. Setup test environment"
	@echo "  2. Start service (existing binary)"
	@echo "  3. Run integration tests"
	@echo "  4. Collect results and logs"
	@cd test && ./local_test_run.sh run
	@echo "✅ Quick run completed!"
	@echo "📁 Check test/results/ for latest run data"

# Development cleanup
dev-clean:
	@echo "🧹 Cleaning development environment..."
	@pkill -f "service" 2>/dev/null || true
	@pkill -f "testrunner" 2>/dev/null || true
	@rm -rf test/results/ test/coverage/ test/*.pid 2>/dev/null || true
	@rm -rf /tmp/kritikos-messagebus* 2>/dev/null || true
	@echo "✅ Development environment cleaned"

# Generate coverage report from latest test run (standalone)
test-coverage:
	@echo "📊 Generating coverage report from latest test run..."
	@if [ -f "test/results/coverage.out" ]; then \
		echo "✅ Using existing coverage data..."; \
		cd src/service && \
		go tool cover -html=../../test/results/coverage.out -o=../../test/results/coverage.html && \
		go tool cover -func=../../test/results/coverage.out > ../../test/results/coverage_summary.txt && \
		cd ../../ && \
		echo "✅ Coverage report updated: test/results/coverage.html"; \
		echo "📋 Coverage summary:"; \
		go tool cover -func=test/results/coverage.out | tail -1; \
	elif [ -d "test/coverage" ] && [ -n "$$(ls -A test/coverage 2>/dev/null)" ]; then \
		echo "✅ Processing raw coverage data..."; \
		mkdir -p test/results && \
		cd src/service && \
		go tool covdata textfmt -i=../../test/coverage -o=../../test/results/coverage.out && \
		go tool cover -html=../../test/results/coverage.out -o=../../test/results/coverage.html && \
		go tool cover -func=../../test/results/coverage.out > ../../test/results/coverage_summary.txt && \
		cd ../../ && \
		echo "✅ Coverage report: test/results/coverage.html"; \
		echo "📋 Coverage summary:"; \
		go tool cover -func=test/results/coverage.out | tail -1; \
	else \
		echo "❌ No coverage data found. Run 'make dev' first."; \
		exit 1; \
	fi

# ==============================================================================
# CI/CD & BUILD SYSTEM TARGETS
# ==============================================================================

# Package target - production build without local development features
# Use this for deployment builds where local testing features should be disabled
package:
	@echo "📦 Running production build (ENTRYPOINT_MODE=$(ENTRYPOINT_MODE))..."
	@echo "   • No local development features"
	@echo "   • Build tags: $(if $(BUILD_TAGS),'$(BUILD_TAGS)','<none>')"
	@$(MAKE) -C src $(ENTRYPOINT_MODE) BUILD_TAGS="$(BUILD_TAGS)"

# Package target for local development - includes local testing features
# Use this for local development and testing where additional debug/test features are needed
package-local:
	@echo "🔧 Running local development build (ENTRYPOINT_MODE=$(ENTRYPOINT_MODE))..."
	@echo "   • Including local development features"
	@echo "   • Build tags: $(LOCAL_BUILD_TAGS)"
	@$(MAKE) -C src $(ENTRYPOINT_MODE) BUILD_TAGS="$(LOCAL_BUILD_TAGS)"

# Pre-build target that depends on package
pre_build: package
	@echo "Pre-build completed"
	@cp ${ROOT}/tools/scripts/initializer/set-common-ni-env.sh .

# Post-build target that depends on clean
post_build: clean
	@echo "Post-build completed"
	-@rm -fR set-common-ni-env.sh

# ==============================================================================
# PRODUCTION BUILD
# ==============================================================================

# Production build with enforced coverage
buildall:
	@echo "🏭 Production build with enforced coverage..."
	@$(MAKE) -C src coverage-enforce
	@$(MAKE) -C src service-build BUILD_TAGS="$(BUILD_TAGS)"
	@$(MAKE) -C src testrunner-build BUILD_TAGS="$(BUILD_TAGS)"
	@echo "✅ Production build completed with enforced coverage!"

# ==============================================================================
# CLEANUP
# ==============================================================================

# Clean all artifacts
clean:
	@echo "🧹 Cleaning all build artifacts..."
	@$(MAKE) -C src clean
	@rm -rf test/results/ test/coverage/ test/*.pid 2>/dev/null || true
	@rm -rf /tmp/kritikos-messagebus* 2>/dev/null || true
	@echo "✅ All artifacts cleaned"


# Copy env.example to .env and set SERVICE_HOME to current root directory
devenv:
	@echo "Setting up .env from env.example..."
	cp test/env.example .env
	@echo "Updating .env with current SERVICE_HOME..."
	sed -i.bak "s|^SERVICE_HOME=.*$$|SERVICE_HOME=$(shell pwd)|g" .env
	@rm .env.bak
	@echo ".env created with SERVICE_HOME=$(shell pwd)"

# Run testrunner directly - useful for debugging test issues
# This bypasses the test orchestration script and runs the testrunner binary directly
testrun:
	@echo "🔍 Running testrunner directly..."
	@echo "Note: This is for debugging only. Use 'make dev' or 'make dev-run' for normal testing."
	@$(MAKE) -C src/testrunner run
