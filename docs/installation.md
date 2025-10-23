# Installation

## Summary

GHES Schedule Scanner supports only Helm installation. Other deployment methods like Kustomize or Kubernetes Operator Pattern are not supported.

## Requirements

- Rust 1.90+ (for building from source)
- Helm v3.0.0+
- Personal Access Token issued by GHES Organization Owner or Enterprise Admin (with `repo:*` scope to access all repositories)
- Kubernetes cluster to deploy GHES Schedule Scanner using Helm chart

## Local Development Setup

For local development and testing, you'll need to set up the following environment variables:

```bash
# Required environment variables
export GITHUB_TOKEN="ghp_token"                      # GitHub Personal Access Token with repo:* scope
export GITHUB_ORG="your_org"                         # Your GitHub organization name
export GITHUB_BASE_URL="https://your-ghes-domain"    # Your GitHub Enterprise Server URL

# Optional environment variables
export LOG_LEVEL="info"                              # Log level: debug, info, warn, error (default: info)
export PUBLISHER_TYPE="console"                      # Publisher type: console, slack-canvas (default: console)
export REQUEST_TIMEOUT="60"                          # HTTP request timeout for scanning in seconds (default: 60)
export CONCURRENT_SCANS="10"                         # Number of concurrent repository scans (default: 10)

# Optional Slack integration (required when PUBLISHER_TYPE=slack-canvas)
export SLACK_TOKEN="xoxb-token"                      # Slack Bot Token (must start with xoxb-)
export SLACK_CHANNEL_ID="C01234ABCD"                 # Slack Channel ID
export SLACK_CANVAS_ID="F01234ABCD"                  # Slack Canvas ID
```

To run the scanner locally:

```bash
# Run with default settings
cargo run --release

# Run with specific log level
LOG_LEVEL=debug cargo run --release

# Run with Slack Canvas publisher
PUBLISHER_TYPE=slack-canvas \
SLACK_TOKEN=xoxb-your-token \
SLACK_CHANNEL_ID=C01234ABCD \
SLACK_CANVAS_ID=F01234ABCD \
cargo run --release
```

## Environment Variables and ConfigMap

The following table shows the mapping between environment variables and ConfigMap values:

| Environment Variable | ConfigMap Key    | Description                                 | Required | Default |
| -------------------- | ---------------- | ------------------------------------------- | -------- | ------- |
| GITHUB_TOKEN         | GITHUB_TOKEN     | GitHub PAT with repo:\* scope               | Yes      | -       |
| GITHUB_ORG           | GITHUB_ORG       | GitHub organization name                    | Yes      | -       |
| GITHUB_BASE_URL      | GITHUB_BASE_URL  | GitHub Enterprise Server URL                | Yes      | -       |
| LOG_LEVEL            | LOG_LEVEL        | Log level (debug, info, warn, error)        | No       | info    |
| PUBLISHER_TYPE       | PUBLISHER_TYPE   | Output format (console, slack-canvas)       | No       | console |
| REQUEST_TIMEOUT      | REQUEST_TIMEOUT  | HTTP request timeout for scanning (seconds) | No       | 60      |
| CONCURRENT_SCANS     | CONCURRENT_SCANS | Number of parallel repository scans         | No       | 10      |
| SLACK_TOKEN          | SLACK_TOKEN      | Slack Bot Token (xoxb-)                     | No\*     | -       |
| SLACK_CHANNEL_ID     | SLACK_CHANNEL_ID | Slack Channel ID                            | No\*     | -       |
| SLACK_CANVAS_ID      | SLACK_CANVAS_ID  | Slack Canvas ID                             | No\*     | -       |

\* Required when `PUBLISHER_TYPE=slack-canvas`

Example ConfigMap in values.yaml:

```yaml
# charts/ghes-schedule-scanner/values.yaml
configMap:
  data:
    GITHUB_ORG: "your-org"
    GITHUB_BASE_URL: "https://your-ghes-domain"
    LOG_LEVEL: "info"
    PUBLISHER_TYPE: "slack-canvas"
    REQUEST_TIMEOUT: "60"
    CONCURRENT_SCANS: "10"
    SLACK_TOKEN: "xoxb-your-token"
    SLACK_CHANNEL_ID: "C01234ABCD"
    SLACK_CANVAS_ID: "F01234ABCD"
```

## Helm Installation

### 1. Create Kubernetes Secret

GSS pod uses GitHub API to scan repositories in the specified organization and find scheduled workflows. PAT only needs `repo:*` scope to work properly. The secret's data key must be `GITHUB_TOKEN` containing the PAT value.

First, create a dedicated namespace to isolate GSS kubernetes resources:

```bash
kubectl create namespace gss
```

