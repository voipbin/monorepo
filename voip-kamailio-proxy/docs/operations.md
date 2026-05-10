# voip-kamailio-proxy — Operations

## Common Failure Modes

### RabbitMQ connection refused

**Symptom:** Service exits immediately after startup with a connection error.

**Cause:** RabbitMQ is not reachable at `RABBITMQ_ADDRESS`.

**Resolution:**
1. Verify the address: `echo $RABBITMQ_ADDRESS`
2. Confirm RabbitMQ is running and the AMQP port (5672) is reachable from the pod.
3. Check for credential mismatch (default: `amqp://guest:guest@localhost:5672`).

### Interface not found / no MAC address

**Symptom:** Service exits at startup with `Could not get kamailio ID from interface`.

**Cause:** The interface named by `--interface_name` does not exist or has no hardware address.

**Resolution:**
1. Check available interfaces: `ip link show`
2. Confirm the pod's primary interface is `eth0` (or set `INTERFACE_NAME` appropriately).
3. In test environments, virtual interfaces (e.g., `lo`) have no MAC address — use a real interface.

### SIP health check always returns unhealthy

**Symptom:** POST `/v1/providers/health` returns `{"status":"unhealthy","result_code":"timeout"}` for all hostnames.

**Possible causes:**
- UDP port 5060 blocked by firewall or network policy.
- DNS resolution failure for the hostname.
- SIP_TIMEOUT too short; the provider responds slowly.

**Resolution:**
1. Test manually: `nc -u <hostname> 5060` or `tcpdump -i eth0 udp port 5060`
2. Increase timeout: `SIP_TIMEOUT=10s`
3. Verify DNS: `nslookup <hostname>` from within the pod.

### Queue not being consumed

**Symptom:** Requests pile up in RabbitMQ; no responses returned.

**Cause:** `listenRun` goroutine exited silently after a queue declare error.

**Resolution:**
1. Check logs for `Could not declare permanent queue` or `Could not declare volatile queue`.
2. Verify RabbitMQ permissions allow queue declare on the vhost.

## Debugging Guide

### View live logs

```bash
kubectl logs -f <pod-name> -c kamailio-proxy
```

The service logs at DEBUG level by default (joonix/fluentd JSON format).

### Check which queues are active

```bash
rabbitmqctl list_queues name messages consumers | grep kamailio
```

Look for:
- `voip.kamailio.request` — permanent queue
- `voip.kamailio.<mac>.request` — volatile queue (one per running instance)

### Simulate a health check via RabbitMQ

Using `rabbitmqadmin` or any AMQP client, publish to `voip.kamailio.request`:

```json
{
  "method": "POST",
  "uri": "/v1/providers/health",
  "data": {"hostname": "sip.example.com"}
}
```

### Check Prometheus metrics

```bash
curl http://<pod-ip>:2112/metrics
```

Default endpoint: `:2112/metrics`.

## Configuration

All configuration is read from environment variables (or CLI flags of the same name).

| Env var | Flag | Default | Description |
|---------|------|---------|-------------|
| `RABBITMQ_ADDRESS` | `--rabbitmq_address` | `amqp://guest:guest@localhost:5672` | RabbitMQ connection URL |
| `RABBITMQ_QUEUE_LISTEN` | `--rabbitmq_queue_listen` | `voip.kamailio.request` | Permanent RabbitMQ queue name |
| `INTERFACE_NAME` | `--interface_name` | `eth0` | Network interface used to derive the instance MAC |
| `PROMETHEUS_ENDPOINT` | `--prometheus_endpoint` | `/metrics` | Prometheus scrape path |
| `PROMETHEUS_LISTEN_ADDRESS` | `--prometheus_listen_address` | `:2112` | Prometheus HTTP listener address |
| `SIP_TIMEOUT` | `--sip_timeout` | `5s` | UDP read deadline for SIP OPTIONS checks |

Configuration is loaded via Viper. Both env vars and flags are supported; env vars take precedence over flag defaults.

## Prometheus Metrics

The service exposes the default Go runtime metrics via `promhttp.Handler()` at `PROMETHEUS_LISTEN_ADDRESS/PROMETHEUS_ENDPOINT`.

No service-specific counters or histograms are defined. Available built-in metrics include:

| Metric | Description |
|--------|-------------|
| `go_goroutines` | Number of running goroutines |
| `go_memstats_*` | Go heap and GC statistics |
| `process_cpu_seconds_total` | Process CPU usage |
| `process_open_fds` | Open file descriptors |
