# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-rag-manager is a Go microservice that provides Retrieval-Augmented Generation (RAG) for VoIPBin documentation. It indexes documentation sources (RST dev docs, OpenAPI specs, design docs, CLAUDE.md files), embeds them using OpenAI, stores embeddings in memory, and answers natural language questions grounded in the documentation.

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
5. **pkg/embedder/** - OpenAI embedding client
6. **pkg/store/** - In-memory vector store with file persistence
7. **pkg/retriever/** - Query embedding and cosine similarity search
8. **pkg/generator/** - Prompt building and OpenAI chat completion

### Request Flow

```
RabbitMQ Request -> listenhandler (regex routing) -> raghandler
                                                       |
                                                       +-> retriever -> embedder (OpenAI)
                                                       |             -> store (in-memory)
                                                       +-> generator -> OpenAI Chat API
```

### API Endpoints (via RabbitMQ RPC)

- `POST /v1/rags/query` - Query the RAG system
- `POST /v1/rags/index` - Trigger full re-indexing
- `POST /v1/rags/index/incremental` - Re-index specific files
- `GET /v1/rags/index/status` - Get indexing status

### Configuration

Environment variables:
- `RABBITMQ_ADDRESS` - RabbitMQ connection
- `OPENAI_API_KEY` - OpenAI API key
- `OPENAI_EMBEDDING_MODEL` - Embedding model (default: text-embedding-3-small)
- `RAG_LLM_MODEL` - LLM for answers (default: gpt-4o)
- `RAG_TOP_K` - Chunks to retrieve (default: 5)
- `RAG_CHUNK_MAX_TOKENS` - Max chunk size (default: 800)
- `GCS_BUCKET` - GCS bucket for persistence
- `GCS_EMBEDDINGS_PATH` - Path for embeddings file
- `RAG_DOCS_BASE_PATH` - Base path to document sources
- `PROMETHEUS_ENDPOINT` - Metrics endpoint (default: /metrics)
- `PROMETHEUS_LISTEN_ADDRESS` - Metrics address (default: :2112)
