# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GHES Schedule Scanner (GSS) is a high-performance Kubernetes add-on for monitoring and analyzing CI/CD scheduled workflows in GitHub Enterprise Server. Written in Rust, it runs as a Kubernetes CronJob that scans repositories for scheduled workflows and publishes results to console or Slack Canvas.

## Technology Stack

- **Language**: Rust 1.90+ (2024 edition)
- **Async Runtime**: Tokio with timeout support
- **HTTP Client**: Reqwest (with rustls)
- **GitHub API**: Octocrab
- **Logging**: Tracing with structured JSON formatting (single-line for Kubernetes)
- **Serialization**: Serde (JSON, YAML)
- **Error Handling**: anyhow, thiserror

## Development Commands

### Build and Run

```bash
# Run the application locally
cargo run --release

# Build the binary
cargo build --release

# Build Docker image
docker build -t ghes-schedule-scanner .

# Or use Makefile
make build
make docker-build
```

### Testing

```bash
# Run all tests
cargo test

# Run tests with verbose output
cargo test -- --nocapture

# Run specific test
cargo test test_config_load

# Run tests for specific module
cargo test --lib config
cargo test --lib scanner
```

### Code Quality

```bash
# Format Rust code
cargo fmt

# Check formatting
cargo fmt -- --check

# Run linter
cargo clippy -- -D warnings

# Check code without building
cargo check

# Run all CI checks
make ci
```

## Architecture

### Module Structure

The project follows Rust 2018+ module conventions with a flat structure:

- **src/main.rs**: Application entry point
  - Initializes configuration, logger, connectivity checker
  - Orchestrates scanning and publishing workflow

- **src/config.rs**: Configuration management
  - Loads environment variables
  - Validates required and optional settings
  - Manages defaults

- **src/connectivity.rs**: Network connectivity checks
  - Validates GitHub Enterprise Server connectivity before scanning
  - Implements retry logic with configurable attempts and intervals
  - Measures and logs response time (in milliseconds) for monitoring
  - Verifies API accessibility with `/api/v3/meta` endpoint

- **src/scanner.rs**: Core scanning logic
  - Interfaces with GitHub API via Octocrab
  - Implements concurrent repository scanning with semaphore-based rate limiting
  - Applies REQUEST_TIMEOUT to all GitHub API calls using `tokio::time::timeout`
  - Extracts workflow schedules from YAML files (base64 decoded)
  - Tracks workflow status and author information (GitHub username)
  - Loads repository exclude list from `/etc/gss/exclude-repos.txt`

- **src/reporter.rs**: Report generation and formatting
  - Processes scan results into structured reports
  - Handles timezone conversions (UTC to KST)
  - Implements cron schedule conversion logic

