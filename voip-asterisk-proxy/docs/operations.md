# voip-asterisk-proxy — Operations

## Common Failure Modes

| Symptom | Likely cause | Resolution |
|---------|-------------|------------|
| ARI WebSocket connection errors in logs, events stop flowing | Asterisk not yet started or ARI service not enabled | Verify Asterisk is running; check `--ari_address`; ensure `ari.conf` enables HTTP and the correct application |
| AMI TCP connection failures | Wrong AMI host/port/credentials; `manager.conf` not enabled | Check `--ami_host`, `--ami_port`, `--ami_username`, `--ami_password`; verify `manager.conf` in Asterisk |
| RabbitMQ consumer not receiving requests | Wrong queue name or RabbitMQ address | Verify `--rabbitmq_queue_listen` matches what upstream services publish to; check connectivity |
| Pod annotation patch failing | Not running in Kubernetes or service account lacks patch permission | Set `--kubernetes_disabled=true` for non-k8s environments; grant `patch` verb on pods to service account |
| Recording upload fails | GCS bucket not configured or ADC not available | Set `--recording_bucket_name`; ensure the pod's service account has `roles/storage.objectCreator` on the bucket |
| Redis write errors | Wrong `--redis_address` or Redis unavailable | Verify connectivity; service continues without Redis but upstream routing by address will fail |
| Proxy responds with 400 to all RPC requests | URI pattern not matching — wrong prefix in request | Confirm request URI starts with `/ari/`, `/ami/`, or `/proxy/recording_file_move` |

## Debugging Guide

### Check ARI connection health

```bash
kubectl logs -n voip deploy/<asterisk-proxy-pod> | grep "eventARIRun\|ARI\|ari"
```

ARI reconnects every 1 second on failure. Repeated reconnect messages indicate Asterisk is not ready.

### Verify AMI connection

```bash
kubectl logs -n voip deploy/<asterisk-proxy-pod> | grep "amigo\|AMI\|ami"
```

### Test ARI proxy manually

```bash
# Port-forward to the Asterisk ARI port (not the proxy itself)
kubectl port-forward -n voip pod/<asterisk-pod> 8088:8088

# Test directly against Asterisk
curl -u asterisk:asterisk http://localhost:8088/ari/asterisk/info
```

### Check Prometheus metrics

```bash
kubectl port-forward -n voip pod/<asterisk-proxy-pod> 2112:2112
curl -s http://localhost:2112/metrics
```

### Inspect Redis address registration

```bash
redis-cli -h <redis-host> get "asterisk.<mac-address>.address-internal"
```

Should return the pod's internal IPv4 address. If empty, check Redis connectivity and logs.

### Check Kubernetes annotation

```bash
kubectl get pod <asterisk-proxy-pod> -n voip -o jsonpath='{.metadata.annotations.asterisk-id}'
```

Should return the MAC address of the configured network interface.

## Configuration

All parameters can be set via command-line flags or environment variables. Flags take precedence.

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--ari_address` | `ARI_ADDRESS` | `localhost:8088` | Asterisk ARI HTTP server address |
| `--ari_account` | `ARI_ACCOUNT` | `asterisk:asterisk` | ARI credentials (`user:password`) |
| `--ari_subscribe_all` | `ARI_SUBSCRIBE_ALL` | `true` | Subscribe to all ARI event types |
| `--ari_application` | `ARI_APPLICATION` | `voipbin` | ARI Stasis application name |
| `--ami_host` | `AMI_HOST` | `127.0.0.1` | Asterisk AMI host |
| `--ami_port` | `AMI_PORT` | `5038` | Asterisk AMI port |
| `--ami_username` | `AMI_USERNAME` | `asterisk` | AMI username |
| `--ami_password` | `AMI_PASSWORD` | `asterisk` | AMI password |
| `--ami_event_filter` | `AMI_EVENT_FILTER` | `` | Comma-separated AMI event types to forward (empty = all) |
| `--interface_name` | `INTERFACE_NAME` | `eth0` | Network interface for MAC-based identity |
| `--rabbitmq_address` | `RABBITMQ_ADDRESS` | `amqp://guest:guest@localhost:5672` | RabbitMQ server |
| `--rabbitmq_queue_listen` | `RABBITMQ_QUEUE_LISTEN` | `asterisk.call.request` | Permanent RabbitMQ listen queue |
| `--redis_address` | `REDIS_ADDRESS` | `localhost:6379` | Redis server |
| `--redis_password` | `REDIS_PASSWORD` | `` | Redis password |
| `--redis_database` | `REDIS_DATABASE` | `1` | Redis database index |
| `--prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Prometheus metrics path |
| `--prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Prometheus listen address |
| `--recording_bucket_name` | `RECORDING_BUCKET_NAME` | `` | GCS bucket for recordings |
| `--recording_asterisk_directory` | `RECORDING_ASTERISK_DIRECTORY` | `/var/spool/asterisk/recording` | Local Asterisk recording directory |
| `--recording_bucket_directory` | `RECORDING_BUCKET_DIRECTORY` | `/mnt/media/recording` | GCS path prefix for uploads |
| `--kubernetes_disabled` | `KUBERNETES_DISABLED` | `false` | Disable Kubernetes annotation patching |

## Prometheus Metrics

Metrics are served at `<prometheus_listen_address><prometheus_endpoint>` (default: `:2112/metrics`).

The proxy uses the standard `bin-common-handler` metric namespace. Check `pkg/` packages for any service-specific counters. The Prometheus listener is initialized in `cmd/asterisk-proxy/init.go` via `initProm()`.

Scrape configuration (add to Kubernetes pod annotations):

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "2112"
  prometheus.io/path: "/metrics"
```
