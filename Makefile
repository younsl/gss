.PHONY: build run test clean docker-build docker-run fmt lint check

# Variables
BINARY_NAME := ghes-schedule-scanner
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
RUSTC_VERSION := $(shell rustc --version | cut -d' ' -f2)

# Build the binary
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	cargo build --release

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	cargo run

# Run tests
test:
	@echo "Running tests..."
	cargo test

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	cargo tarpaulin --out Html --output-dir coverage

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	cargo clean

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -f Dockerfile.rust \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg RUSTC_VERSION=$(RUSTC_VERSION) \
		-t $(BINARY_NAME):$(VERSION) \
		-t $(BINARY_NAME):latest \
		.

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run --rm \
		-e GITHUB_TOKEN \
		-e GITHUB_ORG \
		-e GITHUB_BASE_URL \
		$(BINARY_NAME):latest

# Format code
fmt:
	@echo "Formatting code..."
	cargo fmt

# Check formatting
fmt-check:
	@echo "Checking code formatting..."
	cargo fmt -- --check

# Run linter
lint:
	@echo "Running linter..."
	cargo clippy -- -D warnings

# Check code without building
check:
	@echo "Checking code..."
	cargo check

# Run all checks (format, lint, test)
ci: fmt-check lint test
	@echo "All CI checks passed!"

# Install development dependencies
dev-deps:
	@echo "Installing development dependencies..."
	cargo install cargo-tarpaulin
	cargo install cargo-watch

# Watch and rebuild on changes
watch:
	@echo "Watching for changes..."
	cargo watch -x run

# Show version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Rustc Version: $(RUSTC_VERSION)"
