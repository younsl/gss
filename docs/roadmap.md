# Public Roadmap

## Summary

This roadmap document outlines upcoming features and improvements for the project. Each item includes current status, expected timeline, and brief description. This open roadmap helps track development progress and share plans with team members and community users.

## Status Board

This table tracks the status of major development tasks. Each task is categorized and includes current progress status and additional notes.

| # | Task | Category | Status | Notes |
|---|------|----------|--------|-------|
| 1 | Network Connectivity Check | Infrastructure | Backlog | Required for system stability |
| 2 | Prometheus Metrics Integration | Monitoring | Backlog | Improves observability |
| 3 | Code Refactoring | Maintenance | Backlog | Reduces technical debt |

Each task must be in one of these statuses: Backlog, Todo, In Progress, Done, Cancelled

## Technical Specifications

### Network Connectivity Check

:label: Infrastructure, Required for system stability

Main process needs network validation during initialization. This ensures stable connection to external services before starting core operations.

### Prometheus Metrics Integration

:label: Observability, Improves observability

Expose workflow scan results via Prometheus metrics to improve observability. Will track scan duration, success rates, and resource usage.

### Code Refactoring

:label: Maintenance, Reduce technical debt

Clean up duplicate code by moving common functions to utils package following DRY principle. Focus on HTTP client, validation logic and helper functions.

This refactoring is essential to improve code maintainability and reduce technical debt. Centralizing common functions will make the codebase more consistent and easier to update. Following the standard Go project layout will also make it more familiar to new developers and align with industry best practices.

Key implementation details:

- Move common functions to utils package
- Restructure project layout to follow [Go standard project layout](https://github.com/golang-standards/project-layout)