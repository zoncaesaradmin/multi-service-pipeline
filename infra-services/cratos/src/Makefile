# Optimized Root Makefile for Golang Service Skeleton
.PHONY: help build build-with-coverage build-local test test-local clean clean-coverage clean-all-coverage deep-clean run-service run-service-local run-service-dev run-tests run-tests-local install-deps fmt vet lint coverage coverage-enforce start start-local service-build service-build-local service-build-with-coverage service-coverage service-test service-run testrunner-build testrunner-build-local testrunner-test testrunner-run shared-build shared-build-local shared-test shared-coverage ci dev-setup e2e-test

# Coverage enforcement thresholds
SERVICE_COVERAGE_THRESHOLD = 45

# Build tags (empty by default, but can be set from environment/command line/parent)
BUILD_TAGS ?=

# Helper function to safely append a tag if not already present
define append_tag
$(if $(filter $(1),$(BUILD_TAGS)),$(BUILD_TAGS),$(if $(BUILD_TAGS),$(BUILD_TAGS) $(1),$(1)))
endef

# Local build tags - safely append 'local' to existing BUILD_TAGS
LOCAL_BUILD_TAGS = $(call append_tag,local)

# Coverage enforcement always uses local tags for consistency and reliability
COVERAGE_BUILD_TAGS = $(LOCAL_BUILD_TAGS)

