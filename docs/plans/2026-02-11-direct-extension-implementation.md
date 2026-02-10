# Direct Extension Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add direct extension feature to registrar-manager — when enabled on an extension, generates a hash enabling external calls via `sip:direct.<hash>@sip.voipbin.net`.

**Architecture:** New `extensiondirect` model + `extensiondirecthandler` in registrar-manager. DB table `registrar_directs` maps hash → extension. Extension API (get/list/update/delete) integrates with the new handler. New RPC endpoint for hash lookup.

**Tech Stack:** Go, squirrel (SQL builder), gomock (testing), SQLite (test DB), RabbitMQ RPC

**Design doc:** `docs/plans/2026-02-11-direct-extension-design.md`

**Worktree:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design`

---

### Task 1: Create extensiondirect model files

**Files:**
- Create: `bin-registrar-manager/models/extensiondirect/extensiondirect.go`
- Create: `bin-registrar-manager/models/extensiondirect/field.go`
- Create: `bin-registrar-manager/models/extensiondirect/filters.go`
- Create: `bin-registrar-manager/models/extensiondirect/webhook.go`
- Create: `bin-registrar-manager/models/extensiondirect/event.go`

**Step 1: Create model files**

`bin-registrar-manager/models/extensiondirect/extensiondirect.go`:
```go
package extensiondirect

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// ExtensionDirect struct
type ExtensionDirect struct {
	commonidentity.Identity

	ExtensionID uuid.UUID `json:"extension_id" db:"extension_id,uuid"`
	Hash        string    `json:"hash" db:"hash"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
```

`bin-registrar-manager/models/extensiondirect/field.go`:
```go
package extensiondirect

// Field represents a typed field name for extension direct queries
type Field string

// Field constants for extension direct
const (
	FieldID          Field = "id"
	FieldCustomerID  Field = "customer_id"
	FieldExtensionID Field = "extension_id"
	FieldHash        Field = "hash"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
```

`bin-registrar-manager/models/extensiondirect/filters.go`:
```go
package extensiondirect

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for ExtensionDirect queries
type FieldStruct struct {
	ID          uuid.UUID `filter:"id"`
	CustomerID  uuid.UUID `filter:"customer_id"`
	ExtensionID uuid.UUID `filter:"extension_id"`
	Hash        string    `filter:"hash"`
	Deleted     bool      `filter:"deleted"`
}
```

`bin-registrar-manager/models/extensiondirect/webhook.go`:
```go
package extensiondirect

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	ExtensionID uuid.UUID `json:"extension_id"`
	Hash        string    `json:"hash"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *ExtensionDirect) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		ExtensionID: h.ExtensionID,
		Hash:        h.Hash,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *ExtensionDirect) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
```

`bin-registrar-manager/models/extensiondirect/event.go`:
```go
package extensiondirect

// list of event types
const (
	EventTypeExtensionDirectCreated string = "extension_direct_created"
	EventTypeExtensionDirectUpdated string = "extension_direct_updated"
	EventTypeExtensionDirectDeleted string = "extension_direct_deleted"
)
```

**Step 2: Verify model compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-registrar-manager && go build ./models/extensiondirect/...`
Expected: No errors

**Step 3: Commit**

```bash
git add bin-registrar-manager/models/extensiondirect/
git commit -m "NOJIRA-Add-direct-extension-design

- bin-registrar-manager: Add extensiondirect model, field types, filters, webhook, events"
```

---

### Task 2: Add test SQL script and DB handler methods

**Files:**
- Create: `bin-registrar-manager/scripts/database_scripts_test/table_registrar_directs.sql`
- Create: `bin-registrar-manager/pkg/dbhandler/extension_direct.go`
- Modify: `bin-registrar-manager/pkg/dbhandler/main.go` (add methods to DBHandler interface)

**Step 1: Create test SQL script**

`bin-registrar-manager/scripts/database_scripts_test/table_registrar_directs.sql`:
```sql
create table registrar_directs (
  -- identity
  id            binary(16),
  customer_id   binary(16),

  extension_id  binary(16),
  hash          varchar(255),

  -- timestamps
  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create unique index idx_registrar_directs_extension_id on registrar_directs(extension_id);
create unique index idx_registrar_directs_hash on registrar_directs(hash);
create index idx_registrar_directs_customer_id on registrar_directs(customer_id);
```

