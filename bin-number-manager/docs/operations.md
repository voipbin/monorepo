# bin-number-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| Number purchase fails | 402 or provider error | Insufficient billing balance or provider API credentials invalid |
| Telnyx/Twilio API error | Number not created | Provider credentials (`telnyx_token`, `twilio_sid`/`twilio_token`) wrong or expired |
| Cascade not triggered | Numbers persist after customer delete | subscribehandler not receiving `customer_deleted` event; check RabbitMQ queue |
| Flow reference stale | Inbound calls fail after flow deleted | subscribehandler not receiving `flow_deleted` event; flow IDs not cleared |
| Available number search empty | No numbers returned | Country code not supported by provider, or provider API rate limited |
| Cache stale | Old number data served | Redis out of sync; restart flushes and forces DB reads |

## Debugging Guide

**Get a number:**
```bash
./bin/number-control number get --id <uuid>
```

**List numbers for a customer:**
```bash
./bin/number-control number list --customer-id <uuid> [--limit 100] [--token]
```

**Search available numbers:**
```bash
./bin/number-control number get-available --country-code US [--limit 10]
```

**Update call flow on a number:**
```bash
./bin/number-control number update --id <uuid> --call-flow-id <flow-uuid>
```

**Delete (release) a number:**
```bash
./bin/number-control number delete --id <uuid>
```

**Run service locally:**
```bash
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/bin_manager" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
TELNYX_TOKEN="<token>" \
TELNYX_CONNECTION_ID="<connection-id>" \
TELNYX_PROFILE_ID="<profile-id>" \
./bin/number-manager
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
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `""` | Metrics path (e.g. `/metrics`) |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `""` | Metrics listen address (e.g. `:2112`) |
| `telnyx_token` | `TELNYX_TOKEN` | required if Telnyx used | Telnyx API authentication token |
| `telnyx_connection_id` | `TELNYX_CONNECTION_ID` | required if Telnyx used | Telnyx connection ID for call routing |
| `telnyx_profile_id` | `TELNYX_PROFILE_ID` | required if Telnyx used | Telnyx messaging profile ID |
| `twilio_sid` | `TWILIO_SID` | required if Twilio used | Twilio account SID |
| `twilio_token` | `TWILIO_TOKEN` | required if Twilio used | Twilio auth token |

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event processing duration |
| `request_external_process_time` | Histogram | `provider`, `method` | External provider API call duration |
| `number_create_total` | Counter | — | Total number purchase operations |
| `telnyx_number_create_total` | Counter | — | Telnyx-specific number purchases |
| `twilio_number_create_total` | Counter | — | Twilio-specific number purchases |

**Alert guidance:**
- `telnyx_number_create_total` or `twilio_number_create_total` dropping while `number_create_total` stays constant → provider-specific failures; check provider API credentials.
- `request_external_process_time` p99 > 5s → provider API latency; purchases will time out.
- `receive_subscribe_event_process_time` high → cascade operations (customer/flow deletes) backing up.
