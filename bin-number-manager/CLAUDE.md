# bin-number-manager

PSTN phone number lifecycle management — purchases, manages, and releases DID numbers via Telnyx and Twilio providers, and routes inbound calls/messages to flows.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — components, layers, request routing
- [docs/domain.md](docs/domain.md) — Number, AvailableNumber, ProviderNumber entities; lifecycle rules; provider strategy
- [docs/dependencies.md](docs/dependencies.md) — subscribed events, outbound RPCs, monorepo deps
- [docs/operations.md](docs/operations.md) — failure modes, debugging, config reference, metrics

## CRITICAL: Balance Check Before Purchase

Every number purchase calls `bin-billing-manager` for balance validation **before** contacting the provider. Do not bypass this check.

## CRITICAL: Provider Credentials

Both Telnyx and Twilio credentials must be valid for number operations to succeed. Missing or expired credentials cause silent purchase failures at the provider level.

## Key Commands

```bash
# Build
go build -o ./bin/ ./cmd/...

# Test
go test -coverprofile cp.out -v $(go list ./...)

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# number-control — direct DB/cache access, bypasses RabbitMQ
./bin/number-control number get --id <uuid>
./bin/number-control number list --customer-id <uuid>
./bin/number-control number get-available --country-code US
./bin/number-control number update --id <uuid> --call-flow-id <flow-uuid>
./bin/number-control number delete --id <uuid>

# Run locally
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/bin_manager" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
TELNYX_TOKEN="<token>" \
./bin/number-manager
```

## Architecture Summary

- **ListenHandler**: consumes `bin-manager.number-manager.request`; routes to numberhandler
- **SubscribeHandler**: `customer_deleted` → release all customer numbers; `flow_deleted` → clear flow references
- **Provider dispatch**: numberhandler delegates to `numberhandlertelnyx` or `numberhandlertwilio`
- **ProviderNumber record**: created alongside Number; holds provider's own reference ID for release operations

## Testing

Uses `go.uber.org/mock` (gomock). Table-driven tests. Mocks co-located with handlers.

```bash
go test -v -run TestName ./pkg/numberhandler/...
```

## Database Tables

- `number_numbers` — phone numbers (soft-delete with `tm_delete`)
- `number_provider_numbers` — provider-to-number mapping for release operations
