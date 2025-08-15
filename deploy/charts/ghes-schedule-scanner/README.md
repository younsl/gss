# ghes-schedule-scanner

![Version: 0.4.2](https://img.shields.io/badge/Version-0.4.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.4.2](https://img.shields.io/badge/AppVersion-0.4.2-informational?style=flat-square)

A Helm chart for deploying the GHES Schedule Scanner

**Homepage:** <https://github.com/younsl/ghes-schedule-scanner>

## Prerequisites

- Helm 3.8.0+ (OCI support)
- Kubernetes 1.19+

## Installation

### Install using OCI Registry

> **Note**: OCI registry support requires Helm 3.8.0 or later.

Install the chart directly from OCI registry with the release name `ghes-schedule-scanner`:

```console
helm install ghes-schedule-scanner oci://ghcr.io/younsl/charts/ghes-schedule-scanner --version 0.4.2
```

Install with custom values:

```console
helm install ghes-schedule-scanner oci://ghcr.io/younsl/charts/ghes-schedule-scanner --version 0.4.2 -f values.yaml
```

### Install from local chart

Download ghes-schedule-scanner chart and install from local directory:

```console
helm pull oci://ghcr.io/younsl/charts/ghes-schedule-scanner --version 0.4.2
tar -xzf ghes-schedule-scanner-0.4.2.tgz
helm install ghes-schedule-scanner ./ghes-schedule-scanner
```

Note: The `--untar` option is not available when pulling from OCI registries. You need to manually extract the chart using `tar -xzf`.

## Upgrade

```console
helm upgrade ghes-schedule-scanner oci://ghcr.io/younsl/charts/ghes-schedule-scanner --version 0.4.2
```

## Uninstall

```console
helm uninstall ghes-schedule-scanner
```

## Configuration

The following table lists the configurable parameters and their default values.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Pod affinity settings affinity is used to configure additional pod settings |
| annotations | object | `{}` | CronJob annotations annotations are used to configure additional CronJob settings |
| configMap | object | `{"data":{"CONCURRENT_SCANS":"10","GITHUB_BASE_URL":"https://github.example.com","GITHUB_ORG":"example-org","LOG_LEVEL":"INFO","PUBLISHER_TYPE":"slack-canvas","REQUEST_TIMEOUT":"30","SLACK_CANVAS_ID":null,"SLACK_CHANNEL_ID":null,"SLACK_TOKEN":null}}` | ConfigMap data containing application configuration |
| configMap.data.CONCURRENT_SCANS | string | `"10"` | Number of concurrent repository scans This value is used to limit the number of concurrent goroutines that are scanning repositories. Recommended CONCURRENT_SCANS value depends on several factors: - GitHub API rate limits - GitHub API response time (latency) - Network conditions between your cluster and GitHub Enterprise Typical values range from 10-50, but can be higher if needed. |
| configMap.data.GITHUB_BASE_URL | string | `"https://github.example.com"` | GitHub Enterprise base URL The API endpoint will be automatically appended with '/api/v3' For example: https://github.example.com/api/v3 |
| configMap.data.GITHUB_ORG | string | `"example-org"` | GitHub Enterprise organization name Organization name is used to scan all repositories for the given organization |
| configMap.data.LOG_LEVEL | string | `"INFO"` | Application log level |
| configMap.data.PUBLISHER_TYPE | string | `"slack-canvas"` | Publisher type to use (Available values: console, slack-canvas) This value determines which publisher will be used to output scan results |
| configMap.data.REQUEST_TIMEOUT | string | `"30"` | HTTP request timeout in seconds |
| configMap.data.SLACK_CANVAS_ID | string | `nil` | Slack Canvas ID to update a canvas page in Slack channel. Slack Canvas URL have the following format: https://<WORKSPACE>.slack.com/docs/<CHANNEL_ID>/<CANVAS_ID> How to get: 1. Copy the last part from Canvas URL you want to update Canvas URL format: https://workspace.slack.com/docs/CHANNEL_ID/CANVAS_ID |
| configMap.data.SLACK_CHANNEL_ID | string | `nil` | Slack Channel ID to create a canvas page in Slack channel How to get: 1. Click channel name in Slack 2. Click "View channel details" 3. Scroll to bottom and copy Channel ID starting with `C` |
| configMap.data.SLACK_TOKEN | string | `nil` | Slack Bot Token to create a canvas page in Slack channel. Do not use a slack app token. How to get: 1. Go to https://api.slack.com/apps 2. Select your app > "OAuth & Permissions" 3. Copy "Bot User OAuth Token" starting with `xoxb-` |
| dnsConfig | object | `{}` | DNS config for the CronJob pod |
| excludedRepositoriesList | list | `[]` | List of repositories to exclude from the scan Note: Please exclude the organization name, only the repository name. |
| failedJobsHistoryLimit | int | `1` | Number of failed jobs to keep in history This value is used to limit the number of failed jobs |
| fullnameOverride | string | `""` | Override the full name template |
| image | object | `{"pullPolicy":"IfNotPresent","repository":"ghcr.io/containerelic/gss","tag":null}` | Container image configuration |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy (Available values: Always, IfNotPresent, Never) |
| image.repository | string | `"ghcr.io/containerelic/gss"` | Container image repository This value is used to specify the container image repository. |
| image.tag | string | `nil` | Container image tag (If not set, will use Chart's appVersion by default.) |
| nameOverride | string | `""` | Override the chart name |
| nodeSelector | object | `{}` | Node selector for pod assignment nodeSelector is used to configure additional pod settings |
| podAnnotations | object | `{}` | Pod annotations annotations are used to configure additional pod settings |
| resources | object | `{"limits":{"cpu":"100m","memory":"128Mi"},"requests":{"cpu":"50m","memory":"64Mi"}}` | Container resource requirements |
| restartPolicy | string | `"Never"` | Restart policy. Available values: Always, Never, OnFailure (default: Never) |
| schedule | string | `"0 1 * * *"` | CronJob schedule in Cron format (UTC) This value is used to configure the schedule for the CronJob. Cron expression details: minute (0-59), hour (0-23), day of month (1-31), month (1-12), day of week (0-7), `*` means all |
| secretName | string | `"ghes-schedule-scanner-secret"` | Name of the secret containing sensitive data This secret is used to store the GitHub access token with permissions to scan repositories. |
| successfulJobsHistoryLimit | int | `3` | Number of successful jobs to keep in history This value is used to limit the number of successful jobs |
| timeZone | string | `"Etc/UTC"` | Timezone for the CronJob This value is used to configure the timezone for the CronJob. Available timezone list: https://en.wikipedia.org/wiki/List_of_tz_database_time_zones |
| tolerations | list | `[]` | Pod tolerations tolerations are used to configure additional pod settings |
| topologySpreadConstraints | list | `[]` | Pod scheduling constraints for spreading pods across nodes or zones topologySpreadConstraints are used to configure additional pod settings |
| ttlSecondsAfterFinished | int | `3600` | TTL in seconds for finished jobs This value is used to delete finished jobs after a certain period of time. This helps to reduce the number of old job pods that are kept in the cluster. |

## Source Code

* <https://github.com/younsl/ghes-schedule-scanner>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| younsl | <cysl@kakao.com> | <https://github.com/younsl> |
| ddukbg | <wowrebong@gmail.com> | <https://github.com/ddukbg> |

## License

This chart is licensed under the Apache License 2.0. See [LICENSE](https://github.com/younsl/younsl.github.io/blob/main/LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit a [Pull Request](https://github.com/younsl/younsl.github.io/pulls).

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
