# Cratos Test Runner

## Overview

A comprehensive testing framework for validating the Cratos processing pipeline through message bus integration, supporting both local development (file-based) and production (Kafka) environments.

## Directory Structure

```
testrunner/
├── cmd/                    # Application entry point
│   └── testmain.go        # Main test runner
├── impl/                  # Core implementation
│   ├── comm.go           # Message bus communication
│   ├── context.go        # Test context management
│   └── handler.go        # Message handlers
├── steps/                # Test step definitions
│   ├── common_steps.go   # Common test steps
│   └── handler_steps.go  # Handler-specific steps
├── testdata/            # Test resources
│   ├── scenarios/       # Test scenarios
│   └── configs/        # Test configurationstrunner (BDD)

## Directory Structure

```
infra-services/cratos/src/testrunner/
├── features/         # Gherkin scenario files
├── steps/            # Step definitions (common, rules, etc.)
├── testdata/         # Input/output data files for scenarios
├── testrunner.go     # Main BDD runner
├── reportserver/     # HTTP server for report retrieval
```

### Features

### Test Infrastructure
- **Message Bus Integration**: 
  - Local: File-based at `/tmp/cratos-messagebus/`
  - Production: Kafka-based with configurable brokers
- **Test Scenarios**: 
  - Rules testing
  - Message processing validation
  - Performance benchmarking
- **Environment Support**:
  - Development mode with file-based messaging
  - Production mode with Kafka integration
  - Configurable via environment variables

### Test Capabilities
- **Integration Testing**: End-to-end validation
- **Performance Testing**: Throughput and latency
- **Error Handling**: Edge cases and recovery
- **Parallel Execution**: Concurrent test groups

## Quick Start

### Local Development
```bash
# Build with local message bus
make build-local

# Run test suite
make run

# Run specific tests
make run-integration-tests
make run-performance-tests
```
```
### Environment Configuration
```bash
# Required
export SERVICE_HOME="/path/to/repo/root"

# Optional topic overrides
export PROCESSING_RULES_TOPIC="cisco_nir-alertRules"
export PROCESSING_INPUT_TOPIC="cisco_nir-anomalies"
export PROCESSING_OUTPUT_TOPIC="cisco_nir-prealerts"
export RULE_TASKS_TOPIC="cisco_nir-ruletasks"
```

### Test Configuration
Test scenarios are defined in YAML:
```yaml
# testdata/scenarios/rule_test.yaml
name: "Rule Processing Test"
description: "Validates rule engine processing"
input:
  topic: "test_input"
  messages:
    - type: "RULE_UPDATE"
      payload:
        ruleId: "test-rule-1"
        condition: "severity > 5"
expected:
  topic: "test_output"
  validation:
    - field: "status"
      value: "processed"
    - field: "ruleId"
      value: "test-rule-1"
```

## Implementation Details

- **Kafka Integration**: Replace simulated Kafka send/receive with real producer/consumer logic using [segmentio/kafka-go](https://github.com/segmentio/kafka-go) or [Shopify/sarama](https://github.com/Shopify/sarama).
- **Timeouts & Retries**: Use Go's `context` and `time` packages to implement step timeouts and retry logic for message receipt.
- **Field Validation**: Unmarshal received data (JSON/Protobuf) and compare fields using Go's `encoding/json` or `proto` libraries.
- **Reusable Background Steps**: Use `Background:` in feature files for setup steps (e.g., start Kafka, load configs).
- **Parameterized Scenarios**: Use scenario outlines in Gherkin for data-driven tests.
- **Test Data Management**: Organize testdata/ with subfolders per scenario or test type for scalability.
- **CI/CD Integration**: Use the JSON report for automated result collection and dashboarding.
- **Hooks**: Add `BeforeScenario`/`AfterScenario` hooks in Godog for setup/teardown.

### Message Bus Integration
```go
// Producer setup
producer := messagebus.NewProducer(config, "test-producer")
defer producer.Close()

// Consumer setup
consumer := messagebus.NewConsumer(config, "test-group")
defer consumer.Close()

// Message handling
consumer.Subscribe(topics, func(msg Message) {
    // Process message
})
```

### Test Implementation
```go
type TestContext struct {
    Producer     messagebus.Producer
    Consumer     messagebus.Consumer
    ScenarioData map[string]interface{}
}

func (tc *TestContext) SendTestMessage(msg TestMessage) error {
    return tc.Producer.SendMessage(context.Background(), msg)
}

func (tc *TestContext) WaitForResponse(timeout time.Duration) (*Response, error) {
    select {
    case resp := <-tc.ResponseChan:
        return resp, nil
    case <-time.After(timeout):
        return nil, ErrTimeout
    }
}
```

## Build and Run

### Make Targets
```bash
make help           # Show available commands
make build         # Build test runner (production)
make build-local   # Build for local development
make test          # Run unit tests
make run           # Run all tests
make clean         # Clean artifacts
```

### Test Execution
```bash
# Run all tests
./bin/testrunner

# Run specific scenario
./bin/testrunner -scenario rule_processing

# Run performance tests
./bin/testrunner -mode performance
```

## Message Bus Testing

### Local Development
```bash
# Check message bus
ls -la /tmp/cratos-messagebus/

# View test messages
cat /tmp/cratos-messagebus/test_input/*.json

# Monitor responses
tail -f /tmp/cratos-messagebus/test_output/*.json
```

### Production Testing
```bash
# Configure Kafka
export KAFKA_BROKERS="localhost:9092"
export KAFKA_GROUP="test-group"

# Run tests
./bin/testrunner -env production
```
