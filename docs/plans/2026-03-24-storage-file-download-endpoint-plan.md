# Storage File Download Endpoint Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `GET /storage_files/{id}/file` and `GET /service_agents/files/{id}/file` endpoints that return HTTP 307 redirects to signed GCS download URLs, with automatic URL refresh when expired.

**Architecture:** OpenAPI-first workflow. New endpoints in bin-api-manager redirect to signed GCS URLs. A new RPC in bin-storage-manager regenerates expired URLs. The creation code is fixed from 10-year to 7-day expiration.

**Tech Stack:** Go, OpenAPI (oapi-codegen), RabbitMQ RPC, GCS signed URLs, Gin HTTP framework

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/`

**Design doc:** `docs/plans/2026-03-24-storage-file-download-endpoint-design.md`

---

### Task 1: Fix file creation expiration (10 years → 7 days)

**Files:**
- Modify: `bin-storage-manager/pkg/filehandler/file.go:72`

**Step 1: Change expiration duration**

In `bin-storage-manager/pkg/filehandler/file.go`, replace line 72:

```go
// Before
expireDuration := 3650 * 24 * time.Hour // valid for 10 years

// After
expireDuration := 7 * 24 * time.Hour // valid for 7 days
```

**Step 2: Run verification for bin-storage-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/bin-storage-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All tests pass, no lint errors.

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint
git add bin-storage-manager/pkg/filehandler/file.go
git commit -m "NOJIRA-Add-storage-file-download-endpoint

Fix GCS signed URL expiration from 10 years to 7 days to match actual GCS limit.

- bin-storage-manager: Change file creation download URL expiration to 7 days"
```

---

### Task 2: Add download URI refresh to storage-manager (filehandler + storagehandler)

**Files:**
- Modify: `bin-storage-manager/pkg/filehandler/main.go` (interface)
- Create: `bin-storage-manager/pkg/filehandler/download.go` (implementation)
- Modify: `bin-storage-manager/pkg/storagehandler/main.go` (interface)
- Create: `bin-storage-manager/pkg/storagehandler/download.go` (implementation)

**Step 1: Add `DownloadURIRefresh` to FileHandler interface**

In `bin-storage-manager/pkg/filehandler/main.go`, add to the `FileHandler` interface after `DownloadURIGet`:

```go
DownloadURIRefresh(ctx context.Context, id uuid.UUID) (string, error)
```

**Step 2: Create filehandler download implementation**

Create `bin-storage-manager/pkg/filehandler/download.go`:

```go
package filehandler

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-storage-manager/models/file"
)

// DownloadURIRefresh generates a fresh signed download URL for the given file,
// updates the database with the new URL and expiration, and returns the URL.
func (h *fileHandler) DownloadURIRefresh(ctx context.Context, id uuid.UUID) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "DownloadURIRefresh",
		"id":   id,
	})

	// get file info
	f, err := h.db.FileGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get file info. err: %v", err)
		return "", err
	}
	log.WithField("file", f).Debugf("Retrieved file info. file_id: %s", f.ID)

	// generate fresh download URI
	expireDuration := 7 * 24 * time.Hour
	tmExpire := time.Now().UTC().Add(expireDuration)
	tmDownloadExpire := h.utilHandler.TimeNowAdd(expireDuration)

	downloadURI, err := h.bucketfileGenerateDownloadURI(f.BucketName, f.Filepath, tmExpire)
	if err != nil {
		log.Errorf("Could not generate download URI. err: %v", err)
		return "", err
	}
	log.Debugf("Generated fresh download URI. file_id: %s", f.ID)

	// update database with fresh URL and expiration
	fields := map[file.Field]any{
		file.FieldURIDownload:      downloadURI,
		file.FieldTMDownloadExpire: tmDownloadExpire,
	}

	if err := h.db.FileUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update file download URI. err: %v", err)
		return "", err
	}

	return downloadURI, nil
}
```

**Step 3: Add `FileDownloadURIRefresh` to StorageHandler interface**

In `bin-storage-manager/pkg/storagehandler/main.go`, add to the `StorageHandler` interface after `FileDelete`:

```go
FileDownloadURIRefresh(ctx context.Context, id uuid.UUID) (string, error)
```

**Step 4: Create storagehandler download implementation**

