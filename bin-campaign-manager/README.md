# bin-campaign-manager

Outbound calling campaign orchestration service for VoIPbin. Manages Campaign configuration, Outplan dial settings, and individual Campaigncall attempts, coordinating with `bin-call-manager` to place calls and `bin-flow-manager` to execute on-connect actions.

## Key Concepts

- **Campaign**: Outbound campaign entity; statuses `stop`, `run`, `stopping`; references an outdial (target list), outplan, and optional queue
- **Campaigncall**: Single call attempt; up to 5 destination slots with independent retry counters; references a Call or Activeflow
- **Outplan**: Dialing configuration — `source` (caller ID), `dial_timeout`, `try_interval`, `max_try_count_0..4`; shared across campaigns
- **Service level**: Percentage throttle (0–100) based on available agents in the linked queue; 0 means no dialing
- **Next campaign chaining**: `next_campaign_id` enables sequential campaign execution after current campaign completes

## Public RPC Entrypoints

| Pattern | Operations |
|---------|-----------|
| `POST /v1/campaigns` | Create campaign |
| `GET /v1/campaigns` | List campaigns |
| `GET /v1/campaigns/<id>` | Get campaign |
| `PUT /v1/campaigns/<id>` | Update campaign |
| `DELETE /v1/campaigns/<id>` | Delete campaign |
| `POST /v1/campaigns/<id>/execute` | Execute one dial cycle |
| `POST /v1/campaigns/<id>/start` | Start campaign |
| `POST /v1/campaigns/<id>/stop` | Stop campaign |
| `POST /v1/outplans` | Create outplan |
| `GET /v1/outplans` | List outplans |
| `GET /v1/outplans/<id>` | Get outplan |
| `PUT /v1/outplans/<id>` | Update outplan |
| `DELETE /v1/outplans/<id>` | Delete outplan |
| `GET /v1/campaigncalls` | List campaigncalls |
| `GET /v1/campaigncalls/<id>` | Get campaigncall |

## Dependencies

- **MySQL** — campaign, outplan, campaigncall records
- **Redis** — campaign and campaigncall cache
- **RabbitMQ** — listen queue `bin-manager.campaign-manager.request`; subscribes to `bin-manager.call-manager.event`
- **bin-call-manager** — place outbound calls
- **bin-outdial-manager** — fetch and update dial targets
- **bin-queue-manager** — service level throttle check

## Local Development

```bash
# Build
cd bin-campaign-manager
go build -o ./bin/ ./cmd/...

# Run all tests
go test ./...

# Verify before commit (mandatory)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# CLI tool (bypasses RabbitMQ)
./bin/campaign-control campaign get --id <uuid>
```

## Further Reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)
