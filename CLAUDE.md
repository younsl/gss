# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GHES Schedule Scanner (GSS) is a Kubernetes add-on for monitoring and analyzing CI/CD scheduled workflows in GitHub Enterprise Server. It runs as a Kubernetes CronJob that scans repositories for scheduled workflows and publishes results to console or Slack Canvas.

## Development Commands

### Build and Run

```bash
# Run the application locally
go run cmd/ghes-schedule-scanner/main.go

# Build the binary
go build -o ghes-schedule-scanner ./cmd/ghes-schedule-scanner

# Build Docker image
docker build -t ghes-schedule-scanner .

# Generate Helm chart documentation
make docs
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for specific package
go test ./pkg/scanner/...
go test ./pkg/publisher/...
go test ./pkg/reporter/...

# Run tests with coverage
go test -cover ./...
```

### Code Quality

```bash
# Format Go code
go fmt ./...

# Vet Go code
go vet ./...

# Update dependencies
go mod tidy

# Download dependencies
go mod download
```

## Architecture

### Package Structure

- **cmd/ghes-schedule-scanner**: Main application entry point
  - Initializes configuration, scanner, reporter, and publisher
  - Orchestrates the workflow scanning process

- **pkg/config**: Configuration management
  - Loads environment variables and validates configuration
  - Manages defaults and required settings

- **pkg/scanner**: Core scanning logic
  - Interfaces with GitHub API to scan repositories
  - Identifies scheduled workflows and extracts metadata
  - Implements concurrent scanning with goroutines

- **pkg/reporter**: Report generation
  - Processes scan results into structured reports
  - Handles timezone conversions (UTC/KST)

- **pkg/publisher**: Output publishing
  - **console**: Outputs to stdout/logs
  - **slack**: Publishes to Slack Canvas via API
  - Interface-based design for extensibility

- **pkg/connectivity**: Network connectivity checks
  - Validates GitHub Enterprise Server connectivity before scanning

- **pkg/models**: Data structures
  - Defines WorkflowInfo and other shared types

### Key Design Patterns

1. **Interface-based design**: GitHubClient and Publisher interfaces allow for easy testing and extensibility
2. **Concurrent processing**: Uses goroutines with semaphore pattern for parallel repository scanning
3. **Configuration via environment**: All settings loaded from environment variables for Kubernetes compatibility
4. **Modular publishers**: Publisher interface allows adding new output formats without changing core logic

## Environment Variables

Required for local development:
- `GITHUB_TOKEN`: GitHub personal access token
- `GITHUB_ORG`: Target GitHub organization
- `GITHUB_BASE_URL`: GitHub Enterprise Server URL

Optional:
- `LOG_LEVEL`: Logging level (DEBUG, INFO, WARN, ERROR)
- `PUBLISHER_TYPE`: Output type (console, slack-canvas)
- `CONCURRENT_SCANS`: Number of parallel scans (default: 10)

For Slack Canvas publisher:
- `SLACK_TOKEN`: Slack Bot Token (xoxb-*)
- `SLACK_CHANNEL_ID`: Target channel ID
- `SLACK_CANVAS_ID`: Canvas ID to update

## Kubernetes Deployment

The application is deployed as a Helm chart with:
- CronJob for scheduled execution
- ConfigMap for exclude list configuration
- Support for timezone configuration
- TTL for job cleanup

## Testing Considerations

- Tests currently have some failing cases that need fixing
- Mock interfaces are used for GitHub API testing
- Test files follow Go conventions (*_test.go)