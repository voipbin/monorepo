# bin-rag-manager

Multi-tenant RAG (Retrieval-Augmented Generation) service. Indexes document sources as vector embeddings and retrieves the most relevant chunks for AI query requests. Answer generation is done by `bin-ai-manager` — this service retrieves only.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file covers only what is specific to `bin-rag-manager`.

## Key facts

- **PostgreSQL + pgvector** (not MySQL). DSN via `POSTGRESQL_DSN`.
- **Google Gemini** (`text-embedding-004`, 768-dim vectors). Auth via GKE Workload Identity — no API keys.
- **RabbitMQ queue**: `bin-manager.rag-manager.request`
- **No SubscribeHandler** — no event subscriptions.

## Package layout

| Package | Role |
|---------|------|
| `cmd/rag-manager` | Daemon entry point (Cobra/Viper) |
| `pkg/listenhandler` | RabbitMQ RPC router (regex dispatch) |
| `pkg/raghandler` | Business logic: CRUD, ingest pipeline, query |
| `pkg/chunker` | Parsers for RST, Markdown, OpenAPI YAML |
| `pkg/embedder` | Gemini embedding client |
| `pkg/dbhandler` | PostgreSQL reads/writes; pgvector nearest-neighbor |

## Request routing

| Pattern | Methods |
|---------|---------|
| `^/v1/rags(\?.*)?$` | GET (list), POST (create) |
| `^/v1/rags/<uuid>(\?.*)?$` | GET, DELETE |
| `^/v1/rags/<uuid>/sources(\?.*)?$` | GET, POST |
| `^/v1/rags/<uuid>/sources/<uuid>(\?.*)?$` | GET, DELETE |
| `^/v1/query$` | POST (embed query, return top-K chunks) |

## Common commands

```bash
go build -o ./bin/ ./cmd/...
go test ./...
go generate ./...
golangci-lint run -v --timeout 5m
```

## Configuration

| Env | Description | Default |
|-----|-------------|---------|
| `RABBITMQ_ADDRESS` | RabbitMQ server | required |
| `POSTGRESQL_DSN` | PostgreSQL + pgvector | required |
| `GCP_PROJECT_ID` | Vertex AI project | required |
| `GCP_REGION` | Vertex AI region | required |
| `GOOGLE_EMBEDDING_MODEL` | Embedding model | `text-embedding-004` |
| `RAG_TOP_K` | Chunks per query | `5` |
| `GCP_BUCKET_NAME_MEDIA` | GCS media bucket | required |
| `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

## Further reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)
