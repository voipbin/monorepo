# bin-conference-manager

Multi-party audio conference session management service for VoIPbin. Owns Conference and Conferencecall (participant) entities, coordinates with `bin-call-manager`'s confbridge for audio mixing, and handles conference lifecycle including recording and transcription.

## Key Concepts

- **Conference**: Top-level session entity; types: `conference` (manual stop), `connect` (auto-stops when only 1 participant remains), `queue` (managed by queue-manager)
- **Conferencecall**: A participant in a conference; tracks `reference_id` (call UUID) and join/leave status
- **Confbridge**: Actual audio-mixing bridge owned by `bin-call-manager`; this service stores the `confbridge_id` only
- **Recording**: Initiated/stopped via `bin-call-manager`; `recording_id` stored on the conference
- **Transcription**: Live transcription via `bin-transcribe-manager`; `transcribe_id` stored on the conference
- **Pre/post flows**: Optional flow IDs executed before participants speak and after conference ends

## Public RPC Entrypoints

| Pattern | Operations |
|---------|-----------|
| `POST /v1/conferences` | Create conference |
| `GET /v1/conferences` | List conferences |
| `GET /v1/conferences/<id>` | Get conference |
| `PUT /v1/conferences/<id>` | Update conference |
| `DELETE /v1/conferences/<id>` | Delete conference |
| `POST /v1/conferences/<id>/start` | Start conference |
| `POST /v1/conferences/<id>/stop` | Stop conference |
| `POST /v1/conferences/<id>/recording_start` | Start recording |
| `POST /v1/conferences/<id>/recording_stop` | Stop recording |
| `GET /v1/conferencecalls` | List participants |
| `GET /v1/conferencecalls/<id>` | Get participant |
| `POST /v1/conferencecalls/<id>/kick` | Kick participant |

## Dependencies

- **MySQL** — conference and conferencecall records
- **Redis** — conference and conferencecall cache
- **RabbitMQ** — listen queue `bin-manager.conference-manager.request`; subscribes to `bin-manager.call-manager.event`
- **bin-call-manager** — confbridge create/destroy, recording start/stop
- **bin-transcribe-manager** — transcription start/stop

## Local Development

```bash
# Build
cd bin-conference-manager
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
