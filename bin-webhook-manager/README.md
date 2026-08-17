# webhook-manager

Webhook-manager receives webhook requests from other services and delivers webhook events to customer-configured endpoints.

## Configuration

Configuration is via environment variables (or equivalent CLI flags).

| Environment Variable | CLI Flag | Description |
|---|---|---|
| `DATABASE_DSN` | `--database_dsn` | MySQL connection DSN |
| `RABBITMQ_ADDRESS` | `--rabbitmq_address` | RabbitMQ server address |
| `REDIS_ADDRESS` | `--redis_address` | Redis server address |
| `REDIS_PASSWORD` | `--redis_password` | Redis password |
| `REDIS_DATABASE` | `--redis_database` | Redis database index |
| `PROMETHEUS_ENDPOINT` | `--prometheus_endpoint` | Prometheus metrics endpoint path |
| `PROMETHEUS_LISTEN_ADDRESS` | `--prometheus_listen_address` | Prometheus HTTP listen address |

## Build

```bash
go build -o ./bin/webhook-manager ./cmd/webhook-manager
go build -o ./bin/webhook-control ./cmd/webhook-control
```

## Test

```bash
go test ./...
```

<!-- Updated dependencies: 2026-02-20 -->

# Deploy

`bin-webhook-manager-build` pushes the image, and a single `build-approval` gate covers
the test -> build pipeline. The CircleCI `bin-webhook-manager-deploy` job (direct SSH
deploy to bm-nyc-01) has been removed. Deploys to bm-nyc-01 are manual until
this service migrates to the Komodo-managed deploy path (see bin-call-manager
for the pattern).
