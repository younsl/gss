# GHES Schedule Scanner (GSS)

[![Rust Version](https://img.shields.io/badge/rust-1.90+-000000?style=flat-square&logo=rust&logoColor=white)](https://www.rust-lang.org/)
[![GitHub release](https://img.shields.io/github/v/release/younsl/gss?style=flat-square&color=black&logo=github&logoColor=white&label=release)](https://github.com/younsl/gss/releases)
[![Container Image](https://img.shields.io/badge/ghcr.io-container%20image-blue?style=flat-square&logo=docker&logoColor=white&color=black)](https://github.com/younsl/gss/pkgs/container/gss)
[![Helm Chart](https://img.shields.io/badge/ghcr.io-helm%20chart-blue?style=flat-square&logo=helm&logoColor=white&color=black)](https://github.com/younsl/gss/pkgs/container/charts%2Fghes-schedule-scanner)
[![License](https://img.shields.io/github/license/younsl/gss?style=flat-square&color=black&logo=github&logoColor=white)](https://github.com/younsl/gss/blob/main/LICENSE)

> _GSS stands for GHES(GitHub Enterprise Server) Schedule Scanner._

GSS is a high-performance Kubernetes add-on for DevOps and SRE teams to monitor and analyze CI/CD workflows in [GitHub Enterprise Server](https://docs.github.com/ko/enterprise-server/admin/all-releases). Written in Rust, GSS runs as a kubernetes [cronJob](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) that scans and analyzes scheduled workflows across your GHES environment.

![System Architecture](./docs/assets/images/1.png)

## Overview

GHES Schedule Scanner runs as a kubernetes cronJob that periodically scans GitHub Enterprise Server repositories for scheduled workflows. It collects information about:

- Workflow names and schedules
- Last execution status
- Last committer details
- Repository information

The scanner is designed for high performance with async/concurrent scanning capabilities and provides timezone conversion between UTC and KST for better schedule visibility.

## Features

- **GitHub Enterprise Server Integration**: Compatible with self-hosted [GitHub Enterprise Server (3.11+)](https://docs.github.com/ko/enterprise-server/admin/all-releases)
- **Organization-wide Scanning**: Scan scheduled workflows across all repositories in an organization
- **Timezone Support**: UTC/KST timezone conversion for better schedule visibility
- **Status Monitoring**: Track workflow execution status and identify failed workflows
- **High Performance**: Async concurrent scanning (scans 900+ repositories in about 15-18 seconds)
- **Multiple Publishers**: Publish results to console or Slack Canvas
- **Kubernetes Native**: Runs as a Kubernetes cronJob for periodic scanning
- **Low Resource Usage**: Optimized for minimal CPU and memory consumption

## Quick Start

### Prerequisites

- Rust 1.90+ (2024 edition)
- GitHub Personal Access Token with `repo` and `workflow` scopes
- Access to GitHub Enterprise Server instance

### Building

```bash
# Build release binary
cargo build --release

# Or use Makefile
make build
```

### Running Locally

Set environment variables needed for local development:

```bash
# Required
export GITHUB_TOKEN="ghp_token"
export GITHUB_ORG="your_organization"
export GITHUB_BASE_URL="https://your-ghes-domain"

# Optional
export LOG_LEVEL="info"
export PUBLISHER_TYPE="console" # Available values: `console`, `slack-canvas`
export CONCURRENT_SCANS="10"    # Number of parallel repository scans

# For Slack Canvas Publisher
export SLACK_TOKEN="xoxb-token"
export SLACK_CHANNEL_ID="C01234ABCD"
export SLACK_CANVAS_ID="F01234ABCD"
```

Run the application:

```bash
# Using cargo
cargo run --release

# Or using the binary
./target/release/ghes-schedule-scanner
```

## Output Examples

### Console Output

```bash
Version: 1.0.0
Build Date: 2025-01-23T10:30:00Z
Git Commit: abc1234
Rust Version: 1.83.0

NO   REPOSITORY                        WORKFLOW                            UTC SCHEDULE  KST SCHEDULE  LAST COMMITTER  LAST STATUS
1    api-test-server                   api unit test                       0 15 * * *    0 0 * * *     younsl          completed
2    daily-batch                       daily batch service                 0 0 * * *     0 9 * * *     ddukbg          completed

Total: 2 scheduled workflows found in 100 repositories (5 excluded)
Scan duration: 18.5s
```

### Slack Canvas Output

![Slack Canvas Output](./docs/assets/images/2.png)

![Slack Canvas Output](./docs/assets/images/3.png)

## Configuration

### Required Environment Variables

| Variable          | Description                  | Example                      |
| ----------------- | ---------------------------- | ---------------------------- |
| `GITHUB_TOKEN`    | GitHub Personal Access Token | `ghp_xxxxxxxxxxxx`           |
| `GITHUB_ORG`      | Target GitHub organization   | `my-company`                 |
| `GITHUB_BASE_URL` | GitHub Enterprise Server URL | `https://github.example.com` |

### Optional Environment Variables

| Variable                      | Description                                 | Default   |
| ----------------------------- | ------------------------------------------- | --------- |
| `LOG_LEVEL`                   | Logging level (debug, info, warn, error)    | `info`    |
| `PUBLISHER_TYPE`              | Output format (console, slack-canvas)       | `console` |
| `REQUEST_TIMEOUT`             | HTTP request timeout for scanning (seconds) | `60`      |
| `CONCURRENT_SCANS`            | Max concurrent repository scans             | `10`      |
| `CONNECTIVITY_MAX_RETRIES`    | Connection retry attempts                   | `3`       |
| `CONNECTIVITY_RETRY_INTERVAL` | Retry delay (seconds)                       | `5`       |
| `CONNECTIVITY_TIMEOUT`        | Connectivity check timeout (seconds)        | `5`       |

## Publishers

GSS supports multiple publishers to display scan results:

### Console Publisher

Outputs scan results to the console/logs with structured JSON logging. This is the default publisher.

```bash
export PUBLISHER_TYPE="console"
```

### Slack Canvas Publisher

Publishes scan results to a Slack Canvas, providing a rich, interactive view of your scheduled workflows.

Required environment variables:

- `SLACK_TOKEN`: Slack Bot Token (must start with `xoxb-`)
- `SLACK_CHANNEL_ID`: Slack Channel ID
- `SLACK_CANVAS_ID`: Slack Canvas ID

```bash
export PUBLISHER_TYPE="slack-canvas"
export SLACK_TOKEN="xoxb-your-token"
export SLACK_CHANNEL_ID="C01234ABCD"
export SLACK_CANVAS_ID="F01234ABCD"
```

## Development

### Running Tests

```bash
# Run all tests
cargo test

# Run tests with output
cargo test -- --nocapture

# Run specific test
cargo test test_config_load
```

### Code Quality

```bash
# Format code
cargo fmt

# Check formatting
cargo fmt -- --check

# Run linter
cargo clippy -- -D warnings

# Run all checks
make ci
```

## Docker

### Building Docker Image

```bash
# Build using Makefile
make docker-build

# Or manually
docker build -t ghes-schedule-scanner:latest .
```

### Running with Docker

```bash
docker run --rm \
  -e GITHUB_TOKEN=ghp_xxxx \
  -e GITHUB_ORG=my-org \
  -e GITHUB_BASE_URL=https://github.example.com \
  ghes-schedule-scanner:latest
```

## Kubernetes Deployment

See the [Installation Guide](./docs/installation.md) for detailed instructions on deploying to Kubernetes using Helm.

Quick example:

```bash
# Install using Helm
helm install ghes-schedule-scanner \
  ./charts/ghes-schedule-scanner \
  --set image.repository=ghes-schedule-scanner \
  --set image.tag=latest
```

## Documentation

- [Installation Guide](./docs/installation.md) - Kubernetes deployment with Helm
- [Troubleshooting](./docs/troubleshooting.md) - Common issues and solutions
- [Roadmap](./docs/roadmap.md) - Future plans and features
- [Contributing Guidelines](./docs/contributing.md) - How to contribute
- [Acknowledgements](./docs/acknowledgements.md) - Credits and thanks

## Performance

| Metric                | Value            |
| --------------------- | ---------------- |
| Binary Size           | 3.8MB (stripped) |
| Memory Usage          | ~40MB            |
| Startup Time          | ~50ms            |
| Scan Time (100 repos) | ~18s             |
| Scan Time (900 repos) | ~35s             |

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
