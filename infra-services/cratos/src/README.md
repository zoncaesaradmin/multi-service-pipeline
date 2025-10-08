# Cratos Service System

This project provides a comprehensive Go service system with a message bus-based processing pipeline and integration test infrastructure using Go workspace for multi-module management.

## Project Architecture

The system uses a **message bus architecture** for cross-process communication:
- **Service**: Main processing service with input/output pipeline
- **Testrunner**: Integration test service that communicates via message bus
- **Shared**: Common utilities and message bus implementations

### Message Bus Modes
- **Local Development**: File-based message bus (`//go:build local`)
- **Production**: Kafka-based message bus (configurable)

## Project Structure

```
src/                          # Source code root
├── go.work                   # Go workspace file
├── go.work.sum              # Workspace checksums
├── Makefile                 # Build orchestration with coverage enforcement
├── README.md                # This documentation
│
├── service/               # Main processing service
│   ├── go.mod               # Module: servicegomodule
│   ├── go.sum               # Go module checksums
│   ├── Makefile             # Service build automation
│   ├── cmd/                 # Application entry point
│   │   ├── main.go          # Service main with processing pipeline
│   │   └── main_test.go     # Main function tests
│   ├── internal/            # Private application code
│   │   ├── api/             # HTTP handlers (health, stats)
│   │   │   ├── handlers.go  # API handlers
│   │   │   └── handlers_test.go
│   │   ├── app/             # Application lifecycle
│   │   │   ├── app.go       # App initialization and shutdown
│   │   │   └── app_test.go
│   │   ├── config/          # Configuration management
│   │   │   ├── config.go    # Config structures and parsing
│   │   │   └── config_test.go
│   │   ├── models/          # Data models
│   │   │   ├── models.go    # Message and processing models
│   │   │   └── models_test.go
│   │   └── processing/      # Message processing pipeline
│   │       ├── input.go     # Message bus consumer
│   │       ├── input_test.go
│   │       ├── output.go    # Message bus producer  
│   │       ├── output_test.go
│   │       ├── processing.go # Processing coordinator
│   │       ├── processing_test.go
│   │       ├── processor.go  # Business logic processor
│   │       └── processor_test.go
│   └── bin/                 # Build output
│
├── testrunner/              # Integration test service
│   ├── go.mod               # Module: testrunner
│   ├── go.sum               # Go module checksums
│   ├── Makefile             # Test runner build automation
│   ├── (no config.yaml)     # Configuration moved to conf/testconfig.yaml
│   ├── cmd/                 # Test runner entry point
│   │   └── testmain.go      # Test runner main with message bus
│
└── shared/                  # Shared modules
    ├── go.mod               # Module: sharedgomodule  
    ├── go.sum               # Go module checksums
    ├── Makefile             # Shared module automation
    ├── README.md            # Shared module documentation
    ├── logging/             # Logging utilities
    │   ├── logger.go        # Logger interface
    │   ├── logger_test.go
    │   ├── zerolog.go       # Zerolog implementation
    │   ├── zerolog_test.go
    │   └── README.md
    ├── messagebus/          # Message bus implementations
    │   ├── localbus.go      # File-based message bus (local dev)
    │   ├── kafkabus.go      # Kafka message bus (production) 
    │   ├── mbinterfaces.go  # Message bus interfaces
    │   ├── localbus_test.go
    │   ├── kafkabus_test.go
    │   └── mbinterfaces_test.go
    ├── types/               # Common types
    │   ├── types.go         # Shared data types
    │   └── types_test.go
    └── utils/               # Utility functions
        ├── utils.go         # Common utilities
        └── utils_test.go
```

## Features

### Message Bus Architecture
- **Cross-Process Communication**: File-based message bus for local development
- **Producer/Consumer Pattern**: Testrunner produces test messages, service consumes and processes
- **Topic-Based Messaging**: Organized message flow via `test_input` and `test_output` topics
- **Offset Tracking**: Sequential message processing with offset management
- **Build Tag Support**: Switch between local (`-tags local`) and production message bus implementations

