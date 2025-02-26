# GHES Schedule Scanner (GSS)

![GitHub Go](https://img.shields.io/badge/go-1.21+-00ADD8?logo=go)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/younsl/ghes-schedule-scanner)
![License](https://img.shields.io/github/license/younsl/ghes-schedule-scanner)

A Kubernetes add-on for DevOps and SRE teams to monitor and analyze CI/CD workflows in GitHub Enterprise Server. GSS runs as a kubernetes [cronJob](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) that scans and analyzes scheduled workflows across your GHES environment.

![System Architecture](./docs/assets/images/1.png)

## 🚀 Overview

GHES Schedule Scanner runs as a kubernetes cronJob that periodically scans GitHub Enterprise Server repositories for scheduled workflows. It collects information about:

- Workflow names and schedules
- Last execution status
- Last committer details
- Repository information

The scanner is designed for high performance with parallel scanning capabilities using Go routines and provides timezone conversion between UTC and KST for better schedule visibility.

## ✨ Features

- **GitHub Enterprise Server Integration**: Compatible with self-hosted [GitHub Enterprise Server (3.11+)](https://docs.github.com/ko/enterprise-server/admin/all-releases)
- **Organization-wide Scanning**: Scan scheduled workflows across all repositories in an organization
- **Timezone Support**: UTC/KST timezone conversion for better schedule visibility
- **Status Monitoring**: Track workflow execution status and identify failed workflows
- **High Performance**: Parallel scanning using [Goroutines](https://go.dev/tour/concurrency/1) (scans 900+ repositories in about 20-22 seconds)
- **Multiple Publishers**: Publish results to console or Slack Canvas
- **Kubernetes Native**: Runs as a Kubernetes cronJob for periodic scanning

## 📊 Output Examples

### Console Output

```
Scheduled Workflows Summary:
NO  REPOSITORY                          WORKFLOW                            UTC SCHEDULE  KST SCHEDULE  LAST COMMITTER  LAST STATUS
1   api-test-server                     api unit test                       0 15 * * *    0 0 * * *     younsl        completed
2   daily-batch                       daily batch service                   0 * * * *     0 9 * * *     ddukbg        completed
```

### Slack Canvas Output

![Slack Canvas Output](./docs/assets/images/2.png)
![Slack Canvas Output](./docs/assets/images/3.png)

## 🔧 Installation

### Prerequisites

- Kubernetes cluster (1.19+)
- Helm v3.0.0+
- GitHub Personal Access Token with `repo:*` scope
- Slack Bot Token (for Slack Canvas publishing)

### Quick Start

1. Create a namespace for GSS:
```bash
kubectl create namespace gss
```

2. Create a Kubernetes secret with your GitHub token:
```bash
kubectl create secret generic ghes-schedule-scanner-secret \
    --namespace gss \
    --from-literal GITHUB_TOKEN=ghp_<YOUR_TOKEN>
```

3. Install using Helm:
```bash
helm upgrade \
    --install \
    --values values.yaml \
    --namespace gss \
    --create-namespace \
    ghes-schedule-scanner . \
    --wait
```

For detailed installation instructions, see the [Installation Guide](./docs/installation.md).

## 🔄 Publishers

GSS supports multiple publishers to display scan results:

### Console Publisher

Outputs scan results to the console/logs. This is the default publisher.

### Slack Canvas Publisher

Publishes scan results to a Slack Canvas, providing a rich, interactive view of your scheduled workflows.

Required environment variables:
- `SLACK_BOT_TOKEN`: Slack Bot Token (starts with `xoxb-`)
- `SLACK_CHANNEL_ID`: Slack Channel ID
- `SLACK_CANVAS_ID`: Slack Canvas ID

## 🛠️ Local Development

Set up environment variables:

```bash
# Required
export GITHUB_TOKEN="ghp_token"
export GITHUB_ORG="your_org"
export GITHUB_BASE_URL="https://your-ghes-domain"

# Optional
export LOG_LEVEL="INFO"  # Default log level
export REQUEST_TIMEOUT="30"  # Default timeout in seconds
export PUBLISHER_TYPE="console"  # or "slack-canvas"

# For Slack Canvas
export SLACK_TOKEN="xoxb-token"
export SLACK_CHANNEL_ID="F01234ABCD"
export SLACK_CANVAS_ID="C01234ABCD"
```

Run locally:

```bash
go run cmd/ghes-schedule-scanner/main.go
```

## 📚 Documentation

- [Installation Guide](./docs/installation.md)
- [Roadmap](./docs/roadmap.md)
- [Contributing Guidelines](./docs/contributing.md)

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgements

- [GitHub API](https://docs.github.com/en/rest)
- [Slack API](https://api.slack.com/)
- [Kubernetes CronJobs](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/)
