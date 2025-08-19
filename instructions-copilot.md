# Copilot Instructions for Go (Golang) Projects

## Overview
When generating Go code, always follow these standards to ensure high quality, maintainability, and compliance with SonarQube static analysis.

## 1. Code Style
- Use `gofmt` formatting (indentation, spacing, braces).
- Use PascalCase for exported functions/types, camelCase for local variables.
- Place package-level variables and constants at the top of files.
- Group imports and order them: standard library, external packages, internal packages.

## 2. Naming Conventions
- Use meaningful, descriptive names for variables, functions, and types.
- Avoid single-letter variable names except for loop indices.
- Prefix test functions with `Test`.

## 3. Error Handling
- Always check and handle returned errors.
- Avoid ignoring errors with `_ = ...` or blank error checks.
- Wrap errors with context using `fmt.Errorf` or `errors.Wrap` (if using `pkg/errors`).

## 4. Security
- Validate all input parameters.
- Avoid hardcoded credentials, secrets, or API keys.
- Sanitize user input to prevent injections.

## 5. Code Quality
- Avoid global state and mutable package-level variables.
- Prefer composition over inheritance.
- Limit function complexity (max 15 LOC per function recommended).
- Avoid magic numbers; use named constants.

## 6. Testing
- Write unit tests for all public functions/types.
- Use table-driven tests where possible.
- Cover edge cases and error conditions.

## 7. Documentation
- Document all exported functions, types, and packages.
- Use GoDoc style comments.

## 8. SonarQube-Specific Guidelines
- Avoid duplicated code.
- Remove unused variables, imports, and functions.
- Do not use deprecated Go packages or functions.
- Ensure functions do not exceed cyclomatic complexity threshold.

## 9. Dependency Management
- Use Go modules (`go.mod`, `go.sum`).
- Prefer semantic versioning for dependencies.

## 10. Miscellaneous
- Prefer explicit type declarations over implicit ones where clarity is needed.
- Do not commit generated files or secrets.
- Avoid using `panic` except for unrecoverable errors in `main` or test code.

---

> **Always review the generated code for compliance before committing.**

