# bin-billing-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| Balance check rejected | Services fail to create resources | Account balance at zero or account not found |
| Paddle webhook rejected | Subscriptions not activated | `paddle_api_key` misconfigured or webhook signature invalid |
| FailedEvent accumulation | `failed_event_save_total` rising | Downstream RPC failures during billing event processing |
| Event processing lag | Billing records not created | subscribehandler consumer stalled; check RabbitMQ queue depth |
| Cache stale | Old balance returned | Redis out of sync; restart flushes cache and forces DB reads |
| Missing billing record | Calls billed incorrectly | `call_hangup` event dropped before billing record finalized |

## Debugging Guide

**Check account balance:**
```bash
billing-control account get --id <account-uuid>
```

**List recent billing records for a customer:**
```bash
billing-control billing list --customer-id <uuid> --limit 20
```

**Force balance adjustment (admin only):**
```bash
# Add balance
billing-control account add-balance --id <account-uuid> --amount 10.00

# Subtract balance
billing-control account subtract-balance --id <account-uuid> --amount 5.00
```

**Check failed events (via database):**
```sql
SELECT * FROM billing_failed_events WHERE status != 'finished' ORDER BY tm_create DESC LIMIT 20;
```

**Run service locally:**
```bash
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/bin_manager" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
REDIS_DATABASE=1 \
./bin/billing-manager
```

**Check Paddle webhook delivery:**
- Paddle dashboard → Notifications → Event log
- Look for events with status `failed` and check the response body

## Configuration

All flags can also be set via environment variable (uppercase, underscores). No defaults shown means the flag is required.

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | required | RabbitMQ server address |
| `database_dsn` | `DATABASE_DSN` | required | MySQL DSN |
| `redis_address` | `REDIS_ADDRESS` | required | Redis address |
| `redis_password` | `REDIS_PASSWORD` | `""` | Redis auth password |
| `redis_database` | `REDIS_DATABASE` | `0` | Redis DB index |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `""` | Metrics path (e.g. `/metrics`) |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `""` | Metrics listen address (e.g. `:2112`) |
| `paddle_api_key` | `PADDLE_API_KEY` | required | Paddle API key for webhook validation |
| `paddle_price_id_basic` | `PADDLE_PRICE_ID_BASIC` | required | Paddle price ID for basic plan |
| `paddle_price_id_professional` | `PADDLE_PRICE_ID_PROFESSIONAL` | required | Paddle price ID for professional plan |

## Prometheus Metrics

Metrics are exposed on the configured `prometheus_listen_address` + `prometheus_endpoint`.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `billing_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `billing_manager_receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event processing duration |
| `account_balance_check_total` | Counter | — | Total balance check operations |
| `account_create_total` | Counter | — | Total account creation operations |
| `billing_create_total` | Counter | — | Total billing records created |
| `billing_duration_seconds` | Histogram | — | Billing event duration in seconds |
| `billing_end_total` | Counter | — | Total billing records finalized |
| `failed_event_exhausted_total` | Counter | — | Failed events that exhausted retries |
| `failed_event_retry_total` | Counter | — | Failed event retry attempts |
| `failed_event_save_total` | Counter | — | Failed events persisted to retry queue |

**Alert guidance:**
- `failed_event_exhausted_total` increasing → billing records are being permanently lost; investigate downstream RPC failures.
- `account_balance_check_total` spike → elevated resource creation attempts; normal under load, but check for abuse.
- `billing_manager_receive_subscribe_event_process_time` p99 > 1s → subscribehandler processing too slow; check DB query performance.
