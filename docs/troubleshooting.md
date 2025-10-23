# Troubleshooting

## Summary

Troubleshooting guide for DevOps engineers, SREs, and platform engineers running GHES Schedule Scanner in Kubernetes.

## TL;DR

**Quick Navigation:**

- [Connectivity Issues](#connectivity-issues) - GitHub Enterprise Server connection problems
- [Authentication](#authentication-problems) - Token validation and permissions
- [Timeouts](#timeout-errors) - Request timeout during scanning
- [Performance](#performance-issues) - Slow scans, high memory usage
- [Kubernetes](#kubernetes-deployment-issues) - Pod crashes, CronJob failures
- [Logging](#logging-and-debugging) - Debug mode, log analysis with `jq`
- [Slack](#slack-integration-issues) - Canvas update problems

## Table of Contents

- [Connectivity Issues](#connectivity-issues)
- [Authentication Problems](#authentication-problems)
- [Timeout Errors](#timeout-errors)
- [Performance Issues](#performance-issues)
- [Kubernetes Deployment Issues](#kubernetes-deployment-issues)
- [Logging and Debugging](#logging-and-debugging)
- [Slack Integration Issues](#slack-integration-issues)

## Connectivity Issues

### Cannot Connect to GitHub Enterprise Server

**Symptoms:**

```
Connectivity verification failed: Failed to send request to GitHub Enterprise Server
```

**Solutions:**

1. **Verify GitHub Enterprise Server is reachable**

   ```bash
   curl https://github.example.com/api/v3/meta
   ```

   Expected response: JSON with server metadata

2. **Check DNS resolution**

   ```bash
   nslookup github.example.com
   ```

3. **Verify network connectivity from Kubernetes**

   ```bash
   kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
     curl https://github.example.com/api/v3/meta
   ```

4. **Check proxy settings**
   - Ensure HTTP_PROXY and HTTPS_PROXY are set if needed
   - Verify NO_PROXY excludes your GitHub Enterprise Server

5. **Increase connectivity timeout**
   ```yaml
   # In values.yaml
   configMap:
     data:
       CONNECTIVITY_TIMEOUT: "10" # Increase from default 5 seconds
       CONNECTIVITY_MAX_RETRIES: "5" # Increase from default 3
   ```

### Slow Response Times

**Symptoms:**

- Logs show high `response_time_ms` values
- Scanner takes longer than expected

**Check:**

```bash
# View response times in logs
kubectl logs <pod-name> -n gss | jq '.fields.response_time_ms'
```

**Solutions:**

- Increase `REQUEST_TIMEOUT` if operations are timing out
- Reduce `CONCURRENT_SCANS` to lower API load
- Check GitHub Enterprise Server performance
- Verify network latency between Kubernetes and GHES

## Authentication Problems

### Invalid Token Error

**Symptoms:**

```
Failed to connect to GitHub Enterprise Server: Unauthorized
```

**Solutions:**

1. **Verify token format**

   ```bash
   # Token should start with 'ghp_' or 'gho_'
   echo $GITHUB_TOKEN | cut -c1-4
   ```

2. **Check token permissions**

   ```bash
   curl -H "Authorization: token $GITHUB_TOKEN" \
     https://github.example.com/api/v3/user
   ```

   Should return your user information

3. **Verify token has required scopes**
   Required scopes: `repo` (Full control of private repositories)

   ```bash
   curl -I -H "Authorization: token $GITHUB_TOKEN" \
     https://github.example.com/api/v3/user | grep X-OAuth-Scopes
   ```

4. **Check token expiration**
   - Personal Access Tokens may have expiration dates
   - Generate a new token if expired

5. **Verify secret in Kubernetes**
   ```bash
   kubectl get secret ghes-schedule-scanner-secret -n gss -o yaml
   # Decode the token
   kubectl get secret ghes-schedule-scanner-secret -n gss \
     -o jsonpath='{.data.GITHUB_TOKEN}' | base64 -d
   ```

## Timeout Errors

### Request Timeout During Scanning

**Symptoms:**

```
Timeout getting workflow runs
Timeout getting workflow file
```

**Solutions:**

1. **Increase REQUEST_TIMEOUT**

   ```yaml
   # In values.yaml
   configMap:
     data:
       REQUEST_TIMEOUT: "120" # Increase from default 60 seconds
   ```

2. **Reduce concurrent scans**

   ```yaml
   configMap:
     data:
       CONCURRENT_SCANS: "5" # Reduce from default 10
   ```

3. **Check GitHub API rate limits**

   ```bash
   curl -H "Authorization: token $GITHUB_TOKEN" \
     https://github.example.com/api/v3/rate_limit
   ```

4. **Monitor timeout occurrences**
   ```bash
   kubectl logs <pod-name> -n gss | grep -i timeout
   ```

## Performance Issues

### Scanning Takes Too Long

**Symptoms:**

- Scan duration exceeds expectations
- Job timeout in Kubernetes

**Solutions:**

1. **Increase concurrent scans** (if API rate limits allow)

   ```yaml
   configMap:
     data:
       CONCURRENT_SCANS: "20" # Increase from default 10
   ```

2. **Exclude unnecessary repositories**

   ```yaml
   excludedRepositoriesList:
     - archived-repo-1
     - archived-repo-2
     - test-repo
   ```

3. **Adjust Kubernetes job timeout**

   ```yaml
   # In cronjob.yaml
   spec:
     activeDeadlineSeconds: 3600 # 1 hour
   ```

4. **Check resource limits**
   ```yaml
   resources:
     limits:
       cpu: "200m" # Increase if CPU-bound
       memory: "256Mi" # Increase if memory-bound
   ```

### High Memory Usage

**Check current usage:**

```bash
kubectl top pod -n gss
```

**Solutions:**

- Reduce `CONCURRENT_SCANS` to lower memory footprint
- Increase memory limits in Helm chart
- Check for memory leaks in logs

## Kubernetes Deployment Issues

### CronJob Not Running

**Check CronJob status:**

```bash
kubectl get cronjob -n gss
kubectl describe cronjob ghes-schedule-scanner -n gss
```

**Check recent jobs:**

```bash
kubectl get jobs -n gss
kubectl get pods -n gss
```

**Solutions:**

1. **Verify schedule format**

   ```yaml
   schedule: "0 1 * * *" # Valid cron format
   ```

2. **Check suspend status**

   ```bash
   kubectl get cronjob ghes-schedule-scanner -n gss -o yaml | grep suspend
   ```

3. **Manually trigger job**
   ```bash
   kubectl create job --from=cronjob/ghes-schedule-scanner manual-run -n gss
   ```

### Pod Crash or Restart

**Check pod status:**

```bash
kubectl get pods -n gss
kubectl describe pod <pod-name> -n gss
```

**View logs:**

```bash
# Current pod logs
kubectl logs <pod-name> -n gss

# Previous pod logs (if crashed)
kubectl logs <pod-name> -n gss --previous
```

**Common causes:**

- Missing required environment variables
- Invalid configuration
- Out of memory (OOMKilled)
- Network connectivity issues

### ImagePullBackOff

**Symptoms:**

```
Failed to pull image: unauthorized or not found
```

**Solutions:**

1. **Verify image exists**

   ```bash
   docker pull ghcr.io/containerelic/gss:latest
   ```

2. **Check image pull secrets** (if using private registry)

   ```bash
   kubectl get secrets -n gss
   ```

3. **Verify image tag in values.yaml**
   ```yaml
   image:
     repository: ghcr.io/containerelic/gss
     tag: "1.0.0" # Or latest
   ```

## Logging and Debugging

### Enable Debug Logging

**Local development:**

```bash
export LOG_LEVEL=debug
cargo run --release
```

**Kubernetes:**

```yaml
# In values.yaml
configMap:
  data:
    LOG_LEVEL: "debug"
```

### View Logs in Kubernetes

**Real-time logs:**

```bash
kubectl logs -f <pod-name> -n gss
```

**Filter logs by level:**

```bash
kubectl logs <pod-name> -n gss | jq 'select(.level == "ERROR")'
kubectl logs <pod-name> -n gss | jq 'select(.level == "WARN")'
```

**View configuration at startup:**

```bash
kubectl logs <pod-name> -n gss | jq 'select(.fields.message == "Configuration loaded")'
```

**View response times:**

```bash
kubectl logs <pod-name> -n gss | jq 'select(.fields.response_time_ms != null)'
```

**View scan results:**

```bash
kubectl logs <pod-name> -n gss | jq 'select(.fields.message | contains("Scan completed"))'
```

### Debug with Local Build

**Run with backtrace:**

```bash
RUST_BACKTRACE=1 cargo run --release
```

**Run with full backtrace:**

```bash
RUST_BACKTRACE=full cargo run --release
```

**Use debugger:**

```bash
rust-gdb target/debug/ghes-schedule-scanner
# or
rust-lldb target/debug/ghes-schedule-scanner
```

## Slack Integration Issues

### Slack Canvas Not Updating

**Symptoms:**

- No error in logs but Canvas not updated
- Old data showing in Canvas

**Solutions:**

1. **Verify Slack token format**

   ```bash
   # Token should start with 'xoxb-'
   echo $SLACK_TOKEN | cut -c1-5
   ```

2. **Check Slack API permissions**
   Required scopes:
   - `chat:write`
   - `channels:read`
   - `files:read`
   - `files:write`

3. **Verify Canvas ID**

   ```bash
   # Canvas URL format: https://workspace.slack.com/docs/CHANNEL_ID/CANVAS_ID
   # Extract CANVAS_ID from URL
   ```

4. **Test Slack API connection**

   ```bash
   curl -X POST https://slack.com/api/auth.test \
     -H "Authorization: Bearer $SLACK_TOKEN"
   ```

5. **Check Slack API errors in logs**
   ```bash
   kubectl logs <pod-name> -n gss | jq 'select(.fields.error | contains("Slack"))'
   ```

### Slack Token Validation Error

**Symptoms:**

```
SLACK_TOKEN must start with 'xoxb-' (Bot User OAuth Token)
```

**Solution:**

- Use Bot User OAuth Token, not User OAuth Token
- Token must start with `xoxb-`
- Get token from: https://api.slack.com/apps → Your App → OAuth & Permissions

## Getting Help

If you continue to experience issues:

1. **Enable debug logging** and collect relevant logs

   ```yaml
   # In values.yaml
   configMap:
     data:
       LOG_LEVEL: "debug"
   ```

2. **Gather system information**:
   - Kubernetes version: `kubectl version`
   - Helm version: `helm version`
   - Application version: Check image tag or logs
   - Configuration: Review values.yaml and ConfigMap
   - Error logs with context

3. **Analyze logs systematically**:

   ```bash
   # Check all error logs
   kubectl logs <pod-name> -n gss | jq 'select(.level == "ERROR")'

   # Check warnings
   kubectl logs <pod-name> -n gss | jq 'select(.level == "WARN")'

   # Check configuration
   kubectl logs <pod-name> -n gss | jq 'select(.fields.message == "Configuration loaded")'
   ```

4. **Review documentation**:
   - [Installation Guide](./installation.md)
   - [README.md](../README.md)
   - [CLAUDE.md](../CLAUDE.md)
