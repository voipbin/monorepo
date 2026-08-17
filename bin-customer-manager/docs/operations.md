# bin-customer-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| Customer not found | 404 from API manager | Customer UUID wrong or customer soft-deleted |
| Access key auth failure | All API requests rejected | AccessKey expired, deleted, or Redis cache stale |
| Email verification not sent | Signup appears stuck | `email_verify_base_url` misconfigured or email service down |
| Agent username conflict | Customer create fails | Email already used as agent username in bin-agent-manager |
| Cache miss cascade | High DB latency | Redis unavailable; all reads fall through to MySQL |
| Event publish failure | Downstream services miss customer lifecycle events | RabbitMQ unavailable or exchange misconfigured |

## Debugging Guide

**Get a customer directly (bypasses RabbitMQ):**
```bash
./bin/customer-control customer get --id <uuid>
```

**List customers:**
```bash
./bin/customer-control customer list [--limit 100] [--token <cursor>]
```

**Get access key:**
```bash
./bin/customer-control accesskey get --id <uuid>
```

**List access keys for a customer:**
```bash
./bin/customer-control accesskey list --customer-id <uuid>
```

**Create customer manually (testing):**
```bash
./bin/customer-control customer create --email test@example.com --name "Test Customer"
```

**Link billing account:**
```bash
./bin/customer-control customer update-billing-account --id <customer-uuid> --billing-account-id <billing-uuid>
```

**Run service locally:**
```bash
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
REDIS_DATABASE=1 \
PROMETHEUS_ENDPOINT="/metrics" \
PROMETHEUS_LISTEN_ADDRESS=":2112" \
./bin/customer-manager
```

## Deployment

bin-customer-manager deploys via Komodo (VOIP-1348 Tier 2 rollout,
following the VOIP-1342/bin-call-manager pilot and VOIP-1347/Tier 1
pattern) instead of the older SSH + `versions.lock` (`ssh-deploy.sh`)
path.

- **Stack definition:** `bin-customer-manager/komodo/docker-compose.yml`
  (git is the source of truth for structure; Komodo only executes it on
  request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes
  the built image tag, then `.circleci/scripts/komodo-api-deploy.sh`
  pushes the file's content to Komodo and triggers a deploy, gated
  by the `bin-customer-manager-deploy` job's poll/running checks.
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier2-design.md](../../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier2-design.md)
  (in the monorepo root, not this service's own `docs/`).

## Configuration

All flags can also be set via environment variable (uppercase, underscores).

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | required | RabbitMQ server address |
| `database_dsn` | `DATABASE_DSN` | required | MySQL DSN |
| `redis_address` | `REDIS_ADDRESS` | required | Redis address |
| `redis_password` | `REDIS_PASSWORD` | `""` | Redis auth password |
| `redis_database` | `REDIS_DATABASE` | `0` | Redis DB index |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | required | Metrics path (e.g. `/metrics`) |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `""` | Metrics listen address (e.g. `:2112`) |
| `email_verify_base_url` | `EMAIL_VERIFY_BASE_URL` | required | Base URL for email verification links |

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `cache_operation_total` | Counter | — | Total Redis cache operations |
| `db_operation_duration` | Histogram | — | MySQL query duration |
| `db_operation_total` | Counter | — | Total MySQL operations |
| `email_verification_total` | Counter | — | Total email verification attempts |
| `rpc_call_duration` | Histogram | — | Outbound RPC call duration |
| `rpc_call_total` | Counter | — | Total outbound RPC calls |
| `signup_total` | Counter | — | Total customer signup attempts |

**Alert guidance:**
- `cache_operation_total` drop → Redis may be unavailable; expect increased `db_operation_total`
- `rpc_call_total` failures → agent-manager or billing-manager unreachable; customer creation will fail
- `signup_total` spikes without matching `email_verification_total` → signup flow stuck; check email service
