# Installation

## Summary

GHES Schedule Scanner supports only Helm installation. Other deployment methods like Kustomize or Kubernetes Operator Pattern are not supported.

## Requirements

- Go +1.21
- Helm v3.0.0+
- Personal Access Token issued by GHES Organization Owner or Enterprise Admin (with `repo:*` scope to access all repositories)
- Kubernetes cluster to deploy GHES Schedule Scanner by using Helm chart

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