### Main Service
- **Processing Pipeline**: Input → Processor → Output pipeline with message bus integration
- **Configurable Processing**: Customizable processing delays and batch sizes
- **Health Check API**: HTTP endpoint for service monitoring (`/health`)
- **Status API**: Service status endpoint (`/api/v1/status`)
- **Graceful Shutdown**: Clean shutdown with processing pipeline cleanup
- **Structured Logging**: JSON logging via zerolog with configurable levels
- **Coverage Instrumentation**: Built-in coverage tracking for development

### Test Runner
- **Message Bus Integration**: Sends test scenarios via message bus and validates responses
- **YAML Test Scenarios**: Configurable test scenarios with input/expected output
- **Timeout Management**: Configurable test timeouts with proper polling
- **Response Validation**: Flexible validation that checks for processing indicators
- **Comprehensive Reporting**: Detailed test reports with success/failure analytics
- **JSON Fixture Support**: Test data fixtures for complex scenarios

### Shared Modules
- **Message Bus Interfaces**: Common interfaces for producer/consumer patterns
- **Local Development Bus**: File-based message bus for cross-process communication
- **Production Bus Support**: Kafka integration for production environments (with build tags)
- **Logging Framework**: Structured logging with multiple backend support
- **Common Types**: Shared data structures across all modules
- **Utilities**: Common helper functions and utilities

## Quick Start

### Prerequisites
- Go 1.22.5 or higher
- Make

### Running from Root Directory

The recommended approach is to use the test infrastructure from the root directory:

```bash
# Navigate to root (parallel to src/)
cd ..

# Build and run integration tests (primary workflow)
make test

# Quick test run with existing binaries
make test-run

# View test help
make test-help
```

### Running Services Individually

#### 1. Build and Run Service (Processing Pipeline)
```bash
# From src/ directory
cd service
make build BUILD_TAGS="local"
./bin/service
```

The service will start:
- Processing pipeline listening on message bus
- Health API on `localhost:4477`
- Processing `test_input` messages and sending responses to `test_output`

#### 2. Run Test Runner (Message Bus Tests)
```bash
# In another terminal, from src/ directory  
cd testrunner
make build BUILD_TAGS="local"
./bin/testrunner
```

The testrunner will:
- Send test scenarios via message bus
- Wait for processed responses
- Validate and report results

### Message Bus Development

The system uses a file-based message bus for local development:

```bash
# Check message bus activity
ls -la /tmp/cratos-messagebus/

# View input messages
cat /tmp/cratos-messagebus/test_input/0000000000.json

# View output responses (after processing)
cat /tmp/cratos-messagebus/test_output/0000000000.json | jq .

# Clean message bus for fresh test
rm -rf /tmp/cratos-messagebus
```

### Using Go Workspace

This project uses Go 1.18+ workspace feature:
```bash
go work sync    # Sync workspace modules
go work use ./service ./testrunner ./shared  # Add modules to workspace
```

## Message Bus Architecture

### File-Based Message Bus

For local development, the system uses a file-based message bus implementation:

- **Bus Location**: `/tmp/cratos-messagebus/`
- **Topic Structure**: Each topic is a directory containing JSON message files
- **Message Format**: Sequential numbered files (0000000000.json, 0000000001.json, etc.)
- **Cross-Process Communication**: Service processes messages, testrunner validates responses

### Message Format

All messages follow the shared JSON structure:

```json
{
  "id": "unique-message-id",
  "timestamp": "2024-01-01T00:00:00Z",
  "type": "TEST_USER_CREATION",
  "payload": {
    "name": "John Doe",
    "email": "john@example.com"
  }
}
```

### Processing Pipeline

1. **Input Handler**: Receives messages from `test_input` topic
2. **Processor**: Transforms input according to message type
3. **Output Handler**: Sends processed messages to `test_output` topic

### Development vs Production

The codebase supports build tags for different environments:

```bash
# Local development (file-based message bus)
go build -tags local

# Production (would use Kafka or other message brokers)
go build
```

### Health Endpoints

The service still provides HTTP endpoints for monitoring:

- **GET** `/health` - Service health status
- **GET** `/api/v1/status` - Processing statistics

## Configuration

The service and testrunner can be configured using environment variables and config files:

