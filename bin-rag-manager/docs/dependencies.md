# bin-rag-manager Dependencies

## Upstream Services (consumed via RabbitMQ RPC)

| Service | Purpose |
|---------|---------|
| `bin-ai-manager` | AI engine coordination (queries RAG results) |
| `bin-storage-manager` | Retrieves document source files for ingestion |
| `bin-billing-manager` | Billing checks for RAG operations |
| `bin-customer-manager` | Customer validation |
| `bin-agent-manager` | Agent context lookup |

## Infrastructure Dependencies

| Dependency | Use |
|-----------|-----|
| RabbitMQ | Listens on `bin-manager.rag-manager.request` for RPC requests |
| PostgreSQL + pgvector | Stores RAG configs, document metadata, and 768-dim embedding vectors |
| Google Vertex AI (Gemini) | `text-embedding-004` model for chunk and query embedding; authenticated via GKE Workload Identity |
| GCS (`gcp_bucket_name_media`) | Source document files are read from this bucket via `bin-storage-manager` |

## Monorepo Module Dependencies

This service declares `replace` directives in `go.mod` for every other `bin-*` service in the monorepo. At compile time only the packages actually imported are linked; the full list of replace directives allows cross-service type sharing.

Key local imports:
- `monorepo/bin-common-handler` — sockhandler, requesthandler, notifyhandler
- `monorepo/bin-storage-manager` — file retrieval models
- `monorepo/bin-ai-manager` — AI engine types

## Reverse Dependencies

`bin-ai-manager` calls this service to retrieve relevant chunks before generating AI responses. No other runtime services call `bin-rag-manager` directly.
