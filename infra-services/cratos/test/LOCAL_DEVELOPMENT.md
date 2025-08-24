# Local Development Guide

This guide covers the local development setup for the Cratos project, including building, testing, and generating coverage reports with comprehensive log management.

## Quick Start

### One-Step Build and Test
```bash
# From the root directory (cratos/)
make test

# Or from the test directory
cd test/
./run_tests_local.sh
```

### Available Commands

#### From Root Directory
```bash
# Primary workflow - build and test
make test

# Quick test run (with existing binaries)  
make test-run

# Build services only
make test-build

# Generate coverage report
make test-coverage

# Clean test artifacts
make test-clean

# Setup test environment
make test-setup

# Show test help
make test-help
```

#### Direct Script Usage
```bash
# Navigate to test directory first
cd test/

# Build and run (default)
./run_tests_local.sh build

# Run only (assumes binaries exist)
./run_tests_local.sh run
```

## Test Infrastructure Architecture

The test infrastructure has been moved to the `test/` directory parallel to `src/` for better organization:

```
cratos/
├── src/                    # Source code
│   ├── service/          # Main service source
│   ├── testrunner/         # Test runner source
│   └── shared/             # Shared modules
├── test/                   # Test infrastructure (NEW LOCATION)
│   ├── run_tests_local.sh  # Main test script
│   ├── results/            # Test results and reports
│   │   ├── logs/          # Service and testrunner logs
│   │   ├── coverage.html  # Coverage report
│   │   └── report.txt
│   ├── coverage/          # Coverage binary data
│   └── LOCAL_DEVELOPMENT.md # This file
└── Makefile               # Root makefile with test targets
```

## Message Bus Architecture

The local development environment uses a **file-based message bus** for cross-process communication:

### File-Based Message Bus Features
- **Location**: `/tmp/cratos-messagebus/`
- **Topics**: Organized in subdirectories (`test_input/`, `test_output/`)
- **Message Format**: JSON files with sequential offsets (`0000000000.json`, `0000000001.json`)
- **Cross-Process**: Enables testrunner and service to communicate via filesystem
- **Local Development**: Uses `//go:build local` tag for local-only implementation

### Message Flow
1. **Testrunner** sends test data to `test_input` topic
2. **Service** consumes from `test_input`, processes the data
3. **Service** sends responses to `test_output` topic  
4. **Testrunner** polls `test_output` for responses and validates results

## Output Files and Logs

All test results, logs, and reports are generated in the `test/results/` directory:

### Main Reports
- `report.txt` - Comprehensive report of the script execution
- `coverage.html` - Interactive HTML coverage report
- `coverage.out` - Raw coverage data
- `coverage_summary.txt` - Coverage statistics

### Log Files (organized in `test/results/logs/`)
- `service.log` - Service application logs (structured JSON)
- `service_stdout.log` - Service standard output
- `service_stderr.log` - Service standard error output
- `testrunner_stdout.log` - Testrunner standard output
- `testrunner_stderr.log` - Testrunner standard error output

### Consolidated Logs
- `all_logs.txt` - All logs consolidated in one file with sections
- `testrunner_output.log` - Combined testrunner output (legacy format)

## Current Test Status

✅ **All Tests Passing**: 1/1 scenarios (100% success rate)
✅ **Message Bus Working**: File-based cross-process communication functional  
✅ **Coverage**: ~51.4% baseline coverage with proper instrumentation
✅ **Processing Pipeline**: Service properly processes and responds to test data

### Test Scenarios
Currently includes:
- **user_workflow**: Tests complete user creation and retrieval workflow
- **Message Bus Validation**: Verifies processing pipeline responds with valid data
- **Cross-Process Communication**: Confirms testrunner ↔ service message exchange

## Logging Configuration

The local development setup automatically configures logging for both services:

### Service Logging
- **Structured Logging**: Uses JSON format via zerolog library
- **Configurable Path**: Set via `LOG_FILE_NAME` environment variable
- **Path**: `test/results/logs/service.log` (automatically set)
- **Level Control**: Configurable via `LOG_LEVEL` environment variable

### Testrunner Logging  
- **Standard Logging**: Uses Go's standard log package
- **Output Capture**: Both stdout and stderr are captured separately
- **Organized Storage**: All logs stored in `test/results/logs/` directory

### Environment Variables for Log Control
```bash
# Service logging (automatically set by run_tests_local.sh)
export LOG_FILE_NAME="../../test/results/logs/service.log"
export GOCOVERDIR="../../test/coverage"

# Manual override if needed
export LOG_LEVEL="info"  # debug, info, warn, error
```

## Coverage Reports

The system automatically instruments binaries with coverage tracking and generates:

1. **HTML Report**: Open `test/results/coverage.html` in your browser for interactive coverage exploration
2. **Summary Report**: View `test/results/coverage_summary.txt` for quick coverage statistics  
3. **Console Output**: Coverage percentage is displayed after test completion

