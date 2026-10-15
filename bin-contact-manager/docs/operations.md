# bin-contact-manager — Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Contact lookup returns 404 for known contact | Redis cache stale or evicted | Check cache connectivity; lookup falls back to DB if cache miss is propagated correctly |
| `customer_deleted` events not triggering cascades | Subscribe queue consumer stopped | Check `subscribehandler` logs; verify RabbitMQ queue binding |
| Tag assignments failing | `bin-tag-manager` RPC timeout | Verify tag-manager is healthy; check RabbitMQ for dead-lettered messages |
| Slow list queries | Missing index on `customer_id` + `tm_delete` | Verify database indexes; check slow query log |
| Duplicate phone number in lookup | Cache inconsistency after partial update | Cache key collision is unlikely; force cache invalidation via contact update |
| RPC requests timing out | DB connection pool exhaustion | Check `DATABASE_DSN` pool settings; monitor active connections |

## Debugging Guide

**Check service health:**
```bash
curl -s http://<pod-ip>:2112/metrics | grep contact_manager
```

**Inspect live contacts (CLI tool — bypasses RabbitMQ):**
```bash
./bin/contact-control contact list --customer-id <uuid> --limit 10
./bin/contact-control contact get --id <contact-uuid>
./bin/contact-control contact lookup --customer-id <uuid> --phone-e164 +15551234567
```

**Check RabbitMQ queue depth:**
```bash
rabbitmqctl list_queues name messages consumers | grep contact
```

**Check Redis cache:**
```bash
redis-cli -n 1 keys "contact:*" | head -20
```

**Database queries:**
```sql
-- Active contacts for a customer
SELECT id, display_name, tm_create FROM contact_contacts
WHERE customer_id = '<uuid>' AND tm_delete IS NULL
LIMIT 20;

-- Addresses (phone + email) for a contact
SELECT id, type, target, is_primary FROM contact_addresses
WHERE contact_id = '<uuid>';
```

## Database Migrations

The service applies the migrations of its nine tables itself (`migrations/`, embedded) on every start, after the database connection and before the cache and RabbitMQ; version table `contact_schema_migrations`.

| Situation | Behavior | Action |
|-----------|----------|--------|
| No pending migration | log `no pending migrations`, normal start | none |
| Pending migration | applied under a MySQL named lock, then normal start; two instances serialize | none |
| Database version higher than the image | skipped with a warning, normal start | none |
| Database marked dirty (a migration failed half way) | the start is refused, the container restarts in a loop | stop the container, check whether the failed file was applied, then `contact-control migrate force <version>` against that database |
| Lock not obtained within 10 seconds | the start fails and the container restarts | none, a restart retries |

A database whose tables differ from the Alembic head (for example an Alembic revision that was never applied) starts as version 1 without a warning, because the baseline is `CREATE TABLE IF NOT EXISTS`; compare the normalized dump of the database with the head before the first deployment (`REFERENCE_DUMP` of `scripts/verify-service-migrations.sh`).

The baseline lists the generated columns of `contact_addresses`, `contact_address_ownership_periods` and `contact_cases` after `tm_update` (and `local` right after `peer`), the column order of the existing tables; the Alembic head orders them differently, so these three tables differ from the head in the column ordinals only (`column-order` in `E1_ALLOWED_DIFF`). The two CHECK names of `contact_cases` are `chk_cases_peer_type_nn` and `chk_cases_peer_target_nn`, the names of the tables of databases created earlier; the Alembic head names them `chk_contact_cases_peer_type_not_null` and `chk_contact_cases_peer_target_not_null`. The expressions are the same, so the baseline differs from the head in these names only (`check-name contact_cases` in `E1_ALLOWED_DIFF`), and the comparison with the dump of an existing database (`REFERENCE_DUMP`) needs no `ALLOWED_DIFF`.

Check the state with `./bin/contact-control migrate status` (database only). The image is distroless and has no shell, so run it as `docker run --rm --entrypoint ./contact-control <image> migrate status` with the service environment. The new database must be created with `CHARACTER SET utf8 COLLATE utf8_general_ci`. File rules and checks: [docs/workflows/service-owned-migrations.md](../../docs/workflows/service-owned-migrations.md).

## Deployment

bin-contact-manager deploys via Komodo (VOIP-1347 Tier 1 rollout, following the
VOIP-1342/bin-call-manager pilot pattern) instead of the older SSH +
`versions.lock` (`ssh-deploy.sh`) path.

- **Stack definition:** `bin-contact-manager/komodo/docker-compose.yml` (git is
  the source of truth for structure; Komodo only executes it on
  request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes
  the built image tag, then `.circleci/scripts/komodo-api-deploy.sh`
  pushes the file's content to Komodo and triggers a deploy, gated
  by the `bin-contact-manager-deploy` job's poll/running checks.
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md](../../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md)
  (in the monorepo root, not this service's own `docs/`).

## Configuration

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--database_dsn` | `DATABASE_DSN` | `testid:testpassword@tcp(127.0.0.1:3306)/test` | MySQL connection string |
| `--rabbitmq_address` | `RABBITMQ_ADDRESS` | `amqp://guest:guest@localhost:5672` | RabbitMQ server |
| `--redis_address` | `REDIS_ADDRESS` | `127.0.0.1:6379` | Redis server |
| `--redis_password` | `REDIS_PASSWORD` | `` | Redis auth password |
| `--redis_database` | `REDIS_DATABASE` | `1` | Redis DB index |
| `--prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Metrics path |
| `--prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Metrics listen address |

## Prometheus Metrics

Metrics are served at `<prometheus_listen_address><prometheus_endpoint>` (default: `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `contact_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing latency |
| `contact_manager_receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event subscription processing latency |

**Useful PromQL:**
```promql
# P99 RPC latency
histogram_quantile(0.99, rate(contact_manager_receive_request_process_time_bucket[5m]))

# Event processing rate
rate(contact_manager_receive_subscribe_event_process_time_count[5m])
```