# Reusable function for coverage calculation  
define calculate_coverage
	@rm -f coverage.out; \
	echo "mode: set" > coverage.out; \
	if [ -f service/service_coverage.tmp ]; then \
		tail -n +2 service/service_coverage.tmp >> coverage.out; \
	fi; \
	if [ -f shared/shared_coverage.tmp ]; then \
		tail -n +2 shared/shared_coverage.tmp >> coverage.out; \
	fi; \
	if [ ! -s coverage.out ] || [ $$(wc -l < coverage.out) -eq 1 ]; then \
		echo "❌ No coverage data found"; \
		rm -f coverage.out; \
		exit 1; \
	fi; \
	export SERVICE_COV=$$(cd service && go tool cover -func=service_coverage.tmp 2>/dev/null | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	if [ -z "$$SERVICE_COV" ]; then export SERVICE_COV="0"; fi; \
	export SHARED_COV=$$(cd shared && go tool cover -func=shared_coverage.tmp 2>/dev/null | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	if [ -z "$$SHARED_COV" ]; then export SHARED_COV="0"; fi; \
	export COMBINED_COV=$$(go tool cover -func=coverage.out 2>/dev/null | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	if [ -z "$$COMBINED_COV" ]; then \
		echo "⚠️  Warning: Failed to calculate combined coverage, trying alternative method..."; \
		export COMBINED_COV=$$(go tool cover -func=coverage.out 2>&1 | grep "total:" | awk '{print $$3}' | sed 's/%//'); \
		if [ -z "$$COMBINED_COV" ]; then export COMBINED_COV="0"; fi; \
	fi
endef

# Function for running module commands
define run_module_cmd
	@echo "$(1) $(2)..."
	@cd $(2) && $(MAKE) $(3) BUILD_TAGS="$(4)"
endef

# Default target
help:
	@echo "Golang Service Skeleton - Optimized Makefile"
	@echo "=============================================="
	@echo ""
	@echo "🎯 Main Targets:"
	@echo "  start           - Build with enforced combined coverage (production ready)"
	@echo "  build           - Build all modules (production: '$(BUILD_TAGS)', local: '$(LOCAL_BUILD_TAGS)')"
	@echo "  build-with-coverage - Build with coverage enforcement (always local mode)"
	@echo "  test            - Run all tests (production/local modes available)"
	@echo "  coverage        - Generate and display combined coverage report"
	@echo "  coverage-enforce- Enforce coverage threshold ($(SERVICE_COVERAGE_THRESHOLD)%)"
	@echo ""
	@echo "🚀 Service Operations:"
	@echo "  run-service   - Build with coverage + run service (with/without coverage variants)"
	@echo "  run-tests       - Build and run integration tests"
	@echo ""
	@echo "🛠️  Development:"
	@echo "  install-deps    - Download dependencies for all projects"
	@echo "  fmt             - Format code in all projects"
	@echo "  vet             - Vet code in all projects"
	@echo "  lint            - Lint code in all projects"
	@echo "  dev-setup       - Complete development setup"
	@echo ""
	@echo "🧹 Cleanup:"
	@echo "  clean           - Clean build artifacts (preserves main coverage.out)"
	@echo "  clean-coverage  - Clean individual coverage files only"
	@echo "  clean-all-coverage - Clean all coverage files including combined"
	@echo "  deep-clean      - Deep clean including vendor and module cache"
	@echo ""
	@echo "🔧 CI/CD:"
	@echo "  ci              - Full CI pipeline with coverage enforcement"
	@echo "  e2e-test        - End-to-end integration test"
	@echo ""
	@echo "📦 Module-Specific (service|testrunner|shared):"
	@echo "  <module>-build     - Build specific module"
	@echo "  <module>-test      - Test specific module"
	@echo "  <module>-coverage  - Coverage for specific module"
	@echo ""
	@echo "Parameters:"
	@echo "  BUILD_TAGS='$(BUILD_TAGS)' (current), LOCAL_BUILD_TAGS='$(LOCAL_BUILD_TAGS)'"
	@echo "  Append -local to most targets for local development mode"
	@echo ""
	@echo "Coverage Scope: service/* + shared/* (excludes testrunner/*)"

# ==============================================================================
# MAIN WORKFLOW TARGETS
# ==============================================================================

# Start target - Production ready build with coverage enforcement
start: coverage-enforce service-build
	@echo "✅ Service built with enforced coverage and ready for deployment!"

automation: testrunner-build
	@echo "✅ Test runner built for production environment test automation!"

# Alias for start (for backward compatibility)
start-local: start

# Build all projects
build: service-build testrunner-build shared-build

build-local: service-build-local testrunner-build-local shared-build-local

# Build with coverage enforcement
build-with-coverage: coverage-enforce build
	@echo "✅ Build completed with coverage enforcement!"

# Run all tests
test: service-test testrunner-test shared-test

test-local: service-test-local testrunner-test-local shared-test-local

# ==============================================================================
# COVERAGE TARGETS (OPTIMIZED)
# ==============================================================================

# Generate combined coverage report
coverage: service-coverage shared-coverage
	@echo ""
	@echo "=========================================="
	@echo "COMBINED SERVICE COVERAGE REPORT"
	@echo "=========================================="
	@echo "Scope: service/* + shared/* (excludes testrunner/*)"
	@echo "Build Tags: $(COVERAGE_BUILD_TAGS)"
	@echo ""
	$(call calculate_coverage)
	@echo "📊 Service Module: $$SERVICE_COV%"
	@echo "📊 Shared Module:  $$SHARED_COV%"
	@echo ""
	@echo "🎯 TRUE COMBINED COVERAGE: $$COMBINED_COV%"
	@echo "🎯 THRESHOLD: $(SERVICE_COVERAGE_THRESHOLD)%"
	@echo ""
	@echo "📁 Coverage files preserved:"
	@echo "   • service/service_coverage.tmp"
	@echo "   • shared/shared_coverage.tmp"
	@echo "   • coverage.out"

# Enforce coverage threshold
coverage-enforce: service-coverage shared-coverage
	@echo "=========================================="
	@echo "SERVICE COVERAGE ENFORCEMENT"
	@echo "=========================================="
	@echo "Scope: service/* + shared/* (excludes testrunner/*)"
	@echo "Build Tags: $(COVERAGE_BUILD_TAGS)"
	@echo "Threshold: $(SERVICE_COVERAGE_THRESHOLD)%"
	@echo ""
	@rm -f coverage.out; \
	echo "mode: set" > coverage.out; \
	if [ -f service/service_coverage.tmp ]; then \
		tail -n +2 service/service_coverage.tmp >> coverage.out; \
	fi; \
	if [ -f shared/shared_coverage.tmp ]; then \
		tail -n +2 shared/shared_coverage.tmp >> coverage.out; \
	fi; \
	if [ ! -s coverage.out ] || [ $$(wc -l < coverage.out) -eq 1 ]; then \
		echo "❌ No coverage data found"; \
		rm -f coverage.out; \
		exit 1; \
	fi; \
	SERVICE_COV=$$(cd service && go tool cover -func=service_coverage.tmp 2>/dev/null | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	if [ -z "$$SERVICE_COV" ]; then SERVICE_COV="0"; fi; \
	SHARED_COV=$$(cd shared && go tool cover -func=shared_coverage.tmp 2>/dev/null | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	if [ -z "$$SHARED_COV" ]; then SHARED_COV="0"; fi; \
	COMBINED_COV=$$(go tool cover -func=coverage.out 2>/dev/null | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	if [ -z "$$COMBINED_COV" ]; then \
		echo "⚠️  Warning: Failed to calculate combined coverage, trying alternative method..."; \
		COMBINED_COV=$$(go tool cover -func=coverage.out 2>&1 | grep "total:" | awk '{print $$3}' | sed 's/%//'); \
		if [ -z "$$COMBINED_COV" ]; then COMBINED_COV="0"; fi; \
	fi; \
	echo "📊 Service Module: $$SERVICE_COV%"; \
	echo "📊 Shared Module:  $$SHARED_COV%"; \
	echo ""; \
	echo "🎯 TRUE COMBINED COVERAGE: $$COMBINED_COV%"; \
	echo "🎯 REQUIRED: $(SERVICE_COVERAGE_THRESHOLD)%"; \
	echo ""; \
	if [ $$(awk -v cov="$$COMBINED_COV" -v thresh="$(SERVICE_COVERAGE_THRESHOLD)" 'BEGIN{print (cov >= thresh) ? 1 : 0}') -eq 1 ]; then \
		echo "✅ SERVICE COVERAGE ENFORCEMENT PASSED!"; \
		echo "   Combined coverage $$COMBINED_COV% meets $(SERVICE_COVERAGE_THRESHOLD)% threshold"; \
	else \
		echo "❌ SERVICE COVERAGE ENFORCEMENT FAILED!"; \
		echo "   Combined coverage $$COMBINED_COV% below $(SERVICE_COVERAGE_THRESHOLD)% threshold"; \
		echo ""; \
		echo "💡 Coverage includes: service/* + shared/* (excludes testrunner/*)"; \
		exit 1; \
	fi; \
	echo "📁 Coverage files preserved: coverage.out, service/service_coverage.tmp, shared/shared_coverage.tmp"

# ==============================================================================
# COMPONENT OPERATIONS
# ==============================================================================

# Run service with coverage enforcement
run-service: service-build-with-coverage service-run

run-service-local: service-build-with-coverage service-run-local

# Run service without coverage (for development)
run-service-dev: service-build service-run

# Run integration tests
run-tests: testrunner-run

run-tests-local: testrunner-run-local

# ==============================================================================
# CLEANUP TARGETS
# ==============================================================================

clean: service-clean testrunner-clean shared-clean
	@echo "Cleaning individual module coverage files..."
	@rm -f service/service_coverage.tmp shared/shared_coverage.tmp 2>/dev/null || true
	@echo "✅ All projects cleaned (main coverage.out preserved)"

clean-coverage:
	@echo "Cleaning individual module coverage files..."
	@rm -f service/service_coverage.tmp shared/shared_coverage.tmp
	@echo "✅ Individual coverage files cleaned (main coverage.out preserved)"

clean-all-coverage:
	@echo "Cleaning all coverage files including main combined coverage..."
	@rm -f service/service_coverage.tmp shared/shared_coverage.tmp coverage.out
	@echo "✅ All coverage files cleaned"

deep-clean: service-deep-clean testrunner-deep-clean shared-deep-clean

# ==============================================================================
# DEVELOPMENT WORKFLOW
# ==============================================================================

install-deps: service-deps testrunner-deps shared-deps

fmt: service-fmt testrunner-fmt shared-fmt

vet: service-vet testrunner-vet shared-vet

lint: service-lint shared-lint

dev-setup: install-deps fmt vet build
	@echo "✅ Development setup completed!"

ci: install-deps fmt vet test coverage-enforce build
	@echo "✅ CI pipeline completed successfully!"

# ==============================================================================
# MODULE-SPECIFIC TARGETS (OPTIMIZED WITH FUNCTIONS)
# ==============================================================================

# Service targets
service-build:
	$(call run_module_cmd,Building,service,build,$(BUILD_TAGS))

service-build-local:
	$(call run_module_cmd,Building,service,build,$(LOCAL_BUILD_TAGS))

service-build-with-coverage: service-coverage shared-coverage
	@echo "Building service with coverage enforcement..."
	$(call calculate_coverage)
	@echo "🎯 Coverage: Service=$$SERVICE_COV% | Shared=$$SHARED_COV% | Combined=$$COMBINED_COV%"
	@if [ $$(awk -v cov="$$COMBINED_COV" -v thresh="$(SERVICE_COVERAGE_THRESHOLD)" 'BEGIN{print (cov >= thresh) ? 1 : 0}') -eq 1 ]; then \
		echo "✅ Coverage $$COMBINED_COV% meets threshold - proceeding with build"; \
		cd service && $(MAKE) build BUILD_TAGS="$(LOCAL_BUILD_TAGS)"; \
	else \
		echo "❌ Coverage $$COMBINED_COV% below $(SERVICE_COVERAGE_THRESHOLD)% threshold - build blocked"; \
		exit 1; \
	fi

service-coverage:
	$(call run_module_cmd,Running coverage for,service,coverage,$(COVERAGE_BUILD_TAGS))

service-test:
	$(call run_module_cmd,Testing,service,test,$(BUILD_TAGS))

service-test-local:
	$(call run_module_cmd,Testing,service,test,$(LOCAL_BUILD_TAGS))

service-clean:
	$(call run_module_cmd,Cleaning,service,clean,)

service-deep-clean:
	$(call run_module_cmd,Deep cleaning,service,deep-clean,)

service-deps:
	$(call run_module_cmd,Installing dependencies for,service,deps,)

service-fmt:
	$(call run_module_cmd,Formatting,service,fmt,)

service-vet:
	$(call run_module_cmd,Vetting,service,vet,)

service-lint:
	$(call run_module_cmd,Linting,service,lint,)

service-run:
	$(call run_module_cmd,Running,service,run,$(BUILD_TAGS))

service-run-local:
	$(call run_module_cmd,Running,service,run,$(LOCAL_BUILD_TAGS))

# Testrunner targets
testrunner-build:
	$(call run_module_cmd,Building,testrunner,build,$(BUILD_TAGS))

testrunner-build-local:
	$(call run_module_cmd,Building,testrunner,build,$(LOCAL_BUILD_TAGS))

testrunner-test:
	$(call run_module_cmd,Testing,testrunner,test,$(BUILD_TAGS))

testrunner-test-local:
	$(call run_module_cmd,Testing,testrunner,test,$(LOCAL_BUILD_TAGS))

testrunner-clean:
	$(call run_module_cmd,Cleaning,testrunner,clean,)

testrunner-deep-clean:
	$(call run_module_cmd,Deep cleaning,testrunner,deep-clean,)

testrunner-deps:
	$(call run_module_cmd,Installing dependencies for,testrunner,deps,)

testrunner-fmt:
	$(call run_module_cmd,Formatting,testrunner,fmt,)

testrunner-vet:
	$(call run_module_cmd,Vetting,testrunner,vet,)

testrunner-run:
	$(call run_module_cmd,Running,testrunner,run,$(BUILD_TAGS))

testrunner-run-local:
	$(call run_module_cmd,Running,testrunner,run,$(LOCAL_BUILD_TAGS))

testrunner-integration:
	@echo "Running integration tests..."
	@cd testrunner && $(MAKE) run-integration-tests

testrunner-performance:
	@echo "Running performance tests..."
	@cd testrunner && $(MAKE) run-performance-tests

# Shared module targets
shared-build:
	$(call run_module_cmd,Building,shared,build,$(BUILD_TAGS))

shared-build-local:
	$(call run_module_cmd,Building,shared,build,$(LOCAL_BUILD_TAGS))

shared-test:
	$(call run_module_cmd,Testing,shared,test,$(BUILD_TAGS))

shared-test-local:
	$(call run_module_cmd,Testing,shared,test,$(LOCAL_BUILD_TAGS))

shared-coverage:
	$(call run_module_cmd,Running coverage for,shared,coverage,$(COVERAGE_BUILD_TAGS))

shared-coverage-local:
	$(call run_module_cmd,Running coverage for,shared,coverage,$(LOCAL_BUILD_TAGS))

shared-coverage-enforce:
	@echo "Enforcing shared module coverage..."
	@cd shared && $(MAKE) coverage-enforce

shared-clean:
	$(call run_module_cmd,Cleaning,shared,clean,)

shared-deep-clean:
	$(call run_module_cmd,Deep cleaning,shared,clean,) # shared module doesn't have vendor/

shared-deps:
	$(call run_module_cmd,Installing dependencies for,shared,deps,)

shared-fmt:
	$(call run_module_cmd,Formatting,shared,fmt,)

shared-vet:
	$(call run_module_cmd,Vetting,shared,vet,)

shared-lint:
	$(call run_module_cmd,Linting,shared,lint,)

shared-benchmark:
	@echo "Running shared module benchmarks..."
	@cd shared && $(MAKE) benchmark

shared-race:
	@echo "Running shared module race detection..."
	@cd shared && $(MAKE) race

# ==============================================================================
# INTEGRATION TESTING
# ==============================================================================

e2e-test: service-build testrunner-build
	@echo "Starting end-to-end test..."
	@echo "Starting service in background..."
	@cd ./.. && ./bin/service.bin & SERVICE_PID=$$!; \
	sleep 3; \
	echo "Running integration tests..."; \
	cd ./.. && ./bin/testrunner.bin; \
	TEST_RESULT=$$?; \
	echo "Stopping service..."; \
	kill $$SERVICE_PID 2>/dev/null || true; \
	exit $$TEST_RESULT
