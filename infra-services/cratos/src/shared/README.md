# Shared Module

This module provides common functionality shared between the service and testrunner modules, with a focus on message bus implementations, logging, and utilities.

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

### Utilities (`utils/`)
- **Configuration**: YAML/ENV config management
- **File Operations**: Safe file handling utilities
- **Error Handling**: Common error types and helpers
- **Testing Utilities**: Test helper functions

### Types (`types/`)
- **Shared Models**: Common data structures
- **Message Formats**: Standard message definitions
- **Type Safety**: Go type checking for messages

## Usage

### Importing
```go
import (
    "sharedgomodule/messagebus"
    "sharedgomodule/logging"
    "sharedgomodule/utils"
    "sharedgomodule/types"
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
shared/
├── messagebus/           # Message bus implementations
│   ├── localbus.go      # Local file-based bus
│   ├── kafkabus.go      # Kafka production bus
│   └── mbinterfaces.go  # Common interfaces
├── logging/             # Logging framework
│   ├── logger.go       # Logger interface
│   └── zerolog.go      # Zerolog implementation
├── utils/              # Common utilities
│   └── utils.go       # Utility functions
└── types/             # Shared data types
    └── types.go      # Common models
```
