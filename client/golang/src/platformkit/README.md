# Shared Module

This module provides common functionality shared between the service and testrunner modules, with a focus on message bus implementations, logging, configuration helpers, and contextual request metadata.

## Core Packages

### Message Bus (`messagebus/`)
- **Dual Implementation**: 
  - Local development: File-based message bus (`localbus.go`)
  - Production: Kafka-based message bus (`kafkabus.go`)
- **Common Interfaces**: Producer/Consumer patterns in `mbinterfaces.go`
- **Build Tags**: Use `local` tag for development mode
- **Offset Management**: Sequential message processing support
- **Error Handling**: Transient vs. fatal error distinction

### Logging (`logging/`)
- **Structured Logging**: JSON-formatted logs via zerolog
- **Log Levels**: DEBUG, INFO, WARN, ERROR support
- **Context Support**: Trace ID and metadata handling
- **File/Console Output**: Configurable output targets
- **Performance Optimized**: Zero-allocation logging

### Context Utilities (`ctxutil/`)
- **Flow Metadata**: Trace ID, request ID, correlation ID, tenant ID, and user ID helpers
- **Debug Overrides**: Request-scoped debug enablement support
- **Header Mapping**: Standard trace/debug header extraction and propagation

### Utility Packages
- **Configuration**: `configutil/` for YAML/ENV-backed config loading
- **Environment**: `envutil/` for typed environment access
- **Filesystem**: `fsutil/` for common file and directory helpers
- **Time**: `timeutil/` for UTC timestamp helpers

## Usage

### Importing
```go
import (
    "platformkit/configutil"
    "platformkit/ctxutil"
    "platformkit/logging"
    "platformkit/messagebus"
)
```

### Message Bus Example
```go
// Local development (file-based)
producer := messagebus.NewProducer(configMap, "local-producer")
consumer := messagebus.NewConsumer(configMap, "local-consumer")

// Production (Kafka)
producer := messagebus.NewProducer(kafkaConfig, "kafka-producer")
consumer := messagebus.NewConsumer(kafkaConfig, "kafka-consumer")
```

### Logging Example
```go
logger := logging.NewLogger(logging.Config{
    Level:      "info",
    Format:     "json",
    Output:     "service.log",
    ServiceName: "cratos",
})

logger.Info("Processing message", "msgId", msg.ID)
```

## Configuration

### Message Bus Settings
- Local mode uses `/tmp/cratos-messagebus/`
- Production mode requires Kafka configuration:
  ```yaml
  kafka:
    bootstrap.servers: "localhost:9092"
    group.id: "cratos-group"
    client.id: "cratos-client"
    security.protocol: "PLAINTEXT"
  ```

### Logging Configuration
```yaml
logging:
  level: "info"
  format: "json"
  filename: "service.log"
  service_name: "cratos"
```

## Development

### Build Tags
- Use `-tags local` for local development
- Omit tags for production build

### Testing
```bash
# Run all tests
make test

# Generate coverage report
make coverage-html

# Enforce coverage (85% threshold)
make coverage-enforce
```

### Quality Standards
- **Coverage Target**: 85% minimum
- **Documentation**: All exported functions documented
- **Error Handling**: All errors properly handled
- **Thread Safety**: Concurrent access supported
- **Performance**: Zero-allocation where possible

## Directory Structure
```
platformkit/
├── configstore/         # Shared config path constants
├── configutil/          # Config loading helpers
├── ctxutil/             # Request/message context helpers
├── datastore/           # OpenSearch and local datastore helpers
├── envutil/             # Environment variable helpers
├── fsutil/              # Filesystem utilities
├── logging/             # Logging framework
├── messagebus/          # Local and Kafka message bus implementations
└── timeutil/            # UTC time helpers
```
