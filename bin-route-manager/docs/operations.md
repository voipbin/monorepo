# bin-route-manager — Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Outbound calls failing with "no route" | Customer has no route + no default (`CustomerIDBasicRoute`) route for target | Add a route for `target=all` under `CustomerIDBasicRoute` as fallback |
| Dialroute returns stale provider | Redis cache not invalidated after provider update | Force cache invalidation; route-manager should invalidate on provider `PUT` |
| Provider calls not tracked | `providercalls` table growth / write errors | Check DB disk space; verify provider call insert in logs |
| All outbound calls going to wrong carrier | Route priority misconfigured | List routes for customer + target; check priority ordering |
| RPC requests timing out | DB connection pool exhaustion | Check `DATABASE_DSN` pool settings; inspect DB slow query log |

## Debugging Guide

**Check service health:**
```bash
curl -s http://<pod-ip>:2112/metrics | grep route_manager
```

**Inspect routes (CLI tool — bypasses RabbitMQ):**
```bash
./bin/route-control route list --customer_id <uuid>
./bin/route-control route list-by-target --customer_id <uuid> --target 1
./bin/route-control route dialroute-list --customer_id <uuid> --target 1
```

**Inspect providers:**
```bash
./bin/route-control route list --customer_id 00000000-0000-0000-0000-000000000001
```

**Check system default routes:**
```sql
-- System-default routes (all customers)
SELECT id, target, provider_id, priority FROM route_manager_routes
WHERE customer_id = '00000000-0000-0000-0000-000000000001'
  AND tm_delete = '9999-01-01 00:00:00.000000'
ORDER BY target, priority;
```

**Check RabbitMQ queue depth:**
```bash
rabbitmqctl list_queues name messages consumers | grep route
```

**Trace dialroute merge:**
```sql
-- Customer routes for target
SELECT r.id, r.target, p.hostname, r.priority FROM route_manager_routes r
JOIN route_manager_providers p ON r.provider_id = p.id
WHERE r.customer_id = '<uuid>' AND r.target IN ('<country_code>', 'all')
  AND r.tm_delete = '9999-01-01 00:00:00.000000'
ORDER BY r.priority;

-- Default routes for same target
SELECT r.id, r.target, p.hostname, r.priority FROM route_manager_routes r
JOIN route_manager_providers p ON r.provider_id = p.id
WHERE r.customer_id = '00000000-0000-0000-0000-000000000001'
  AND r.target IN ('<country_code>', 'all')
  AND r.tm_delete = '9999-01-01 00:00:00.000000'
ORDER BY r.priority;
```

## Deployment

bin-route-manager deploys via Komodo (VOIP-1347 Tier 1 rollout, following the
VOIP-1342/bin-call-manager pilot pattern) instead of the older SSH +
`versions.lock` (`ssh-deploy.sh`) path.

- **Stack definition:** `bin-route-manager/komodo/docker-compose.yml` (git is
  the source of truth for structure; Komodo only executes it on
  request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes
  the built image tag, then `.circleci/scripts/komodo-api-deploy.sh`
  pushes the file's content to Komodo and triggers a deploy, gated
  by the `bin-route-manager-deploy` job's poll/running checks.
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md](../../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md)
  (in the monorepo root, not this service's own `docs/`).
- **Route-manager-specific deviation:** this service runs a background
  health-check ticker (`pkg/healthcheckhandler`) that probes every
  configured carrier via outbound SIP OPTIONS on an interval. During
  old/new container overlap this doubles outbound probe traffic to
  carriers, so — unlike the other Tier 1 services — cut over **promptly**
  once the new container is confirmed healthy at the log level; do not
  leave both containers running for an extended soak. See the design doc's
  cutover section for the full rationale.
- **P23b — standing 2-replica operation:** since the fleet rollout
  (docs/plans/2026-08-28-bin-manager-two-replica-rollout-design.md), this
  service normally runs 2 replicas, guarded by a redsync cross-replica
  lock around the healthcheck cycle (`pkg/healthcheckhandler/health.go`)
  so only one replica probes carriers per tick. The lock fails **open** on
  a Redis error (proceeds unlocked rather than skipping the cycle) — this
  is a deliberate tradeoff, since `ProviderUpdateHealthStatus` is a blind
  UPDATE with no CAS, so a concurrent unlocked write is harmless and the
  only cost is the same transient doubled-probe traffic already accepted
  above for cutover overlaps. An **extended** Redis outage would make that
  doubled probing persist for the outage's duration rather than a brief
  window — watch for `WARN`-level "healthcheck lock" log lines during a
  Redis incident, which is the passive signal this is happening.

## Configuration

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--database_dsn` | `DATABASE_DSN` | required | MySQL connection string |
| `--rabbitmq_address` | `RABBITMQ_ADDRESS` | required | RabbitMQ server |
| `--redis_address` | `REDIS_ADDRESS` | required | Redis cache |
| `--redis_password` | `REDIS_PASSWORD` | `` | Redis auth password |
| `--redis_database` | `REDIS_DATABASE` | `` | Redis DB index |
| `--external_sip_gateway_fqdn_for_pstn` | `EXTERNAL_SIP_GATEWAY_FQDN_FOR_PSTN` | `` | Public SIP gateway FQDN:port (e.g. `pstn.voipbin.net:5060`) registered on Telnyx as an FQDN connection for provider setup |
| `--prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Metrics path |
| `--prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Metrics listen address |

## Prometheus Metrics

Metrics are served at `<prometheus_listen_address><prometheus_endpoint>` (default: `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `route_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing latency |

**Useful PromQL:**
```promql
# P99 RPC latency
histogram_quantile(0.99, rate(route_manager_receive_request_process_time_bucket[5m]))

# Request rate
rate(route_manager_receive_request_process_time_count[5m])
```
