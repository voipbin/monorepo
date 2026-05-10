# bin-rag-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|-----------|
| Embedding calls fail / timeout | Vertex AI quota exhausted or GKE Workload Identity misconfigured | Check pod IAM binding; verify `GCP_PROJECT_ID` and `GCP_REGION` |
| "missing go.sum entry" at Docker build | `go mod tidy` not run after dependency change | Run full verification workflow |
| pgvector extension missing | PostgreSQL instance provisioned without pgvector | Run `CREATE EXTENSION IF NOT EXISTS vector` on the database |
| Query returns 0 chunks | RAG config has no indexed sources, or embedding dimension mismatch | Verify documents were successfully ingested; check chunk count in DB |
| Source ingestion fails silently | Unsupported file format or corrupted file in GCS | Check `pkg/chunker` format detection; verify source file is readable |
| RabbitMQ connection refused | RABBITMQ_ADDRESS misconfigured | Verify network policy and secret value in k8s |

## Debugging Guide

```bash
# Check pod logs
kubectl logs -n voipbin -l app=rag-manager --tail=100

# Verify Workload Identity is working
kubectl exec -n voipbin <pod> -- curl -s "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token" -H "Metadata-Flavor: Google"

# Check pgvector extension
psql $POSTGRESQL_DSN -c "SELECT extname FROM pg_extension WHERE extname='vector';"

# Count indexed chunks for a customer
psql $POSTGRESQL_DSN -c "SELECT COUNT(*) FROM chunks WHERE customer_id='<uuid>';"

# Build locally
cd bin-rag-manager && go build -o ./bin/ ./cmd/...

# Run tests
go test ./...

# Run with race detector
go test -race ./...
```

## Configuration

Environment variables (sourced from the `voipbin` k8s secret via `secretKeyRef`):

| Flag / Env | Description | Default |
|-----------|-------------|---------|
| `RABBITMQ_ADDRESS` | RabbitMQ connection string | required |
| `POSTGRESQL_DSN` | PostgreSQL connection (secret key: `DATABASE_DSN_POSTGRES`) | required |
| `GCP_PROJECT_ID` | GCP project ID for Vertex AI | required |
| `GCP_REGION` | GCP region for Vertex AI | required |
| `GOOGLE_EMBEDDING_MODEL` | Gemini embedding model name | `text-embedding-004` |
| `RAG_TOP_K` | Number of chunks returned per query | `5` |
| `GCP_BUCKET_NAME_MEDIA` | GCS bucket for media/source files | required |
| `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

## Prometheus Metrics

Metrics are served at `PROMETHEUS_LISTEN_ADDRESS` (default `:2112`) on `PROMETHEUS_ENDPOINT` (default `/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `rag_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
