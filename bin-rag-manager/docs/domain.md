# bin-rag-manager Domain

## Domain Entities

### RAG Configuration
Per-customer settings for a knowledge base instance. Controls which documents belong to the knowledge base and what parameters are used during retrieval (e.g., `top_k`).

### Document Source
A source file added to a RAG configuration. Supported formats: RST (Sphinx-style), Markdown, OpenAPI YAML. On ingestion the document is parsed by `pkg/chunker`, split into semantic chunks, each chunk is embedded by `pkg/embedder`, and the resulting vectors are stored in PostgreSQL.

### Chunk
A semantically meaningful slice of a document. Stored with its 768-dimension embedding vector in a pgvector column. Chunks are the retrieval unit — query results return ranked chunks, not whole documents.

### Embedding
A 768-dimension floating-point vector produced by Google Gemini `text-embedding-004`. Embeddings are computed by `pkg/embedder` using GKE Workload Identity — no API keys are stored in the service.

## Key Business Rules

- **Customer isolation.** All RAG configurations and their documents are scoped to a `customer_id`. Queries only search within the customer's own knowledge base.
- **Storage dependency.** Document source files are retrieved from `bin-storage-manager` before parsing. A valid file reference is required to add a source.
- **top_k is per-RAG.** The number of chunks returned per query (`RAG_TOP_K`) defaults to 5 and can be tuned globally; per-RAG overrides may exist in the configuration.
- **No answer generation here.** This service returns ranked chunks only. The calling service (`bin-ai-manager`) is responsible for sending those chunks along with the original query to a language model.
- **pgvector, not MySQL.** This service is the only one in the monorepo that uses PostgreSQL with the pgvector extension. Use `POSTGRESQL_DSN` (not `DATABASE_DSN`) in configuration.
- **Chunker format detection.** The `pkg/chunker` selects the parser based on the document's MIME type or file extension. Adding a new document format requires a new parser implementation in `pkg/chunker`.