### Service Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| SERVER_HOST | localhost | Health check server host |
| SERVER_PORT | 4477 | Health check server port |
| LOG_LEVEL | info | Log level (debug, info, warn, error) |
| LOG_FORMAT | json | Log format (json, text) |
| PROCESSING_DELAY | 100ms | Processing delay for each message |
| PROCESSING_BATCH_SIZE | 10 | Batch size for processing |

### Testrunner Configuration

Configuration is managed via `conf/testconfig.yaml` (automatically loaded using SERVICE_HOME):

```yaml
# Message bus configuration
messagebus:
  timeout: "30s"
  polling_interval: "100ms"

# Test scenarios
scenarios:
  - name: "user_workflow"
    file: "testdata/scenarios/user_workflow.yaml"

# Validation settings
validation:
  timeout: "10s"
  strict_mode: false
```

### Message Bus Configuration

Local development uses file-based message bus at `/tmp/cratos-messagebus/`.
Production configuration would specify Kafka brokers and topics.

## Testing

The project has comprehensive test infrastructure in the parallel `test/` directory. Refer to `test/LOCAL_DEVELOPMENT.md` for detailed testing information.

### Quick Testing (Recommended)

Run tests from the root directory using the comprehensive test infrastructure:

```bash
# Build and run integration tests
make test

# Quick test with existing binaries  
make test-run

# View test options
make test-help
```

### Service Unit Tests

```bash
cd service
make test
```

### Testrunner Unit Tests

```bash
cd testrunner  
make test
```

### Shared Module Tests

```bash
cd shared
make test
```

### Test Coverage

```bash
# Service coverage
cd service
make test-coverage

# Testrunner coverage  
cd testrunner
make test-coverage

# Combined coverage (via integration tests)
make test  # Coverage reported in test/results/
```

### Message Bus Testing

Test the message bus functionality directly:

```bash
# Start service
cd service && make run BUILD_TAGS="local"

# In another terminal, send test messages
cd testrunner && make run BUILD_TAGS="local"

# Monitor message bus
ls -la /tmp/cratos-messagebus/
```

## Cleanup Commands

### Quick Clean (removes build artifacts)
```bash
# Clean all services from root
make clean

# Clean individual services
cd service && make clean
cd testrunner && make clean  
cd shared && make clean
```

### Deep Clean (removes build artifacts, vendor, and module cache)
```bash
# Deep clean all services from root
make deep-clean

# Deep clean individual services
cd service && make deep-clean
cd testrunner && make deep-clean
cd shared && make deep-clean
```

### Message Bus Clean

```bash
# Clean message bus data (fresh start)
rm -rf /tmp/cratos-messagebus
```

The clean commands remove:
- `bin/` directories and binaries
- `coverage.out` and `coverage.html` files
- `*.log` files
- `tmp/` and `temp/` directories
- `*.test` and `*.out` files

The deep-clean commands additionally remove:
- `vendor/` directories
- Go module cache

### Test Results Clean

```bash
# Clean test outputs (from root directory)
rm -rf test/results/
```

## Build Commands

### Service (Processing Pipeline)

```bash
cd service

# Build for local development (file-based message bus)
make build BUILD_TAGS="local"

# Build for production (would use Kafka message bus)
make build

# Run locally
make run BUILD_TAGS="local"

# Clean build artifacts
make clean
```

### Testrunner (Integration Tests)

```bash
cd testrunner

# Build for local development  
make build BUILD_TAGS="local"

# Run integration tests
make run BUILD_TAGS="local"

# Clean build artifacts
make clean
```

### Shared Modules

```bash
cd shared

# Run tests for shared modules
make test

# Clean artifacts
make clean
```

### Workspace-Level Commands

```bash
# From root directory - builds all services
make build

# Clean all modules
make clean

# Run comprehensive integration tests  
make test
```

### Root Level Commands (Recommended)

From the root directory (parallel to src/), use the comprehensive test infrastructure:

```bash
make help           # Show all available targets  
make test           # Build and run integration tests (recommended workflow)
make test-run       # Quick test with existing binaries
make test-help      # Show test-specific options
make test-clean     # Clean test results and artifacts
```

### Service Commands
```bash
cd service
make help           # Show available targets
make build          # Build with production settings
make build BUILD_TAGS="local"  # Build for local development
make run BUILD_TAGS="local"    # Run service with file-based message bus
make test           # Run unit tests
make test-coverage  # Run tests with coverage
make clean          # Clean build artifacts
```