**Step 2: Add DB handler interface methods**

Add to `bin-registrar-manager/pkg/dbhandler/main.go` interface, after the Extension section:

```go
	// ExtensionDirect
	ExtensionDirectCreate(ctx context.Context, ed *extensiondirect.ExtensionDirect) error
	ExtensionDirectDelete(ctx context.Context, id uuid.UUID) error
	ExtensionDirectGet(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error)
	ExtensionDirectGetByExtensionID(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error)
	ExtensionDirectGetByExtensionIDs(ctx context.Context, extensionIDs []uuid.UUID) ([]*extensiondirect.ExtensionDirect, error)
	ExtensionDirectGetByHash(ctx context.Context, hash string) (*extensiondirect.ExtensionDirect, error)
	ExtensionDirectUpdate(ctx context.Context, id uuid.UUID, fields map[extensiondirect.Field]any) error
```

Add import: `"monorepo/bin-registrar-manager/models/extensiondirect"`

**Step 3: Create DB handler implementation**

`bin-registrar-manager/pkg/dbhandler/extension_direct.go`:
```go
package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-registrar-manager/models/extensiondirect"
)

const (
	extensionDirectsTable = "registrar_directs"
)

// extensionDirectGetFromRow gets the extension direct from the row
func (h *handler) extensionDirectGetFromRow(row *sql.Rows) (*extensiondirect.ExtensionDirect, error) {
	res := &extensiondirect.ExtensionDirect{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. extensionDirectGetFromRow. err: %v", err)
	}

	return res, nil
}

// ExtensionDirectCreate creates new ExtensionDirect record.
func (h *handler) ExtensionDirectCreate(ctx context.Context, ed *extensiondirect.ExtensionDirect) error {
	now := h.utilHandler.TimeNow()

	ed.TMCreate = now
	ed.TMUpdate = nil
	ed.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(ed)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ExtensionDirectCreate. err: %v", err)
	}

	sb := squirrel.
		Insert(extensionDirectsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ExtensionDirectCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. ExtensionDirectCreate. err: %v", err)
	}

	return nil
}

// extensionDirectGetFromDB returns ExtensionDirect from the DB.
func (h *handler) extensionDirectGetFromDB(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	fields := commondatabasehandler.GetDBFields(&extensiondirect.ExtensionDirect{})
	query, args, err := squirrel.
		Select(fields...).
		From(extensionDirectsTable).
		Where(squirrel.Eq{string(extensiondirect.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. extensionDirectGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. extensionDirectGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. extensionDirectGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.extensionDirectGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. extensionDirectGetFromDB. id: %s, err: %v", id, err)
	}

	return res, nil
}

// ExtensionDirectGet returns extension direct.
func (h *handler) ExtensionDirectGet(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	return h.extensionDirectGetFromDB(ctx, id)
}

// ExtensionDirectGetByExtensionID returns extension direct of the given extension ID.
func (h *handler) ExtensionDirectGetByExtensionID(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	fields := commondatabasehandler.GetDBFields(&extensiondirect.ExtensionDirect{})
	sb := squirrel.
		Select(fields...).
		From(extensionDirectsTable).
		Where(squirrel.Eq{string(extensiondirect.FieldExtensionID): extensionID.Bytes()}).
		Where(squirrel.Eq{string(extensiondirect.FieldTMDelete): nil}).
		Limit(1).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. ExtensionDirectGetByExtensionID. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ExtensionDirectGetByExtensionID. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.extensionDirectGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. ExtensionDirectGetByExtensionID. err: %v", err)
	}

	return res, nil
}

// ExtensionDirectGetByExtensionIDs returns extension directs for the given extension IDs.
func (h *handler) ExtensionDirectGetByExtensionIDs(ctx context.Context, extensionIDs []uuid.UUID) ([]*extensiondirect.ExtensionDirect, error) {
	if len(extensionIDs) == 0 {
		return []*extensiondirect.ExtensionDirect{}, nil
	}

	ids := make([][]byte, len(extensionIDs))
	for i, id := range extensionIDs {
		ids[i] = id.Bytes()
	}

	fields := commondatabasehandler.GetDBFields(&extensiondirect.ExtensionDirect{})
	sb := squirrel.
		Select(fields...).
		From(extensionDirectsTable).
		Where(squirrel.Eq{string(extensiondirect.FieldExtensionID): ids}).
		Where(squirrel.Eq{string(extensiondirect.FieldTMDelete): nil}).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. ExtensionDirectGetByExtensionIDs. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ExtensionDirectGetByExtensionIDs. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*extensiondirect.ExtensionDirect{}
	for rows.Next() {
		ed, err := h.extensionDirectGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. ExtensionDirectGetByExtensionIDs. err: %v", err)
		}
		res = append(res, ed)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. ExtensionDirectGetByExtensionIDs. err: %v", err)
	}

	return res, nil
}

// ExtensionDirectGetByHash returns extension direct of the given hash.
func (h *handler) ExtensionDirectGetByHash(ctx context.Context, hash string) (*extensiondirect.ExtensionDirect, error) {
	fields := commondatabasehandler.GetDBFields(&extensiondirect.ExtensionDirect{})
	sb := squirrel.
		Select(fields...).
		From(extensionDirectsTable).
		Where(squirrel.Eq{string(extensiondirect.FieldHash): hash}).
		Where(squirrel.Eq{string(extensiondirect.FieldTMDelete): nil}).
		Limit(1).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. ExtensionDirectGetByHash. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ExtensionDirectGetByHash. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.extensionDirectGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. ExtensionDirectGetByHash. err: %v", err)
	}

	return res, nil
}

// ExtensionDirectDelete deletes given extension direct (soft delete)
func (h *handler) ExtensionDirectDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	fields := map[extensiondirect.Field]any{
		extensiondirect.FieldTMUpdate: ts,
		extensiondirect.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ExtensionDirectDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(extensionDirectsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(extensiondirect.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("ExtensionDirectDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ExtensionDirectDelete: exec failed: %w", err)
	}

	return nil
}

// ExtensionDirectUpdate updates extension direct record with given fields.
func (h *handler) ExtensionDirectUpdate(ctx context.Context, id uuid.UUID, fields map[extensiondirect.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[extensiondirect.FieldTMUpdate] = h.utilHandler.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ExtensionDirectUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(extensionDirectsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(extensiondirect.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("ExtensionDirectUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ExtensionDirectUpdate: exec failed: %w", err)
	}

	return nil
}
```

