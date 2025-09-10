# Parallel Test Execution Guide

## Overview

The testrunner now supports parallel execution of multiple feature files or test suites based on configuration. This enhancement allows for faster test execution while maintaining proper resource management and trace isolation.

## Configuration

Set the following environment variables to enable and configure parallel execution:

### Basic Parallel Configuration
```bash
# Enable parallel execution
export PARALLEL_EXECUTION=true

# Set maximum number of parallel groups (default: 2)
export MAX_PARALLEL_GROUPS=3

# Set concurrency within each group for Godog (default: 1)
export TEST_CONCURRENCY=2
```

### Feature Selection
```bash
# Option 1: Use specific features list
export FEATURES="features/basic_tests.feature,features/one_to_one_tests.feature"

# Option 2: Use feature list file
export FEATURE_LIST_FILE="test_features.txt"

# Option 3: Run single feature
./testrunner features/basic_tests.feature
```

## Execution Modes

### Sequential Execution (Default)
```bash
# Run normally - all features run sequentially
./testrunner

# Or explicitly disable parallel
export PARALLEL_EXECUTION=false
./testrunner
```

### Parallel Execution
```bash
# Enable parallel execution with 3 groups, each running 2 scenarios concurrently
export PARALLEL_EXECUTION=true
export MAX_PARALLEL_GROUPS=3
export TEST_CONCURRENCY=2
./testrunner
```

## How It Works

1. **Feature Grouping**: Features are distributed across the specified number of parallel groups
2. **Isolated Resources**: Each group gets its own:
   - Logger instance (`testrunner_group_N.log`)
   - Test execution report (`test_execution_report_group_N.txt`)
   - Kafka producer/consumer instances (for feature-level resource management)
   - Trace ID context and scenario state

3. **Resource Management**: 
   - Feature-level resources (like `one_to_one_tests.feature`) are properly isolated per group
   - Scenario-level resources (like `basic_tests.feature`) work within each group independently

4. **Report Aggregation**: Individual group reports are merged into a final `test_execution_report.txt`

## Examples

### Example 1: Run 2 features in parallel with 2 groups
```bash
export PARALLEL_EXECUTION=true
export MAX_PARALLEL_GROUPS=2
export FEATURES="features/basic_tests.feature,features/one_to_one_tests.feature"
./testrunner
```
Result: Each feature runs in its own parallel group.

### Example 2: Run 4 features in parallel with 2 groups
```bash
export PARALLEL_EXECUTION=true
export MAX_PARALLEL_GROUPS=2
export FEATURES="features/basic_tests.feature,features/one_to_one_tests.feature,features/advanced_tests.feature,features/integration_tests.feature"
./testrunner
```
Result: Features are distributed - Group 0 gets basic_tests and advanced_tests, Group 1 gets one_to_one_tests and integration_tests.

### Example 3: High concurrency within groups
```bash
export PARALLEL_EXECUTION=true
export MAX_PARALLEL_GROUPS=2
export TEST_CONCURRENCY=4
./testrunner
```
Result: 2 groups run in parallel, each group runs up to 4 scenarios concurrently.

## Logging and Reports

### Log Files
- `testrunner_group_0.log`, `testrunner_group_1.log`, etc. - Individual group logs
- `test_execution_report_group_0.txt`, etc. - Individual group reports
- `test_execution_report.txt` - Final merged report

### Log Structure
```
2024-01-XX Starting feature group    {"group": 0, "features": ["features/basic_tests.feature"]}
2024-01-XX Setting up group feature resources    {"feature": "basic_tests", "group": 0}
2024-01-XX Starting scenario in group    {"scenario": "Basic scenario", "feature": "basic_tests", "group": 0}
```

## Trace IDs in Parallel Execution

Each parallel group maintains its own trace context:
- Group-specific trace IDs: `scenario_name_group_0`, `scenario_name_group_1`, etc.
- Isolated Kafka headers with `X-Trace-Id` per group
- No trace ID conflicts between parallel groups

## Performance Benefits

- **Faster Execution**: Multiple features run simultaneously
- **Better Resource Utilization**: CPU and I/O operations distributed across cores
- **Scalable**: Adjust `MAX_PARALLEL_GROUPS` based on available resources
- **Maintained Isolation**: No interference between parallel test groups

## Best Practices

1. **Group Size**: Start with `MAX_PARALLEL_GROUPS=2` and increase based on system capacity
2. **Concurrency**: Use `TEST_CONCURRENCY=1` initially, increase for CPU-bound tests
3. **Resource Monitoring**: Monitor system resources when running parallel tests
4. **Feature Compatibility**: Ensure features don't have conflicting resource requirements
5. **Debugging**: Use individual group logs for troubleshooting specific failures

## Fallback Behavior

If parallel execution encounters issues:
- Individual group failures are reported but don't stop other groups
- Final exit code reflects any group failures
- All group reports are still merged for complete test coverage analysis