### Testrunner Commands
```bash
cd testrunner  
make help           # Show available targets
make build BUILD_TAGS="local"  # Build for local development
make run BUILD_TAGS="local"    # Run integration tests via message bus
make test           # Run unit tests
make clean          # Clean build artifacts
```

### Shared Module Commands
```bash
cd shared
make help           # Show available targets
make test           # Run shared module tests
make clean          # Clean build artifacts
```

## Development

### Prerequisites
- Go 1.22.5 or higher
- Make

### Project Architecture
- **Multi-module workspace**: Uses Go 1.18+ workspace feature with service, testrunner, and shared modules
- **Message Bus Communication**: File-based message bus for local development with producer/consumer pattern
- **Processing Pipeline**: Input → Processor → Output pipeline architecture
- **Build Tag Support**: Local development vs production message bus implementations
- **Clean separation**: Service and testrunner communicate via message bus, completely independent processes

### Adding New Features

#### Service Features
1. Add models in `service/internal/models/`
2. Implement processing logic in `service/internal/processing/processor.go`
3. Update message types in `shared/types/types.go`
4. Add unit tests for new processing logic
5. Add integration tests in `testrunner/internal/tests/`

#### Message Bus Features
1. Add new interfaces in `shared/messagebus/mbinterfaces.go`
2. Implement in `shared/messagebus/localbus.go` for local development
3. Implement in `shared/messagebus/kafkabus.go` for production
4. Update build tags as needed

### Module Management
```bash
# Working with workspace
go work sync                 # Sync all modules in workspace
go work use ./service      # Add service module
go work use ./testrunner     # Add testrunner module
go work use ./shared         # Add shared module

# Working with individual modules
cd service && go mod tidy  # Manage service dependencies
cd testrunner && go mod tidy # Manage testrunner dependencies
cd shared && go mod tidy     # Manage shared dependencies
```

### Testing Strategy
1. **Unit Tests**: Test individual functions in each module (service, testrunner, shared)
2. **Integration Tests**: Test cross-process communication via message bus
3. **Message Bus Tests**: Test producer/consumer patterns and offset management
4. **Processing Pipeline Tests**: End-to-end processing validation
5. **Cross-Process Tests**: Service and testrunner interaction via file-based message bus

## Examples

### Basic Message Bus Workflow

1. **Start the service**:
```bash
cd service
make build BUILD_TAGS="local"
./bin/service
```

2. **Send test messages via testrunner**:
```bash
cd testrunner
make build BUILD_TAGS="local"  
./bin/testrunner
```

3. **Monitor message bus activity**:
```bash
# Check message files
ls -la /tmp/cratos-messagebus/test_input/
ls -la /tmp/cratos-messagebus/test_output/

# View message content
cat /tmp/cratos-messagebus/test_input/0000000000.json | jq .
cat /tmp/cratos-messagebus/test_output/0000000000.json | jq .
```

### Custom Test Scenarios

Create custom test scenarios in `testrunner/testdata/scenarios/`:

```yaml
# custom_test.yaml
name: "Custom User Test"
description: "Test custom user processing"
input:
  type: "TEST_USER_CREATION"
  payload:
    name: "Custom User"
    email: "custom@example.com"
expected_output:
  type: "TEST_USER_CREATION_RESULT" 
  payload:
    status: "processed"
    processed_name: "Custom User"
    processed_email: "custom@example.com"
```

### Health Check (Service Monitoring)
```bash
curl http://localhost:4477/health
```

## Go Workspace Benefits

This project uses Go 1.18+ workspace feature which provides:

- **Multi-module management**: Work with service, testrunner, and shared modules in a single workspace
- **Cross-module development**: Make changes across service, testrunner, and shared modules simultaneously
- **Shared dependencies**: Efficient dependency management across modules
- **IDE support**: Better code navigation and refactoring across modules

### Workspace Commands
```bash
go work init                 # Initialize workspace
go work use ./service      # Add service module to workspace
go work use ./testrunner     # Add testrunner module to workspace
go work use ./shared         # Add shared module to workspace
go work sync                 # Sync workspace modules
go work edit                 # Edit go.work file
```

## License

MIT License - see LICENSE file for details.
