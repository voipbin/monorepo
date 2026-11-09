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

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|-----------|
| Every send answers `sent_count: 0`, `not_configured` grows | FCM service account file missing, empty or invalid | Check that the deployment secret exists and holds a valid service account; restart the stack |
| `failed` grows | FCM is unreachable or rejects the message | Check the service logs (no token is logged) and the FCM status |
| `unregistered` grows | Uninstalled apps | Expected; the devices are deleted |
| Register answers 400 `INVALID_TOKEN` | Empty token or more than 512 bytes | Fix the client |

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
