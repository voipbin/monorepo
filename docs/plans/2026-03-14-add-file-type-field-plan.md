# Add `type` Field to Storage File — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `type` field to the File model to categorize file purpose (rag, talk, recording), rename `tmp/` upload directory to `storage/`, and enforce type validation at API endpoints.

**Architecture:** Bottom-up 7-layer change: DB migration → model → storage-manager internals → common-handler interface → api-manager → call-manager → OpenAPI spec. Each layer builds on the previous.

**Tech Stack:** Go, MySQL (Alembic migrations), OpenAPI 3.0, oapi-codegen, RabbitMQ RPC

---

### Task 1: Database Migration — Add `type` Column

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/fd2a3b4c5d6e_storage_files_add_type_column.py`

**Step 1: Create Alembic migration file**

Create the migration file manually (do NOT run `alembic revision` — it requires DB connection):

```python
"""storage_files_add_type_column

Revision ID: fd2a3b4c5d6e
Revises: fc1a2b3d4e5f
Create Date: 2026-03-14 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'fd2a3b4c5d6e'
down_revision = 'fc1a2b3d4e5f'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE storage_files
        ADD COLUMN type VARCHAR(255) NOT NULL DEFAULT '' AFTER reference_id
    """)


def downgrade():
    op.execute("""
        ALTER TABLE storage_files
        DROP COLUMN type
    """)
```

**Step 2: Commit**

```bash
git add bin-dbscheme-manager/bin-manager/main/versions/fd2a3b4c5d6e_storage_files_add_type_column.py
git commit -m "NOJIRA-Add-file-type-field

- bin-dbscheme-manager: Add migration to add type column to storage_files table"
```

---

### Task 2: Model Layer — Add Type to File Model

**Files:**
- Modify: `bin-storage-manager/models/file/main.go`
- Modify: `bin-storage-manager/models/file/field.go`
- Modify: `bin-storage-manager/models/file/webhook.go`

**Step 1: Add Type type and constants to `main.go`**

In `bin-storage-manager/models/file/main.go`:

After the `ReferenceType` constants block (line 46), add:

```go
// Type defines the file type/category
type Type string

// list of file types
const (
	TypeNone      Type = ""
	TypeRAG       Type = "rag"
	TypeTalk      Type = "talk"
	TypeRecording Type = "recording"
)
```

Add the `Type` field to the `File` struct, after `ReferenceID` (line 19):

```go
	ReferenceType ReferenceType `json:"reference_type" db:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id" db:"reference_id,uuid"`

	Type Type `json:"type" db:"type"`
```

**Step 2: Add `FieldType` to `field.go`**

In `bin-storage-manager/models/file/field.go`, after `FieldReferenceID` (line 13):

```go
	FieldType Field = "type" // type
```

**Step 3: Add `Type` to `WebhookMessage` and `ConvertWebhookMessage()` in `webhook.go`**

In `bin-storage-manager/models/file/webhook.go`:

Add to `WebhookMessage` struct after `ReferenceID`:

```go
	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	Type Type `json:"type"`
```

Add to `ConvertWebhookMessage()` after `ReferenceID`:

```go
		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		Type: h.Type,
```

**Step 4: Commit**

```bash
git add bin-storage-manager/models/file/main.go bin-storage-manager/models/file/field.go bin-storage-manager/models/file/webhook.go
git commit -m "NOJIRA-Add-file-type-field

- bin-storage-manager: Add Type field to File model, WebhookMessage, and field constants"
```

---

### Task 3: Storage Manager Internals — Pass Type Through Handlers

**Files:**
- Modify: `bin-storage-manager/pkg/listenhandler/models/request/files.go`
- Modify: `bin-storage-manager/pkg/listenhandler/v1_files.go`
- Modify: `bin-storage-manager/pkg/storagehandler/main.go`
- Modify: `bin-storage-manager/pkg/storagehandler/file.go`
- Modify: `bin-storage-manager/pkg/filehandler/main.go`
- Modify: `bin-storage-manager/pkg/filehandler/file.go`

**Step 1: Add `Type` to RPC request struct**

In `bin-storage-manager/pkg/listenhandler/models/request/files.go`, add after `ReferenceID`:

```go
	ReferenceType file.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID          `json:"reference_id"`

	Type file.Type `json:"type"`
