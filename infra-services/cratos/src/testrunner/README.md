# Infra Testrunner (BDD)

## Directory Structure

```
infra-services/cratos/src/testrunner/
├── features/         # Gherkin scenario files
├── steps/            # Step definitions (common, rules, etc.)
├── testdata/         # Input/output data files for scenarios
├── testrunner.go     # Main BDD runner
├── reportserver/     # HTTP server for report retrieval
```

## How to Run

### 1. Install dependencies
```
go mod init testrunner
# Add godog dependency
go get github.com/cucumber/godog@latest
```

### 2. Run the BDD testrunner
```
go run testrunner.go | tee testrunner_report.txt
# For JSON report, use:
go run testrunner.go -format cucumber > testrunner_report.json
```

### 3. Start the report HTTP server
```
cd reportserver
# Serve both text and JSON reports
# (Make sure testrunner_report.txt and testrunner_report.json exist in parent dir)
go run report_server.go
```
- Access text report: http://localhost:8080/report/text
- Access JSON report: http://localhost:8080/report/json

## Advanced Step Implementations

- **Kafka Integration**: Replace simulated Kafka send/receive with real producer/consumer logic using [segmentio/kafka-go](https://github.com/segmentio/kafka-go) or [Shopify/sarama](https://github.com/Shopify/sarama).
- **Timeouts & Retries**: Use Go's `context` and `time` packages to implement step timeouts and retry logic for message receipt.
- **Field Validation**: Unmarshal received data (JSON/Protobuf) and compare fields using Go's `encoding/json` or `proto` libraries.
- **Reusable Background Steps**: Use `Background:` in feature files for setup steps (e.g., start Kafka, load configs).
- **Parameterized Scenarios**: Use scenario outlines in Gherkin for data-driven tests.
- **Test Data Management**: Organize testdata/ with subfolders per scenario or test type for scalability.
- **CI/CD Integration**: Use the JSON report for automated result collection and dashboarding.
- **Hooks**: Add `BeforeScenario`/`AfterScenario` hooks in Godog for setup/teardown.

## Example: Advanced Kafka Step
```go
import (
    "github.com/segmentio/kafka-go"
    "context"
    "time"
)

func sendKafkaMessage(topic, dataFile string) error {
    // Read data from file
    // Create Kafka writer
    // Write message
    // Handle errors
    return nil
}
```

## Example: Field Validation Step
```go
import "encoding/json"

func verifyFieldEquals(field, expected string) error {
    var msg map[string]interface{}
    json.Unmarshal(receivedBytes, &msg)
    if msg[field] != expected {
        return fmt.Errorf("field %s mismatch: got %v, want %v", field, msg[field], expected)
    }
    return nil
}
```

---
For more advanced needs, ask for specific step code or integration samples!
