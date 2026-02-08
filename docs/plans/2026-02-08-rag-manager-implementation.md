RAG Manager Implementation Plan

Reference: docs/plans/2026-02-08-rag-manager-design.md

Phase 1: Foundation (bin-common-handler updates)

Step 1.1: Add queue names and service name constants

File: bin-common-handler/models/outline/queuename.go
- Add QueueNameRagEvent, QueueNameRagRequest, QueueNameRagSubscribe constants

File: bin-common-handler/models/outline/servicename.go
- Add ServiceNameRagManager constant

Step 1.2: Add requesthandler methods for RAG service

File: bin-common-handler/pkg/requesthandler/rag.go (new)
- Add RagV1QueryPost method (sends query to rag-manager via RabbitMQ RPC)
- Add RagV1IndexPost method (triggers full re-indexing)
- Add RagV1IndexIncrementalPost method (triggers incremental re-indexing)
- Add RagV1IndexStatusGet method (returns indexing status)

Step 1.3: Run verification for bin-common-handler

- go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

Phase 2: bin-rag-manager service scaffold

Step 2.1: Create directory structure

Create bin-rag-manager/ with:
- cmd/rag-manager/main.go
- internal/config/config.go
- pkg/listenhandler/main.go
- pkg/raghandler/main.go
- go.mod, Dockerfile, k8s/, CLAUDE.md

Step 2.2: Configuration (internal/config/config.go)

Environment variables:
- RABBITMQ_ADDRESS -- RabbitMQ connection
- PROMETHEUS_ENDPOINT, PROMETHEUS_LISTEN_ADDRESS -- metrics
- OPENAI_API_KEY -- OpenAI API key
- OPENAI_EMBEDDING_MODEL -- default "text-embedding-3-small"
- RAG_LLM_MODEL -- default "gpt-4o"
- RAG_TOP_K -- default 5
- RAG_CHUNK_MAX_TOKENS -- default 800
- GCS_BUCKET -- GCS bucket name
- GCS_EMBEDDINGS_PATH -- path to embeddings file
- RAG_DOCS_BASE_PATH -- base path to document sources

Step 2.3: Main entry point (cmd/rag-manager/main.go)

Follow bin-tag-manager pattern:
- Cobra root command
- Signal handling
- Prometheus metrics endpoint
- Connect to RabbitMQ
- Initialize ragHandler and listenHandler
- No database or Redis needed (in-memory vector store)

Step 2.4: go.mod setup

Module: monorepo/bin-rag-manager
Dependencies:
- monorepo/bin-common-handler (shared models, sockhandler, requesthandler)
- github.com/sashabaranov/go-openai (OpenAI Go client)
- cloud.google.com/go/storage (GCS client)
- Replace directives for all sibling bin-* modules

Phase 3: Document chunking

Step 3.1: Chunker interface and types

File: pkg/chunker/chunker.go
- Define Chunk struct: ID, Text, SourceFile, DocType, SectionTitle, LastUpdated
- Define Chunker interface: Chunk(filePath string) ([]Chunk, error)

Step 3.2: RST chunker

File: pkg/chunker/rst.go
- Parse RST files by section headers (underlined titles)
- Split sections exceeding RAG_CHUNK_MAX_TOKENS at paragraph boundaries
- Extract section titles from RST header syntax

Step 3.3: Markdown chunker

File: pkg/chunker/markdown.go
- Parse Markdown files by ## headings
- Split sections exceeding RAG_CHUNK_MAX_TOKENS at paragraph boundaries
- Extract section titles from heading text

Step 3.4: OpenAPI chunker

File: pkg/chunker/openapi.go
- Parse OpenAPI YAML by path+method combinations
- Each endpoint (path + method + description + schemas) becomes one chunk
- Shared component schemas get their own chunks

Step 3.5: Tests for all chunkers

Phase 4: Embedding and vector store

Step 4.1: OpenAI embedder

File: pkg/embedder/openai.go
- EmbedTexts(texts []string) ([][]float32, error)
- Calls OpenAI text-embedding-3-small API
- Handles batching (OpenAI supports up to 2048 inputs per request)

Step 4.2: In-memory vector store

File: pkg/store/memory.go
- Store struct holding []StoredChunk (chunk + embedding vector)
- Add(chunks []Chunk, embeddings [][]float32) -- add chunks with embeddings
- Search(queryEmbedding []float32, topK int, docTypes []string) []SearchResult
- DeleteByFile(sourceFile string) -- remove all chunks for a file
- Save(bucket, path string) error -- serialize to GCS
- Load(bucket, path string) error -- load from GCS on startup
- Stats() StoreStats -- chunk count, last updated

