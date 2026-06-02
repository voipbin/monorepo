# bin-queue-manager

Inbound call queue management service for VoIPbin. Holds callers in a waiting state, routes them to available agents using tag-based matching, and coordinates the conference bridge used to connect agent and caller.

## Key Concepts

- **Queue**: Configuration entity — routing method, `tag_ids` (agent filter), `wait_timeout`, `service_timeout`
- **Queuecall**: Single call waiting in a queue; status: `initiating` → `waiting` → `connecting` → `service` → `done` / `abandoned`
- **Routing**: `random` method picks a random available agent whose tag IDs overlap with the queue's `tag_ids`
- **Conference bridge**: When an agent is matched, a conference in `bin-conference-manager` bridges caller and agent
- **Wait timeout**: Maximum time a caller waits before being abandoned; triggered by external scheduler
- **Service timeout**: Maximum agent service duration; triggers forced disconnect via `timeout_service`

## Public RPC Entrypoints

| Pattern | Operations |
|---------|-----------|
| `POST /v1/queues` | Create queue |
| `GET /v1/queues` | List queues |
| `GET /v1/queues/<id>` | Get queue |
| `PUT /v1/queues/<id>` | Update queue |
| `DELETE /v1/queues/<id>` | Delete queue |
| `POST /v1/queuecalls` | Enqueue a call |
| `GET /v1/queuecalls` | List queuecalls |
| `GET /v1/queuecalls/<id>` | Get queuecall |
| `DELETE /v1/queuecalls/<id>` | Remove from queue |
| `POST /v1/queuecalls/<id>/timeout_wait` | Trigger wait timeout |
| `POST /v1/queuecalls/<id>/timeout_service` | Trigger service timeout |

## Dependencies

- **MySQL** — queue and queuecall records
- **Redis** — queue and queuecall cache
- **RabbitMQ** — listen queue `bin-manager.queue-manager.request`; subscribes to `bin-manager.call-manager.event`, `bin-manager.agent-manager.event`
- **bin-agent-manager** — fetch available agents for routing
- **bin-conference-manager** — create conference bridge for agent-caller connection
- **bin-call-manager** — call state tracking via events

## Local Development

```bash
# Build
cd bin-queue-manager
go build -o ./bin/ ./cmd/...

# Run all tests
go test ./...

# Verify before commit (mandatory)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Further Reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)
