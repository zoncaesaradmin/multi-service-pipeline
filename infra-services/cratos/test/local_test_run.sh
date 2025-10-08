#!/bin/bash

# Local Development Test Runner
# Usage: ./local_test_run.sh [build|run]
# Default mode: build (builds and runs)

set -e
# Note: We handle test failures gracefully to ensure log collection

# kafka topic overrides
INPUT_TOPIC=cisco_nir-anomalies
RULES_TOPIC=cisco_nir-alertRules
RULE_TASKS_TOPIC=cisco_nir-ruletasks
OUTPUT_TOPIC=cisco_nir-prealerts

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Configuration - Updated paths to work from both root and test directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ "$(basename "$SCRIPT_DIR")" == "test" ]]; then
    # Running from test directory or script in test directory
    ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
    SERVICE_DIR="$ROOT_DIR/src/service"
    TESTRUNNER_DIR="$ROOT_DIR/src/testrunner"
    RESULTS_DIR="$SCRIPT_DIR/results"
    LOGS_DIR="$SCRIPT_DIR/results/logs"
    COVERAGE_DIR="$SCRIPT_DIR/coverage"
else
    # Fallback: assume we're in root and test is a subdirectory
    ROOT_DIR="$(pwd)"
    SERVICE_DIR="$ROOT_DIR/src/service"
    TESTRUNNER_DIR="$ROOT_DIR/src/testrunner"
    RESULTS_DIR="$ROOT_DIR/test/results"
    LOGS_DIR="$ROOT_DIR/test/results/logs"
    COVERAGE_DIR="$ROOT_DIR/test/coverage"
fi

log_info "Repository root: $ROOT_DIR"

# Load environment variables from .env file (if present)
if [ -f "$ROOT_DIR/.env" ]; then
    log_info "Loading environment variables from $ROOT_DIR/.env"
    set -a
    source "$ROOT_DIR/.env"
    set +a
else
    log_warning ".env file not found at $ROOT_DIR/.env; using default environment."
fi

BUILD_MODE="${1:-build}"

# Function to cleanup processes and directories
cleanup() {
    log_info "Cleaning up processes and temporary files..."
    
    # Kill any running service or testrunner processes
    pkill -f "service.bin" 2>/dev/null || true
    pkill -f "testrunner.bin" 2>/dev/null || true
    
    # Clean up coverage data
    if [ -d "$COVERAGE_DIR" ]; then
        rm -rf "$COVERAGE_DIR"
    fi
    
    # Clean up LocalBus data
    if [ -d "/tmp/cratos-messagebus-test" ]; then
        rm -rf "/tmp/cratos-messagebus-test"
        log_info "Cleaned up LocalBus message data"
    fi
    
    if [ -d "/tmp/cratos-messagebus-offsets" ]; then
        rm -rf "/tmp/cratos-messagebus-offsets"
        log_info "Cleaned up LocalBus offset data"
    fi
    
    log_success "Cleanup completed"
}

# Function to setup directories
setup_directories() {
    log_info "Setting up directories..."
    mkdir -p "$RESULTS_DIR"
    mkdir -p "$LOGS_DIR"
    mkdir -p "$COVERAGE_DIR"
    log_success "Directories created"
}

# Function to build service with coverage
build_service() {
    log_info "Building service with coverage instrumentation and local tags..."
    
    # Verify service directory exists and has Makefile
    if [ ! -f "$SERVICE_DIR/Makefile" ]; then
        log_error "Service Makefile not found at $SERVICE_DIR/Makefile"
        return 1
    fi
    
    cd "$SERVICE_DIR"
    make build-local
    BUILD_RESULT=$?
    cd - > /dev/null
    
    if [ $BUILD_RESULT -eq 0 ]; then
        log_success "Service built successfully with coverage instrumentation"
        return 0
    else
        log_error "Service build failed"
        return 1
    fi
}

