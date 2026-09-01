# bin-sentinel-manager — Operations

## Common Failure Modes

| Symptom | Likely cause | Resolution |
|---------|-------------|------------|
| Service exits immediately at startup, crash-loops | The docker-socket-proxy sidecar is unreachable or denies `/containers/json` | Check the proxy container is running and healthy; confirm `CONTAINERS=1` in its env. This exit is intentional (fail-loud), not a bug |
| Service exits at startup with a Redis error | `REDIS_ADDRESS` wrong, or Redis unreachable from the `production` network | Verify connectivity; `Connect()` pings before the watcher starts |
| Service crash-loops after running fine for a while, log says "could not be established after N consecutive attempts" | The socket proxy died or became unreachable post-boot | Intentional: sentinel refuses to keep running blind. Check the proxy sidecar's health; `sentinel_manager_container_event_stream_reconnect_total{result="empty"}` shows how long it had been failing |
| `sentinel_manager_container_event_stream_reconnect_total{result="empty"}` rising but the service stays up | The stream is flapping — reconnecting successfully often enough to reset the consecutive counter | Check the proxy sidecar and the host's docker daemon; events may be being lost in the reconnect gaps |
| `sentinel_manager_container_asterisk_id_conflict_total` incrementing | A refresh pass found a different asterisk-id for an already-resolved container. Most likely a die+start pair was lost in an event-stream gap and the replacement reused the same static IP | The id sentinel is holding is probably the DEAD generation's, so the next death for that container will publish a wrong asterisk-id. Cross-check the reconnect panel for a matching gap; restarting sentinel re-seeds that entry cleanly from boot reconciliation |
| No events published, no errors in the log | The container-name prefixes no longer match reality (e.g. the Compose project or service was renamed) | Compare the live container names against `watchedContainerPrefixes` in `pkg/dockerwatchhandler/main.go`; the match requires a bare replica index after the prefix |
| `container_died` events publish with an empty `asterisk_id` | The asterisk-id never resolved — most often a Redis DB mismatch, a container that never registered, or an IP that resolves from an unexpected network | Check `sentinel_manager_container_unresolved_asterisk_id_total`; confirm `REDIS_DATABASE=1` matches voip-asterisk-proxy's; verify `asterisk.*.address-internal` keys exist and their values match the containers' `production`-network IPs |
| `sentinel_manager_container_asterisk_id_refresh_miss_total` climbing | A container's Redis key has gone stale while the container is still alive (the proxy sidecar stopped refreshing it) | Leading indicator — the id is still held (sticky last-known) but the NEXT recreation will re-resolve from nothing. Check that container's `-proxy` sidecar |
| RabbitMQ publish errors in logs | RabbitMQ unreachable or wrong `RABBITMQ_ADDRESS` | Verify connectivity; check the address format `amqp://user:pass@host:5672` |
| Downstream calls not recovered after an Asterisk container died | Sentinel published but call-manager filtered or guarded the event | Check the event's `service` (must be `asterisk-call`) and `asterisk_id` (must be non-empty); then check call-manager's `EventSMContainerDied` logs |
| Repeated deaths stop producing events | Flap damping engaged (>3 deaths / 60s for that container) | Expected. Find and fix the crash-loop; the window drains on its own |

## Debugging Guide

### Check that the watcher started

```bash
docker logs bin-sentinel-manager-sentinel-manager-1 | grep "Completed the boot-time reconciliation"
docker logs bin-sentinel-manager-sentinel-manager-1 | grep "Opening the docker event stream"
```

The reconciliation line reports how many containers were seeded. A `seeded: 0` on a host that is running Asterisk containers means the name-prefix match is failing.

### Watch resolution happen

```bash
docker logs -f bin-sentinel-manager-sentinel-manager-1 | grep "Resolved the asterisk id"
```

### Verify events are being published

```bash
docker logs -f bin-sentinel-manager-sentinel-manager-1 | grep "Container started\|Container died"
```

### Inspect Prometheus metrics

```bash
curl -s http://<host>:2112/metrics | grep sentinel_manager_container
```

Expected output format:

```
sentinel_manager_container_state_change_total{container_name="voip-asterisk-call-docker-1",service="asterisk-call",state="started"} 3
sentinel_manager_container_state_change_total{container_name="voip-asterisk-call-docker-1",service="asterisk-call",state="died"} 3
sentinel_manager_container_unresolved_asterisk_id_total{container_name="voip-asterisk-call-docker-1"} 1
sentinel_manager_container_asterisk_id_refresh_miss_total{container_name="voip-asterisk-call-docker-2"} 0
```

### Use the sentinel-control CLI

`sentinel-control` queries the same read-only socket proxy without waiting for events. All output is JSON on stdout; logs go to stderr.

```bash
# list containers visible through the proxy, filtered by name substring
./bin/sentinel-control container list --name voip-asterisk-call-docker

# inspect one container (includes per-network IP and MAC — the MAC is what the
# asterisk-id derives from)
./bin/sentinel-control container get --name voip-asterisk-call-docker-1
```