- **src/publisher.rs** + **src/publisher/**: Output publishing (Rust 2018+ pattern)
  - **publisher.rs**: Publisher trait and factory pattern
  - **publisher/console.rs**: Console output with structured logging
  - **publisher/slack.rs**: Slack Canvas integration via API
  - Interface-based design for extensibility
  - Note: Uses modern `publisher.rs` + `publisher/` pattern instead of `publisher/mod.rs`

- **src/models.rs**: Data structures
  - Defines WorkflowInfo, ScanResult, and workflow parsing types
  - Implements serialization/deserialization

- **src/logger.rs**: Logging configuration
  - Sets up tracing-subscriber with JSON formatting
  - Single-line JSON output optimized for Kubernetes log aggregation
  - Includes thread IDs, thread names, and targets for debugging
  - Configurable log levels via LOG_LEVEL environment variable

### Key Design Patterns

1. **Async/Await with Timeouts**: Tokio runtime with timeout wrappers on all external API calls
2. **Trait-based abstractions**: Publisher trait for extensible output formats
3. **Factory pattern**: PublisherFactory for creating appropriate publisher instances
4. **Semaphore-based rate limiting**: Concurrent scanning with controlled parallelism
5. **Configuration via environment**: All settings from environment variables
6. **Structured logging**: Tracing with structured fields for Kubernetes log aggregation
7. **Test isolation**: `Config::new_for_test()` pattern avoids environment variable pollution in tests
8. **Modern module structure**: Rust 2018+ pattern with `module.rs` + `module/` instead of `module/mod.rs`

## Environment Variables

Required for local development:

- `GITHUB_TOKEN`: GitHub personal access token
- `GITHUB_ORG`: Target GitHub organization
- `GITHUB_BASE_URL`: GitHub Enterprise Server URL

Optional:

- `LOG_LEVEL`: Logging level (debug, info, warn, error) - default: info
- `PUBLISHER_TYPE`: Output type (console, slack-canvas) - default: console
- `CONCURRENT_SCANS`: Number of parallel scans - default: 10
- `REQUEST_TIMEOUT`: HTTP request timeout for scanning in seconds - default: 60
- `CONNECTIVITY_MAX_RETRIES`: Connection retry attempts - default: 3
- `CONNECTIVITY_RETRY_INTERVAL`: Retry delay in seconds - default: 5
- `CONNECTIVITY_TIMEOUT`: Connectivity check timeout in seconds - default: 5

For Slack Canvas publisher:

- `SLACK_TOKEN`: Slack Bot Token (xoxb-\*)
- `SLACK_CHANNEL_ID`: Target channel ID
- `SLACK_CANVAS_ID`: Canvas ID to update

## Kubernetes Deployment

The application is deployed as a Helm chart with:

- CronJob for scheduled execution
- ConfigMap for exclude list configuration
- Support for timezone configuration
- TTL for job cleanup
- SecurityContext for non-root execution (UID/GID 1000)

Helm chart location: `charts/ghes-schedule-scanner/`

### Helm Chart Development

```bash
# Lint the chart
helm lint charts/ghes-schedule-scanner

# Template and verify
helm template test ./charts/ghes-schedule-scanner

# Test with custom values
helm template test ./charts/ghes-schedule-scanner --set configMap.enabled=false

# Package the chart
helm package charts/ghes-schedule-scanner
```

### Helm Chart Architecture

Key features:

- **External ConfigMap support**: Set `configMap.enabled=false` and `configMap.name=external-cm` to use pre-existing ConfigMaps
- **Flexible resources**: Uses `{{- toYaml . | nindent 14 }}` pattern for dynamic resource definitions
- **Helper functions**: `configMapName` and `excludeConfigMapName` in `_helpers.tpl` for dynamic name resolution
- **Conditional volumes**: Volumes only mount when ConfigMap is available (enabled or external)
- **CronJob controls**: `suspend`, `concurrencyPolicy` (default: Forbid), `imagePullSecrets` support
- **Type annotations**: All values.yaml fields have `(string)`, `(int)`, `(bool)`, `(object)`, `(list)` type hints

## Testing Considerations

- **Test Pattern**: Use `Config::new_for_test()` to avoid environment variable pollution
  - Tests run in parallel without interference
  - No need for `unsafe` blocks or environment variable cleanup
  - Example: `Config::new_for_test("token".to_string(), "org".to_string(), "url".to_string())`
- **Test Coverage**: 25 tests covering all modules
- **Mock implementations**: For external GitHub API dependencies
- **Test modules**: Follow Rust conventions with `#[cfg(test)]` modules in same file

## Build Configuration

- **Cargo.toml package metadata**: Includes all recommended fields for crates.io publishing
  - `keywords`: 5 search keywords for discoverability
  - `categories`: Proper categorization (command-line-utilities, development-tools)
  - `homepage`, `documentation`, `readme`: Documentation links
  - `exclude`: Minimizes package size by excluding CI configs, docs, charts

- **Release profile**: Optimized for size and performance
  - LTO enabled
  - Strip symbols
  - Single codegen unit
  - Size optimization (`opt-level = "z"`)

- **Build script**: `build.rs` captures build metadata
  - Git commit hash
  - Build date
  - Rustc version

## Docker

- **Multi-stage build**: Builder (rust:1.90-alpine) + runtime (alpine:3.22)
- **Security**: Non-root user (UID/GID 1000)
- **Binary size**: ~3.8MB (stripped, optimized)
- **Runtime dependencies**: ca-certificates for HTTPS

## Common Tasks

### Adding a new publisher

1. Create new module in `src/publisher/`
2. Implement the `Publisher` trait
3. Add to `PublisherFactory::create()` method
4. Update documentation and tests

### Modifying scan logic

1. Update `src/scanner.rs`
2. Consider impact on concurrency and rate limiting
3. Update tests in scanner module
4. Verify with local testing before deployment

### Adding new configuration

1. Add field to `Config` struct in `src/config.rs`
2. Implement loading logic with defaults
3. Update validation if required
4. Document in README.md

### Debugging

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Run with backtraces
RUST_BACKTRACE=1 cargo run

# Use rust-gdb or rust-lldb
rust-gdb target/debug/ghes-schedule-scanner
```

## Performance Considerations

- **Concurrent scanning**: Semaphore-based rate limiting (default: 10 parallel scans)
- **Timeout protection**: All GitHub API calls have configurable timeout (default: 60s)
- **Async I/O**: Non-blocking operations with Tokio runtime
- **Memory efficiency**: Streaming responses, no buffering of large payloads
- **Binary size**: 3.8MB with LTO and size optimization
- **Logging overhead**: Minimal with structured fields, single-line JSON

## Security

- Secrets loaded from environment variables only
- HTTPS-only communication with rustls
- No hardcoded credentials
- Token validation on startup

## Logging Best Practices

- **Use tracing macros**: `info!()`, `debug!()`, `warn!()`, `error!()`
- **Structured fields**: Use field syntax for searchable data
  ```rust
  info!(
      github_org = %config.github_organization,
      concurrent_scans = config.concurrent_scans,
      "Configuration loaded"
  );
  ```
- **Single-line JSON**: All logs output as single-line JSON for Kubernetes
- **Response time logging**: Always log `response_time_ms` for external API calls
- **Console output**: Only use `println!()` in Console publisher for actual program output

## Documentation Structure

- **README.md**: Project overview, features, quick start
- **docs/installation.md**: Kubernetes deployment with Helm
- **docs/troubleshooting.md**: Common issues and solutions (no GitHub Issues)
- **docs/roadmap.md**: Future plans
- **docs/contributing.md**: Contribution guidelines
- **CLAUDE.md**: This file - guidance for Claude Code instances

## Contributing Guidelines

1. Run `cargo fmt` before committing
2. Ensure `cargo clippy -- -D warnings` passes
3. Add tests using `Config::new_for_test()` pattern for isolation
4. Update documentation for API changes
5. Use conventional commit messages
6. All logging must use tracing, never println! (except Console publisher)

## Important Notes

- **GitHub Issues are disabled** for this repository
- Documentation should be **direct and concise** with Summary and TL;DR sections
- All troubleshooting is self-service via docs/troubleshooting.md
- Logs are single-line JSON optimized for Kubernetes log aggregation tools
