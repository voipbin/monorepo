# bin-push-manager Operations

## Configuration

| Variable | Flag | Meaning |
|----------|------|---------|
| `DATABASE_DSN` | `database_dsn` | MySQL DSN |
| `RABBITMQ_ADDRESS` | `rabbitmq_address` | RabbitMQ address |
| `PROMETHEUS_ENDPOINT` | `prometheus_endpoint` | Metrics path (default `/metrics`) |
| `PROMETHEUS_LISTEN_ADDRESS` | `prometheus_listen_address` | Metrics listen address (default `:2112`) |
| `FCM_SERVICE_ACCOUNT_FILE` | `fcm_service_account_file` | Path of the FCM service account JSON file. Empty means FCM is not configured |

The service account is mounted as a file by the deployment secret. Its content is never logged.

## Metrics

| Metric | Labels | Meaning |
|--------|--------|---------|
| `push_manager_notification_total` | `kind`, `result` | Deliveries. `result` is `sent`, `unregistered`, `failed` or `not_configured` |
| `push_manager_receive_request_process_time` | `type`, `method` | RPC processing time |
| `push_manager_receive_subscribe_event_process_time` | `publisher`, `type` | Event processing time |
| `push_manager_recipient_truncated_total` | `kind` | Events whose recipients were cut to the limit of 200 (trigger to look at sending in batches) |

The request handler adds `push_manager_request_process_time`, `push_manager_event_publish_total` and `push_manager_circuitbreaker_*`.

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|-----------|
| Every send answers `sent_count: 0`, `not_configured` grows | FCM service account file missing, empty or invalid | Check that the deployment secret exists and holds a valid service account; restart the stack |
| `failed` grows | FCM is unreachable or rejects the message | Check the service logs (no token is logged) and the FCM status |
| `unregistered` grows | Uninstalled apps | Expected; the devices are deleted |
| Register answers 400 `INVALID_TOKEN` | Empty token or more than 512 bytes | Fix the client |

## Event Processing

At most 32 events are processed at the same time. When all slots are taken, the consumer callback waits for a free slot and does not take more events (the consumer has 10 workers, so up to 10 events wait at a time); the events wait in the queue. A chat message to a large chat keeps its slot until all its sends are done or its 30 seconds are over, so a burst of such messages slows every other event down: look at `push_manager_receive_subscribe_event_process_time` and the log line "The event ran out of time". The agent and customer delete events use the same pool.

| Symptom | Likely Cause | Resolution |
|---------|-------------|-----------|
| "The recipients of the chat message are cut to the limit" | A chat with more than 200 agents that want the notification | Expected; the participants who joined first are not notified. Look at sending in batches if it happens often |
| "The event ran out of time" | A slow FCM or a very large chat | Check FCM and the metric above |
| A mention does not notify | The mention expression of this build differs from the one of `bin-talk-manager`, or the agent switched `chat_mentioned` off | Deploy both services together |
| `case_assigned` does not arrive | Only an open case that gets a different agent as its owner publishes it | See `bin-contact-manager/docs/domain.md` |

## Schema Migration Failures

Migrations run at startup before the service listens. A failure exits the process with a non-zero code. The recovery steps (dirty database, lock timeout) are in the migration section of `CLAUDE.md` and in `docs/workflows/service-owned-migrations.md`; the CLI is `push-control migrate status|up|down|force`.

## Debugging Guide

```bash
cd bin-push-manager && go build -o ./bin/ ./cmd/...
go test ./...
golangci-lint run -v --timeout 5m

# Migration status against a local throwaway database
./bin/push-control migrate status
```
