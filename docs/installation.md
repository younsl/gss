# Installation

## Summary

GHES Schedule Scanner supports only Helm installation. Other deployment methods like Kustomize or Kubernetes Operator Pattern are not supported.

## Requirements

- Go +1.21
- Helm v3.0.0+
- Personal Access Token issued by GHES Organization Owner or Enterprise Admin (with `repo:*` scope to access all repositories)
- Kubernetes cluster to deploy GHES Schedule Scanner by using Helm chart

## Local Development Setup

For local development and testing, you'll need to set up the following environment variables:

```bash
# Required environment variables
export GITHUB_TOKEN="ghp_token"              # GitHub Personal Access Token with repo:* scope
export GITHUB_ORGANIZATION="your_org"        # Your GitHub organization name
export GITHUB_BASE_URL="https://your-ghes-domain"  # Your GitHub Enterprise Server URL

# Optional environment variables
export LOG_LEVEL="INFO"                      # Log level (default: INFO)
export REQUEST_TIMEOUT="30"                  # Request timeout in seconds (default: 30)

# Optional Slack integration
export SLACK_BOT_TOKEN="xoxb-token"         # Slack Bot Token
export SLACK_CHANNEL_ID="F01234ABCD"        # Slack Channel ID
export SLACK_CANVAS_ID="C01234ABCD"         # Slack Canvas ID
```

To run the scanner locally:

```bash
# Run with default log level
go run cmd/ghes-schedule-scanner/main.go

# Run with specific log level
LOG_LEVEL=DEBUG go run cmd/ghes-schedule-scanner/main.go
```

## Environment Variables and ConfigMap

The following table shows the mapping between environment variables and configMap values:

| Environment Variable   | ConfigMap Key       | Description                                    | Required |
|-----------------------|---------------------|------------------------------------------------|----------|
| GITHUB_TOKEN          | GITHUB_TOKEN        | GitHub PAT with repo:* scope                   | Yes      |
| GITHUB_ORGANIZATION   | GITHUB_ORGANIZATION | GitHub organization name                        | Yes      |
| GITHUB_BASE_URL      | GITHUB_BASE_URL     | GitHub Enterprise Server URL                    | Yes      |
| LOG_LEVEL            | LOG_LEVEL           | Log level (DEBUG, INFO, WARN, ERROR)           | No       |
| REQUEST_TIMEOUT      | REQUEST_TIMEOUT     | API request timeout in seconds                 | No       |
| SLACK_BOT_TOKEN      | SLACK_BOT_TOKEN     | Slack Bot Token (xoxb-)                        | No       |
| SLACK_CHANNEL_ID     | SLACK_CHANNEL_ID    | Slack Channel ID                               | No       |
| SLACK_CANVAS_ID      | SLACK_CANVAS_ID     | Slack Canvas ID                                | No       |

Example configMap in values.yaml:

```yaml
# hack/charts/ghes-schedule-scanner/values.yaml
configMap:
  data:
    GITHUB_ORGANIZATION: "your-org"
    GITHUB_BASE_URL: "https://your-ghes-domain"
    LOG_LEVEL: "INFO"
    REQUEST_TIMEOUT: "30"
    SLACK_BOT_TOKEN: "xoxb-your-token"
    SLACK_CHANNEL_ID: "F01234ABCD"
    SLACK_CANVAS_ID: "C01234ABCD"
```

## Helm installation

1. Create kubernetes secret

GSS pod uses GitHub API to scan repositories in specified organization and find scheduled workflows. PAT only needs `repo:*` scope to work properly. Secret's data key must be `GITHUB_TOKEN` containing PAT value.

First, create a dedicated namespace to isolate GSS kubernetes resources.

```bash
kubectl create namespace gss
```

The GitHub Personal Access Token (PAT) with `repo:*` scope must be created as a [kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/) before helm installation

```bash
# Create secret resource storing GitHub access token with repo:* scope
kubectl create secret generic ghes-schedule-scanner-secret \
    --namespace gss \
    --from-literal GITHUB_TOKEN=ghp_<CLASSIC_TOKEN>
```

2. Create slack credentials to publish canvas page

GSS pod uses Slack Bot Token to create a canvas page in Slack channel. Slack Bot Token can be created from [Slack API](https://api.slack.com/apps).

![Canvases API architecture](./assets/images/2.png)

- **Slack Bot Token**: Not slack app token, only slack bot token is supported. Slack bot token starts with `xoxb-`.
- **Slack Channel ID**: Channel ID where the canvas will be created. Channel ID is a string of numbers starting with `C`.
- **Slack Canvas ID**: Canvas ID to update. Canvas ID is a string of numbers starting with `F`.

Configure these credentials in `values.yaml` file. These credentials are used by GSS pod to publish a canvas page in your slack channel.

```yaml
# hack/charts/ghes-schedule-scanner/values.yaml
configMap:
  data:
    # ... omitted for brevity ...
    SLACK_BOT_TOKEN: xoxb-<SLACK_BOT_TOKEN>
    SLACK_CHANNEL_ID: F01234ABCD
    SLACK_CANVAS_ID: C01234ABCD
```

3. Install Helm Chart

Install [ghes-schedule-scanner](https://github.com/younsl/gss/tree/main/hack/charts/ghes-schedule-scanner) helm chart in the `gss` namespace:

```bash
helm upgrade \
    --install \
    --values values.yaml \
    --namespace gss \
    --create-namespace \
    ghes-schedule-scanner . \
    --wait
```

You can use the same command to update and apply helm chart configurations later.

### Output Example

You can run the following command to check the scanning output:

```bash
kubectl logs -l app.kubernetes.io/name=ghes-schedule-scanner -n gss
```

You can see all scheduled workflows in the canvas page. Canvas URL format is follows: `https://<WORKSPACE>.slack.com/docs/<CHANNEL_ID>/<CANVAS_ID>`.

Scheduled workflow scanning output example in slack canvas page formatting with markdown:

```bash
GHES Scheduled Workflows

📊 Scan Summary • Total Repositories: 152 • Scheduled Workflows: 12 • Unknown Committers: 3 
Last Updated: 2024-03-19T09:15:33Z by GHES Schedule Scanner

* [1] backend-api
  * Workflow: Clean up old artifacts
  * UTC Schedule: 0 0 * * *
  * KST Schedule: 0 9 * * *
  * Last Status: ✅ completed
  * Last Committer: mike-zhang
* [2] monitoring
  * Workflow: Refresh metrics dashboard
  * UTC Schedule: */30 * * * *
  * KST Schedule: */30 * * * * 
  * Last Status: ✅ completed
  * Last Committer: sarah-kim
* [3] infrastructure
  * Workflow: Backup database
  * UTC Schedule: 0 18 * * *
  * KST Schedule: 0 3 * * *
  * Last Status: ✅ completed
  * Last Committer: alice-park
```

## Helm uninstall

Delete the GSS helm chart from `gss` namespace:

```bash
helm uninstall ghes-schedule-scanner -n gss
helm list -n gss
```

Then delete kubernetes secret and namespace:

```bash
kubectl delete secret ghes-schedule-scanner-secret -n gss
kubectl delete namespace gss
```