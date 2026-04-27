# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-rag-manager` is a Go microservice that provides a multi-tenant knowledge base with Retrieval-Augmented Generation (RAG) for VoIPbin. It indexes documentation sources and customer-uploaded documents, embeds them using Google Gemini, stores embeddings in PostgreSQL with pgvector, and retrieves relevant chunks for queries. Answer generation is handled by `bin-ai-manager`, not this service.

**Key Concepts:**
- **RAG configuration**: Per-customer settings for the knowledge base.
- **Document**: A source file (RST, Markdown, or OpenAPI YAML) parsed into chunks.
- **Chunk**: A semantically meaningful slice of a document with an embedding vector.
- **Embedding**: 768-dimension vector produced by Google Gemini `text-embedding-004`.
- **pgvector**: PostgreSQL extension storing the embedding vectors and providing nearest-neighbor search.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-rag-manager`.

## Architecture

### Service Communication Pattern

This service uses **RabbitMQ for RPC-style communication**:
- **ListenHandler** (`pkg/listenhandler/`): Consumes RPC requests from queue `bin-manager.rag-manager.request` and routes to `raghandler` via regex-based URI matching.
- **NotifyHandler**: Publishes RAG lifecycle events when configurations or documents change.

### Service Layer Structure

1. **cmd/rag-manager/** — Main daemon entry point (Cobra/Viper configuration)
2. **pkg/listenhandler/** — RabbitMQ RPC request handler
3. **pkg/raghandler/** — Core business logic orchestrating the RAG pipeline
4. **pkg/chunker/** — Document parsers for RST, Markdown, and OpenAPI YAML
5. **pkg/embedder/** — Google Gemini embedding client (`text-embedding-004`, 768 dimensions)
6. **pkg/dbhandler/** — PostgreSQL operations (rags, documents, chunks with pgvector)

### Request Flow

```
RabbitMQ Request -> listenhandler (regex routing) -> raghandler
                                                       |
                                                       +-> embedder (Google Gemini)
                                                       +-> dbhandler -> PostgreSQL (pgvector)
```

## Common Commands

```bash
# Build the service
go build -o ./bin/ ./cmd/...

# Run the daemon
./bin/rag-manager

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Lint
golangci-lint run -v --timeout 5m

# Generate mocks
go generate ./...
```

## Request Routing

Multi-tenant CRUD operations for RAG configurations, documents, and queries (see `pkg/listenhandler/main.go` for the authoritative regex patterns):

**RAGs API (`/v1/rags/*`):**
- `POST /v1/rags` — Create RAG configuration
- `GET /v1/rags?<filters>` — List RAGs
- `GET /v1/rags/<uuid>` — Get RAG
- `DELETE /v1/rags/<uuid>` — Delete RAG

**Documents API (`/v1/documents/*`):**
- `POST /v1/documents` — Add document (parses, chunks, embeds, stores)
- `GET /v1/documents?<filters>` — List documents
- `DELETE /v1/documents/<uuid>` — Remove a document

**Queries API (`/v1/queries/*`):**
- `POST /v1/queries` — Embed query and return top-K relevant chunks

## Event Subscriptions

This service does not subscribe to external events. There is no SubscribeHandler.

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared utilities (sockhandler, requesthandler, notifyhandler)

Always run `go mod vendor` after changing dependencies.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock) plus pgvector-aware integration tests:
- Mock interfaces co-located with handlers (`mock_*.go`)
- Table-driven tests with struct slices

```go
tests := []struct {
    name      string
    input     InputType
    mockSetup func(*MockHandler)
    expectRes ResultType
    expectErr bool
}{
    {"success case", input1, setupMock1, expected1, false},
    {"error case", input2, setupMock2, nil, true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        // test implementation
    })
}
```

## Key Implementation Details

### PostgreSQL with pgvector
Unlike most services in the monorepo (which use MySQL), this service uses PostgreSQL with the pgvector extension to store and search 768-dimension embedding vectors. The DSN is provided via `POSTGRESQL_DSN`.

### GKE Workload Identity Auth
Authentication to Google Vertex AI for embeddings uses GKE Workload Identity (Application Default Credentials) — no API keys required in production.

### Document Parsers
`pkg/chunker/` contains parsers for RST (Sphinx-style), Markdown, and OpenAPI YAML — each producing a stream of semantically coherent chunks before embedding.

## Configuration

Environment variables (sourced from the `voipbin` k8s secret via `secretKeyRef`):

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `RABBITMQ_ADDRESS` | RabbitMQ server | required |
| `GCP_PROJECT_ID` | GCP project ID for Vertex AI | required |
| `GCP_REGION` | GCP region for Vertex AI | required |
| `POSTGRESQL_DSN` | PostgreSQL connection (secret key: `DATABASE_DSN_POSTGRES`) | required |
| `GOOGLE_EMBEDDING_MODEL` | Embedding model | `text-embedding-004` |
| `RAG_TOP_K` | Chunks to retrieve per query | `5` |
| `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

## Prometheus Metrics

Service exposes metrics on the configured endpoint (default `:2112/metrics`):
- `rag_manager_receive_request_process_time` — histogram of RPC request processing time (labels: type, method)
