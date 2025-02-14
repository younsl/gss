# GHES Schedule Scanner (GSS)

A Kubernetes add-on for DevOps and SRE teams to monitor and analyze CI/CD workflows in GitHub Enterprise Server. GSS runs as a kubernetes [cronJob](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) that scans and analyzes scheduled workflows across your GHES environment.

![System Architecture](./docs/assets/images/1.png)

GHES Schedule Scanner runs as a kubernetes cronJob that periodically scans GitHub Enterprise Server repositories for scheduled workflows. It collects information about workflow name, workflow schedules, last execution status, and last committer details across all repositories in an organization. GHES schedule scanner is designed for high performance with parallel scanning capabilities using Go routines and provides timezone conversion between UTC and KST for better schedule visibility.

## Features

- Compatible with self-hosted GitHub Enterprise Server (3.11+)
- Scan scheduled workflows across all repositories in an organization
- UTC/KST timezone conversion support
- Workflow execution status monitoring
- Parallel scanning support
- High performance repository scanning (scans 900+ repositories in about 20-22 seconds)

## Installation

### Requirements

- Go +1.21
- GitHub Enterprise Server access token (with `repo:*` scope to access all repositories)
- Kubernetes cluster to deploy GHES Schedule Scanner

### Helm installation

GHES Schedule Scanner supports only Helm installation.

1. Create Secret

`GITHUB_TOKEN` data in the secret resource must be an access token issued from GitHub Enterprise Server. Only repository permissions are required, so the `repo:*` scope is sufficient.

> [!IMPORTANT]
> The secret resource storing GitHub access token must be created before helm installation

```bash
# Create dedicated namespace for GHES Schedule Scanner (gss)
kubectl create namespace gss

# Create secret resource storing GitHub access token with repo:* scope
kubectl create secret generic ghes-schedule-scanner-secret \
    --namespace gss \
    --from-literal GITHUB_TOKEN=ghp_<CLASSIC_TOKEN>
```

2. Install Helm Chart

Requires `values.yaml`, Helm v3.0.0+, correct kubectl context and admin privileges.

```bash
helm upgrade \
    --install \
    --values values.yaml \
    --namespace gss \
    --create-namespace \
    ghes-schedule-scanner . \
    --wait
```

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

## License

MIT License