**Step 4: Regenerate mocks and verify build**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-registrar-manager
go generate ./pkg/dbhandler/...
go build ./...
```

**Step 5: Write DB handler tests**

Create `bin-registrar-manager/pkg/dbhandler/extension_direct_test.go` following the extension_test.go patterns (table-driven, gomock, SQLite in-memory).

**Step 6: Run tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-registrar-manager && go test ./pkg/dbhandler/...`
Expected: All pass

**Step 7: Commit**

```bash
git add bin-registrar-manager/scripts/ bin-registrar-manager/pkg/dbhandler/ bin-registrar-manager/models/extensiondirect/
git commit -m "NOJIRA-Add-direct-extension-design

- bin-registrar-manager: Add registrar_directs test SQL and DB handler with CRUD operations
- bin-registrar-manager: Add ExtensionDirect methods to DBHandler interface"
```

---

### Task 3: Create extensiondirecthandler

**Files:**
- Create: `bin-registrar-manager/pkg/extensiondirecthandler/main.go`
- Create: `bin-registrar-manager/pkg/extensiondirecthandler/handler.go`

**Step 1: Create handler interface**

`bin-registrar-manager/pkg/extensiondirecthandler/main.go`:
```go
package extensiondirecthandler

//go:generate mockgen -package extensiondirecthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-registrar-manager/models/extensiondirect"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

// ExtensionDirectHandler is interface for extension direct handle
type ExtensionDirectHandler interface {
	Create(ctx context.Context, customerID, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error)
	Delete(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error)
	Get(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error)
	GetByExtensionID(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error)
	GetByExtensionIDs(ctx context.Context, extensionIDs []uuid.UUID) ([]*extensiondirect.ExtensionDirect, error)
	GetByHash(ctx context.Context, hash string) (*extensiondirect.ExtensionDirect, error)
	Regenerate(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error)
}

// extensionDirectHandler structure for service handle
type extensionDirectHandler struct {
	utilHandler utilhandler.UtilHandler
	db          dbhandler.DBHandler
}

// NewExtensionDirectHandler returns new handler
func NewExtensionDirectHandler(db dbhandler.DBHandler) ExtensionDirectHandler {
	h := &extensionDirectHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
	}
	return h
}
```

