LANG=python
IMAGE_NAME=cratos

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
	@echo "                 • Build service with enforced coverage"
	@echo "                 • Build testrunner (no coverage)"
	@echo "                 • Run integration tests"
	@echo "                 • Generate reports"
	@echo "                 • Auto cleanup"
	@echo ""
	@echo "  dev-run      - Quick development iteration:"
	@echo "                 • Use existing binaries"
	@echo "                 • Run integration tests only"
	@echo "                 • Faster for testing changes"
	@echo ""
	@echo "  dev-clean    - Development cleanup:"
	@echo "                 • Kill all processes"
	@echo "                 • Clean artifacts and logs"
	@echo "                 • Reset environment"
	@echo ""
	@echo "  test-coverage- Generate coverage report from latest test run"
	@echo "                 • Convert binary coverage data to HTML"
	@echo "                 • Show coverage summary"
	@echo ""
	@echo "🏭 PRODUCTION BUILD:"
	@echo "  buildall       - Production build with enforced coverage:"
	@echo "                 • Build all services with coverage enforcement"
	@echo "                 • Ready for deployment"
	@echo "                 • No test execution"
	@echo ""
	@echo "📦 CI/CD & BUILD SYSTEM:"
	@echo "  package      - Run specified ENTRYPOINT_MODE target in src/ (production mode)"
	@echo "  package-local- Run specified ENTRYPOINT_MODE target in src/ (local development mode)"
	@echo "  pre_build    - Pre-build target (depends on package)"
	@echo "  post_build   - Post-build target (depends on clean)"
	@echo ""
	@echo "🧹 CLEANUP:"
	@echo "  clean        - Clean all build artifacts"
	@echo ""
	@echo "Parameters:"
	@echo "  ENTRYPOINT_MODE - Mode to pass to src/Makefile (default: start)"
	@echo "  BUILD_TAGS      - Additional build tags to use (empty by default)"
	@echo ""
	@echo "Examples:"
	@echo "  make dev                           # Full local development cycle"
	@echo "  make dev-run                       # Quick test run"
	@echo "  make test-coverage                 # Generate coverage report from latest test"
	@echo "  make buildall                      # Production build"
	@echo "  make package                       # Uses default ENTRYPOINT_MODE=start (production)"
	@echo "  make package-local                 # Uses default ENTRYPOINT_MODE=start (local dev)"
	@echo "  make package ENTRYPOINT_MODE=test  # Run tests via CI/CD"

# ==============================================================================
# LOCAL DEVELOPMENT WORKFLOW
# ==============================================================================

# Full development workflow
dev: dev-clean
	@echo "🚀 Starting full development workflow..."
	@echo ""
	@echo "📋 Workflow Steps:"
	@echo "  1. Setup environment and directories"
	@echo "  2. Build service with enforced coverage"
	@echo "  3. Build testrunner (no coverage)"
	@echo "  4. Run integration tests"
	@echo "  5. Generate coverage and test reports"
	@echo "  6. Auto cleanup processes"
	@echo ""
	@cd test && ./run_tests_local.sh build
	@echo ""
	@echo "✅ Development workflow completed!"
	@echo "📁 Check test/results/ for reports and logs"

# Quick development iteration (use existing binaries)
dev-run:
	@echo "⚡ Quick development run (using existing binaries)..."
	@cd test && ./run_tests_local.sh run
	@echo "✅ Quick run completed!"

# Development cleanup
dev-clean:
	@echo "🧹 Cleaning development environment..."
	@pkill -f "service" 2>/dev/null || true
	@pkill -f "testrunner" 2>/dev/null || true
	@rm -rf test/results/ test/coverage/ test/*.pid 2>/dev/null || true
	@rm -rf /tmp/cratos-messagebus* 2>/dev/null || true
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

# Package target - runs the specified ENTRYPOINT_MODE in src/ (production mode)
package:
	@echo "Running package target with ENTRYPOINT_MODE=$(ENTRYPOINT_MODE) (production mode)"
	@$(MAKE) -C src $(ENTRYPOINT_MODE) BUILD_TAGS="$(BUILD_TAGS)"

# Package target with local development build tags
package-local:
	@echo "Running package target with ENTRYPOINT_MODE=$(ENTRYPOINT_MODE) (local development mode)"
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
	@rm -rf /tmp/cratos-messagebus* 2>/dev/null || true
	@echo "✅ All artifacts cleaned"


# Copy env.example to .env and set SERVICE_HOME to current root directory
devenv:
	@echo "Setting up .env from env.example..."
	cp test/env.example .env
	@echo "Updating .env with current SERVICE_HOME..."
	sed -i.bak "s|^SERVICE_HOME=.*$$|SERVICE_HOME=$(shell pwd)|g" .env
	@rm .env.bak
	@echo ".env created with SERVICE_HOME=$(shell pwd)"

testrun:
	@echo "run testrunner "
	@$(MAKE) -C src/testrunner run
