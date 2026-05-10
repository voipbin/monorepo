# bin-sentinel-manager — Operations

## Common Failure Modes

| Symptom | Likely cause | Resolution |
|---------|-------------|------------|
| Service exits immediately at startup | `rest.InClusterConfig()` fails — not running inside a Kubernetes cluster | Deploy to cluster; do not run locally without a kubeconfig override |
| Informer never delivers events | RBAC misconfigured — pod-reader Role or RoleBinding missing in `voip` namespace | Apply Role + RoleBinding (see Kubernetes Deployment section of CLAUDE.md) |
| RabbitMQ publish errors in logs | RabbitMQ unreachable or wrong `RABBITMQ_ADDRESS` | Verify connectivity; check address format `amqp://user:pass@host:5672` |
| No `pod_state_change_total` increments in Prometheus | Pods in watched namespaces are stable (no updates/deletes) — may be normal | Cross-check by triggering a pod restart and watching the counter |
| Downstream services not cleaning up hung calls | Sentinel is running but publishing to wrong queue or consumer not subscribed | Verify queue name matches `QueueNameSentinelEvent` constant in bin-common-handler |

## Debugging Guide

### Check that the informer started

Sentinel logs `"Starting pod informer"` for each (namespace, selector) pair at startup. If these log lines are absent, the informer loop never launched.

```bash
kubectl logs -n bin-manager deploy/sentinel-manager | grep "Starting pod informer"
```

### Verify events are being published

Trigger a pod update (e.g., restart an asterisk-call pod) and watch for publish log lines:

```bash
kubectl logs -n bin-manager deploy/sentinel-manager -f | grep "Pod updated\|Pod deleted"
```

### Inspect Prometheus metrics

```bash
kubectl port-forward -n bin-manager deploy/sentinel-manager 2112:2112
curl -s http://localhost:2112/metrics | grep pod_state_change_total
```

Expected output format:

```
sentinel_manager_pod_state_change_total{namespace="voip",pod="asterisk-call",state="updated"} 12
sentinel_manager_pod_state_change_total{namespace="voip",pod="asterisk-call",state="deleted"} 1
```

### Use the sentinel-control CLI

The `sentinel-control` binary can query pod state for debugging without waiting for events:

```bash
# List monitored pods in the voip namespace
./bin/sentinel-control pod list --namespace voip

# Get a specific pod
./bin/sentinel-control pod get --namespace voip --name asterisk-call-abc123
```

All output is JSON on stdout; logs go to stderr.

### RBAC troubleshooting

If the informer fails to list pods, check the service account permissions:

```bash
kubectl auth can-i list pods -n voip --as=system:serviceaccount:bin-manager:sentinel-manager
```

Should return `yes`. If not, the RoleBinding is missing or points to the wrong namespace.

## Configuration

All parameters can be set via command-line flags or environment variables. Flags take precedence.

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--rabbitmq_address` | `RABBITMQ_ADDRESS` | `amqp://guest:guest@localhost:5672` | RabbitMQ server address |
| `--prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Prometheus metrics path |
| `--prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Address/port for Prometheus scraping |

No database or Redis configuration is required — sentinel is stateless beyond the Kubernetes watch stream.

## Prometheus Metrics

Metrics are served at `<prometheus_listen_address><prometheus_endpoint>` (default: `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `sentinel_manager_pod_state_change_total` | Counter | `namespace`, `pod` (app label value), `state` (`updated`\|`deleted`) | Total number of pod state changes observed and published |

The `pod` label contains the value of the Kubernetes `app` label on the pod (e.g., `asterisk-call`), not the pod name. This allows aggregation across all replicas of the same workload.

Scrape configuration (already set in `k8s/deployment.yml`):

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "2112"
  prometheus.io/path: "/metrics"
```
