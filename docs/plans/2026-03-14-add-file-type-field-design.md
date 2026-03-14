# Design: Add `type` Field to Storage File

## Problem

1. Customer-uploaded files have no category metadata — can't distinguish RAG documents from chat attachments from general uploads.
2. All customer uploads land in `tmp/<uuid>` — misleading directory name for permanent files.

## Solution

Add a new `type` field to the File model to categorize file purpose. Rename the `tmp/` bucket directory to `storage/`.

### New Field: `type`

A new `Type` field on the `File` model, **separate from** `reference_type`/`reference_id`:

- `type` — What the file **is** (category/purpose)
- `reference_type` + `reference_id` — What the file **belongs to** (linked resource, unchanged)

### Enum Values

| Value         | Meaning               | Set by                                              |
|---------------|-----------------------|-----------------------------------------------------|
| `""`          | General / unspecified  | Default for existing files                          |
| `"rag"`       | RAG document          | Customer via `POST /v1/storage_files` (required)    |
| `"talk"`      | Chat attachment       | Agent via `POST /v1/service_agents/files` (required)|
| `"recording"` | Call recording         | Internal, `bin-call-manager/recordinghandler`       |

### Validation Rules

| Endpoint                          | `type` field | Allowed values  |
|-----------------------------------|-------------|-----------------|
| `POST /v1/storage_files`         | Required    | `"rag"` only    |
| `POST /v1/service_agents/files`  | Required    | `"talk"` only   |
| Internal (call-manager)           | Programmatic| `"recording"`   |

### Storage Path Change

Rename `tmp/` to `storage/` for customer-uploaded files. The `type` field does **not** affect the directory — all customer uploads go to `storage/<uuid>`.

| Source                             | Bucket path            | Change?                    |
|------------------------------------|------------------------|----------------------------|
| Customer uploads (both endpoints)  | `storage/<uuid>`       | Yes (was `tmp/<uuid>`)     |
| Call recordings                    | `recording/<filename>` | No                         |

Existing files in `tmp/` remain accessible — their `filepath` column in DB still points to `tmp/<uuid>`.

## Change Chain (7 layers)

### Layer 1: Database

- `bin-dbscheme-manager` — Alembic migration: add `type VARCHAR` column to files table, default `""`

### Layer 2: Model (`bin-storage-manager/models/file/`)

- `main.go` — Add `Type Type` field with `db:"type"` tag, add `Type` string type and constants (`TypeNone`, `TypeRAG`, `TypeTalk`, `TypeRecording`)
- `webhook.go` — Add `Type` to `WebhookMessage` and `ConvertWebhookMessage()`
- `field.go` — Add `FieldType Field = "type"`

### Layer 3: Storage Manager internal (`bin-storage-manager/`)

- `pkg/listenhandler/models/request/files.go` — Add `Type file.Type` to `V1DataFilesPost`
- `pkg/listenhandler/v1_files.go` — Pass `req.Type` to `storageHandler.FileCreate()`
- `pkg/storagehandler/file.go` — Add `fileType file.Type` param to `FileCreate()`, set it on the created file
- `pkg/storagehandler/main.go` — Update `StorageHandler` interface
- `pkg/dbhandler/` — Ensure `type` column is handled (via existing `PrepareFields`/`ScanRow` pattern)

### Layer 4: Common handler (`bin-common-handler/`)

- `pkg/requesthandler/main.go` — Add `fileType smfile.Type` param to `StorageV1FileCreate()` and `StorageV1FileCreateWithDelay()` interface methods
- `pkg/requesthandler/storage_files.go` — Add `fileType` param to both functions, include in `V1DataFilesPost`

### Layer 5: API Manager (`bin-api-manager/`)

- `server/storage_files.go` — Read `type` from multipart form, validate it's `"rag"`, pass to service handler
- `server/service_agents_files.go` — Read `type` from multipart form, validate it's `"talk"`, pass to service handler
- `pkg/servicehandler/storage_file.go` — Accept `fileType` param, pass through to `storageFileCreate()`, change `tmp/` to `storage/`
- `pkg/servicehandler/serviceagent_file.go` — Accept `fileType` param, pass through, change `tmp/` to `storage/`

### Layer 6: Call Manager (`bin-call-manager/`)

- `pkg/recordinghandler/stop.go` — Pass `smfile.TypeRecording` to `StorageV1FileCreate()`

### Layer 7: OpenAPI (`bin-openapi-manager/`)

- `openapi/openapi.yaml` — Add `StorageManagerFileType` enum (`""`, `"rag"`, `"talk"`, `"recording"`), add `type` property to `StorageManagerFile` schema
- `openapi/paths/storage_files/main.yaml` — Add `type` as required string field in multipart form
- `openapi/paths/service_agents/files.yaml` — Add `type` as required string field in multipart form

## Signature Change Risk

`StorageV1FileCreate` and `StorageV1FileCreateWithDelay` in `bin-common-handler` are shared interfaces. Changing their signatures requires updating **all callers**:

| Caller                             | File                   | New `type` value     |
|------------------------------------|------------------------|----------------------|
| `bin-api-manager` servicehandler   | `storage_file.go`      | passed from API      |
| `bin-api-manager` servicehandler   | `serviceagent_file.go` | passed from API      |
| `bin-call-manager` recordinghandler| `stop.go`              | `TypeRecording`      |

Per monorepo coding conventions — must grep for all import aliases and multi-line call patterns across the monorepo, not just these known callers.

## What's NOT Changing

- `reference_type` / `reference_id` — untouched
- Recording storage path (`recording/<filename>`) — untouched
- Existing `tmp/` files in GCS — remain accessible via their DB `filepath` value
- `StorageV1FileList` / `StorageV1FileGet` / `StorageV1FileDelete` — no signature changes