Create `bin-storage-manager/pkg/storagehandler/download.go`:

```go
package storagehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// FileDownloadURIRefresh generates a fresh signed download URL for the given file.
func (h *storageHandler) FileDownloadURIRefresh(ctx context.Context, id uuid.UUID) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "FileDownloadURIRefresh",
		"id":   id,
	})

	res, err := h.fileHandler.DownloadURIRefresh(ctx, id)
	if err != nil {
		log.Errorf("Could not refresh download URI. err: %v", err)
		return "", err
	}

	return res, nil
}
```

**Step 5: Run verification for bin-storage-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/bin-storage-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All tests pass. Mocks are regenerated for updated interfaces.

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint
git add bin-storage-manager/pkg/filehandler/main.go bin-storage-manager/pkg/filehandler/download.go \
  bin-storage-manager/pkg/filehandler/mock_main.go \
  bin-storage-manager/pkg/storagehandler/main.go bin-storage-manager/pkg/storagehandler/download.go \
  bin-storage-manager/pkg/storagehandler/mock_main.go
git commit -m "NOJIRA-Add-storage-file-download-endpoint

Add download URI refresh capability to storage-manager.

- bin-storage-manager: Add DownloadURIRefresh to filehandler for generating fresh signed URLs
- bin-storage-manager: Add FileDownloadURIRefresh to storagehandler as business logic layer"
```

---

### Task 3: Add RPC endpoint for download URI refresh in storage-manager

**Files:**
- Modify: `bin-storage-manager/pkg/listenhandler/main.go` (add regex + route)
- Create: `bin-storage-manager/pkg/listenhandler/v1_files_download.go` (RPC handler)

**Step 1: Add regex and route for the new RPC endpoint**

In `bin-storage-manager/pkg/listenhandler/main.go`, add regex after line 49 (`regV1FilesID`):

```go
regV1FilesIDDownloadURIRefresh = regexp.MustCompile("/v1/files/" + regUUID + "/download_uri_refresh$")
```

Add route case in `processRequest` switch, after the `regV1FilesID DELETE` case (around line 174):

```go
case regV1FilesIDDownloadURIRefresh.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
	requestType = "/files/<file-id>/download_uri_refresh"
	response, err = h.v1FilesIDDownloadURIRefresh(ctx, m)
