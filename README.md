# GHES Schedule Scanner (GSS)

A Kubernetes CronJob-based scanning server that scans and analyzes scheduled workflows in GitHub Enterprise Server.

## Features

- Scan scheduled workflows across all repositories in an organization
- UTC/KST timezone conversion support
- Workflow execution status monitoring
- Parallel scanning support
- High performance repository scanning (900+ repositories in ~19 seconds)

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
2024/11/14 12:48:19 Scan completed: 930 repositories in 18.570823763s with max 10 concurrent goroutines
Scheduled Workflows Summary:
NO    REPOSITORY                            WORKFLOW                              UTC SCHEDULE    KST SCHEDULE    LAST STATUS
1     mock-service-a                        Cleanup Job                           0 0 * * *       0 9 * * *       Unknown
2     mock-service-b                        Health Check                          */30 * * * *    */30 * * * *    completed
3     mock-repo-c                           Backup Database                       0 18 * * *      0 3 * * *       completed
...

Scanned: 930 repos, 27 workflows
```

## License

MIT License