**Step 2: Create handler implementation**

`bin-registrar-manager/pkg/extensiondirecthandler/handler.go`:
```go
package extensiondirecthandler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-registrar-manager/models/extensiondirect"
)

const (
	hashLength  = 6 // 6 bytes = 12 hex chars
	maxRetries  = 3
)

// generateHash generates a random 12-character hex string
func generateHash() (string, error) {
	b := make([]byte, hashLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("could not generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Create creates a new extension direct with a generated hash
func (h *extensionDirectHandler) Create(ctx context.Context, customerID, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Create",
		"customer_id":  customerID,
		"extension_id": extensionID,
	})

	// check if already exists
	existing, err := h.db.ExtensionDirectGetByExtensionID(ctx, extensionID)
	if err == nil && existing != nil {
		log.Debugf("Extension direct already exists. extension_id: %s", extensionID)
		return existing, nil
	}

	id := h.utilHandler.UUIDCreate()

	for i := 0; i < maxRetries; i++ {
		hash, err := generateHash()
		if err != nil {
			return nil, fmt.Errorf("could not generate hash: %w", err)
		}

		ed := &extensiondirect.ExtensionDirect{
			Identity: commonidentity.Identity{
				ID:         id,
				CustomerID: customerID,
			},
			ExtensionID: extensionID,
			Hash:        hash,
		}

		if err := h.db.ExtensionDirectCreate(ctx, ed); err != nil {
			log.Debugf("Could not create extension direct (attempt %d). err: %v", i+1, err)
			continue
		}

		res, err := h.db.ExtensionDirectGet(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not get created extension direct: %w", err)
		}

		return res, nil
	}

	return nil, fmt.Errorf("could not create extension direct after %d attempts", maxRetries)
}

// Delete deletes extension direct
func (h *extensionDirectHandler) Delete(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Delete",
		"id":   id,
	})

	if err := h.db.ExtensionDirectDelete(ctx, id); err != nil {
		log.Errorf("Could not delete extension direct. err: %v", err)
		return nil, err
	}

	res, err := h.db.ExtensionDirectGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted extension direct. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns extension direct by ID
func (h *extensionDirectHandler) Get(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	return h.db.ExtensionDirectGet(ctx, id)
}

// GetByExtensionID returns extension direct by extension ID
func (h *extensionDirectHandler) GetByExtensionID(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	return h.db.ExtensionDirectGetByExtensionID(ctx, extensionID)
}

// GetByExtensionIDs returns extension directs by extension IDs
func (h *extensionDirectHandler) GetByExtensionIDs(ctx context.Context, extensionIDs []uuid.UUID) ([]*extensiondirect.ExtensionDirect, error) {
	return h.db.ExtensionDirectGetByExtensionIDs(ctx, extensionIDs)
}

// GetByHash returns extension direct by hash
func (h *extensionDirectHandler) GetByHash(ctx context.Context, hash string) (*extensiondirect.ExtensionDirect, error) {
	return h.db.ExtensionDirectGetByHash(ctx, hash)
}

// Regenerate generates a new hash for an existing extension direct
func (h *extensionDirectHandler) Regenerate(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Regenerate",
		"id":   id,
	})

	for i := 0; i < maxRetries; i++ {
		hash, err := generateHash()
		if err != nil {
			return nil, fmt.Errorf("could not generate hash: %w", err)
		}

		fields := map[extensiondirect.Field]any{
			extensiondirect.FieldHash: hash,
		}

		if err := h.db.ExtensionDirectUpdate(ctx, id, fields); err != nil {
			log.Debugf("Could not update extension direct hash (attempt %d). err: %v", i+1, err)
			continue
		}

		res, err := h.db.ExtensionDirectGet(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not get updated extension direct: %w", err)
		}

		return res, nil
	}

	return nil, fmt.Errorf("could not regenerate hash after %d attempts", maxRetries)
}
```

