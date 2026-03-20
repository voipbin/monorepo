# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-rag-manager is a Go microservice that provides a multi-tenant knowledge base with Retrieval-Augmented Generation (RAG) for VoIPBin. It indexes documentation sources and customer-uploaded documents, embeds them using Google Gemini, stores embeddings in PostgreSQL with pgvector, and retrieves relevant chunks for queries. Answer generation is handled by ai-manager, not this service.

## Build and Test Commands

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

## Architecture

### Service Layer Structure

1. **cmd/rag-manager/** - Main daemon entry point with configuration via pflag/Viper
2. **pkg/listenhandler/** - RabbitMQ RPC request handler with regex-based URI routing
3. **pkg/raghandler/** - Core business logic orchestrating the RAG pipeline
4. **pkg/chunker/** - Document parsers for RST, Markdown, and OpenAPI YAML
5. **pkg/embedder/** - Google Gemini embedding client (text-embedding-004, 768 dimensions)
6. **pkg/dbhandler/** - PostgreSQL database operations (rags, documents, chunks with pgvector)

### Request Flow

```
RabbitMQ Request -> listenhandler (regex routing) -> raghandler
                                                       |
                                                       +-> embedder (Google Gemini)
                                                       +-> dbhandler -> PostgreSQL (pgvector)
```

### API Endpoints (via RabbitMQ RPC)

Multi-tenant CRUD operations for RAG configurations, documents, and queries (endpoints TBD in Phase 2).

### Configuration

Environment variables (sourced from `bin-manager-secrets` k8s secret via `secretKeyRef`):
- `RABBITMQ_ADDRESS` - RabbitMQ connection (secret key: `RABBITMQ_ADDRESS`)
- `GCP_PROJECT_ID` - GCP project ID for Vertex AI (secret key: `GCP_PROJECT_ID`)
- `GCP_REGION` - GCP region for Vertex AI (secret key: `GCP_REGION`)
- `POSTGRESQL_DSN` - PostgreSQL connection string (secret key: `DATABASE_DSN_POSTGRES`)

Hardcoded in deployment.yml:
- `GOOGLE_EMBEDDING_MODEL` - Embedding model (default: text-embedding-004)
- `RAG_TOP_K` - Chunks to retrieve (default: 5)
- `PROMETHEUS_ENDPOINT` - Metrics endpoint (default: /metrics)
- `PROMETHEUS_LISTEN_ADDRESS` - Metrics address (default: :2112)

Authentication uses GKE Workload Identity (Application Default Credentials) — no API keys required.
