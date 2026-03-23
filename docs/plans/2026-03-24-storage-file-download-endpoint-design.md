# Storage File Download Endpoint Design

## Problem Statement

Storage files have a `uri_download` field containing a signed GCS URL, but:

1. The creation code sets `tm_download_expire` to 10 years, while GCS signed URLs have a hard maximum of 7 days. The stored expiration is inaccurate — URLs expire silently after 7 days.
2. There is no dedicated download endpoint. Clients must extract `uri_download` from `GET /storage_files/{id}` and hope it's still valid.
3. There is no mechanism to refresh an expired download URL.

## Solution

Add download endpoints that return an HTTP 307 redirect to a working signed GCS URL. If the stored URL has expired, the endpoint regenerates a fresh one before redirecting.

## Endpoints

### Customer API

```
GET /storage_files/{id}/file
```

- **Auth**: JWT required
- **Permission**: `PermissionCustomerAdmin | PermissionCustomerManager`
- **Response**: HTTP 307 redirect to signed GCS URL
- **Errors**: 400 (bad request / not found / no permission)

### Service Agent API

```
GET /service_agents/files/{id}/file
```

- **Auth**: JWT required
- **Permission**: `f.CustomerID == a.CustomerID` (customer ownership check, same as `ServiceAgentFileGet`)
- **Response**: HTTP 307 redirect to signed GCS URL
- **Errors**: 400 (bad request / not found / no permission)

## Request Flow

```
Client
  → GET /storage_files/{id}/file
  → api-manager (JWT auth)
  → api-manager: storageFileGet(fileID)
      → RPC: StorageV1FileGet(fileID)
      ← File{ID, CustomerID, URIDownload, TMDownloadExpire, ...}
  → api-manager: permission check (PermissionCustomerAdmin|PermissionCustomerManager)
  → api-manager: check TMDownloadExpire
      │
      ├─ Still valid → 307 redirect to stored URIDownload
      │
      └─ Expired → RPC: StorageV1FileDownloadURIRefresh(fileID)  [NEW]
                      → storage-manager: looks up BucketName + Filepath from DB
                      → storage-manager: bucketfileGenerateDownloadURI(bucket, path, 7 days)
                      → storage-manager: updates DB (uri_download, tm_download_expire)
                      ← fresh signed URL
                   → 307 redirect to fresh URL
```

## New RPC: StorageV1FileDownloadURIRefresh

- **Input**: fileID (UUID)
- **What it does**:
  1. Fetches the file record from DB (gets `BucketName`, `Filepath`)
  2. Calls `bucketfileGenerateDownloadURI()` with 7-day expiry
  3. Updates DB: `uri_download` = fresh URL, `tm_download_expire` = now + 7 days
  4. Returns the fresh signed URL
- **No webhook event**: URL refresh is internal bookkeeping, not a meaningful state change

## Expiration Fix

### Creation Code Change

In `bin-storage-manager/pkg/filehandler/file.go`, change:

```go
// Before
expireDuration := 3650 * 24 * time.Hour // valid for 10 years

// After
expireDuration := 7 * 24 * time.Hour // valid for 7 days
```

This makes `tm_download_expire` trustworthy — it now matches the actual GCS signed URL expiration.

### Existing Data Fix (Manual SQL)

All existing files have `tm_download_expire` set 10 years out but their actual GCS URLs expired after 7 days. Reset to force regeneration:

```sql
UPDATE storage_files
SET tm_download_expire = tm_create
WHERE tm_delete IS NULL;
```

No Alembic migration file needed — this is a one-time manual fix.

## Services Touched

### bin-openapi-manager

- Add `GET /storage_files/{id}/file` path definition (307 redirect response, matching `GET /recordingfiles/{id}` pattern)
- Add `GET /service_agents/files/{id}/file` path definition

### bin-common-handler

- Add `StorageV1FileDownloadURIRefresh(ctx, fileID)` request handler method

### bin-storage-manager

- Add RPC handler for the download URI refresh endpoint
- Add business logic: fetch file → generate signed URL → update DB → return URL
- Fix creation expiry: 3650 days → 7 days

### bin-api-manager

- Add server handler for `GET /storage_files/{id}/file` (307 redirect)
- Add server handler for `GET /service_agents/files/{id}/file` (307 redirect)
- Add service handler: `StorageFileDownload` — permission check + expiry check + redirect
- Add service handler: `ServiceAgentFileDownload` — ownership check + expiry check + redirect

## What Does NOT Change

- `StorageFileGet` permission stays `PermissionCustomerAll`
- `GET /storage_files/{id}` continues to return `uri_download` in JSON (may be expired)
- No webhook events triggered on URL refresh
- No changes to `ServiceAgentFileGet` or `ServiceAgentFileDelete`

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| HTTP method | GET | Semantically "get the file", consistent with `GET /recordingfiles/{id}` |
| Response | 307 redirect | Client downloads directly from GCS, no proxy through api-manager |
| Expiry check | Trust `tm_download_expire` | After fixing creation code + running migration, field is accurate |
| URL refresh updates DB | Yes | Next request within 7 days uses cached URL, avoids unnecessary signing |
| Service agent permission | CustomerID check | Download is a read operation, matches `ServiceAgentFileGet` |
| Webhook on refresh | No | Internal bookkeeping, not a user-visible state change |
