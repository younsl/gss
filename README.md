# GHES Schedule Scanner (GSS)

A Kubernetes add-on for DevOps and SRE teams to monitor and analyze CI/CD workflows in GitHub Enterprise Server. GSS runs as a kubernetes [cronJob](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) that scans and analyzes scheduled workflows across your GHES environment.

![System Architecture](./docs/assets/images/1.png)

GHES Schedule Scanner runs as a kubernetes cronJob that periodically scans GitHub Enterprise Server repositories for scheduled workflows. It collects information about workflow name, workflow schedules, last execution status, and last committer details across all repositories in an organization. GHES schedule scanner is designed for high performance with parallel scanning capabilities using Go routines and provides timezone conversion between UTC and KST for better schedule visibility.

## Features

- Compatible with self-hosted [GitHub Enterprise Server (3.11+)](https://docs.github.com/ko/enterprise-server/admin/all-releases)
- Scan scheduled workflows across all repositories in an organization
- UTC/KST timezone conversion support
- Workflow execution status monitoring
- Parallel scanning support using [Goroutines](https://go.dev/tour/concurrency/1)
- High performance repository scanning (scans 900+ repositories in about 20-22 seconds)

## Documentation

- [Installation](./docs/installation.md)
- [Roadmap](./docs/public-roadmap.md)