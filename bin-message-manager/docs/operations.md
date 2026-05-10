# bin-message-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| SMS not sent | Message created but target stays `queued` | Provider API credentials invalid or provider API down |
| Balance rejection | `POST /v1/messages` returns 402 | Customer billing balance insufficient |
| Webhook not processed | Target status never updates | `bin-hook-manager` not forwarding; or provider webhook URL misconfigured |
| MessageBird auth failure | Targets fail with 401 | `authtoken_messagebird` expired or wrong |
| Telnyx auth failure | Fallback provider fails | `authtoken_telnyx` expired or wrong |
| Cache stale | Stale message data returned | Redis out of sync; restart clears cache |

## Debugging Guide

**Get a message (bypasses RabbitMQ):**
```bash
./bin/message-control message get --id <uuid>
```

**List messages for a customer:**
```bash
./bin/message-control message list --customer_id <uuid> [--limit 100] [--token]
```

**Delete a message:**
```bash
./bin/message-control message delete --id <uuid>
```

**Check provider webhook configuration:**
- MessageBird dashboard → Channels → your number → Delivery Reports URL
- Telnyx portal → Messaging Profiles → your profile → Webhooks

**Provider webhook endpoint** (via hook-manager):
```
POST https://hook.voipbin.net/v1.0/messages/messagebird
POST https://hook.voipbin.net/v1.0/messages/telnyx
```

**Run service locally:**
```bash
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
AUTHTOKEN_MESSAGEBIRD="your-token" \
AUTHTOKEN_TELNYX="your-token" \
./bin/message-manager
```

## Configuration

All flags can also be set via environment variable (uppercase, underscores).

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | required | RabbitMQ server address |
| `database_dsn` | `DATABASE_DSN` | required | MySQL DSN |
| `redis_address` | `REDIS_ADDRESS` | required | Redis address |
| `redis_password` | `REDIS_PASSWORD` | `""` | Redis auth password |
| `redis_database` | `REDIS_DATABASE` | `0` | Redis DB index |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Metrics path |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Metrics listen address |
| `authtoken_messagebird` | `AUTHTOKEN_MESSAGEBIRD` | required | MessageBird API authentication token |
| `authtoken_telnyx` | `AUTHTOKEN_TELNYX` | required | Telnyx API authentication token |

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `request_external_process_time` | Histogram | `provider`, `method` | Provider API call duration |
| `messagebird_number_send_total` | Counter | `type` | Total SMS sent via MessageBird |
| `telnyx_number_send_total` | Counter | `type` | Total SMS sent via Telnyx |

**Alert guidance:**
- `messagebird_number_send_total` dropping to zero → MessageBird auth failure or API down; SMS will fail silently.
- `request_external_process_time` p99 > 10s → provider API slow; messages may time out.
- `receive_request_process_time` rising → RabbitMQ queue backing up; check consumer health.
