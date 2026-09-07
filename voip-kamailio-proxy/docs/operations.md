# voip-kamailio-proxy — Operations

## Common Failure Modes

### RabbitMQ connection refused

**Symptom:** Container keeps running, but logs repeat `Could not connect to rabbitmq. Will retry again after 1 sec` and no queue consumer is ever registered. The RabbitMQ handler retries the connection forever at 1s intervals (`bin-common-handler/pkg/rabbitmqhandler/main.go:250`); the service never exits over this.

**Cause:** RabbitMQ is not reachable at `RABBITMQ_ADDRESS`.

**Resolution:**
1. Verify the address: `echo $RABBITMQ_ADDRESS`
2. Confirm RabbitMQ is running and the AMQP port (5672) is reachable from the container.
3. Check for credential mismatch (default: `amqp://guest:guest@localhost:5672`).

### Interface not found / no MAC address

**Symptom:** Service exits at startup with `Could not get kamailio ID from interface`.

**Cause:** The interface named by `--interface_name` does not exist or has no hardware address.

**Resolution:**
1. Check available interfaces: `ip link show`
2. Confirm the container's primary interface is `eth0` (or set `INTERFACE_NAME` appropriately).
3. In test environments, virtual interfaces (e.g., `lo`) have no MAC address — use a real interface.

### SIP health check always returns unhealthy

**Symptom:** POST `/v1/providers/health` returns `{"status":"unhealthy","result_code":"timeout"}` for all hostnames.

**Possible causes:**
- UDP port 5060 blocked by firewall or network policy.
- DNS resolution failure for the hostname.
- SIP_TIMEOUT too short; the provider responds slowly.

**Resolution:**
1. Test manually from somewhere that has tooling. The service image is
   `gcr.io/distroless/static-debian12` (`Dockerfile:12`): no shell, no `nc`, no
   `nslookup`, so `docker exec` into it is not an option. Use a throwaway
   container on the same network instead:
   `docker run --rm --network production nicolaka/netshoot nc -u <hostname> 5060`
   Or capture on the host: `tcpdump -i any udp port 5060`
2. Increase timeout, but keep it below the caller's RPC deadline: `SIP_TIMEOUT=8s`. bin-route-manager gives the RPC 10s (`bin-common-handler/pkg/requesthandler/main.go:152`), so a probe budget of 10s or more turns a slow provider into an RPC timeout at the caller instead of an unhealthy verdict.
3. Verify DNS the same way:
   `docker run --rm --network production nicolaka/netshoot nslookup <hostname>`

### Queue not being consumed

**Symptom:** Requests pile up in RabbitMQ; no responses returned.

**Cause:** `listenRun` goroutine exited silently after a queue declare error.

**Resolution:**
1. Check logs for `Could not declare permanent queue` or `Could not declare volatile queue`.
2. Verify RabbitMQ permissions allow queue declare on the vhost.

## Debugging Guide

### View live logs

```bash
docker ps --filter name=kamailio-proxy
docker logs -f <container-name>
```

The service logs at DEBUG level by default (joonix/fluentd JSON format).

### Check which queues are active

```bash
docker exec infra-rabbitmq rabbitmqctl list_queues name messages consumers \
  | grep kamailio
```

`rabbitmqctl` is not installed on the host; the broker runs in the
`infra-rabbitmq` container.

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
curl http://<container-ip>:2112/metrics
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