The GitHub Personal Access Token (PAT) with `repo:*` scope must be created as a [Kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/) before Helm installation:

```bash
# Create secret resource storing GitHub access token with repo:* scope
kubectl create secret generic ghes-schedule-scanner-secret \
    --namespace gss \
    --from-literal GITHUB_TOKEN=ghp_<CLASSIC_TOKEN>
```

### 2. Configure Slack Credentials (Optional)

If you want to publish results to Slack Canvas, configure the Slack credentials.

![Canvases API architecture](./assets/images/2.png)

- **Slack Bot Token**: Not slack app token, only slack bot token is supported. Slack bot token starts with `xoxb-`.
- **Slack Channel ID**: Channel ID where the canvas will be created. Channel ID starts with `C`.
- **Slack Canvas ID**: Canvas ID to update. Canvas ID starts with `F`.

Configure these credentials in `values.yaml`:

```yaml
# charts/ghes-schedule-scanner/values.yaml
configMap:
  data:
    PUBLISHER_TYPE: "slack-canvas"
    SLACK_TOKEN: "xoxb-<SLACK_BOT_TOKEN>"
    SLACK_CHANNEL_ID: "C01234ABCD"
    SLACK_CANVAS_ID: "F01234ABCD"
```

### 3. Install Helm Chart

Install [ghes-schedule-scanner](https://github.com/younsl/gss/tree/main/charts/ghes-schedule-scanner) Helm chart in the `gss` namespace:

```bash
helm upgrade \
    --install \
    --values values.yaml \
    --namespace gss \
    --create-namespace \
    ghes-schedule-scanner ./charts/ghes-schedule-scanner \
    --wait
```

You can use the same command to update and apply Helm chart configurations later.

## Verify Installation

### Check CronJob

```bash
kubectl get cronjob -n gss
kubectl get jobs -n gss
```

### View Logs

You can run the following command to check the scanning output:

```bash
kubectl logs -l app.kubernetes.io/name=ghes-schedule-scanner -n gss --tail=100
```

### Console Output Example

```json
{"level":"info","msg":"Starting GHES Schedule Scanner","target":"ghes_schedule_scanner"}
{"level":"info","msg":"Version: 1.0.0","target":"ghes_schedule_scanner"}
{"level":"info","msg":"Verifying connectivity to GitHub Enterprise Server","target":"ghes_schedule_scanner"}
{"level":"info","msg":"Successfully connected to GitHub Enterprise Server","target":"ghes_schedule_scanner::connectivity"}
{"level":"info","msg":"Scanning organization: your-org","target":"ghes_schedule_scanner"}
{"level":"info","msg":"Found 152 repositories to scan","target":"ghes_schedule_scanner::scanner"}
{"level":"info","msg":"Scan completed: found 12 scheduled workflows","target":"ghes_schedule_scanner"}
```

### Slack Canvas Output Example

You can see all scheduled workflows in the canvas page. Canvas URL format: `https://<WORKSPACE>.slack.com/docs/<CHANNEL_ID>/<CANVAS_ID>`.

Scheduled workflow scanning output example in Slack Canvas:

```markdown
# GitHub Scheduled Workflows Report

**Version:** 1.0.0
**Build Date:** 2025-01-23T10:30:00Z
**Git Commit:** abc1234

## Summary

- **Total Workflows:** 12
- **Total Repositories:** 152
- **Excluded Repositories:** 5
- **Scan Duration:** 18.5s

## Scheduled Workflows

### 1. Clean up old artifacts

- **Repository:** `backend-api`
- **Workflow File:** `.github/workflows/cleanup.yml`
- **UTC Schedule:** `0 0 * * *`
- **KST Schedule:** `0 9 * * *`
- **Last Status:** ✅ completed
- **Last Committer:** mike-zhang (✅ Active)

### 2. Refresh metrics dashboard

- **Repository:** `monitoring`
- **Workflow File:** `.github/workflows/metrics.yml`
- **UTC Schedule:** `*/30 * * * *`
- **KST Schedule:** `*/30 * * * *`
- **Last Status:** ✅ completed
- **Last Committer:** sarah-kim (✅ Active)

### 3. Backup database

- **Repository:** `infrastructure`
- **Workflow File:** `.github/workflows/backup.yml`
- **UTC Schedule:** `0 18 * * *`
- **KST Schedule:** `0 3 * * *`
- **Last Status:** ✅ completed
- **Last Committer:** alice-park (⚠️ Inactive)
```

## Helm Uninstall

Delete the GSS Helm chart from `gss` namespace:

```bash
helm uninstall ghes-schedule-scanner -n gss
helm list -n gss
```

Then delete Kubernetes secret and namespace:

```bash
kubectl delete secret ghes-schedule-scanner-secret -n gss
kubectl delete namespace gss
```
