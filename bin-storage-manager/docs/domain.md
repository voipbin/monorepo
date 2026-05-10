# bin-storage-manager Domain

## Domain Entities

### Storage Account
A per-customer record that tracks total storage usage and enforces the 10 GB quota. Every customer must have exactly one storage account before files can be uploaded. Managed by `pkg/accounthandler`.

### File
An object stored in the `gcp_bucket_name_media` GCS bucket. Each file record has a `reference_type` (`normal` or `recording`) and an optional `reference_id` that links files to a specific call or resource.

- `normal` — generic uploaded files (e.g., audio prompts, TTS outputs)
- `recording` — call recordings; files sharing the same `reference_id` can be compressed together

### Compressfile
An on-demand zip archive generated from all files that share a `reference_id`. The zip is stored in the `gcp_bucket_name_tmp` bucket with a SHA-1 named path. The signed URL expires; re-requesting generates a new zip if needed.

### Recording
A logical grouping of one or more `recording`-type files under the same `reference_id`. The `GET /v1/recordings/<reference_id>` endpoint creates a compressed zip of all matching files and returns a signed URL.

## Key Business Rules

- **10 GB quota per customer.** The `accounthandler` checks total usage before accepting any new file upload. Uploads that would exceed the quota are rejected.
- **Cascading deletes.** When `bin-customer-manager` emits `customer_deleted`, all storage accounts and all files belonging to that customer are removed. This is handled by `pkg/subscribehandler`.
- **Signed URL expiry.** Download URLs are GCS signed URLs with a 24-hour default expiry. Clients that need a fresh URL can call `POST /v1/files/<uuid>/download_uri_refresh`.
- **Cache consistency.** All mutations (create, update, delete) invalidate the corresponding Redis keys in `pkg/cachehandler`. Reads check Redis first; on miss, fall through to MySQL.
- **Two buckets, two lifetimes.** Media bucket content is persistent; tmp bucket content is transient. Compressfiles in tmp are not tracked in the database and may be garbage-collected by GCS lifecycle rules.
- **Reference types are immutable.** Once a file is created as `normal` or `recording`, the type cannot be changed. Business operations that depend on type (compression, cascading recording deletes) rely on this invariant.
