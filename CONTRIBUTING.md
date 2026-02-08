# Contributing to accio

Thank you for your interest in contributing to accio! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/jvendramin/accio/issues)
2. If not, create a new issue with:
   - Clear title and description
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (OS, Go version)

### Suggesting Features

1. Check existing issues for similar suggestions
2. Create a new issue with the "enhancement" label
3. Describe the feature and its use case

### Pull Requests

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run tests: `go test ./...`
5. Run linter: `golangci-lint run`
6. Commit with clear messages
7. Push and create a Pull Request

## Development Setup

### Prerequisites

- Go 1.22 or later
- golangci-lint (optional, for linting)

### Build

```bash
git clone https://github.com/jvendramin/accio.git
cd accio
go build ./cmd/accio
```

### Run Tests

```bash
go test ./...
```

### Run with Debug Logging

```bash
ACCIO_LOG_LEVEL=debug ./accio
```

## Project Structure

```
accio/
├── cmd/accio/          # Entry point
├── internal/
│   ├── application/    # Use cases / services
│   ├── config/         # Configuration management
│   ├── domain/         # Domain entities and interfaces
│   ├── infrastructure/ # External integrations (AWS, storage)
│   └── tui/            # Terminal UI components
└── pkg/
    ├── plugin/         # Plugin interfaces
    └── provider/       # Cloud provider interfaces
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write tests for new functionality
- Add comments for exported functions
- Keep functions focused and small

## Commit Messages

- Use present tense ("Add feature" not "Added feature")
- Use imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit first line to 72 characters
- Reference issues when relevant

## Questions?

Feel free to open an issue for any questions about contributing.
