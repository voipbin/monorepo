# bin-tag-manager

Lightweight CRUD service for managing customer-scoped tags in VoIPbin. Tags are labels that other services (contacts, queues, campaigns) attach to resources for categorization and routing. Handles event-driven cascading deletes when customers are removed.

## Key Concepts

- **Tag**: Customer-scoped label with name and detail; referenced by agent `tag_ids`, queue `tag_ids`, and campaign filters
- **Cascade delete**: Subscribes to `customer_deleted` event and bulk-deletes all tags for the removed customer

## Public RPC Entrypoints

| Pattern | Operations |
|---------|-----------|
| `POST /v1/tags` | Create tag |
| `GET /v1/tags` | List tags |
| `GET /v1/tags/<id>` | Get tag |
| `PUT /v1/tags/<id>` | Update tag |
| `DELETE /v1/tags/<id>` | Delete tag |

## Dependencies

- **MySQL** — tag records (soft-delete via `tm_delete`)
- **Redis** — tag cache
- **RabbitMQ** — listen queue `bin-manager.tag-manager.request`; subscribes to `bin-manager.customer-manager.event`; publishes `tag_created`, `tag_updated`, `tag_deleted`

## Local Development

```bash
# Build
cd bin-tag-manager
go build -o ./bin/ ./cmd/...

# Run all tests
go test ./...

# Verify before commit (mandatory)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# CLI tool (bypasses RabbitMQ)
./bin/tag-control tag list --customer_id <uuid>
./bin/tag-control tag get --id <uuid>
./bin/tag-control tag create --customer_id <uuid> --name "my-tag"
./bin/tag-control tag delete --id <uuid>
```

## Further Reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)
