# Acknowledgements

This project utilizes and acknowledges the following technologies and services:

## Libraries

- [Go](https://go.dev/): For the programming language
- [logrus](https://github.com/sirupsen/logrus): For logging
- [slack-go/slack](https://github.com/slack-go/slack): For sending messages to Slack
- [stretchr/testify](https://github.com/stretchr/testify): For testing

## Third-party APIs

- [GitHub API](https://docs.github.com/en/rest): For getting the scheduled workflows
- [Slack API](https://api.slack.com/): For writing scan results to Slack Canvas page through Slack App

## Kubernetes

- [Kubernetes CronJobs](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/): For running the application as a cronJob
- [Helm](https://helm.sh/): For deploying the application to the Kubernetes cluster
