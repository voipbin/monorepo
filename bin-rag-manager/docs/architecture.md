# bin-rag-manager Architecture

## Component Overview

`bin-rag-manager` implements a multi-tenant Retrieval-Augmented Generation (RAG) pipeline. It receives requests over RabbitMQ RPC, manages RAG configurations and document sources, embeds text chunks using Google Gemini, stores vectors in PostgreSQL with pgvector, and retrieves the most relevant chunks for query-time lookups. Answer generation is handled by `bin-ai-manager` — this service only retrieves.

```
cmd/rag-manager/        — Cobra/Viper daemon entry point
pkg/listenhandler/      — RabbitMQ RPC request router (regex URI dispatch)
pkg/raghandler/         — Core business logic and pipeline orchestration
pkg/chunker/            — Document parsers (RST, Markdown, OpenAPI YAML)
pkg/embedder/           — Google Gemini embedding client (text-embedding-004)
pkg/dbhandler/          — PostgreSQL operations (pgvector storage and retrieval)
```

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|---------------|
| Entry | `cmd/rag-manager` | Configuration via pflag/Viper; starts ListenHandler |
| Transport | `pkg/listenhandler` | Consumes `bin-manager.rag-manager.request` queue; regex-routes to raghandler |
| Business logic | `pkg/raghandler` | Orchestrates RAG lifecycle: CRUD, ingest pipeline, query execution |
| Document parsing | `pkg/chunker` | Splits RST, Markdown, and OpenAPI YAML into semantic chunks |
| Embedding | `pkg/embedder` | Calls Google Gemini `text-embedding-004`; produces 768-dim vectors |
| Persistence | `pkg/dbhandler` | Reads/writes rags, documents, and chunks; nearest-neighbor search via pgvector |

Unlike most services in the monorepo, this service uses **PostgreSQL** (not MySQL) with the pgvector extension. The DSN is supplied via `POSTGRESQL_DSN`.

## Request Routing

Requests arrive on the RabbitMQ queue `bin-manager.rag-manager.request`. The `listenhandler` dispatches by matching the request URI against compiled regexes in `pkg/listenhandler/main.go`.

| Pattern | Operations |
|---------|-----------|
| `^/v1/rags(\?.*)?$` | List RAGs (GET), Create RAG (POST) |
| `^/v1/rags/<uuid>(\?.*)?$` | Get RAG (GET), Delete RAG (DELETE) |
| `^/v1/rags/<uuid>/sources(\?.*)?$` | List sources (GET), Add source (POST) |
| `^/v1/rags/<uuid>/sources/<uuid>(\?.*)?$` | Get source (GET), Delete source (DELETE) |
| `^/v1/query$` | Embed query and return top-K chunks (POST) |

Request flow:

```
RabbitMQ → listenhandler (regex dispatch)
               |
               v
           raghandler
           |         |
     embedder     dbhandler
     (Gemini)    (pgvector)
```

Authentication to Google Vertex AI uses GKE Workload Identity (Application Default Credentials). No API keys are required in production.
