#!/bin/bash

# Local Development Test Runner
# Usage: ./run_tests_local.sh [build|run]
# Default mode: build (builds and runs)

set -e
# Note: We handle test failures gracefully to ensure log collection

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
    log_info "Service will be built by make run-local-coverage with coverage instrumentation and local tags..."
    
    # Verify service directory exists and has Makefile
    if [ ! -f "$SERVICE_DIR/Makefile" ]; then
        log_error "Service Makefile not found at $SERVICE_DIR/Makefile"
        exit 1
    fi
    
    log_success "Service build preparation completed (will use make run-local-coverage)"
}

# Function to build testrunner
build_testrunner() {
    log_info "Building testrunner..."
    cd "$TESTRUNNER_DIR"
    
    # Build testrunner
    go build -tags local -o ./../../bin/testrunner.bin cmd/testmain.go
    
    if [ $? -eq 0 ]; then
        log_success "Testrunner built successfully"
    else
        log_error "Testrunner build failed"
        exit 1
    fi
    
    cd - > /dev/null
}

# Function to run service
run_service() {
    log_info "Starting service using make run-local-coverage..."
    cd "$SERVICE_DIR"
    
    # Set coverage directory and log file path (using absolute paths)
    export GOCOVERDIR="$COVERAGE_DIR"
    
        # All other environment variables should be set via .env file
    
    # Run in background and capture stdout/stderr
    ./../../bin/service.bin > "$LOGS_DIR/service_stdouterr.log" 2> "$LOGS_DIR/service_stdouterr.log" &
    COMPONENT_PID=$!
    echo $COMPONENT_PID > "$ROOT_DIR/test/service.pid"
    
    log_success "Service started with PID $COMPONENT_PID"
    log_info "Service logs: $LOGS_DIR/service_stdouterr.log"
    cd - > /dev/null
    
    # Wait a moment for service to start
    sleep 2
}

# Function to run testrunner
run_testrunner() {
    log_info "Running testrunner..."
    cd "$TESTRUNNER_DIR"
    
    # Temporarily disable strict error handling for testrunner execution
    set +e
    # Run testrunner and capture output
    ./../../bin/testrunner.bin > "$LOGS_DIR/testrunner_stdouterr.log" 2> "$LOGS_DIR/testrunner_stdouterr.log"
    TEST_RESULT=$?
    set -e
    
    cd - > /dev/null
    
    if [ $TEST_RESULT -eq 0 ]; then
        log_success "Tests completed successfully"
    else
        log_warning "Tests completed with issues (exit code: $TEST_RESULT)"
    fi
    
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
    
    # Create a consolidated log file with timestamps
    CONSOLIDATED_LOG="$RESULTS_DIR/all_logs.txt"
    
    {
        echo "=================================================="
        echo "Consolidated Log Report"
        echo "Generated: $(date)"
        echo "=================================================="
        echo
        
        echo "=== COMPONENT APPLICATION LOGS ==="
        
        echo "=== COMPONENT STDOUT & STDERR ==="
        if [ -f "$LOGS_DIR/service_stdouterr.log" ]; then
            cat "$LOGS_DIR/service_stdouterr.log"
        else
            echo "No service stdout/stderr logs found"
        fi
        echo
        
        echo "=== TESTRUNNER STDOUT & STDERR ==="
        if [ -f "$LOGS_DIR/testrunner_stdouterr.log.log" ]; then
            cat "$LOGS_DIR/testrunner_stdouterr.log.log"
        else
            echo "No testrunner combined logs found"
        fi
        echo
        
    } > "$CONSOLIDATED_LOG"
    
    log_success "Consolidated logs created at $CONSOLIDATED_LOG"
    
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
    log_info "Generating test report..."
    
    REPORT_FILE="$RESULTS_DIR/executionreport.txt"
    
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
            echo "- Status: FAILED (exit code: $TEST_RESULT)"
        fi
        echo
        
        echo "Coverage Report:"
        if [ -f "$RESULTS_DIR/coverage_summary.txt" ]; then
            cat "$RESULTS_DIR/coverage_summary.txt" | tail -1
        else
            echo "- No coverage data available"
        fi
        echo
        
        echo "Log Files:"
        echo "- Service & Testrunner logs are at: $LOGS_DIR/"
        echo "- Service stdout & stderr logs: $LOGS_DIR/service_stdouterr.log"
        echo "- Testrunner stdout & stderr logs: $LOGS_DIR/testrunner_stdouterr.log.log"
        echo "- Consolidated logs: $RESULTS_DIR/all_logs.txt"
        echo
        
        echo "Output Files:"
        echo "- Coverage report: $RESULTS_DIR/coverage.html"
        echo "- Coverage summary: $RESULTS_DIR/coverage_summary.txt"
        echo "- This report: $REPORT_FILE"
        echo
        
    } > "$REPORT_FILE"
    
    log_success "Test report generated at $REPORT_FILE"
    echo
    log_info "=== QUICK SUMMARY ==="
    cat "$REPORT_FILE" | grep -A 10 "Test Execution:"
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
    
    # Setup trap for cleanup on exit - but not for normal script completion
    trap 'cleanup' INT TERM
    
    # Setup directories
    setup_directories

    # Ensure local bus topic directories exist for tests
    LOCAL_BUS_BASE="/tmp/cratos-messagebus"
    for topic in input-topic output-topic rules-topic; do
        if [ ! -d "$LOCAL_BUS_BASE/$topic" ]; then
            mkdir -p "$LOCAL_BUS_BASE/$topic"
            log_info "Created local bus topic directory: $LOCAL_BUS_BASE/$topic"
        fi
    done

    # Ensure at least one dummy message file exists for input-topic
    #DUMMY_MSG="$LOCAL_BUS_BASE/input-topic/0000000000.json"
    #if [ ! -f "$DUMMY_MSG" ]; then
    #    echo '{"dummy":"message"}' > "$DUMMY_MSG"
    #    log_info "Created dummy message file: $DUMMY_MSG"
    #fi
    
    if [ "$BUILD_MODE" = "build" ] || [ "$BUILD_MODE" = "all" ]; then
        # Build phase
        build_service
        build_testrunner
    fi
    
    if [ "$BUILD_MODE" = "run" ] || [ "$BUILD_MODE" = "build" ] || [ "$BUILD_MODE" = "all" ]; then
        # Run phase
        run_service
        
        # Run tests (handle failures gracefully)
        set +e
        run_testrunner
        TEST_RESULT=$?
        set -e
        
        echo "DEBUG: About to stop service"        # Stop service
        stop_service
        
        echo "DEBUG: About to collect logs"        # Collect and organize logs
        collect_logs
        
        # Generate reports
        generate_coverage_report
        generate_report
        
        # Manual cleanup at the end
        cleanup
        
        # Show final status
        echo
        if [ $TEST_RESULT -eq 0 ]; then
            log_success "All tests passed! Check $RESULTS_DIR/ for detailed reports."
            log_info "All logs are organized in $LOGS_DIR/"
        else
            log_error "Some tests failed. Check $RESULTS_DIR/ and $LOGS_DIR/ for detailed logs."
        fi
        echo
    fi
}

# Check if we're being sourced or executed
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
