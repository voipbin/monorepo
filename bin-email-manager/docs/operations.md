# bin-email-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| Email not delivered | Status stays `initiated` | Both SendGrid and Mailgun APIs failing; check credentials |
| SendGrid auth failure | Primary provider rejected | `sendgrid_api_key` invalid or expired |
| Mailgun auth failure | Fallback also fails; email stuck | `mailgun_api_key` invalid or expired |
| Attachment not attached | Email sent without attachment | `bin-storage-manager` RPC failed or recording not found |
| Webhook status not updated | Email stays at `processed` forever | `bin-hook-manager` not forwarding; or provider webhook URL misconfigured |
| Cache stale | Old email data returned | Redis out of sync; restart clears cache |

## Debugging Guide

**Get an email (bypasses RabbitMQ):**
```bash
./bin/email-control email get --id <uuid>
```

**List emails for a customer:**
```bash
./bin/email-control email list --customer_id <uuid> [--limit 100] [--token]
```

**Delete an email:**
```bash
./bin/email-control email delete --id <uuid>
```

**Check provider webhook configuration:**
- SendGrid: Settings → Mail Settings → Event Webhook → URL must be `https://hook.voipbin.net/v1.0/emails/sendgrid`
- Mailgun: Sending → Webhooks → URL must be `https://hook.voipbin.net/v1.0/emails/mailgun`

**Run service locally:**
```bash
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
SENDGRID_API_KEY="SG.xxx" \
MAILGUN_API_KEY="xxx" \
./bin/email-manager
```

**Check email delivery via SendGrid Activity Feed:**
- SendGrid dashboard → Activity → search by recipient email or `provider_reference_id`

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
| `sendgrid_api_key` | `SENDGRID_API_KEY` | required | SendGrid API key |
| `mailgun_api_key` | `MAILGUN_API_KEY` | required | Mailgun API key |

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |

**Alert guidance:**
- `receive_request_process_time` p99 rising → emailhandler slow; check provider API latency and storage-manager RPC.
- No email events published in billing-manager → email billing broken; check `bin-manager.email-manager.event` exchange.
