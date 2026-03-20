# Remove Duplicate /files Endpoint & Fix /storage_files DELETE Bug

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove the duplicate `/files` API endpoint (identical to `/storage_files`) and fix the DELETE `/storage_files/{id}` bug where it calls `ServiceAgentFileDelete` instead of `StorageFileDelete`.

**Architecture:** The `/files` and `/storage_files` endpoints are exact duplicates calling the same servicehandler methods. We remove `/files` from OpenAPI spec, delete its server handlers, and fix the wrong method call in the `/storage_files` DELETE handler. The shared servicehandler layer (`StorageFileCreate/Get/List/Delete`) stays unchanged.

**Tech Stack:** Go, OpenAPI 3.0, oapi-codegen, Sphinx RST docs

---

### Task 1: Fix the DELETE /storage_files/{id} bug in server handler

**Files:**
- Modify: `bin-api-manager/server/storage_files.go:165`

**Step 1: Fix the wrong method call**

In `bin-api-manager/server/storage_files.go`, line 165, change:
```go
res, err := h.serviceHandler.ServiceAgentFileDelete(c.Request.Context(), &a, target)
```
to:
```go
res, err := h.serviceHandler.StorageFileDelete(c.Request.Context(), &a, target)
```

This fixes the permission model from owner-only (`ServiceAgentFileDelete` checks `f.OwnerID == a.ID`) to role-based (`StorageFileDelete` checks Admin/Manager permission), consistent with the other `/storage_files` operations.

**Step 2: No commit yet** — continue to Task 2.

---

### Task 2: Fix the DELETE /storage_files/{id} test

**Files:**
- Modify: `bin-api-manager/server/storage_files_test.go:407`

**Step 1: Update the test expectation**

In `bin-api-manager/server/storage_files_test.go`, line 407, change:
```go
mockSvc.EXPECT().ServiceAgentFileDelete(req.Context(), &tt.agent, tt.expectFileID).Return(tt.responseFile, nil)
```
to:
```go
mockSvc.EXPECT().StorageFileDelete(req.Context(), &tt.agent, tt.expectFileID).Return(tt.responseFile, nil)
```

**Step 2: Verify the fix compiles and test passes**

Run:
```bash
cd bin-api-manager && go test ./server/ -run Test_DeleteStorageFilesId -v
```
Expected: PASS

**Step 3: No commit yet** — continue to Task 3.

---

### Task 3: Move constMaxFileSize and delete files.go

**Files:**
- Modify: `bin-api-manager/server/storage_files.go` (add constant)
- Delete: `bin-api-manager/server/files.go`
- Delete: `bin-api-manager/server/files_test.go`

**Step 1: Add constMaxFileSize to storage_files.go**

Add the constant block at the top of `storage_files.go`, after the import block (before `func (h *server) PostStorageFiles`):

```go
const (
	constMaxFileSize = int64(30 << 20) // Max upload file size. 30 MB.
)
```

**Step 2: Delete files.go and files_test.go**

```bash
cd bin-api-manager
rm server/files.go
rm server/files_test.go
```

**Step 3: Verify compilation still works**

Note: This will fail until the OpenAPI spec is updated (Task 4) and code is regenerated, because the generated `ServerInterface` still requires `PostFiles`, `GetFiles`, `GetFilesId`, `DeleteFilesId`. That's expected — proceed to Task 4.

**Step 4: No commit yet** — continue to Task 4.

---

### Task 4: Remove /files from OpenAPI spec

**Files:**
- Delete: `bin-openapi-manager/openapi/paths/files/main.yaml`
- Delete: `bin-openapi-manager/openapi/paths/files/id.yaml`
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Delete the /files path spec files**

```bash
cd bin-openapi-manager
rm openapi/paths/files/main.yaml
rm openapi/paths/files/id.yaml
rmdir openapi/paths/files
```

**Step 2: Remove /files path references from openapi.yaml**

Remove these 4 lines (around lines 6771-6774):
```yaml
  /files/{id}:
    $ref: './paths/files/id.yaml'
  /files:
    $ref: './paths/files/main.yaml'
```

**Step 3: Update descriptions referencing "POST /files" to "POST /storage_files"**

In `openapi/openapi.yaml`:

- Line ~5235: Change `POST /files` to `POST /storage_files`
  ```yaml
  description: "The storage file ID if the source is an uploaded file. Returned from the `POST /storage_files` response."
  ```

- Line ~5599: Change `POST /files` and `GET /files` to `POST /storage_files` and `GET /storage_files`
  ```yaml
  description: "The unique identifier of the file. Returned from the `POST /storage_files` or `GET /storage_files` response."
  ```