**Step 3: Generate mocks and verify build**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-registrar-manager
go generate ./pkg/extensiondirecthandler/...
go build ./...
```

**Step 4: Commit**

```bash
git add bin-registrar-manager/pkg/extensiondirecthandler/
git commit -m "NOJIRA-Add-direct-extension-design

- bin-registrar-manager: Add extensiondirecthandler with Create, Delete, Get, Regenerate operations
- bin-registrar-manager: Add hash generation with crypto/rand and collision retry"
```

---

### Task 4: Integrate with extension model and extensionhandler

**Files:**
- Modify: `bin-registrar-manager/models/extension/extension.go` (add DirectHash field)
- Modify: `bin-registrar-manager/models/extension/webhook.go` (add DirectHash field)
- Modify: `bin-registrar-manager/pkg/extensionhandler/main.go` (add dependency)
- Modify: `bin-registrar-manager/pkg/extensionhandler/extension.go` (integrate direct in Get/List/Update/Delete)

**Step 1: Add DirectHash to extension model**

In `bin-registrar-manager/models/extension/extension.go`, add before TMCreate:
```go
	DirectHash string `json:"direct_hash" db:"-"` // populated from registrar_directs table
```

In `bin-registrar-manager/models/extension/webhook.go`, add DirectHash to WebhookMessage struct and ConvertWebhookMessage:
```go
// In WebhookMessage struct:
	DirectHash string `json:"direct_hash"`

// In ConvertWebhookMessage:
	DirectHash: h.DirectHash,
```

**Step 2: Add extensionDirectHandler dependency to extensionhandler**

In `bin-registrar-manager/pkg/extensionhandler/main.go`:

Add import: `"monorepo/bin-registrar-manager/pkg/extensiondirecthandler"`

Update struct:
```go
type extensionHandler struct {
	utilHandler            utilhandler.UtilHandler
	reqHandler             requesthandler.RequestHandler
	dbAst                  dbhandler.DBHandler
	dbBin                  dbhandler.DBHandler
	notifyHandler          notifyhandler.NotifyHandler
	extensionDirectHandler extensiondirecthandler.ExtensionDirectHandler
}
```

Update constructor:
```go
func NewExtensionHandler(r requesthandler.RequestHandler, dbAst dbhandler.DBHandler, dbBin dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler, extensionDirectHandler extensiondirecthandler.ExtensionDirectHandler) ExtensionHandler {
	h := &extensionHandler{
		utilHandler:            utilhandler.NewUtilHandler(),
		reqHandler:             r,
		dbAst:                  dbAst,
		dbBin:                  dbBin,
		notifyHandler:          notifyHandler,
		extensionDirectHandler: extensionDirectHandler,
	}
	return h
}
```

**Step 3: Integrate direct logic into extension.go**

Update `Get` method:
```go
func (h *extensionHandler) Get(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {
	res, err := h.dbBin.ExtensionGet(ctx, id)
	if err != nil {
		return nil, err
	}

	direct, err := h.extensionDirectHandler.GetByExtensionID(ctx, res.ID)
	if err == nil && direct != nil {
		res.DirectHash = direct.Hash
	}

	return res, nil
}
```

Update `List` method:
```go
func (h *extensionHandler) List(ctx context.Context, token string, limit uint64, filters map[extension.Field]any) ([]*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "List",
		"filters": filters,
	})

	res, err := h.dbBin.ExtensionList(ctx, limit, token, filters)
	if err != nil {
		log.Errorf("Could not get extensions. err: %v", err)
		return nil, err
	}

	// batch fetch direct records
	extIDs := make([]uuid.UUID, len(res))
	for i, ext := range res {
		extIDs[i] = ext.ID
	}

	directs, _ := h.extensionDirectHandler.GetByExtensionIDs(ctx, extIDs)
	directMap := make(map[uuid.UUID]string)
	for _, d := range directs {
		directMap[d.ExtensionID] = d.Hash
	}

	for _, ext := range res {
		if hash, ok := directMap[ext.ID]; ok {
			ext.DirectHash = hash
		}
	}

	return res, nil
}
```

Update `Update` method — add direct handling after the existing update logic, before the event publish. Add a new parameter to the Update interface: `direct *bool, directRegenerate *bool`:

Actually, looking at the existing pattern, the Update method takes `fields map[extension.Field]any`. The direct/regenerate flags are not extension fields — they're separate actions. The cleanest approach is to handle them in the listenhandler layer and call extensionDirectHandler directly. But per the design doc, the extensionhandler.Update integrates them.

Better approach: keep the Update signature unchanged. Add two new methods to the ExtensionHandler interface:

```go
	DirectEnable(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error)
	DirectDisable(ctx context.Context, extensionID uuid.UUID) error
	DirectRegenerate(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error)
