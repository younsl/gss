# Public Roadmap

## Summary

This roadmap document outlines upcoming features and improvements for the project. Each item includes current status, expected timeline, and brief description. This open roadmap helps track development progress and share plans with team members and community users.

## Status Board

This table tracks the status of major development tasks. Each task is categorized and includes current progress status and additional notes.

| # | Task | Category | Status | Notes |
|---|------|----------|--------|-------|
| 1 | Multi-Publisher System | Feature | Backlog | Core architecture for output flexibility |
| 2 | Messaging Platform Integration | Integration | Backlog | Discord, Teams webhook support |
| 3 | Network Connectivity Check | Infrastructure | Backlog | Required for system stability |
| 4 | Prometheus Metrics Integration | Monitoring | Backlog | Improves observability |
| 5 | Code Refactoring | Maintenance | Backlog | Reduces technical debt |
| 6 | Markdown Export | Feature | Backlog | Documentation support |
| 7 | JSON/API Support | Integration | Backlog | Enable programmatic access |

Each task must be in one of these statuses: Backlog, Todo, In Progress, Done, Cancelled

## Technical Specifications

### Multi-Publisher System

:label: Feature, Core architecture improvement

Currently, scan results are only published to Slack Canvas. This enhancement introduces a flexible publisher system to support various output formats and destinations.

Key implementation details:
- Create Publisher interface for standardized output handling
- Implement Factory pattern for publisher creation
- Support multiple concurrent publishers
- Add configuration options for each publisher type
- Maintain backward compatibility with existing Canvas publisher

Planned publisher types:
1. SlackCanvasPublisher (existing)
2. WebhookPublisher (for messaging platforms)
3. MarkdownPublisher (documentation)
4. JSONPublisher (API integration)
5. FilePublisher (local storage)

### Messaging Platform Integration

:label: Integration, Multi-platform support

Implement webhook-based integration support for various messaging platforms to increase accessibility and user adoption.

Supported platforms:
- Slack webhook
- Discord webhook
- Microsoft Teams webhook
- Generic webhook for custom integrations

Implementation details:
- Common webhook interface
- Platform-specific message formatters
- Configurable webhook URLs
- Error handling and retry logic
- Message customization options

Configuration example:
```yaml
publishers:
  webhook:
    platforms:
      slack:
        url: "https://hooks.slack.com/..."
        enabled: true
      discord:
        url: "https://discord.com/api/webhooks/..."
        enabled: true
      teams:
        url: "https://outlook.office.com/webhook/..."
        enabled: true
```

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

### Markdown Export

:label: Feature, Documentation support

Add ability to export scan results in markdown format, suitable for GitHub wikis, README files, or other documentation systems.

Implementation details:
- Markdown template system
- Configurable output formatting
- Table of contents generation
- Links to GitHub repositories
- Export to file or return as string

### JSON/API Support

:label: Integration, Enable automation

Provide structured JSON output and API endpoints for programmatic access to scan results. This enables integration with other tools and automation workflows.

Features:
- RESTful API endpoints
- JSON schema documentation
- Authentication support
- Rate limiting
- Bulk export capabilities