- Line ~5651: This is the `uri_download` example URL — no change needed (it's a generic example URL, not an API path reference).

- Line ~5873: Change `GET /files` to `GET /storage_files`
  ```yaml
  description: "The unique identifier of the file. Valid only if the type is `file`. Returned from the `GET /storage_files` response."
  ```

**Step 4: No commit yet** — continue to Task 5.

---

### Task 5: Update /files references in RAG OpenAPI paths

**Files:**
- Modify: `bin-openapi-manager/openapi/paths/rags/main.yaml:51`
- Modify: `bin-openapi-manager/openapi/paths/rags/id_sources.yaml:26`

**Step 1: Update rags/main.yaml**

Line 51, change:
```yaml
description: "List of storage file IDs to ingest. Obtained from the `id` field of `POST /files` response."
```
to:
```yaml
description: "List of storage file IDs to ingest. Obtained from the `id` field of `POST /storage_files` response."
```

**Step 2: Update rags/id_sources.yaml**

Line 26, change:
```yaml
description: "List of storage file IDs to ingest. Obtained from the `id` field of `POST /files` response."
```
to:
```yaml
description: "List of storage file IDs to ingest. Obtained from the `id` field of `POST /storage_files` response."
```

**Step 3: No commit yet** — continue to Task 6.

---

### Task 6: Regenerate and verify bin-openapi-manager

**Step 1: Run full verification workflow**

```bash
cd bin-openapi-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All pass. The generated `gens/models/gen.go` will no longer contain `PostFilesMultipartBodyType`, `GetFilesParams`, or `PostFilesMultipartBody`.

**Step 2: No commit yet** — continue to Task 7.

---

### Task 7: Regenerate and verify bin-api-manager

**Step 1: Run full verification workflow**

```bash
cd bin-api-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All pass. The generated `gens/openapi_server/gen.go` will no longer contain `PostFiles`, `GetFiles`, `GetFilesId`, `DeleteFilesId` routes or interface methods. The deleted `files.go` handlers are no longer needed.

**Step 2: No commit yet** — continue to Task 8.

---

### Task 8: Update RST documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/storage_overview.rst`
- Modify: `bin-api-manager/docsdev/source/storage_struct_file.rst`
- Modify: `bin-api-manager/docsdev/source/rag_struct_rag.rst`

**Step 1: Update storage_overview.rst**

Replace all `/files` references with `/storage_files`. Specific lines:

- Line 10: `POST /files` → `POST /storage_files`, `GET /files/{id}` → `GET /storage_files/{id}`
- Line 70: `GET /files/{id}` → `GET /storage_files/{id}`
- Line 76: `https://api.voipbin.net/v1.0/files` → `https://api.voipbin.net/v1.0/storage_files`
- Line 98: `https://api.voipbin.net/v1.0/files` → `https://api.voipbin.net/v1.0/storage_files`
- Line 104: `https://api.voipbin.net/v1.0/files/<file-id>` → `https://api.voipbin.net/v1.0/storage_files/<file-id>`
- Line 110: `https://api.voipbin.net/v1.0/files/<file-id>` → `https://api.voipbin.net/v1.0/storage_files/<file-id>`
- Line 248: `GET /files?type=recording` → `GET /storage_files?type=recording`
- Line 252: `GET /files/{id}` → `GET /storage_files/{id}`
- Line 256: `DELETE /files/{id}` → `DELETE /storage_files/{id}`
- Line 273: `GET /files?sort=size&order=desc` → `GET /storage_files?sort=size&order=desc`
- Line 276: `GET /files?sort=tm_create&order=asc` → `GET /storage_files?sort=tm_create&order=asc`
- Line 280: `DELETE /files/{id}` → `DELETE /storage_files/{id}`

**Step 2: Update storage_struct_file.rst**

- Line 30: `POST /files` → `POST /storage_files`, `GET /files` → `GET /storage_files`
- Line 39: `GET /files/{id}` → `GET /storage_files/{id}`
- Line 47: `GET /files/{id}` → `GET /storage_files/{id}`

**Step 3: Update rag_struct_rag.rst**

- Line 92: `POST /files` → `POST /storage_files`

**Step 4: Rebuild HTML docs**

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```

Expected: Build completes with no errors.

**Step 5: No commit yet** — continue to Task 9.

---

### Task 9: Delete API validator test for /files

**Files:**
- Delete: `~/gitvoipbin/monorepo-monitoring/api-validator/tests/scenarios/generated/test_files_generated.py`

This file is auto-generated and tests the `/files` endpoint which no longer exists. The equivalent `/storage_files` tests in `test_storage_files_generated.py` already cover the same functionality.

```bash
rm ~/gitvoipbin/monorepo-monitoring/api-validator/tests/scenarios/generated/test_files_generated.py
```

**Note:** This file is in a separate repo (`monorepo-monitoring`). It should be committed separately or alongside depending on your deployment workflow.

---

### Task 10: Commit all changes

**Step 1: Stage changes in monorepo worktree**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Remove-duplicate-files-endpoint

git add bin-api-manager/server/storage_files.go
git add bin-api-manager/server/storage_files_test.go
git add bin-api-manager/server/files.go        # deleted
git add bin-api-manager/server/files_test.go    # deleted
git add bin-openapi-manager/openapi/paths/files/  # deleted directory
git add bin-openapi-manager/openapi/openapi.yaml
git add bin-openapi-manager/openapi/paths/rags/main.yaml
git add bin-openapi-manager/openapi/paths/rags/id_sources.yaml
git add bin-openapi-manager/gens/models/gen.go
git add bin-api-manager/gens/openapi_server/gen.go
git add bin-api-manager/go.mod bin-api-manager/go.sum
git add bin-openapi-manager/go.mod bin-openapi-manager/go.sum
git add bin-api-manager/docsdev/source/storage_overview.rst
git add bin-api-manager/docsdev/source/storage_struct_file.rst
git add bin-api-manager/docsdev/source/rag_struct_rag.rst
git add -f bin-api-manager/docsdev/build/
```

**Step 2: Commit**

```
NOJIRA-Remove-duplicate-files-endpoint

Remove duplicate /files endpoint and fix /storage_files DELETE bug.

- bin-api-manager: Remove /files server handlers (duplicate of /storage_files)
- bin-api-manager: Fix DELETE /storage_files/{id} calling wrong method (ServiceAgentFileDelete → StorageFileDelete)
- bin-api-manager: Move constMaxFileSize constant to storage_files.go
- bin-api-manager: Update RST docs to reference /storage_files instead of /files
- bin-openapi-manager: Remove /files and /files/{id} path specs
- bin-openapi-manager: Update descriptions referencing POST /files → POST /storage_files
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-Remove-duplicate-files-endpoint
```