```

These are called from the listenhandler after the normal Update, based on the request body flags.

Implement in extension.go:
```go
// DirectEnable enables direct extension
func (h *extensionHandler) DirectEnable(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	ext, err := h.dbBin.ExtensionGet(ctx, extensionID)
	if err != nil {
		return nil, fmt.Errorf("could not get extension: %w", err)
	}

	return h.extensionDirectHandler.Create(ctx, ext.CustomerID, ext.ID)
}

// DirectDisable disables direct extension
func (h *extensionHandler) DirectDisable(ctx context.Context, extensionID uuid.UUID) error {
	direct, err := h.extensionDirectHandler.GetByExtensionID(ctx, extensionID)
	if err != nil {
		return nil // already disabled, no-op
	}

	_, err = h.extensionDirectHandler.Delete(ctx, direct.ID)
	return err
}

// DirectRegenerate regenerates the direct extension hash
func (h *extensionHandler) DirectRegenerate(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	direct, err := h.extensionDirectHandler.GetByExtensionID(ctx, extensionID)
	if err != nil {
		return nil, fmt.Errorf("direct extension not enabled: %w", err)
	}

	return h.extensionDirectHandler.Regenerate(ctx, direct.ID)
}
```

Update `Delete` — add cleanup of direct record:
```go
// In Delete, after deleting sipauth and before publishing event:
	// delete extension direct if exists
	direct, errDirect := h.extensionDirectHandler.GetByExtensionID(ctx, id)
	if errDirect == nil && direct != nil {
		if _, errDelete := h.extensionDirectHandler.Delete(ctx, direct.ID); errDelete != nil {
			log.Errorf("Could not delete extension direct. err: %v", errDelete)
		}
	}
```

**Step 4: Regenerate mocks and verify build**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-registrar-manager
go generate ./pkg/extensionhandler/...
go build ./...
```

**Step 5: Run tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-registrar-manager && go test ./...`
Expected: All pass (existing tests need mock updates for new constructor param)

**Step 6: Commit**

```bash
git add bin-registrar-manager/models/extension/ bin-registrar-manager/pkg/extensionhandler/
git commit -m "NOJIRA-Add-direct-extension-design

- bin-registrar-manager: Add DirectHash field to Extension model and webhook
- bin-registrar-manager: Add DirectEnable/DirectDisable/DirectRegenerate to extensionhandler
- bin-registrar-manager: Populate DirectHash in Get and List methods
- bin-registrar-manager: Clean up direct record on extension Delete"
```

---

### Task 5: Update listenhandler and request models

**Files:**
- Modify: `bin-registrar-manager/pkg/listenhandler/models/request/extensions.go`
- Modify: `bin-registrar-manager/pkg/listenhandler/v1_extensions.go`
- Modify: `bin-registrar-manager/pkg/listenhandler/main.go` (add extension-directs route)
- Create: `bin-registrar-manager/pkg/listenhandler/v1_extension_directs.go`

**Step 1: Update request model**

In `bin-registrar-manager/pkg/listenhandler/models/request/extensions.go`, update V1DataExtensionsIDPut:
```go
type V1DataExtensionsIDPut struct {
	Name             string `json:"name"`
	Detail           string `json:"detail"`
	Password         string `json:"password"`
	Direct           *bool  `json:"direct,omitempty"`
	DirectRegenerate *bool  `json:"direct_regenerate,omitempty"`
}
```

**Step 2: Update processV1ExtensionsIDPut**

In `bin-registrar-manager/pkg/listenhandler/v1_extensions.go`, update the PUT handler to call direct methods after the normal update:

```go
// After the existing Update call and marshaling, add:
	// handle direct extension enable/disable
	if req.Direct != nil {
		if *req.Direct {
			if _, err := h.extensionHandler.DirectEnable(ctx, extensionID); err != nil {
				log.Errorf("Could not enable direct extension. err: %v", err)
				return simpleResponse(500), nil
			}
		} else {
			if err := h.extensionHandler.DirectDisable(ctx, extensionID); err != nil {
				log.Errorf("Could not disable direct extension. err: %v", err)
				return simpleResponse(500), nil
			}
		}
	}

	// handle direct regenerate
	if req.DirectRegenerate != nil && *req.DirectRegenerate {
		if _, err := h.extensionHandler.DirectRegenerate(ctx, extensionID); err != nil {
			log.Errorf("Could not regenerate direct extension hash. err: %v", err)
			return simpleResponse(500), nil
		}
	}

	// re-fetch the extension to get updated DirectHash
	tmp, err = h.extensionHandler.Get(ctx, extensionID)
