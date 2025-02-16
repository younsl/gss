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

2. Install Helm Chart

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

## Output Example

You can run the following command to check the scanning output:

```bash
kubectl logs -l app.kubernetes.io/name=ghes-schedule-scanner -n gss
```

Scheduled workflow scanning output example is as follows:

```bash
2025/02/14 10:19:23 Scan completed: 937 repositories in 22.726351349s with max 10 concurrent goroutines
Scheduled Workflows Summary:
NO  REPOSITORY                          WORKFLOW                           UTC SCHEDULE  KST SCHEDULE  LAST COMMITTER  LAST STATUS
1   payment-service                     Cleanup Old Artifacts              0 15 * * *    0 0 * * *     john-doe        completed
2   ip-address-monitor                  IP Range Sync                      0 * * * *     0 9 * * *     sarah-kim       completed
3   docker-images                       Build Docker Images                0 20 * * *    0 5 * * *     mike-zhang      completed
...

Scanned: 937 repos, 27 workflows
```