```

**Step 2: Create the RPC handler**

Create `bin-storage-manager/pkg/listenhandler/v1_files_download.go`:

```go
package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// v1FilesIDDownloadURIRefresh handles /v1/files/<id>/download_uri_refresh POST request
// generates a fresh signed download URL for the given file.
func (h *listenHandler) v1FilesIDDownloadURIRefresh(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1FilesIDDownloadURIRefresh",
		"request": m,
	})

	// "/v1/files/<uuid>/download_uri_refresh"
	tmpVals := strings.Split(m.URI, "/")
	fileID := uuid.FromStringOrNil(tmpVals[3])

	downloadURI, err := h.storageHandler.FileDownloadURIRefresh(ctx, fileID)
	if err != nil {
		log.Errorf("Could not refresh download URI. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(downloadURI)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
```

**Step 3: Run verification for bin-storage-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/bin-storage-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint
git add bin-storage-manager/pkg/listenhandler/main.go bin-storage-manager/pkg/listenhandler/v1_files_download.go
git commit -m "NOJIRA-Add-storage-file-download-endpoint

Add RPC endpoint for download URI refresh in storage-manager.

- bin-storage-manager: Add /v1/files/{id}/download_uri_refresh POST RPC endpoint
- bin-storage-manager: Add listenhandler routing for download URI refresh"
```

---

### Task 4: Add request handler method in bin-common-handler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go:1230` (interface)
- Modify: `bin-common-handler/pkg/requesthandler/storage_files.go` (implementation)

**Step 1: Add `StorageV1FileDownloadURIRefresh` to RequestHandler interface**

In `bin-common-handler/pkg/requesthandler/main.go`, add after line 1230 (`StorageV1FileDelete`):

```go
StorageV1FileDownloadURIRefresh(ctx context.Context, fileID uuid.UUID) (string, error)
```

**Step 2: Add implementation**

In `bin-common-handler/pkg/requesthandler/storage_files.go`, add after `StorageV1FileDelete`:

```go
// StorageV1FileDownloadURIRefresh sends a request to storage-manager
// to refresh the download URI for the given file.
// it returns fresh download URI if it succeeds.
func (r *requestHandler) StorageV1FileDownloadURIRefresh(ctx context.Context, fileID uuid.UUID) (string, error) {
	uri := fmt.Sprintf("/v1/files/%s/download_uri_refresh", fileID)

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodPost, "storage/files/<file-id>/download_uri_refresh", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return "", err
	}

	var res string
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return "", errParse
	}

	return res, nil
}
```

**Step 3: Run verification for bin-common-handler**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: Mocks regenerated, all tests pass.

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint
git add bin-common-handler/pkg/requesthandler/main.go bin-common-handler/pkg/requesthandler/storage_files.go \
  bin-common-handler/pkg/requesthandler/mock_main.go
git commit -m "NOJIRA-Add-storage-file-download-endpoint

Add StorageV1FileDownloadURIRefresh RPC method to common handler.

- bin-common-handler: Add StorageV1FileDownloadURIRefresh to requesthandler interface
- bin-common-handler: Add RPC implementation for file download URI refresh"
```

---

### Task 5: Add OpenAPI path definitions

**Files:**
- Create: `bin-openapi-manager/openapi/paths/storage_files/id_file.yaml`
- Create: `bin-openapi-manager/openapi/paths/service_agents/files_id_file.yaml`
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (register paths)

**Step 1: Create storage_files download path**

Create `bin-openapi-manager/openapi/paths/storage_files/id_file.yaml`:

```yaml
get:
  summary: Download the storage file
  description: Retrieves the specified storage file and redirects to the download URI.
  tags:
    - Storage
  parameters:
    - name: id
      in: path
      required: true
      description: The storage file's ID.
      schema:
        type: string
  responses:
    '307':
      description: The storage file download URL.
      content:
        application/json:
          schema:
            type: string
    '400':
      description: Bad request. Could not find agent information or storage file.
```

**Step 2: Create service_agents file download path**

Create `bin-openapi-manager/openapi/paths/service_agents/files_id_file.yaml`:

```yaml
get:
  summary: Download the service agent file
  description: Retrieves the specified service agent file and redirects to the download URI.
  tags:
    - Service Agent
  parameters:
    - name: id
      in: path
      required: true
      description: The file's ID.
      schema:
        type: string
  responses:
    '307':
      description: The service agent file download URL.
      content:
        application/json:
          schema:
            type: string
    '400':
      description: Bad request. Could not find agent information or file.
```

**Step 3: Register paths in openapi.yaml**

In `bin-openapi-manager/openapi/openapi.yaml`, add the path references.

After the `/storage_files/{id}` entry, add:

```yaml
  /storage_files/{id}/file:
    $ref: './paths/storage_files/id_file.yaml'
```

After the `/service_agents/files/{id}` entry, add:

```yaml
  /service_agents/files/{id}/file:
    $ref: './paths/service_agents/files_id_file.yaml'
```

**Step 4: Run verification for bin-openapi-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint
git add bin-openapi-manager/openapi/paths/storage_files/id_file.yaml \
  bin-openapi-manager/openapi/paths/service_agents/files_id_file.yaml \
  bin-openapi-manager/openapi/openapi.yaml \
  bin-openapi-manager/gens/
git commit -m "NOJIRA-Add-storage-file-download-endpoint

Add OpenAPI path definitions for storage file download endpoints.

- bin-openapi-manager: Add GET /storage_files/{id}/file path (307 redirect)
- bin-openapi-manager: Add GET /service_agents/files/{id}/file path (307 redirect)"
```

---

### Task 6: Add service handler methods in bin-api-manager

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (interface, ~line 841 and ~861)
- Modify: `bin-api-manager/pkg/servicehandler/storage_file.go` (implementation)
- Modify: `bin-api-manager/pkg/servicehandler/serviceagent_file.go` (implementation)

**Step 1: Add interface methods**

In `bin-api-manager/pkg/servicehandler/main.go`:

After `ServiceAgentFileList` (~line 841), add:

```go
ServiceAgentFileDownloadRedirect(ctx context.Context, a *amagent.Agent, id uuid.UUID) (string, error)
```

After `StorageFileDelete` (~line 861), add:

```go
StorageFileDownloadRedirect(ctx context.Context, a *amagent.Agent, id uuid.UUID) (string, error)
```

**Step 2: Implement StorageFileDownloadRedirect**

In `bin-api-manager/pkg/servicehandler/storage_file.go`, add after `StorageFileGet` (line ~50):

```go
// StorageFileDownloadRedirect returns a working download URL for the given file.
// If the stored URL has expired, it refreshes via storage-manager RPC.
func (h *serviceHandler) StorageFileDownloadRedirect(ctx context.Context, a *amagent.Agent, id uuid.UUID) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "StorageFileDownloadRedirect",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"file_id":     id,
	})
	log.Debug("Getting file download URL.")

	// get file
	f, err := h.storageFileGet(ctx, id)
	if err != nil {
		log.Infof("Could not get file info. err: %v", err)
		return "", fmt.Errorf("could not find file info. err: %v", err)
	}
	log.WithField("file", f).Debugf("Retrieved file info. file_id: %s", f.ID)

	if !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return "", fmt.Errorf("user has no permission")
	}

	// check if download URL is still valid
	if f.TMDownloadExpire != nil && f.TMDownloadExpire.After(time.Now()) {
		log.Debugf("Download URL is still valid. file_id: %s", f.ID)
		return f.URIDownload, nil
	}

	// URL expired, refresh it
	log.Debugf("Download URL expired. Refreshing. file_id: %s", f.ID)
	downloadURI, err := h.reqHandler.StorageV1FileDownloadURIRefresh(ctx, id)
	if err != nil {
		log.Errorf("Could not refresh download URI. err: %v", err)
		return "", fmt.Errorf("could not refresh download URI. err: %v", err)
	}

	return downloadURI, nil
}
```

**Note:** Add `"time"` to the import block in `storage_file.go`.

**Step 3: Implement ServiceAgentFileDownloadRedirect**

In `bin-api-manager/pkg/servicehandler/serviceagent_file.go`, add after `ServiceAgentFileDelete` (line ~167):

```go
// ServiceAgentFileDownloadRedirect returns a working download URL for the given service agent file.
// If the stored URL has expired, it refreshes via storage-manager RPC.
func (h *serviceHandler) ServiceAgentFileDownloadRedirect(ctx context.Context, a *amagent.Agent, id uuid.UUID) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentFileDownloadRedirect",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"file_id":     id,
	})
	log.Debug("Getting service agent file download URL.")

	// get file
	f, err := h.storageFileGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get file info from the storage-manager. err: %v", err)
		return "", fmt.Errorf("could not find file info. err: %v", err)
	}
	log.WithField("file", f).Debugf("Retrieved file info. file_id: %s", f.ID)

	// Check permission - file must belong to the same customer
	if f.CustomerID != a.CustomerID {
		log.Info("The user has no permission.")
		return "", fmt.Errorf("user has no permission")
	}

	// check if download URL is still valid
	if f.TMDownloadExpire != nil && f.TMDownloadExpire.After(time.Now()) {
		log.Debugf("Download URL is still valid. file_id: %s", f.ID)
		return f.URIDownload, nil
	}

	// URL expired, refresh it
	log.Debugf("Download URL expired. Refreshing. file_id: %s", f.ID)
	downloadURI, err := h.reqHandler.StorageV1FileDownloadURIRefresh(ctx, id)
	if err != nil {
		log.Errorf("Could not refresh download URI. err: %v", err)
		return "", fmt.Errorf("could not refresh download URI. err: %v", err)
	}

	return downloadURI, nil
}
```

**Note:** Add `"time"` to the import block in `serviceagent_file.go`.

**Step 4: Run verification for bin-api-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint
git add bin-api-manager/pkg/servicehandler/main.go \
  bin-api-manager/pkg/servicehandler/storage_file.go \
  bin-api-manager/pkg/servicehandler/serviceagent_file.go \
  bin-api-manager/pkg/servicehandler/mock_main.go \
  bin-api-manager/go.mod bin-api-manager/go.sum
git commit -m "NOJIRA-Add-storage-file-download-endpoint

Add service handler methods for file download redirect with expiry check.

- bin-api-manager: Add StorageFileDownloadRedirect with Admin|Manager permission
- bin-api-manager: Add ServiceAgentFileDownloadRedirect with customer ownership check
- bin-api-manager: Both check TMDownloadExpire and refresh via RPC when expired"
```

---

### Task 7: Add server handlers in bin-api-manager

**Files:**
- Modify: `bin-api-manager/server/storage_files.go` (add handler)
- Modify: `bin-api-manager/server/service_agents_files.go` (add handler)

**Step 1: Regenerate OpenAPI server code**

The OpenAPI paths were added in Task 5. Now regenerate the server code in api-manager:

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/bin-api-manager
go mod tidy && go mod vendor && go generate ./...
```

This updates `gens/openapi_server/gen.go` with the new `GetStorageFilesIdFile` and `GetServiceAgentsFilesIdFile` interface methods.

**Step 2: Add `GetStorageFilesIdFile` server handler**

In `bin-api-manager/server/storage_files.go`, add after the last handler. Add `"net/http"` to imports if not present:

```go
func (h *server) GetStorageFilesIdFile(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetStorageFilesIdFile",
		"request_address": c.ClientIP(),
		"file_id":         id,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	downloadURI, err := h.serviceHandler.StorageFileDownloadRedirect(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get storage file download URL. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, downloadURI)
}
```

**Step 3: Add `GetServiceAgentsFilesIdFile` server handler**

In `bin-api-manager/server/service_agents_files.go`, add after the last handler. Add `"net/http"` to imports if not present:

```go
func (h *server) GetServiceAgentsFilesIdFile(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsFilesIdFile",
		"request_address": c.ClientIP(),
		"file_id":         id,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	downloadURI, err := h.serviceHandler.ServiceAgentFileDownloadRedirect(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get service agent file download URL. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, downloadURI)
}
```

**Step 4: Run verification for bin-api-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint
git add bin-api-manager/server/storage_files.go \
  bin-api-manager/server/service_agents_files.go \
  bin-api-manager/gens/openapi_server/gen.go \
  bin-api-manager/go.mod bin-api-manager/go.sum
git commit -m "NOJIRA-Add-storage-file-download-endpoint

Add server handlers for file download endpoints with 307 redirect.

- bin-api-manager: Add GetStorageFilesIdFile handler (GET /storage_files/{id}/file)
- bin-api-manager: Add GetServiceAgentsFilesIdFile handler (GET /service_agents/files/{id}/file)
- bin-api-manager: Both return HTTP 307 redirect to signed GCS download URL"
```

---

### Task 8: Add unit tests

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/storage_file_test.go`
- Modify: `bin-api-manager/server/storage_files_test.go`

**Step 1: Add service handler test for StorageFileDownloadRedirect**

In `bin-api-manager/pkg/servicehandler/storage_file_test.go`, add a test function following the existing `Test_StorageFileGet` pattern. Test cases:

1. **Valid URL** — `TMDownloadExpire` in the future → returns stored `URIDownload`
2. **Expired URL** — `TMDownloadExpire` in the past → calls `StorageV1FileDownloadURIRefresh`, returns fresh URL
3. **Permission denied** — agent lacks `Admin|Manager` permission → returns error

**Step 2: Add server handler test for GetStorageFilesIdFile**

In `bin-api-manager/server/storage_files_test.go`, add a test function following the existing `Test_GetStorageFilesId` pattern. Verify:

1. Returns HTTP 307 status
2. `Location` header contains the download URL
3. Calls `StorageFileDownloadRedirect` on the mock

**Step 3: Run verification for bin-api-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint
git add bin-api-manager/pkg/servicehandler/storage_file_test.go \
  bin-api-manager/server/storage_files_test.go
git commit -m "NOJIRA-Add-storage-file-download-endpoint

Add unit tests for storage file download endpoints.

- bin-api-manager: Add service handler tests for download redirect with expiry logic
- bin-api-manager: Add server handler tests for 307 redirect response"
```

---

### Task 9: Final verification across all services

**Step 1: Verify bin-storage-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/bin-storage-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Verify bin-common-handler**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Verify bin-openapi-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Verify bin-api-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-storage-file-download-endpoint/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

---

### Post-Implementation: Manual SQL (Not part of code changes)

Run manually by the user after deployment:

```sql
UPDATE storage_files
SET tm_download_expire = tm_create
WHERE tm_delete IS NULL;
```