### Verify the socket proxy's ACL

The proxy is the security boundary. A mutating call must be refused:

```bash
# from a container on the private docker-socket network
wget -q -O - http://sentinel-docker-socket-proxy:2375/_ping           # -> OK
wget -q -O - --post-data= http://sentinel-docker-socket-proxy:2375/containers/create   # -> 403
```

### Cross-check the Redis side

```bash
redis-cli -n 1 --scan --pattern 'asterisk.*.address-internal'
redis-cli -n 1 ttl 'asterisk.<asterisk-id>.address-internal'
```

A remaining TTL below 23h48m means the key is stale by sentinel's freshness rule and will not resolve a new id (though an already-resolved one is kept).

## Configuration

All parameters can be set via command-line flags or environment variables. Flags take precedence.

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--rabbitmq_address` | `RABBITMQ_ADDRESS` | `amqp://guest:guest@localhost:5672` | RabbitMQ server address |
| `--docker_socket_proxy_address` | `DOCKER_SOCKET_PROXY_ADDRESS` | `tcp://sentinel-docker-socket-proxy:2375` | Read-only docker-socket-proxy endpoint. **Never** point this at `unix:///var/run/docker.sock` |
| `--redis_address` | `REDIS_ADDRESS` | `localhost:6379` | Redis server address |
| `--redis_password` | `REDIS_PASSWORD` | (empty) | Redis password |
| `--redis_database` | `REDIS_DATABASE` | `1` | Redis database index. Must match voip-asterisk-proxy's |
| `--prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Prometheus metrics path |
| `--prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Address/port for Prometheus scraping |

No database configuration is required — sentinel's only persistent-store access is the read-only Redis scan.

## Deployment

`komodo/docker-compose.yml` defines two containers:

| Service | Networks | Purpose |
|---|---|---|
| `sentinel-manager` | `production` + `docker-socket` | The watcher. Single replica |
| `sentinel-docker-socket-proxy` | `docker-socket` only | Read-only Docker API proxy, digest-pinned `tecnativa/docker-socket-proxy:v0.5.0` |

`docker-socket` is `internal: true` and joined by nothing else, which is what bounds the blast radius of `CONTAINERS=1`. That permission is path-prefix based rather than per-endpoint, so it exposes far more than the single inspect call sentinel makes: for *any* container on the host, a client that reaches the proxy can read `/containers/{id}/json` (config plus every env-var secret), `/containers/{id}/archive` (arbitrary files from the container filesystem), `/containers/{id}/export`, `/containers/{id}/logs`, and `/containers/{id}/attach/ws` — effectively read access to all container data on bm-nyc-01. It is read-only (no mutating family is granted), but network scope is the whole mitigation. This mirrors the sidecar shape monorepo-etc's `infra-prometheus` and `infra-loki` already use; do not move the proxy onto `production`.

CI deploys via the `bin-sentinel-manager-deploy` job in `.circleci/config_work.yml`, which renders the image tag and pushes the compose file through `komodo-api-deploy.sh`.

## Prometheus Metrics

Metrics are served at `<prometheus_listen_address><prometheus_endpoint>` (default `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `sentinel_manager_container_state_change_total` | Counter | `container_name`, `service`, `state` (`started`\|`died`) | Watched container state changes observed and published |
| `sentinel_manager_container_unresolved_asterisk_id_total` | Counter | `container_name` | `container_died` events published without a resolved asterisk-id. Each increment is one recovery that will NOT happen |
| `sentinel_manager_container_asterisk_id_refresh_miss_total` | Counter | `container_name` | Refresh passes that found no fresh address for an already-resolved container. Leading indicator: the id is kept, but the next death may go unrecovered |
| `sentinel_manager_container_asterisk_id_conflict_total` | Counter | `container_name` | A refresh pass resolved a *different* asterisk-id for a container that already had one, and kept the existing id. Structurally should never fire. The plausible real cause is a missed die+start pair — in which case the kept id is the dead generation's and the next death publishes a wrong one |
| `sentinel_manager_container_event_stream_reconnect_total` | Counter | `result` (`delivered`\|`idle`\|`empty`) | Docker event stream attempts that ended. `delivered` and `idle` (no events but survived past 30s, i.e. a proxy restart on an idle fleet) are both normal and reset the give-up budget. A rising `empty` rate — attempts ending immediately — means the socket proxy is unreachable, i.e. sentinel is up but watching nothing. `maxConsecutiveEmptyStreams` (20, ≈1 min) consecutive `empty` results exit the process |

Note the rename: `sentinel_manager_pod_state_change_total` (labels `namespace`, `pod`) is gone as of VOIP-1418. `monitoring/grafana/dashboards/sentinel-manager.json` at the repo root was updated in the same change.

Also relevant, from the shared notifyhandler: `sentinel_manager_topic_placeholder_total` no longer tracks `topic_publish_total{ok}` one-for-one. Sentinel now publishes real subscription addresses for resolved deaths; the placeholder is expected for every `container_started` and for unresolved deaths only.