```

**Step 3: Add extension-directs RPC endpoint**

Add regex to `bin-registrar-manager/pkg/listenhandler/main.go`:
```go
	// extension-directs
	regV1ExtensionDirectsGet = regexp.MustCompile(`/v1/extension-directs\?`)
```

Add case in processRequest switch:
```go
	/////////////////
	// extension-directs
	/////////////////
	case regV1ExtensionDirectsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ExtensionDirectsGet(ctx, m)
		requestType = "/v1/extension-directs"
```

Create `bin-registrar-manager/pkg/listenhandler/v1_extension_directs.go`:
```go
package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
)

// processV1ExtensionDirectsGet handles /v1/extension-directs?hash=<hash> GET request
func (h *listenHandler) processV1ExtensionDirectsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ExtensionDirectsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	hash := u.Query().Get("hash")
	if hash == "" {
		log.Debugf("Missing hash parameter")
		return simpleResponse(400), nil
	}

	direct, err := h.extensionHandler.GetDirectByHash(ctx, hash)
	if err != nil {
		log.Errorf("Could not get extension direct by hash. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(direct)
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

This requires adding `GetDirectByHash` to the ExtensionHandler interface:
```go
	GetDirectByHash(ctx context.Context, hash string) (*extensiondirect.ExtensionDirect, error)
```

Implement:
```go
func (h *extensionHandler) GetDirectByHash(ctx context.Context, hash string) (*extensiondirect.ExtensionDirect, error) {
	return h.extensionDirectHandler.GetByHash(ctx, hash)
}
```

**Step 4: Regenerate mocks and verify build**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-registrar-manager
go generate ./...
go build ./...
```

**Step 5: Run tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-registrar-manager && go test ./...`
Expected: All pass

**Step 6: Commit**

```bash
git add bin-registrar-manager/pkg/listenhandler/ bin-registrar-manager/pkg/extensionhandler/
git commit -m "NOJIRA-Add-direct-extension-design

- bin-registrar-manager: Add direct and direct_regenerate fields to extension update request
- bin-registrar-manager: Handle direct enable/disable/regenerate in PUT handler
- bin-registrar-manager: Add /v1/extension-directs?hash= RPC endpoint for hash lookup"
```

---

### Task 6: Update main.go service initialization

**Files:**
- Modify: `bin-registrar-manager/cmd/registrar-manager/main.go`

**Step 1: Update run function**

Add import: `"monorepo/bin-registrar-manager/pkg/extensiondirecthandler"`

In the `run` function, after creating dbBin and before extensionHandler:
```go
	extensionDirectHandler := extensiondirecthandler.NewExtensionDirectHandler(dbBin)
```

Update the extensionHandler creation:
```go
	extensionHandler := extensionhandler.NewExtensionHandler(reqHandler, dbAst, dbBin, notifyHandler, extensionDirectHandler)
```

**Step 2: Verify build**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-registrar-manager && go build ./...`

**Step 3: Commit**

```bash
git add bin-registrar-manager/cmd/registrar-manager/main.go
git commit -m "NOJIRA-Add-direct-extension-design

- bin-registrar-manager: Wire extensiondirecthandler into service initialization"
```

---

### Task 7: Update OpenAPI spec

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`
- Modify: `bin-openapi-manager/openapi/paths/extensions/id.yaml`

**Step 1: Add direct_hash to RegistrarManagerExtension schema**

In `bin-openapi-manager/openapi/openapi.yaml`, in the RegistrarManagerExtension schema, add after `password`:
```yaml
        direct_hash:
          type: string
          description: "Hash for direct extension access via SIP URI sip:direct.<hash>@sip.voipbin.net"
```

**Step 2: Add direct fields to extension update request**

In `bin-openapi-manager/openapi/paths/extensions/id.yaml`, in the PUT requestBody schema, add:
```yaml
            direct:
              type: boolean
              description: "Enable (true) or disable (false) direct extension access"
            direct_regenerate:
              type: boolean
              description: "Regenerate the direct extension hash (only when direct is enabled)"
```

**Step 3: Regenerate OpenAPI types**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-openapi-manager && go generate ./...
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-api-manager && go generate ./...
```

**Step 4: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Add-direct-extension-design

- bin-openapi-manager: Add direct_hash to RegistrarManagerExtension schema
- bin-openapi-manager: Add direct and direct_regenerate to extension update request
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

### Task 8: Add requesthandler RPC method in bin-common-handler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/registrar_extensions.go`
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (add to interface)

**Step 1: Add RegistrarV1ExtensionDirectGetByHash**

In `bin-common-handler/pkg/requesthandler/registrar_extensions.go`, add:
```go
// RegistrarV1ExtensionDirectGetByHash sends a request to registrar-manager
// to get the extension direct by hash.
func (r *requestHandler) RegistrarV1ExtensionDirectGetByHash(ctx context.Context, hash string) (*rmextensiondirect.ExtensionDirect, error) {
	uri := fmt.Sprintf("/v1/extension-directs?hash=%s", url.QueryEscape(hash))

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodGet, "registrar/extension-direct", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res rmextensiondirect.ExtensionDirect
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

Add import: `rmextensiondirect "monorepo/bin-registrar-manager/models/extensiondirect"`

Add to RequestHandler interface in `main.go`:
```go
	RegistrarV1ExtensionDirectGetByHash(ctx context.Context, hash string) (*rmextensiondirect.ExtensionDirect, error)
```

**Step 2: Regenerate mocks**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-common-handler
go generate ./pkg/requesthandler/...
```

**Step 3: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-Add-direct-extension-design

- bin-common-handler: Add RegistrarV1ExtensionDirectGetByHash RPC method for hash lookup"
```

---

### Task 9: Create Alembic migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<auto>_add_table_registrar_directs.py`

**Step 1: Create migration**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-dbscheme-manager/bin-manager/main
alembic -c alembic.ini revision -m "add_table_registrar_directs"
```

**Step 2: Edit the generated migration file**

Add SQL in `upgrade()`:
```python
def upgrade() -> None:
    op.execute("""
        create table registrar_directs(
            -- identity
            id            binary(16),
            customer_id   binary(16),

            extension_id  binary(16),
            hash          varchar(255),

            -- timestamps
            tm_create datetime(6),
            tm_update datetime(6),
            tm_delete datetime(6),

            primary key(id)
        );
    """)

    op.execute("create unique index idx_registrar_directs_extension_id on registrar_directs(extension_id);")
    op.execute("create unique index idx_registrar_directs_hash on registrar_directs(hash);")
    op.execute("create index idx_registrar_directs_customer_id on registrar_directs(customer_id);")
```

Add SQL in `downgrade()`:
```python
def downgrade() -> None:
    op.execute("drop table if exists registrar_directs;")
```

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/
git commit -m "NOJIRA-Add-direct-extension-design

- bin-dbscheme-manager: Add Alembic migration for registrar_directs table"
```

---

### Task 10: Full verification

**Step 1: Run full verification for registrar-manager**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-registrar-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Run verification for bin-common-handler changes**

Since bin-common-handler was modified, run verification for ALL services that depend on it:

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design
for dir in bin-*/; do
  if [ -f "$dir/go.mod" ]; then
    echo "=== $dir ===" && \
    (cd "$dir" && \
      go mod tidy && \
      go mod vendor && \
      go generate ./... && \
      go test ./... && \
      golangci-lint run -v --timeout 5m) || echo "FAILED: $dir"
  fi
done
```

**Step 3: Run verification for bin-openapi-manager and bin-api-manager**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-direct-extension-design/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Fix any issues and commit**

**Step 5: Push and create PR**

```bash
git push origin NOJIRA-Add-direct-extension-design
```