# Function to build testrunner
build_testrunner() {
    log_info "Building testrunner using Makefile..."
    
    # Verify testrunner directory exists and has Makefile
    if [ ! -f "$TESTRUNNER_DIR/Makefile" ]; then
        log_error "Testrunner Makefile not found at $TESTRUNNER_DIR/Makefile"
        exit 1
    fi
    
    cd "$TESTRUNNER_DIR"
    
    # Build testrunner using Makefile
    make build-local
    BUILD_RESULT=$?
    
    cd - > /dev/null
    
    if [ $BUILD_RESULT -eq 0 ]; then
        log_success "Testrunner built successfully using Makefile"
    else
        exit 1
    fi
}

# Function to run service
run_service() {
    log_info "Starting service..."
    cd "$SERVICE_DIR"
    
    # Run in background and capture stdout/stderr
    make run-local > "$LOGS_DIR/service_stdouterr.log" 2>&1 &
    COMPONENT_PID=$!
    
    # Give process a moment to start
    sleep 1
    
    # Check if process started successfully
    if ! kill -0 $COMPONENT_PID 2>/dev/null; then
        log_error "Service failed to start"
        return 1
    fi
    
    echo $COMPONENT_PID > "$ROOT_DIR/test/service.pid"
    log_success "Service started with PID $COMPONENT_PID"
    cd - > /dev/null
    return 0
}

