# bin-outdial-manager

Outbound dialing target management service for VoIPbin. Stores outdial containers, individual call targets (up to 5 destinations per target with independent retry counters), and per-attempt call records. Primary consumer is `bin-campaign-manager`.

## Key Concepts

- **Outdial**: Container grouping a set of dial targets for a campaign
- **OutdialTarget**: Single dial target with up to 5 destination slots (`destination_0`–`destination_4`); statuses: `idle` → `processing` → `done`
- **OutdialTargetCall**: Per-attempt call record linking a target to a specific call UUID
- **Available query**: Filters targets by per-destination try-count thresholds; used by campaign-manager for retry scheduling

## Public RPC Entrypoints

| Pattern | Operations |
|---------|-----------|
| `POST /v1/outdials` | Create outdial |
| `GET /v1/outdials` | List outdials |
| `GET /v1/outdials/<id>` | Get outdial |
| `PUT /v1/outdials/<id>` | Update outdial |
| `DELETE /v1/outdials/<id>` | Delete outdial |
| `POST /v1/outdials/<id>/targets` | Add target |
| `GET /v1/outdials/<id>/targets` | List targets |
| `GET /v1/outdials/<id>/targets/<target-id>` | Get target |
| `PUT /v1/outdials/<id>/targets/<target-id>` | Update target |
| `DELETE /v1/outdials/<id>/targets/<target-id>` | Delete target |
| `GET /v1/outdials/<id>/targets/available` | Get next available targets |

## Dependencies

- **MySQL** — outdial, target, and call records
- **Redis** — outdial and target cache
- **RabbitMQ** — listen queue `bin-manager.outdial-manager.request`; publishes `outdial_created`, `outdial_updated`, `outdial_deleted`

## Local Development

```bash
# Build
cd bin-outdial-manager
go build -o ./bin/ ./cmd/...

# Run all tests
go test ./...

# Verify before commit (mandatory)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# CLI tool (bypasses RabbitMQ)
./bin/outdial-control outdial list --customer_id <uuid>
./bin/outdial-control outdial get --id <uuid>
```

## Further Reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)
