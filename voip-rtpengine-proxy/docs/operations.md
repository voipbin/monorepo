# voip-rtpengine-proxy — Operations

## Common Failure Modes

### RabbitMQ connection refused

**Symptom:** Service exits immediately at startup.

**Cause:** RabbitMQ not reachable at `RABBITMQ_ADDRESS`.

**Resolution:**
1. Verify `RABBITMQ_ADDRESS` is set correctly.
2. Check AMQP port 5672 is reachable from the pod.
3. Confirm credentials in the connection URL.

### No IPv4 address on interface

**Symptom:** `Could not get proxy ID from interface eth0` at startup; service exits.

**Cause:** `INTERFACE_NAME` references an interface with no IPv4 address, or the interface does not exist.

**Resolution:**
1. List interfaces: `ip addr show`
2. Set `INTERFACE_NAME` to the correct interface (typically `eth0` in Kubernetes pods).

### Redis registration fails

**Symptom:** `Could not register proxy in Redis` at startup; service exits.

**Cause:** Redis not reachable at `REDIS_ADDRESS`, wrong password, or wrong database index.

**Resolution:**
1. Verify `REDIS_ADDRESS`, `REDIS_PASSWORD`, and `REDIS_DATABASE`.
2. Test manually: `redis-cli -h <host> -a <password> -n <db> ping`

### RTPEngine NG commands time out

**Symptom:** `ng` commands return 500 with `RTPEngine NG timeout after 5s`.

**Cause:** RTPEngine is not running, the NG port is wrong, or the daemon is overloaded.

**Resolution:**
1. Check RTPEngine status: `systemctl status rtpengine` (or equivalent).
2. Verify `RTPENGINE_NG_ADDRESS` points to the correct host and UDP port (default `127.0.0.1:22222`).
3. Increase timeout: `RTPENGINE_NG_TIMEOUT=10s`
4. Check RTPEngine logs for errors.

### `exec` returns "max concurrent captures reached"

**Symptom:** 500 response with `max concurrent captures reached (20)`.

**Cause:** 20 tcpdump processes are already running.

**Resolution:**
1. Issue `kill` commands for stale captures that were not properly terminated.
2. If processes are orphaned, restart the proxy (orphans are cleaned on startup via `CleanOrphans()`).
3. If 20 concurrent captures is a real bottleneck, the limit is hardcoded in `processmanager/manager.go` (`defaultMaxConcurrent = 20`).

### GCS upload fails silently

**Symptom:** Pcap files accumulate in `/tmp/` on the pod; GCS bucket is empty.

**Cause:** `GCP_BUCKET_NAME_MEDIA` is set but GCS credentials are missing or the bucket does not exist.

**Resolution:**
1. Verify the service account has `storage.objects.create` on the bucket.
2. Check logs for `GCS upload failed` and `GCS upload retry failed`.
3. Files remain at `/tmp/<id>.pcap` for manual recovery after both upload attempts fail.

## Debugging Guide

### View live logs

```bash
kubectl logs -f <pod-name> -c rtpengine-proxy
```

Logs are in joonix/fluentd JSON format at DEBUG level by default.

### Check RabbitMQ queues

```bash
rabbitmqctl list_queues name messages consumers | grep rtpengine
```

Look for:
- `rtpengine.proxy.request` — permanent queue
- `rtpengine.<ip>.request` — volatile queue (one per running instance)

### Check Redis proxy registration

```bash
redis-cli -n 1 keys 'rtpengine.*'
# Expected: rtpengine.<ip>.address-internal
redis-cli -n 1 get 'rtpengine.<ip>.address-internal'
```

### Send a test NG command

Use `rtpengine-ctl` or a custom bencode UDP client to test the NG port directly:

```bash
# Test RTPEngine is alive (from within the pod)
echo "abcd1234 d7:command4:pinge" | nc -u -w1 127.0.0.1 22222
```

### List active captures

The proxy does not expose an API to list captures. Check running processes on the pod:

```bash
ps aux | grep tcpdump
ls /tmp/*.pcap
```

### Check Prometheus metrics

```bash
curl http://<pod-ip>:2112/metrics
```

## Configuration

All configuration is read from environment variables (or CLI flags of the same name).

| Env var | Flag | Default | Description |
|---------|------|---------|-------------|
| `INTERFACE_NAME` | `--interface_name` | `eth0` | Network interface for proxy IP / identity |
| `RTPENGINE_NG_ADDRESS` | `--rtpengine_ng_address` | `127.0.0.1:22222` | RTPEngine NG UDP endpoint |
| `RTPENGINE_NG_TIMEOUT` | `--rtpengine_ng_timeout` | `5s` | Timeout for NG protocol responses |
| `RABBITMQ_ADDRESS` | `--rabbitmq_address` | `amqp://guest:guest@localhost:5672` | RabbitMQ connection URL |
| `RABBITMQ_QUEUE_LISTEN` | `--rabbitmq_queue_listen` | `rtpengine.proxy.request` | Permanent RabbitMQ queue name |
| `REDIS_ADDRESS` | `--redis_address` | `localhost:6379` | Redis server address |
| `REDIS_PASSWORD` | `--redis_password` | (empty) | Redis password |
| `REDIS_DATABASE` | `--redis_database` | `1` | Redis database index |
| `PROMETHEUS_ENDPOINT` | `--prometheus_endpoint` | `/metrics` | Prometheus scrape path |
| `PROMETHEUS_LISTEN_ADDRESS` | `--prometheus_listen_address` | `:2112` | Prometheus HTTP listener |
| `RTPENGINE_RECORDING_DIR` | `--rtpengine_recording_dir` | (empty) | Directory for RTPEngine's own pcap recordings; enables pcap watcher when set |
| `GCP_BUCKET_NAME_MEDIA` | `--gcp_bucket_name_media` | (empty) | GCS bucket for pcap uploads; GCS upload disabled when unset |

## Prometheus Metrics

The service exposes the default Go runtime metrics via `promhttp.Handler()`.

No service-specific counters or histograms are defined. Available built-in metrics include:

| Metric | Description |
|--------|-------------|
| `go_goroutines` | Number of running goroutines |
| `go_memstats_*` | Go heap and GC statistics |
| `process_cpu_seconds_total` | Process CPU usage |
| `process_open_fds` | Open file descriptors |