# Function to run testrunner
run_testrunner() {
    log_info "Running testrunner..."
    cd "$TESTRUNNER_DIR"
    
    # Run testrunner and capture output
    ./../../bin/testrunner.bin > "$LOGS_DIR/testrunner_stdouterr.log" 2>&1 &
    TESTRUNNER_PID=$!
    
    # Check if process started successfully
    if ! kill -0 $TESTRUNNER_PID 2>/dev/null; then
        log_error "Test runner failed to start"
        return 1
    fi
    
    log_success "Test runner started with PID $TESTRUNNER_PID"
    echo $TESTRUNNER_PID > "$ROOT_DIR/test/testrunner.pid"

    STATUS_URL="http://localhost:4478/report/status"
    log_info "Waiting for test runner to complete..."
    
    # Add timeout for test runner (30 minutes)
    local timeout=1800  # 30 minutes in seconds
    local start_time=$(date +%s)
    local current_time
    
    while true; do
        # Check if test runner is still running
        if ! kill -0 $TESTRUNNER_PID 2>/dev/null; then
            log_error "Test runner process died unexpectedly"
            return 1
        fi
        
        # Check timeout
        current_time=$(date +%s)
        if [ $((current_time - start_time)) -gt $timeout ]; then
            log_error "Test runner timed out after ${timeout} seconds"
            kill $TESTRUNNER_PID 2>/dev/null || true
            return 1
        fi
        
        # Get status with timeout
        STATUS=$(curl -s --connect-timeout 5 "$STATUS_URL" | grep -o '"status": *"[^"]*"' | cut -d'"' -f4)
        if [ $? -ne 0 ]; then
            log_warning "Failed to get test runner status, retrying..."
            sleep 4
            continue
        fi
        
        if [ "$STATUS" = "complete" ]; then
            log_success "Test runner completed"
            break
        fi
        
        sleep 4
    done

    log_info "Fetching test report..."
    REPORT=$(curl -s --connect-timeout 5 http://localhost:4478/report/text)
    if [ $? -ne 0 ]; then
        log_error "Failed to fetch test report"
        return 1
    fi
    
    if echo "$REPORT" | grep -q "SUITE RESULT: SUCCESS"; then
        log_success "Test suite PASSED"
        TEST_RESULT=0
    else
        log_error "Test suite FAILED"
        TEST_RESULT=1
    fi
    
    cd - > /dev/null
    return $TEST_RESULT
}

# Function to stop service
stop_service() {
    PIDFILE="$ROOT_DIR/test/service.pid"
    if [ -f "$PIDFILE" ]; then
        COMPONENT_PID=$(cat "$PIDFILE")
        log_info "Stopping service (PID: $COMPONENT_PID)..."
        kill $COMPONENT_PID 2>/dev/null || true
        rm -f "$PIDFILE"
        log_success "Service stopped"
        
        # Wait a moment for clean shutdown and final logs
        sleep 1
    fi
}

# Function to collect and organize logs
collect_logs() {
    log_info "Collecting and organizing logs..."
    
    # Display log file summary
    echo
    log_info "Log Files Summary:"
    if [ -d "$LOGS_DIR" ]; then
        ls -la "$LOGS_DIR" | while read line; do
            echo "  $line"
        done
    fi
    echo
}

# Function to generate coverage report
generate_coverage_report() {
    log_info "Generating coverage report..."
    
    if [ -d "$COVERAGE_DIR" ] && [ "$(ls -A "$COVERAGE_DIR")" ]; then
        # Convert binary coverage data to text format
        # Use the service's working directory for proper module resolution
        cd "$SERVICE_DIR"
        go tool covdata textfmt -i="$COVERAGE_DIR" -o="$RESULTS_DIR/coverage.out"
        
        # Generate HTML coverage report
        go tool cover -html="$RESULTS_DIR/coverage.out" -o="$RESULTS_DIR/coverage.html"
        
        # Generate coverage summary
        go tool cover -func="$RESULTS_DIR/coverage.out" > "$RESULTS_DIR/coverage_summary.txt"
        
        cd - > /dev/null
        
        log_success "Coverage report generated at $RESULTS_DIR/coverage.html"
        
        # Show coverage summary
        echo
        log_info "Coverage Summary:"
        cat "$RESULTS_DIR/coverage_summary.txt" | tail -1
        echo
    else
        log_warning "No coverage data found"
    fi
}

# Function to generate final report
generate_report() {
    log_info "Generating test script report..."

    REPORT_FILE="$RESULTS_DIR/report.txt"
    TEST_EXEC_REPORT="$RESULTS_DIR/logs/complete_test_execution_report.txt"

    {
        echo "=================================================="
        echo "Local Development Test Report"
        echo "Generated: $(date)"
        echo "=================================================="
        echo
        
        echo "Build Results:"
        echo "- Service: Built successfully"
        echo "- Testrunner: Built successfully"
        echo
        
        echo "Test Execution:"
        if [ $TEST_RESULT -eq 0 ]; then
            echo "- Status: PASSED"
        else
            echo "- Status: FAILED"
        fi
        echo
        
        echo "Coverage Report:"
        if [ -f "$RESULTS_DIR/coverage_summary.txt" ]; then
            cat "$RESULTS_DIR/coverage_summary.txt" | tail -1
        else
            echo "- No coverage data available"
        fi
        echo
        
        #echo "Log Files:"
        #echo "- Service & Testrunner logs are at: $LOGS_DIR/"
        #echo "- Service stdout & stderr logs: $LOGS_DIR/service_stdouterr.log"
        #echo "- Testrunner stdout & stderr logs: $LOGS_DIR/testrunner_stdouterr.log.log"
        #echo "- Consolidated logs: $RESULTS_DIR/all_logs.txt"
        #echo
        
        echo "Output Files:"
        echo "- Coverage report: $RESULTS_DIR/coverage.html"
        echo "- Coverage summary: $RESULTS_DIR/coverage_summary.txt"
        echo "- This report: $REPORT_FILE"
        echo "- Test execution details: $TEST_EXEC_REPORT"
        echo
        
    } > "$REPORT_FILE"
    
    log_success "Test script report generated at $REPORT_FILE"
    echo
    log_info "=== QUICK SUMMARY ==="
    cat "$REPORT_FILE" | grep -A 10 "Test Execution:"
    
    # Print last 50 lines of test execution report if it exists
    if [ -f "$TEST_EXEC_REPORT" ]; then
        echo
        log_info "=== LAST 50 LINES OF TEST EXECUTION REPORT ==="
        log_info "File: $TEST_EXEC_REPORT"
        echo "----------------------------------------"
        tail -50 "$TEST_EXEC_REPORT"
        echo "----------------------------------------"
        log_info "=== END OF TEST EXECUTION REPORT ==="
    else
        echo
        log_warning "Test execution report not found at: $TEST_EXEC_REPORT"
    fi
}

# Function to exit with error and cleanup
exit_with_error() {
    local error_message="$1"
    local exit_code="${2:-1}"
    
    log_error "$error_message"
    cleanup
    exit $exit_code
}

# Main execution
main() {
    echo
    log_info "Starting Local Development Test Runner (mode: $BUILD_MODE)"
    log_info "Script directory: $SCRIPT_DIR"
    log_info "Repository root: $ROOT_DIR"
    log_info "Service directory: $SERVICE_DIR"
    log_info "Results directory: $RESULTS_DIR"
    echo
    
    # Setup trap for cleanup on exit
    trap 'cleanup' INT TERM
    
    # Setup directories
    setup_directories || exit_with_error "Failed to set up directories"

    # Ensure local bus topic directories exist for tests
    LOCAL_BUS_BASE="/tmp/cratos-messagebus-test"
    LOCAL_BUS_OFFSETS="/tmp/cratos-messagebus-offsets"
    
    # Clean up any existing local bus data to ensure clean test state
    if [ -d "$LOCAL_BUS_BASE" ]; then
        log_info "Cleaning existing LocalBus message data"
        rm -rf "$LOCAL_BUS_BASE" || exit_with_error "Failed to clean up LocalBus message data"
    fi
    
    if [ -d "$LOCAL_BUS_OFFSETS" ]; then
        log_info "Cleaning existing LocalBus offset data"
        rm -rf "$LOCAL_BUS_OFFSETS" || exit_with_error "Failed to clean up LocalBus offset data"
    fi
    
    # Create fresh directories
    for topic in $INPUT_TOPIC $OUTPUT_TOPIC $RULES_TOPIC $RULE_TASKS_TOPIC; do
        mkdir -p "$LOCAL_BUS_BASE/$topic" || exit_with_error "Failed to create topic directory: $topic"
        log_info "Created clean local bus topic directory: $LOCAL_BUS_BASE/$topic"
    done
    
    # Create offset directory
    mkdir -p "$LOCAL_BUS_OFFSETS" || exit_with_error "Failed to create offset directory"
    log_info "Created clean local bus offset directory: $LOCAL_BUS_OFFSETS"
    
    if [ "$BUILD_MODE" = "build" ] || [ "$BUILD_MODE" = "all" ]; then
        # Build phase
        if ! build_service; then
            cleanup
            exit 1
        fi
        
        if ! build_testrunner; then
            cleanup
            exit 1
        fi
    fi
    
    if [ "$BUILD_MODE" = "run" ] || [ "$BUILD_MODE" = "build" ] || [ "$BUILD_MODE" = "all" ]; then
        # Run phase
        if ! run_service; then
            cleanup
            exit 1
        fi
        
        # Run tests
        if ! run_testrunner; then
            echo "DEBUG: About to stop service"
            stop_service
            
            echo "DEBUG: About to collect logs"
            collect_logs
            
            # Generate reports
            #generate_coverage_report
            generate_report
            
            # Manual cleanup at the end
            cleanup
            
            # Show final status
            echo
            log_error "Some tests failed. Check $RESULTS_DIR/ and $LOGS_DIR/ for detailed logs."
            echo
            exit 1
        fi
        
        echo "DEBUG: About to stop service"
        stop_service
        
        echo "DEBUG: About to collect logs"
        collect_logs
        
        # Generate reports
        #generate_coverage_report
        generate_report
        
        # Manual cleanup at the end
        cleanup
        
        # Show final status
        echo
        log_success "All tests passed! Check $RESULTS_DIR/ for detailed reports."
        log_info "All logs are organized in $LOGS_DIR/"
        echo
        exit 0
    fi
    
    exit 0
}

# Check if we're being sourced or executed
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
