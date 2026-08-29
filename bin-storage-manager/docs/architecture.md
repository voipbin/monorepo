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
| Events | `pkg/subscribehandler` | Consumes customer events via pattern bindings on the global topic exchange `bin-manager.event` (sole intake mechanism since VOIP-1407); handles cascading deletes |
| Business logic | `pkg/storagehandler` | Coordinates file lifecycle, quota checks, recording compression |
| GCS operations | `pkg/filehandler` | Bucket CRUD, signed URL generation (service account JSON key file, local signing) |
| Account management | `pkg/accounthandler` | 10 GB quota enforcement per customer |
| Persistence | `pkg/dbhandler` | MySQL for durable records; Redis for cache |
| Cache | `pkg/cachehandler` | Redis lookups for files and accounts; invalidates on mutation |

### GCS Authentication

`GOOGLE_APPLICATION_CREDENTIALS` should point to a service account JSON key file. The key's
private key is used for local GCS V4 signed-URL signing (`storage.SignedURLOptions.PrivateKey`).
There is no in-cluster metadata-server / IAM Credentials API fallback for signing.

The variable is **optional**, not required. If it is unset, `NewFileHandler` logs a warning
and returns a working handler with no private key: the service starts, and only the
signing-dependent paths degrade (structured `UNAVAILABLE` / `SIGNING_NOT_CONFIGURED`; file
`Create` persists an empty `uri_download`). If it is set but the file cannot be read or
parsed, startup still fails — that is misconfiguration rather than an intentional keyless
deployment. See [operations.md](operations.md) for the full degradation matrix.

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

## Event Subscriptions

SubscribeHandler (`pkg/subscribehandler/`) consumes from the queue `bin-manager.storage-manager.subscribe`. Since VOIP-1406 the queue is bound to the **global topic exchange `bin-manager.event`** with one pattern per dispatched (publisher, event-type) pair — 2 patterns total, pinned byte-for-byte by the binding golden test (`pkg/subscribehandler/binding_golden_test.go`):

| Pattern | Purpose |
|---------|---------|
| `customer-manager.customer.*.created` | Customer created — provisions the customer's storage account |
| `customer-manager.customer.*.deleted` | Customer deleted — cascading delete of the customer's files and account |

As of VOIP-1407 this topic-pattern binding is the **sole intake mechanism**; the old per-service fanout subscription (`QueueSubscribe` to `bin-manager.customer-manager.event`) has been removed from `Run()` entirely, along with the fanout-unbind step that used to follow a successful topic bind.

## Events Published

State changes emit events on the fanout exchange `bin-manager.storage-manager.event`:

| Event | Payload | Trigger |
|-------|---------|---------|
| `Account_created` | `*account.Account` | Account create (`pkg/accounthandler`) |
| `Account_updated` | `*account.Account` | Quota counters changed by `IncreaseFileInfo` / `DecreaseFileInfo` |
| `Account_deleted` | `*account.Account` | Account delete |
| `file_created` | `*file.File` | File record create (`pkg/filehandler`) |
| `file_deleted` | `*file.File` | File delete |

`file_updated` is declared in `models/file` but has no publish site.

### Global topic exchange (VOIP-1404 / VOIP-1405)

All three NotifyHandler construction sites — `cmd/storage-manager/main.go`, and both
`cmd/storage-control/main.go` sites (`initAccountHandler` for the account-only CLI path and
`initHandler` for the full storage path) — construct their NotifyHandler with
`notifyhandler.WithGlobalTopicPublish()`. Every event is therefore published twice: once to the
per-service fanout exchange above (unchanged, still the system of record) and once to the global
topic exchange `bin-manager.event` with the routing key
`storage-manager.<resource>.<id>.<action>`. The three sites must stay in lockstep on this option —
enabling it in only some would leave consumers with gaps depending on which process published. A
topic publish failure never propagates to the caller and never affects the fanout publish.

storage-manager uses no subscription-id override: both `account.Account` and `file.File` are
addressed by their own `id`. Note that the account event-type constants are capitalized
(`Account_created`, …); `eventtopic.RoutingKey` lowercases every segment, so the wire keys still
read `storage-manager.account.<account-id>.created`. The golden table pinning every key is
`models/account/routingkey_golden_test.go`; the schema is defined in monorepo
`docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md`.
