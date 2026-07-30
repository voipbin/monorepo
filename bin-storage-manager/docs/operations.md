# bin-storage-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|-----------|
| File upload rejected | Customer storage quota (10 GB) exceeded | Check account usage; delete old files or contact customer |
| Signed URL expired | Default 24 h expiry passed | Call `POST /v1/files/<uuid>/download_uri_refresh` |
| GCS signed URL generation fails (`503` / reason `SIGNING_NOT_CONFIGURED`) | `GOOGLE_APPLICATION_CREDENTIALS` is not set, so the handler has no private key | Expected degraded mode — the service still runs. Set the env var to a valid service account key file with `storage.objects.get`/`storage.signUrl` and restart to restore signed URLs. See "Signing degradation" below. |
| GCS signed URL generation fails (untyped error, key is configured) | The configured key is present but rejected by the signer, or the service account lacks bucket permissions | Verify the key file content and the service account's IAM roles. Create still succeeds (with an empty download URI); read paths surface the underlying signing error. |
| `storage-manager`/`storage-control` exit non-zero at startup | `GOOGLE_APPLICATION_CREDENTIALS` is set but points to a missing/unparsable file | Fix or unset the variable. A set-but-broken path is misconfiguration and is intentionally fatal; unsetting it selects the degraded keyless mode instead. |
| Every GCS operation fails, `newStorageClient` logged a fallback warning | No credential at all: `GOOGLE_APPLICATION_CREDENTIALS` unset and no metadata server | The process intentionally starts with an unauthenticated client rather than crash-looping. Provide a key file or run under GKE workload identity. |
| File created with empty `uri_download` | Signing was unavailable when the file was created | Expected — `Create` never fails on a signing error (the object is already moved, so failing would orphan it). Call `POST /v1/files/<uuid>/download_uri_refresh` once a usable key is configured. |
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
| `GOOGLE_APPLICATION_CREDENTIALS` | SA JSON key file path (mounted in k8s from `Secret/voipbin` key `GOOGLE_APPLICATION_CREDENTIALS_JSON` at `/var/secrets/google/service-account.json`). Required for signed download URLs; unset runs the service in the degraded keyless mode below | optional |
| `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

## Signing degradation

A missing signing credential is non-fatal. `NewFileHandler` logs a warning and returns a
working handler with no private key, so the service keeps serving everything that does
not need a signed URL: file get/list/delete, account bookkeeping, and the
`customer_deleted` cascade.

`GOOGLE_APPLICATION_CREDENTIALS` is also the storage client's primary
application-default-credentials source. If it is unset and no other ADC source exists
(no GKE workload-identity metadata server), `newStorageClient` logs a warning and falls
back to an unauthenticated client so the process still starts; GCS calls then fail per
request instead of at boot. That fallback is gated on the variable being unset: when a
credential file IS configured, a client-construction failure stays fatal so the pod
crash-loops and self-heals instead of silently running unauthenticated forever.

Monitor with `storage_manager_signing_available` (0 = degraded) and
`storage_manager_download_uri_failure_total` (see Prometheus Metrics below).

| Path | Behavior without a usable signing key |
|------|--------------------------------------|
| `DownloadURIGet`, `DownloadURIRefresh`, `CompressfileCreate`, `RecordingGet` | Return a structured `*cerrors.VoipbinError` (status `UNAVAILABLE`, reason `SIGNING_NOT_CONFIGURED`) when no key is configured, which `listenhandler.errorResponse` turns into a typed HTTP 503 for API callers. A present-but-rejected key surfaces the underlying signing error instead. |
| `Create` (file upload) | Succeeds. Any download-URI failure — missing key or sign-time rejection — is logged, and the record is persisted with an empty `uri_download` and a NULL `tm_download_expire`. Failing here would break the primary write path and orphan the already-moved GCS object. |

Recovery: configure a valid `GOOGLE_APPLICATION_CREDENTIALS`, restart, then call
`POST /v1/files/<uuid>/download_uri_refresh` for any record left with an empty
`uri_download`.

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `storage_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `storage_manager_receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event processing duration |
| `storage_manager_signing_available` | Gauge | — | `1` when a GCS signing credential was loaded at startup, `0` when signed download URLs are unavailable |
| `storage_manager_download_uri_failure_total` | Counter | `reason` | File creations persisted without a download URI. `reason=not_configured` (no credential) or `reason=signer_error` (credential present but rejected) |