Cosine similarity search:
- Brute-force dot product over normalized vectors
- Filter by doc_type if specified
- Return top-k results with relevance scores

Step 4.3: GCS persistence

- Use encoding/gob or protobuf for serialization
- Download embeddings file on startup
- Upload after re-indexing completes
- Handle missing file gracefully (first run = empty store)

Step 4.4: Tests for embedder and store

Phase 5: Retrieval and answer generation

Step 5.1: Retriever

File: pkg/retriever/retriever.go
- Query(ctx, query string, topK int, docTypes []string) ([]RetrievedChunk, error)
- Embeds the query using the embedder
- Searches the vector store
- Returns chunks with relevance scores

Step 5.2: Generator

File: pkg/generator/generator.go
- Generate(ctx, query string, chunks []RetrievedChunk) (string, error)
- Builds system prompt: "Answer using only the provided context. Cite sources. Say I don't know if context is insufficient."
- Builds user message with numbered context blocks + query
- Calls OpenAI Chat API directly (configured model, default gpt-4o)
- Returns generated answer

Step 5.3: Tests

Phase 6: RabbitMQ RPC handlers (listenhandler)

Step 6.1: ListenHandler setup

File: pkg/listenhandler/main.go
- ListenHandler interface with Run method
- processRequest router with regexp patterns
- Prometheus metrics for request processing time

Step 6.2: Query handler

File: pkg/listenhandler/v1_query.go
- POST /v1/rags/query
- Parse request: query, doc_types (optional), top_k (optional)
- Call retriever.Query -> generator.Generate
- Return answer + sources with relevance scores

Step 6.3: Index handlers

File: pkg/listenhandler/v1_index.go
- POST /v1/rags/index -- trigger full re-index of all document sources
- POST /v1/rags/index/incremental -- re-index specific files (list in request body)
- GET /v1/rags/index/status -- return last run time, chunk count, errors

Step 6.4: Request/response models

File: pkg/listenhandler/models/request.go
- QueryRequest: Query, DocTypes, TopK
- QueryResponse: Answer, Sources []Source
- IndexRequest: Files []string
- IndexStatusResponse: LastRun, ChunkCount, Errors

Step 6.5: Tests

Phase 7: RAG handler (business logic orchestration)

Step 7.1: RagHandler

File: pkg/raghandler/main.go
- RagHandler interface
- Query(ctx, req QueryRequest) (*QueryResponse, error)
- IndexFull(ctx) error
- IndexIncremental(ctx, files []string) error
- IndexStatus(ctx) (*IndexStatusResponse, error)

File: pkg/raghandler/query.go
- Orchestrates retriever + generator

File: pkg/raghandler/index.go
- Scans document source directories
- Chunks documents using appropriate chunker per file type
- Embeds chunks via embedder
- Updates vector store
- Persists to GCS

Step 7.2: Tests

Phase 8: bin-api-manager integration

Step 8.1: Add RAG endpoint to OpenAPI spec

File: bin-openapi-manager/openapi/openapi.yaml
- Add POST /v1/rags/query endpoint definition
- Add request/response schemas (QueryRequest, QueryResponse, Source)
- Add internal endpoints (POST /v1/rags/index, GET /v1/rags/index/status)

Step 8.2: Regenerate OpenAPI types

- cd bin-openapi-manager && go generate ./...
- cd bin-api-manager && go generate ./...

Step 8.3: Add API handler for RAG query

File: bin-api-manager (appropriate handler file)
- Wire POST /v1/rags/query to forward via RabbitMQ RPC to bin-rag-manager
- Follow existing patterns for request forwarding

Step 8.4: Run verification for bin-api-manager

Phase 9: Packaging and deployment

Step 9.1: Dockerfile

File: bin-rag-manager/Dockerfile
- Multi-stage build (golang:1.25-alpine -> alpine:latest)
- Build binary from cmd/rag-manager/
- Copy to runtime image

Step 9.2: Kubernetes manifests

File: bin-rag-manager/k8s/
- deployment.yml: 1 replica, 128MB memory, 0.1 CPU, env vars
- kustomization.yml
- namespace.yml (bin-manager)

Step 9.3: CircleCI configuration

File: .circleci/config.yml
- Add path-filtering parameter for bin-rag-manager

File: .circleci/config_work.yml
- Add test, build, release jobs for bin-rag-manager
- Add workflow definition

Phase 10: Final verification

Step 10.1: Run full verification for all changed services

- bin-common-handler: go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
- bin-rag-manager: go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
- bin-openapi-manager: go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
- bin-api-manager: go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

Step 10.2: Commit and push

- Commit with message: NOJIRA-Add-rag-manager-service
- Push to remote
- Create PR
