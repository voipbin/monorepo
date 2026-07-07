# bin-storage-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|-----------|
| File upload rejected | Customer storage quota (10 GB) exceeded | Check account usage; delete old files or contact customer |
| Signed URL expired | Default 24 h expiry passed | Call `POST /v1/files/<uuid>/download_uri_refresh` |
| GCS signed URL generation fails | `GOOGLE_APPLICATION_CREDENTIALS` missing/invalid, or the service account JSON key lacks bucket permissions | Verify the env var points to a valid service account key file with `storage.objects.get`/`storage.signUrl`. **Known failure mode**: if the key file is missing at startup, `storage-manager`/`storage-control` fail fast and exit non-zero (`NewFileHandler` returns an error that aborts `main()`) — the process will not start with a broken signed-URL path. |
| Cascading delete incomplete | `subscribehandler` missed `customer_deleted` event | Check RabbitMQ dead-letter queue; re-publish event manually |
| Redis cache stale | Crash between DB write and cache invalidation | Restart pod — cache keys expire; DB is the source of truth |
| Compressfile generation slow | Many large recordings with same `reference_id` | Expected; zip is built synchronously on first request |

## Debugging Guide

```bash
# Pod logs
kubectl logs -n voipbin -l app=storage-manager --tail=100

# Check account quota for a customer
./bin/storage-control account get --id <uuid>

# List files for a customer
./bin/storage-control file list --customer_id <uuid> --limit 50

# Check recording files
./bin/storage-control recording list --customer_id <uuid> --limit 20

# Delete a specific file
./bin/storage-control file delete --id <uuid>

# Build
cd bin-storage-manager && go build -o ./bin/storage-manager ./cmd/storage-manager

# Run tests
go test ./...

# Coverage report
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html
```

## Configuration

| Flag / Env | Description | Default |
|-----------|-------------|---------|
| `DATABASE_DSN` | MySQL connection string | required |
| `RABBITMQ_ADDRESS` | RabbitMQ server | required |
| `REDIS_ADDRESS` | Redis server | required |
| `REDIS_PASSWORD` | Redis auth | optional |
| `REDIS_DATABASE` | Redis DB index | optional |
| `GCP_PROJECT_ID` | Google Cloud project | required |
| `GCP_BUCKET_NAME_MEDIA` | Persistent media GCS bucket | required |
| `GCP_BUCKET_NAME_TMP` | Temporary zip GCS bucket | required |
| `GOOGLE_APPLICATION_CREDENTIALS` | SA JSON key file path (mounted in k8s from `Secret/voipbin` key `GOOGLE_APPLICATION_CREDENTIALS_JSON` at `/var/secrets/google/service-account.json`) | required |
| `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `storage_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `storage_manager_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event processing duration |
