# bin-storage-manager

File and media storage service for the VoIPbin platform. Manages customer storage accounts with 10 GB quota enforcement, stores files and call recordings in Google Cloud Storage, generates signed download URLs, and handles cascading deletes when customers are removed.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file covers only what is specific to `bin-storage-manager`.

## Key facts

- **MySQL + Redis** for records and cache.
- **Two GCS buckets**: `gcp_bucket_name_media` (persistent) and `gcp_bucket_name_tmp` (transient zips).
- **Signed URLs** default to 24-hour expiry; refresh via `POST /v1/files/<uuid>/download_uri_refresh`.
- **RabbitMQ queue**: `bin-manager.storage-manager.request`
- **Subscribes to**: `bin-manager.customer-manager.event` (`customer_deleted` → cascading delete)

## Package layout

| Package | Role |
|---------|------|
| `cmd/storage-manager` | Daemon entry point |
| `cmd/storage-control` | Admin CLI (JSON output, bypasses RabbitMQ) |
| `pkg/listenhandler` | RabbitMQ RPC router (regex dispatch) |
| `pkg/subscribehandler` | Event consumer (customer_deleted) |
| `pkg/storagehandler` | Core business logic |
| `pkg/filehandler` | GCS operations and signed URL generation (signing needs a `GOOGLE_APPLICATION_CREDENTIALS` service account key file; no in-cluster metadata-server fallback — see [Signing degradation](#signing-degradation)) |
| `pkg/accounthandler` | 10 GB quota enforcement |
| `pkg/dbhandler` | MySQL reads/writes |
| `pkg/cachehandler` | Redis cache |

## Request routing

| Pattern | Operations |
|---------|-----------|
| `/v1/accounts?` | GET (list) |
| `/v1/accounts$` | POST (create) |
| `/v1/accounts/<uuid>$` | GET, DELETE |
| `/v1/files?` | GET (list) |
| `/v1/files$` | POST (create) |
| `/v1/files/<uuid>$` | GET, DELETE |
| `/v1/files/<uuid>/download_uri_refresh$` | POST (refresh URL) |
| `/v1/compressfiles$` | POST (create zip) |
| `/v1/recordings/(.*)` | GET, DELETE (by reference_id) |

## storage-control CLI

```bash
./bin/storage-control file list --customer_id <uuid> --limit 50
./bin/storage-control account get --id <uuid>
./bin/storage-control recording list --customer_id <uuid>
```

## Common commands

```bash
go build -o ./bin/storage-manager ./cmd/storage-manager
go test ./...
go generate ./pkg/filehandler/...
golangci-lint run -v --timeout 5m
```

## Configuration

| Env | Description | Default |
|-----|-------------|---------|
| `DATABASE_DSN` | MySQL DSN | required |
| `RABBITMQ_ADDRESS` | RabbitMQ server | required |
| `REDIS_ADDRESS` | Redis server | required |
| `GCP_PROJECT_ID` | GCP project | required |
| `GCP_BUCKET_NAME_MEDIA` | Persistent bucket | required |
| `GCP_BUCKET_NAME_TMP` | Temporary bucket | required |
| `GOOGLE_APPLICATION_CREDENTIALS` | Service account JSON key path, used to sign download URLs | optional (see below) |
| `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

## Signing degradation

`GOOGLE_APPLICATION_CREDENTIALS` is optional — the service boots without it instead of
crash-looping.

- **Unset**: `NewFileHandler` logs a warning and returns a working handler with no
  private key. Every signing-dependent read path (`DownloadURIGet`,
  `DownloadURIRefresh`, and their `storagehandler` consumers `CompressfileCreate` and
  `RecordingGet`) returns a structured `*cerrors.VoipbinError` with status
  `UNAVAILABLE` and reason `SIGNING_NOT_CONFIGURED`, which `listenhandler.errorResponse`
  translates into a typed 503 for API callers instead of an opaque 500.
- **Unset AND no other ADC source**: the variable is also the storage client's primary
  application-default-credentials source, so on a deployment with no credential at all
  (no key file, no GKE workload-identity metadata server) the authenticated client
  cannot be built either. `newStorageClient` falls back to an unauthenticated client so
  the process still starts; GCS calls then fail per request rather than at boot. Only
  the non-GCS capabilities (file get/list from MySQL, account bookkeeping, the
  `customer_deleted` cascade) are fully functional in that mode.
- **Set but unreadable/unparsable**: still fatal. Absence of the variable is an
  intentional keyless deployment; a broken path is misconfiguration and must be loud.
- **File `Create`**: tolerates ANY download-URI failure — the missing-key error above
  and a sign-time rejection of a present-but-invalid key. The GCS object has already
  been moved to its final location by that point, so failing would break the primary
  write path AND orphan the object. Instead the record is persisted with an empty
  `uri_download` and a NULL `tm_download_expire`, the error is logged, and `Create`
  succeeds. `POST /v1/files/<uuid>/download_uri_refresh` populates the URI later, once
  a usable signing key exists.

Unaffected without a key: file get/list/delete, account bookkeeping, and the
`customer_deleted` cascade.

## Further reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)
