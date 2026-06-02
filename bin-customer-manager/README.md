# bin-customer-manager

Foundational identity service for VoIPbin. Manages tenant organizations (customers) and their API credentials (access keys). All other services depend on customer context — this is the root of the VoIPbin tenant hierarchy.

## Key Concepts

- **Customer**: Tenant organization entity; holds name, status, and billing account link
- **AccessKey**: API credential scoped to a customer; used as `?accesskey=` query parameter on all API calls
- **Cascade delete**: `customer_deleted` event triggers bulk cleanup in `bin-number-manager` and `bin-billing-manager`

## Public RPC Entrypoints

| Pattern | Operations |
|---------|-----------|
| `POST /v1/customers` | Create customer |
| `GET /v1/customers` | List customers |
| `GET /v1/customers/<id>` | Get customer |
| `PUT /v1/customers/<id>` | Update customer |
| `DELETE /v1/customers/<id>` | Delete customer |
| `POST /v1/accesskeys` | Create access key |
| `GET /v1/accesskeys` | List access keys |
| `GET /v1/accesskeys/<id>` | Get access key |
| `DELETE /v1/accesskeys/<id>` | Delete access key |

## Dependencies

- **MySQL** — customer and accesskey records
- **Redis** — customer and accesskey cache
- **RabbitMQ** — listen queue `bin-manager.customer-manager.request`; publishes `customer_created`, `customer_updated`, `customer_deleted`
- **bin-agent-manager** — username uniqueness check on customer create
- **bin-billing-manager** — billing account link validation

## Local Development

```bash
# Build
cd bin-customer-manager
go build -o ./bin/ ./cmd/...

# Run all tests
go test ./...

# Verify before commit (mandatory)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# CLI tool (bypasses RabbitMQ)
./bin/customer-control customer get --id <uuid>
./bin/customer-control customer list
./bin/customer-control accesskey list --customer-id <uuid>
```

## Further Reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)
