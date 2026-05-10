# bin-storage-manager Architecture

## Component Overview

`bin-storage-manager` handles file and media storage for the VoIPbin platform. It manages customer storage accounts with quota enforcement, stores files and call recordings in Google Cloud Storage (GCS), and generates signed download URLs. A CLI tool (`storage-control`) provides direct database access for administrative operations.

```
cmd/storage-manager/    — Daemon entry point (Viper/pflag)
cmd/storage-control/    — Admin CLI (JSON output, bypasses RabbitMQ)
pkg/listenhandler/      — RabbitMQ RPC request router (regex URI dispatch)
pkg/subscribehandler/   — Event subscriber (customer_deleted cascading delete)
pkg/storagehandler/     — Core business logic
pkg/filehandler/        — GCS bucket operations and signed URL generation
pkg/accounthandler/     — Storage account management and quota enforcement
pkg/dbhandler/          — MySQL + Redis cache coordination
pkg/cachehandler/       — Redis cache operations for files and accounts
models/                 — Data structures (file, account, bucketfile, compressfile)
```

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|---------------|
| Entry | `cmd/storage-manager` | Configuration; starts ListenHandler and SubscribeHandler |
| Transport | `pkg/listenhandler` | Consumes `bin-manager.storage-manager.request`; regex-routes to storagehandler |
| Events | `pkg/subscribehandler` | Subscribes to `bin-manager.customer-manager.event`; handles cascading deletes |
| Business logic | `pkg/storagehandler` | Coordinates file lifecycle, quota checks, recording compression |
| GCS operations | `pkg/filehandler` | Bucket CRUD, signed URL generation (local-key or IAM Credentials API) |
| Account management | `pkg/accounthandler` | 10 GB quota enforcement per customer |
| Persistence | `pkg/dbhandler` | MySQL for durable records; Redis for cache |
| Cache | `pkg/cachehandler` | Redis lookups for files and accounts; invalidates on mutation |

### GCS Authentication

Two modes supported:
1. **Service Account JSON** — `GOOGLE_APPLICATION_CREDENTIALS` points to JSON file; private keys used for local signing.
2. **ADC/Workload Identity** — GKE environment; uses Application Default Credentials with IAM Credentials API for signing.

## Request Routing

Requests arrive on the RabbitMQ queue `bin-manager.storage-manager.request`. The `listenhandler` dispatches via regex match in `pkg/listenhandler/main.go`.

| Pattern | Operations |
|---------|-----------|
| `/v1/accounts?` | List accounts (GET with query params) |
| `/v1/accounts$` | Create account (POST) |
| `/v1/accounts/<uuid>$` | Get/Delete account |
| `/v1/files?` | List files (GET with query params) |
| `/v1/files$` | Create file record (POST) |
| `/v1/files/<uuid>$` | Get/Delete file |
| `/v1/files/<uuid>/download_uri_refresh$` | Refresh signed download URL |
| `/v1/compressfiles$` | Create zip archive from multiple files |
| `/v1/recordings/(.*)` | Get/Delete recordings by reference_id |

Request flow:

```
RabbitMQ → listenhandler (regex dispatch)
               |
               v
          storagehandler
          |            |
    filehandler    accounthandler
    (GCS)          (quota check)
          |
       dbhandler → MySQL / Redis
```

### Storage layout in GCS

| Bucket | Directory | Content |
|--------|-----------|---------|
| `gcp_bucket_name_media` | `recording/` | Call recordings (persistent) |
| `gcp_bucket_name_media` | `bin/` | Service-uploaded files (persistent) |
| `gcp_bucket_name_tmp` | `tmp/` | Compressed zip archives (transient, SHA-1 named) |

Download URLs are GCS signed URLs with a default 24-hour expiration.
