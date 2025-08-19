# Shared Module

This module contains common utilities and packages shared between the service and testrunner modules.

## Packages

- `utils/` - Common utility functions
- `types/` - Shared data types and structures
- `logging/` - Comprehensive logging functionality

## Coverage Target

- **Target**: 85% coverage
- **Enforcement**: Enabled via Makefile targets
- **Reporting**: HTML and terminal output available

## Usage

Import from other modules in the workspace:
```go
import "sharedgomodule/utils"
import "sharedgomodule/types"
import "sharedgomodule/logging"
```