### Coverage Scope
- ✅ **service/**: Main business logic (included)
- ✅ **shared/**: Common utilities (included)  
- ❌ **testrunner/**: Testing infrastructure (excluded from coverage)

## Build Tags

The local development setup uses `-tags local` to:
- Use file-based message bus implementation (instead of Kafka)
- Enable local-specific configurations
- Avoid external dependencies in development

## Process Management

The test runner automatically:
- Builds service and testrunner with appropriate tags
- Starts the service with coverage instrumentation and log configuration
- Runs the test suite with organized log capture
- Manages cross-process message bus communication
- Stops all processes on completion
- Organizes and consolidates all logs
- Cleans up temporary files and message bus data

## Requirements

### Required
- Go 1.22.5 or later
- Make utility

### Optional
- `jq` (for JSON processing)

### Installation on macOS
```bash
# Optional dependencies
brew install jq
```

## Development Workflow

### Full Test Cycle
```bash
# From root directory
make test
```

This will:
1. Build service with coverage instrumentation
2. Build testrunner
3. Start service with message bus
4. Run test scenarios via message bus
5. Generate coverage and test reports
6. Clean up processes and organize logs

### Quick Iteration
```bash
# For faster development cycles
make test-run
```

This runs tests with existing binaries (skips build phase).

### Coverage Analysis
```bash
# Generate coverage report from latest test run
make test-coverage

# Then open the HTML report
open test/results/coverage.html
```

## Message Bus Development

### Manual Message Bus Inspection
```bash
# Check message bus state
ls -la /tmp/cratos-messagebus/

# View input messages
cat /tmp/cratos-messagebus/test_input/0000000000.json

# View output responses
cat /tmp/cratos-messagebus/test_output/0000000000.json | jq .
```

### Message Bus Cleanup
```bash
# Clean message bus for fresh test
rm -rf /tmp/cratos-messagebus
```

### Debug Message Flow
The logs show detailed message bus activity:
```bash
# Check testrunner message bus activity
grep "\[MessageBus\]" test/results/logs/testrunner_stderr.log

# Check service processing logs
grep "processing" test/results/logs/service.log
```

## Log Analysis

### Quick Log Review
```bash
# View all logs consolidated
cat test/results/all_logs.txt

# View specific service logs
cat test/results/logs/service.log

# View test execution output
cat test/results/logs/testrunner_stdout.log

# Check for errors across all logs
grep -i error test/results/logs/*.log
```

### Message Bus Debug Logs
The current implementation includes extensive debug logging:
```bash
# Message sending/receiving
grep "Sent message\|Consumed message" test/results/logs/testrunner_stderr.log

# Polling activity
grep "Polling\|Looking for message" test/results/logs/testrunner_stderr.log

# Validation results
grep "Validation" test/results/logs/testrunner_stderr.log
```

## Troubleshooting

### Build Issues
1. Ensure Go modules are properly initialized
2. Check for import cycle issues
3. Verify all dependencies are available
4. Check build tags are correctly applied

### Test Failures
1. Check `test/results/logs/testrunner_stdout.log` for detailed error messages
2. Review `test/results/logs/service.log` for service-side issues
3. Verify service starts successfully in `test/results/logs/service_stdout.log`
4. Check port conflicts (service uses localhost:4477)

### Message Bus Issues
1. Check message bus directory: `ls -la /tmp/cratos-messagebus/`
2. Verify message format: `cat /tmp/cratos-messagebus/test_*/000*.json | jq .`
3. Clean and retry: `rm -rf /tmp/cratos-messagebus && make test`
4. Check process communication in logs

### Coverage Issues
1. Ensure `GOCOVERDIR` environment is set correctly
2. Check that binaries are built with `-cover` flag
3. Verify coverage directory permissions
4. Run coverage generation from service directory context

### Stuck Processes
```bash
# Find stuck service processes
ps aux | grep service

# Kill stuck processes
pkill -f service
```

## Current Architecture Status

### ✅ **Completed**
- File-based message bus for cross-process communication
- Processing pipeline with input/output handlers
- Testrunner with proper message bus integration
- Coverage instrumentation and reporting
- Comprehensive logging and log organization
- Test infrastructure moved to dedicated `test/` directory

### 🔄 **In Development**  
- Enhanced business logic in processor.go
- Additional test scenarios
- Performance optimization

### 🎯 **Next Steps**
1. Expand processor logic for specific business requirements
2. Add more comprehensive test scenarios
3. Production message bus integration (Kafka/Redis)
4. Performance and load testing

## Integration with IDE

The local development setup integrates with VS Code through:
- Build tasks for compilation
- Coverage reports viewable in browser
- Terminal integration for script execution
- Problem detection through build output
- Log files can be opened directly in editor

## Performance Notes

- Initial builds may take longer due to coverage instrumentation
- Message bus polling introduces small latency (~100ms intervals)
- Coverage collection adds minimal runtime overhead
- Log file I/O is optimized for development use
- File-based message bus is suitable for development but not production

## Best Practices

1. **Always run full test suite before committing changes**
2. **Monitor coverage reports to maintain test quality**
3. **Use quick test runs for rapid iteration**
4. **Clean test artifacts regularly with `make test-clean`**
5. **Review message bus logs for communication patterns**
6. **Check consolidated logs for issue patterns**
7. **Use structured logging data for debugging**
8. **Clean message bus data between test runs if debugging issues**

## Log Management Tips

1. **Regular Cleanup**: Use `make test-clean` to remove old log files
2. **Log Analysis**: Use grep, awk, or jq for structured log analysis
3. **Error Tracking**: Monitor error patterns across test runs
4. **Message Bus Debugging**: Check /tmp/cratos-messagebus/ for message flow
5. **Performance Monitoring**: Track service startup and response times in logs
6. **Debug Information**: Enable verbose logging for detailed troubleshooting

## Message Bus Implementation Details

### Local vs Production
- **Local**: File-based message bus in `/tmp/cratos-messagebus/`
- **Production**: Kafka-based message bus (not implemented in local mode)
- **Switching**: Controlled by build tags (`-tags local`)

### Message Format
```json
{
  "topic": "test_input",
  "value": "base64-encoded-payload",
  "timestamp": "2025-08-11T16:38:56.402274+05:30",
  "partition": 0,
  "offset": 0
}
```

### Polling Mechanism
- **Polling Interval**: 100ms
- **Timeout**: 30 seconds (configurable in test scenarios)
- **Offset Tracking**: Per-topic offset management
- **Cross-Process**: File-based coordination between processes