```

**Step 2: Add `fileType` param to `FileHandler.Create()` interface**

In `bin-storage-manager/pkg/filehandler/main.go`, update the `Create` method signature:

```go
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		ownerID uuid.UUID,
		referenceType file.ReferenceType,
		referenceID uuid.UUID,
		fileType file.Type,
		name string,
		detail string,
		filename string,
		bucketName string,
		filepath string,
	) (*file.File, error)
```

**Step 3: Update `fileHandler.Create()` implementation**

In `bin-storage-manager/pkg/filehandler/file.go`:

Update the function signature to add `fileType file.Type` after `referenceID uuid.UUID`:

```go
func (h *fileHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	ownerID uuid.UUID,
	referenceType file.ReferenceType,
	referenceID uuid.UUID,
	fileType file.Type,
	name string,
	detail string,
	filename string,
	bucketName string,
	filepath string,
) (*file.File, error) {
```

Add `"file_type": fileType` to the log fields block.

Set the `Type` field when building the `file.File` struct (after `ReferenceID` at ~line 92):

```go
		ReferenceType:    referenceType,
		ReferenceID:      referenceID,
		Type:             fileType,
```

**Step 4: Add `fileType` param to `StorageHandler.FileCreate()` interface**

In `bin-storage-manager/pkg/storagehandler/main.go`, update:

```go
	FileCreate(
		ctx context.Context,
		customerID uuid.UUID,
		ownerID uuid.UUID,
		referenceType file.ReferenceType,
		referenceID uuid.UUID,
		fileType file.Type,
		name string,
		detail string,
		filename string,
		bucketName string,
		filepath string,
	) (*file.File, error)
```

**Step 5: Update `storageHandler.FileCreate()` implementation**

In `bin-storage-manager/pkg/storagehandler/file.go`, update signature and add `"file_type": fileType` to log fields, then pass through:

```go
func (h *storageHandler) FileCreate(
	ctx context.Context,
	customerID uuid.UUID,
	ownerID uuid.UUID,
	referenceType file.ReferenceType,
	referenceID uuid.UUID,
	fileType file.Type,
	name string,
	detail string,
	filename string,
	bucketName string,
	filepath string,
) (*file.File, error) {
```

Update the `h.fileHandler.Create()` call to pass `fileType`:

```go
	res, err := h.fileHandler.Create(ctx, customerID, ownerID, referenceType, referenceID, fileType, name, detail, filename, bucketName, filepath)
```

**Step 6: Update `v1FilesPost` to pass `req.Type`**

In `bin-storage-manager/pkg/listenhandler/v1_files.go`, update the `h.storageHandler.FileCreate()` call to include `req.Type`:

```go
	tmp, err := h.storageHandler.FileCreate(
		ctx,
		req.CustomerID,
		req.OwnerID,
		req.ReferenceType,
		req.ReferenceID,
		req.Type,
		req.Name,
		req.Detail,
		req.Filename,
		req.BucketName,
		req.Filepath,
	)
```

**Step 7: Run verification**

```bash
cd bin-storage-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 8: Commit**

```bash
git add bin-storage-manager/
git commit -m "NOJIRA-Add-file-type-field

- bin-storage-manager: Pass file type through listenhandler, storagehandler, and filehandler"
```

---

### Task 4: Common Handler — Update Shared Interface

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (lines 1193-1218)
- Modify: `bin-common-handler/pkg/requesthandler/storage_files.go`

**Step 1: Add `fileType` param to interface in `main.go`**

In `bin-common-handler/pkg/requesthandler/main.go`, update both signatures:

```go
	StorageV1FileCreate(
		ctx context.Context,
		customerID uuid.UUID,
		ownerID uuid.UUID,
		referenceType smfile.ReferenceType,
		referenceID uuid.UUID,
		fileType smfile.Type,
		name string,
		detail string,
		filename string,
		bucketName string,
		filepath string,
		requestTimeout int, // milliseconds
	) (*smfile.File, error)
	StorageV1FileCreateWithDelay(
		ctx context.Context,
		customerID uuid.UUID,
		ownerID uuid.UUID,
		referenceType smfile.ReferenceType,
		referenceID uuid.UUID,
		fileType smfile.Type,
		name string,
		detail string,
		filename string,
		bucketName string,
		filepath string,
		delay int, // milliseconds
	) error
```

**Step 2: Update implementations in `storage_files.go`**

Update `StorageV1FileCreate` — add `fileType smfile.Type` param after `referenceID`, set `Type: fileType` in the `V1DataFilesPost` struct:

```go
func (r *requestHandler) StorageV1FileCreate(
	ctx context.Context,
	customerID uuid.UUID,
	ownerID uuid.UUID,
	referenceType smfile.ReferenceType,
	referenceID uuid.UUID,
	fileType smfile.Type,
	name string,
	detail string,
	filename string,
	bucketName string,
	filepath string,
	requestTimeout int,
) (*smfile.File, error) {
	uri := "/v1/files"

	data := &smrequest.V1DataFilesPost{
		CustomerID:    customerID,
		OwnerID:       ownerID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Type:          fileType,
		Name:          name,
		Detail:        detail,
		Filename:      filename,
		BucketName:    bucketName,
		Filepath:      filepath,
	}
```

Do the same for `StorageV1FileCreateWithDelay`.

**Step 3: Run verification**

```bash
cd bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Note: Tests in `storage_files_test.go` will need updating to pass the new `fileType` parameter. Add `smfile.TypeNone` (or `""`) to existing test calls.

**Step 4: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-Add-file-type-field

- bin-common-handler: Add fileType param to StorageV1FileCreate and StorageV1FileCreateWithDelay"
```

---

### Task 5: API Manager — Add Type Validation and Storage Path Rename

**Files:**
- Modify: `bin-api-manager/server/storage_files.go`
- Modify: `bin-api-manager/server/service_agents_files.go`
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (interface, lines 831, 851)
- Modify: `bin-api-manager/pkg/servicehandler/storage_file.go`
- Modify: `bin-api-manager/pkg/servicehandler/serviceagent_file.go`

**Step 1: Update `ServiceHandler` interface in `main.go`**

Add `fileType smfile.Type` param to both methods:

```go
	ServiceAgentFileCreate(ctx context.Context, a *amagent.Agent, f multipart.File, fileType smfile.Type, name string, detail string, filename string) (*smfile.WebhookMessage, error)
```

```go
	StorageFileCreate(ctx context.Context, a *amagent.Agent, f multipart.File, fileType smfile.Type, name string, detail string, filename string) (*smfile.WebhookMessage, error)
```

**Step 2: Update `StorageFileCreate` in `storage_file.go`**

Update signature:

```go
func (h *serviceHandler) StorageFileCreate(ctx context.Context, a *amagent.Agent, f multipart.File, fileType smfile.Type, name string, detail string, filename string) (*smfile.WebhookMessage, error) {
```

Change `tmp/` to `storage/`:

```go
	filepath := fmt.Sprintf("storage/%s", h.utilHandler.UUIDCreate())
```

Pass `fileType` to `storageFileCreate`:

```go
	tmp, err := h.storageFileCreate(ctx, a.CustomerID, a.ID, smfile.ReferenceTypeNone, uuid.Nil, fileType, name, detail, filename, h.bucketName, filepath)
```

Update the private `storageFileCreate` helper to accept and pass `fileType`:

```go
func (h *serviceHandler) storageFileCreate(
	ctx context.Context,
	customerID uuid.UUID,
	ownerID uuid.UUID,
	referenceType smfile.ReferenceType,
	referenceID uuid.UUID,
	fileType smfile.Type,
	name string,
	detail string,
	filename string,
	bucketName string,
	filepath string,
) (*smfile.File, error) {
	res, err := h.reqHandler.StorageV1FileCreate(ctx, customerID, ownerID, referenceType, referenceID, fileType, name, detail, filename, bucketName, filepath, 60000)
```

**Step 3: Update `ServiceAgentFileCreate` in `serviceagent_file.go`**

Update signature:

```go
func (h *serviceHandler) ServiceAgentFileCreate(ctx context.Context, a *amagent.Agent, f multipart.File, fileType smfile.Type, name string, detail string, filename string) (*smfile.WebhookMessage, error) {
```

Change `tmp/` to `storage/`:

```go
	filepath := fmt.Sprintf("storage/%s", h.utilHandler.UUIDCreate())
```

Pass `fileType` to `storageFileCreate`:

```go
	tmp, err := h.storageFileCreate(ctx, a.CustomerID, a.ID, smfile.ReferenceTypeNone, uuid.Nil, fileType, name, detail, filename, h.bucketName, filepath)
```

**Step 4: Update HTTP handlers to read and validate `type` from form**

In `bin-api-manager/server/storage_files.go` `PostStorageFiles`:

After getting the file (line 38), read and validate the type field:

```go
	// read and validate type field
	fileType := c.PostForm("type")
	if fileType != "rag" {
		log.Errorf("Invalid or missing file type. type: %s", fileType)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.StorageFileCreate(c.Request.Context(), &a, f, smfile.Type(fileType), "", "", header.Filename)
```

Add import: `smfile "monorepo/bin-storage-manager/models/file"`

In `bin-api-manager/server/service_agents_files.go` `PostServiceAgentsFiles`:

After getting the file (line 38), read and validate:

```go
	// read and validate type field
	fileType := c.PostForm("type")
	if fileType != "talk" {
		log.Errorf("Invalid or missing file type. type: %s", fileType)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ServiceAgentFileCreate(c.Request.Context(), &a, f, smfile.Type(fileType), header.Filename, "Uploaded by agent", header.Filename)
```

Add import: `smfile "monorepo/bin-storage-manager/models/file"`

**Step 5: Run verification**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```bash
git add bin-api-manager/
git commit -m "NOJIRA-Add-file-type-field

- bin-api-manager: Add file type validation to PostStorageFiles and PostServiceAgentsFiles
- bin-api-manager: Rename tmp/ upload directory to storage/
- bin-api-manager: Pass fileType through servicehandler to storage-manager"
```

---

### Task 6: Call Manager — Pass TypeRecording

**Files:**
- Modify: `bin-call-manager/pkg/recordinghandler/stop.go` (line 98)

**Step 1: Update `StorageV1FileCreate` call**

In `bin-call-manager/pkg/recordinghandler/stop.go`, line 98, add `smfile.TypeRecording` after `r.ID`:

```go
		tmp, err := h.reqHandler.StorageV1FileCreate(ctx, r.CustomerID, uuid.Nil, smfile.ReferenceTypeRecording, r.ID, smfile.TypeRecording, "recording file", "", filename, defaultBucketName, filepath, 30000)
```

**Step 2: Run verification**

```bash
cd bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Note: Tests in `stop_test.go` that mock `StorageV1FileCreate` (lines 71, 153) will need the new `smfile.TypeRecording` argument added.

**Step 3: Commit**

```bash
git add bin-call-manager/
git commit -m "NOJIRA-Add-file-type-field

- bin-call-manager: Pass TypeRecording when creating storage files for recordings"
```

---

### Task 7: OpenAPI Specification

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (components/schemas section)
- Modify: `bin-openapi-manager/openapi/paths/storage_files/main.yaml`
- Modify: `bin-openapi-manager/openapi/paths/service_agents/files.yaml`

**Step 1: Add `StorageManagerFileType` enum**

In `bin-openapi-manager/openapi/openapi.yaml`, after the `StorageManagerFileReferenceType` block (~line 5455), add:

```yaml
    StorageManagerFileType:
      type: string
      description: "The type/category of the file. Indicates the purpose of the uploaded file."
      enum:
        - ""
        - "rag"
        - "talk"
        - "recording"
      x-enum-varnames:
        - StorageManagerFileTypeNone
        - StorageManagerFileTypeRAG
        - StorageManagerFileTypeTalk
        - StorageManagerFileTypeRecording
      example: "rag"
```

**Step 2: Add `type` property to `StorageManagerFile` schema**

In `openapi.yaml`, inside the `StorageManagerFile` properties, after `reference_id` (~line 5490), add:

```yaml
        type:
          $ref: '#/components/schemas/StorageManagerFileType'
          description: "The type/category of the file."
          example: "rag"
```

**Step 3: Add `type` field to POST storage_files form**

In `bin-openapi-manager/openapi/paths/storage_files/main.yaml`, update the post requestBody:

```yaml
post:
  summary: Upload a file and create a call with it
  description: Creates a temporary file and initiates a call with the temporary file.
  tags:
    - Storage
  requestBody:
    content:
      multipart/form-data:
        schema:
          type: object
          properties:
            file:
              type: string
              format: binary
            type:
              type: string
              description: "The type/category of the file. Must be 'rag' for this endpoint."
              enum:
                - "rag"
              example: "rag"
          required:
            - file
            - type
  responses:
    '200':
      description: The created call details.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/StorageManagerFile'
```

**Step 4: Add `type` field to POST service_agents/files form**

In `bin-openapi-manager/openapi/paths/service_agents/files.yaml`, update the post requestBody:

```yaml
post:
  summary: Upload a file
  description: Uploads a file and returns the details of the uploaded file.
  tags:
    - Service Agent
  requestBody:
    required: true
    content:
      multipart/form-data:
        schema:
          type: object
          properties:
            file:
              type: string
              format: binary
              description: The file to upload.
            type:
              type: string
              description: "The type/category of the file. Must be 'talk' for this endpoint."
              enum:
                - "talk"
              example: "talk"
          required:
            - file
            - type
  responses:
    '200':
      description: The details of the uploaded file.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/StorageManagerFile'
```

**Step 5: Regenerate models in bin-openapi-manager**

```bash
cd bin-openapi-manager
go mod tidy && go mod vendor && go generate ./...
```

**Step 6: Regenerate server code in bin-api-manager**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 7: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Add-file-type-field

- bin-openapi-manager: Add StorageManagerFileType enum and type property to StorageManagerFile
- bin-openapi-manager: Add required type field to POST storage_files and service_agents/files
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

### Task 8: Final Verification

**Step 1: Run verification across all changed services**

```bash
# bin-storage-manager
cd bin-storage-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-common-handler
cd ../bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-api-manager
cd ../bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-call-manager
cd ../bin-call-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-openapi-manager
cd ../bin-openapi-manager && go mod tidy && go mod vendor && go generate ./...
```

**Step 2: Verify no other callers were missed**

```bash
grep -r "StorageV1FileCreate\|storageFileCreate\|ServiceAgentFileCreate\|StorageFileCreate" --include="*.go" . | grep -v "_test.go" | grep -v "mock_" | grep -v "vendor/"
```

**Step 3: Check for conflicts with main**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```

---

## Caller Impact Summary

| Caller | File | Change |
|--------|------|--------|
| `bin-api-manager` `storageFileCreate` | `pkg/servicehandler/storage_file.go:140-158` | Add `fileType` param |
| `bin-api-manager` `StorageFileCreate` | `pkg/servicehandler/storage_file.go:95` | Add `fileType` param |
| `bin-api-manager` `ServiceAgentFileCreate` | `pkg/servicehandler/serviceagent_file.go:19` | Add `fileType` param |
| `bin-api-manager` `PostStorageFiles` | `server/storage_files.go:13` | Read+validate `type` form field |
| `bin-api-manager` `PostServiceAgentsFiles` | `server/service_agents_files.go:13` | Read+validate `type` form field |
| `bin-call-manager` `storeRecordingFiles` | `pkg/recordinghandler/stop.go:98` | Add `smfile.TypeRecording` |
| `bin-common-handler` `StorageV1FileCreate` tests | `pkg/requesthandler/storage_files_test.go` | Add `smfile.TypeNone` to test calls |
| `bin-call-manager` `storeRecordingFiles` tests | `pkg/recordinghandler/stop_test.go:71,153` | Add `smfile.TypeRecording` to mock expectations |
