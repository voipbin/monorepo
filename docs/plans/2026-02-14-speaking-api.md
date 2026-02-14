# Speaking API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a `/v1/speakings` API that lets customers control real-time streaming TTS on active calls — send text, flush queued messages, and stop sessions.

**Architecture:** New `speaking` resource backed by MySQL for persistence, orchestrating the existing `streaminghandler` for live ElevenLabs WebSocket + AudioSocket connections. API requests flow through bin-api-manager → RabbitMQ RPC → bin-tts-manager. Pod-targeted routing ensures say/flush/stop reach the pod with the live session.

**Tech Stack:** Go, MySQL (squirrel query builder), RabbitMQ RPC, ElevenLabs WebSocket, Asterisk AudioSocket, OpenAPI 3.0, oapi-codegen, Gin HTTP framework.

**Design document:** `docs/plans/2026-02-14-speaking-api-design.md`

---

## Task 1: Database Migration

Create the Alembic migration for the `tts_manager_speaking` table.

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<revision>_add_tts_manager_speaking.py`

**Step 1: Create the migration file**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api/bin-dbscheme-manager
alembic -c bin-manager/alembic.ini.sample revision -m "add tts_manager_speaking"
```

Expected: Creates a new file in `bin-manager/main/versions/` with a revision ID.

**Step 2: Write the migration SQL**

Edit the generated file to add:

```python
def upgrade():
    op.execute("""
        CREATE TABLE tts_manager_speaking (
            id binary(16) NOT NULL,
            customer_id binary(16) NOT NULL,
            reference_type varchar(32) NOT NULL DEFAULT '',
            reference_id binary(16) NOT NULL DEFAULT '',
            language varchar(16) NOT NULL DEFAULT '',
            provider varchar(32) NOT NULL DEFAULT '',
            voice_id varchar(255) NOT NULL DEFAULT '',
            direction varchar(8) NOT NULL DEFAULT '',
            status varchar(16) NOT NULL DEFAULT '',
            pod_id varchar(64) NOT NULL DEFAULT '',
            tm_create datetime(6) DEFAULT NULL,
            tm_update datetime(6) DEFAULT NULL,
            tm_delete datetime(6) DEFAULT NULL,
            PRIMARY KEY (id),
            INDEX idx_tts_manager_speaking_customer_id (customer_id),
            INDEX idx_tts_manager_speaking_reference (reference_type, reference_id),
            INDEX idx_tts_manager_speaking_status (status)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8;
    """)


def downgrade():
    op.execute("""DROP TABLE IF EXISTS tts_manager_speaking;""")
```

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/bin-manager/main/versions/*_add_tts_manager_speaking.py
git commit -m "NOJIRA-Add-speaking-api

- bin-dbscheme-manager: Add Alembic migration for tts_manager_speaking table"
```

**IMPORTANT:** Do NOT run `alembic upgrade`. The migration will be applied by a human.

---

## Task 2: Speaking Model (`bin-tts-manager`)

Create the Go model struct and Field type for the speaking resource.

**Files:**
- Create: `bin-tts-manager/models/speaking/speaking.go`
- Create: `bin-tts-manager/models/speaking/field.go`
- Create: `bin-tts-manager/models/speaking/status.go`

**Step 1: Create the status type**

Create `bin-tts-manager/models/speaking/status.go`:

```go
package speaking

// Status represents the status of a speaking session
type Status string

const (
	StatusInitiating Status = "initiating"
	StatusActive     Status = "active"
	StatusStopped    Status = "stopped"
)
```

**Step 2: Create the model struct**

Create `bin-tts-manager/models/speaking/speaking.go`:

```go
package speaking

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
)

// Speaking represents an active streaming TTS session attached to a call or conference.
type Speaking struct {
	commonidentity.Identity

	ReferenceType streaming.ReferenceType `json:"reference_type" db:"reference_type"`
	ReferenceID   uuid.UUID               `json:"reference_id"   db:"reference_id,uuid"`
	Language      string                  `json:"language"        db:"language"`
	Provider      string                  `json:"provider"       db:"provider"`
	VoiceID       string                  `json:"voice_id"       db:"voice_id"`
	Direction     streaming.Direction     `json:"direction"       db:"direction"`
	Status        Status                  `json:"status"          db:"status"`
	PodID         string                  `json:"pod_id"          db:"pod_id"`

	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
}
```

**Step 3: Create the field type**

Create `bin-tts-manager/models/speaking/field.go`:

```go
package speaking

// Field represents a database field name for Speaking
type Field string

const (
	FieldID            Field = "id"
	FieldCustomerID    Field = "customer_id"
	FieldReferenceType Field = "reference_type"
	FieldReferenceID   Field = "reference_id"
	FieldLanguage      Field = "language"
	FieldProvider      Field = "provider"
	FieldVoiceID       Field = "voice_id"
	FieldDirection     Field = "direction"
	FieldStatus        Field = "status"
	FieldPodID         Field = "pod_id"
	FieldTMCreate      Field = "tm_create"
	FieldTMUpdate      Field = "tm_update"
	FieldTMDelete      Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
```

**Step 4: Verify it compiles**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api/bin-tts-manager
go build ./models/speaking/...
```
Expected: No errors.

**Step 5: Commit**

```bash
git add bin-tts-manager/models/speaking/
git commit -m "NOJIRA-Add-speaking-api

- bin-tts-manager: Add speaking model with Status type and Field type"
```

---

## Task 3: Add MySQL to tts-manager Config and Main

The tts-manager currently has no database connection. Add `DatabaseDSN` to config and `*sql.DB` initialization to main.go.

**Files:**
- Modify: `bin-tts-manager/internal/config/main.go`
- Modify: `bin-tts-manager/cmd/tts-manager/main.go`

**Step 1: Add DatabaseDSN to config**

In `bin-tts-manager/internal/config/main.go`, add `DatabaseDSN` to the Config struct:

```go
type Config struct {
	DatabaseDSN            string
	RabbitMQAddress        string
	PrometheusEndpoint     string
	PrometheusListenAddress string
	AWSAccessKey           string
	AWSSecretKey           string
	ElevenlabsAPIKey       string
}
```

And add to the `init()` function (or wherever env vars are read):

```go
c.DatabaseDSN = os.Getenv("DATABASE_DSN")
```

**Step 2: Add database initialization to main.go**

In `bin-tts-manager/cmd/tts-manager/main.go`, add:

```go
import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)
```

In `main()`, after config initialization:

```go
db, err := sql.Open("mysql", config.Get().DatabaseDSN)
if err != nil {
	logrus.Fatalf("Could not open database connection. err: %v", err)
}
defer db.Close()

if err := db.Ping(); err != nil {
	logrus.Fatalf("Could not ping database. err: %v", err)
}
```

Pass `db` to `NewDBHandler` (will be updated in next task).

**Step 3: Add mysql driver to go.mod if needed**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api/bin-tts-manager
go mod tidy
```

**Step 4: Verify it compiles**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api/bin-tts-manager
go build ./cmd/...
```

Note: This may have compilation errors until the DBHandler is updated in the next task. If so, make the DBHandler changes first or do them together.

**Step 5: Commit**

```bash
git add bin-tts-manager/internal/config/main.go bin-tts-manager/cmd/tts-manager/main.go
git commit -m "NOJIRA-Add-speaking-api

- bin-tts-manager: Add DatabaseDSN to config and database initialization to main"
```

---

## Task 4: DB Handler — Add MySQL and Speaking CRUD

Add `*sql.DB` to dbHandler and implement speaking CRUD operations.

**Files:**
- Modify: `bin-tts-manager/pkg/dbhandler/main.go`
- Create: `bin-tts-manager/pkg/dbhandler/speaking.go`
- Create: `bin-tts-manager/pkg/dbhandler/speaking_test.go`

**Step 1: Update DBHandler interface and struct**

In `bin-tts-manager/pkg/dbhandler/main.go`:

```go
package dbhandler

import (
	"context"
	"database/sql"

	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/cachehandler"

	"github.com/gofrs/uuid"
)

type DBHandler interface {
	// existing streaming methods (cache-only)
	StreamingCreate(ctx context.Context, s *streaming.Streaming) error
	StreamingGet(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error)
	StreamingUpdate(ctx context.Context, s *streaming.Streaming) error

	// new speaking methods (MySQL)
	SpeakingCreate(ctx context.Context, s *speaking.Speaking) error
	SpeakingGet(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error)
	SpeakingGets(ctx context.Context, token string, size uint64, filters map[speaking.Field]any) ([]*speaking.Speaking, error)
	SpeakingUpdate(ctx context.Context, id uuid.UUID, fields map[speaking.Field]any) error
	SpeakingDelete(ctx context.Context, id uuid.UUID) error
}

type dbHandler struct {
	db    *sql.DB
	util  utilhandler.UtilHandler
	cache cachehandler.CacheHandler
}

func NewDBHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	return &dbHandler{
		db:    db,
		util:  utilhandler.NewUtilHandler(),
		cache: cache,
	}
}
```

**Step 2: Implement speaking CRUD**

Create `bin-tts-manager/pkg/dbhandler/speaking.go`:

```go
package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-tts-manager/models/speaking"
)

const speakingTable = "tts_manager_speaking"

func (h *dbHandler) speakingGetFromRow(row *sql.Rows) (*speaking.Speaking, error) {
	res := &speaking.Speaking{}
	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. speakingGetFromRow. err: %v", err)
	}
	return res, nil
}

func (h *dbHandler) SpeakingCreate(ctx context.Context, s *speaking.Speaking) error {
	s.TMCreate = h.util.TimeGetCurTime()

	fields, err := commondatabasehandler.PrepareFields(s)
	if err != nil {
		return fmt.Errorf("could not prepare fields. SpeakingCreate. err: %v", err)
	}

	query, args, err := squirrel.
		Insert(speakingTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. SpeakingCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. SpeakingCreate. err: %v", err)
	}

	return nil
}

func (h *dbHandler) SpeakingGet(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error) {
	cols := commondatabasehandler.GetDBFields(&speaking.Speaking{})

	query, args, err := squirrel.
		Select(cols...).
		From(speakingTable).
		Where(squirrel.Eq{string(speaking.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. SpeakingGet. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. SpeakingGet. err: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("row error. SpeakingGet. err: %v", err)
		}
		return nil, fmt.Errorf("not found")
	}

	return h.speakingGetFromRow(rows)
}

func (h *dbHandler) SpeakingGets(ctx context.Context, token string, size uint64, filters map[speaking.Field]any) ([]*speaking.Speaking, error) {
	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(&speaking.Speaking{})

	sb := squirrel.
		Select(cols...).
		From(speakingTable).
		Where(squirrel.Lt{string(speaking.FieldTMCreate): token}).
		OrderBy(string(speaking.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. SpeakingGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. SpeakingGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. SpeakingGets. err: %v", err)
	}
	defer rows.Close()

	var res []*speaking.Speaking
	for rows.Next() {
		s, err := h.speakingGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan row. SpeakingGets. err: %v", err)
		}
		res = append(res, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error. SpeakingGets. err: %v", err)
	}

	return res, nil
}

func (h *dbHandler) SpeakingUpdate(ctx context.Context, id uuid.UUID, fields map[speaking.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[speaking.FieldTMUpdate] = h.util.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("could not prepare fields. SpeakingUpdate. err: %v", err)
	}

	query, args, err := squirrel.
		Update(speakingTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{"id": id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. SpeakingUpdate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. SpeakingUpdate. err: %v", err)
	}

	return nil
}

func (h *dbHandler) SpeakingDelete(ctx context.Context, id uuid.UUID) error {
	now := h.util.TimeGetCurTime()
	fields := map[speaking.Field]any{
		speaking.FieldTMDelete: now,
		speaking.FieldTMUpdate: now,
	}

	return h.SpeakingUpdate(ctx, id, fields)
}
```

**Step 3: Update all callers of NewDBHandler**

Since we changed `NewDBHandler` signature to accept `*sql.DB`, update `cmd/tts-manager/main.go`:

```go
dbHandler := dbhandler.NewDBHandler(db, cacheHandler)
```

**Step 4: Write tests for speaking CRUD**

Create `bin-tts-manager/pkg/dbhandler/speaking_test.go` with table-driven tests. Use `sqlmock` for database mocking.

**Step 5: Verify it compiles and tests pass**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api/bin-tts-manager
go mod tidy && go mod vendor && go build ./...
go test ./pkg/dbhandler/...
```

**Step 6: Commit**

```bash
git add bin-tts-manager/pkg/dbhandler/ bin-tts-manager/cmd/tts-manager/main.go
git commit -m "NOJIRA-Add-speaking-api

- bin-tts-manager: Add MySQL to dbHandler and implement speaking CRUD operations"
```

---

## Task 5: Add SayFlush to Streamer Interface and ElevenLabs

Add `SayFlush()` to the `streamer` interface and implement it in `elevenlabsHandler`.

**Files:**
- Modify: `bin-tts-manager/pkg/streaminghandler/main.go` (streamer interface)
- Modify: `bin-tts-manager/pkg/streaminghandler/elevenlabs.go` (SayFlush implementation)

**Step 1: Add SayFlush to streamer interface**

In `bin-tts-manager/pkg/streaminghandler/main.go`, add to the `streamer` interface:

```go
type streamer interface {
	Init(ctx context.Context, st *streaming.Streaming) (any, error)
	Run(vendorConfig any) error
	SayStop(vendorConfig any) error
	SayAdd(vendorConfig any, text string) error
	SayFinish(vendorConfig any) error
	SayFlush(vendorConfig any) error // new
}
```

**Step 2: Add SayFlush to StreamingHandler interface**

In `bin-tts-manager/pkg/streaminghandler/main.go`, add to the `StreamingHandler` interface:

```go
SayFlush(ctx context.Context, id uuid.UUID) error
```

**Step 3: Implement SayFlush in elevenlabsHandler**

In `bin-tts-manager/pkg/streaminghandler/elevenlabs.go`, add:

```go
func (h *elevenlabsHandler) SayFlush(vendorConfig any) error {
	cf, ok := vendorConfig.(*ElevenlabsConfig)
	if !ok || cf == nil {
		return fmt.Errorf("the vendorConfig is not a *ElevenlabsConfig or is nil")
	}

	cf.muConnWebsock.Lock()
	defer cf.muConnWebsock.Unlock()

	if cf.ConnWebsock == nil {
		return fmt.Errorf("the ConnWebsock is nil")
	}

	msg := ElevenlabsMessage{Text: "", Flush: true}
	return cf.ConnWebsock.WriteJSON(msg)
}
```

**Step 4: Implement SayFlush in streamingHandler**

Create or add to `bin-tts-manager/pkg/streaminghandler/say.go`:

```go
func (h *streamingHandler) SayFlush(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "SayFlush",
		"streaming_id": id,
	})

	st, err := h.Get(id)
	if err != nil {
		log.Infof("Could not get streaming. err: %v", err)
		return err
	}

	st.VendorLock.Lock()
	defer st.VendorLock.Unlock()

	switch st.VendorName {
	case streaming.VendorNameElevenlabs:
		if errFlush := h.elevenlabsHandler.SayFlush(st.VendorConfig); errFlush != nil {
			log.Errorf("Could not flush the say streaming. err: %v", errFlush)
			return errFlush
		}

	default:
		log.Errorf("Unsupported vendor. vendor_name: %s", st.VendorName)
		return fmt.Errorf("unsupported vendor: %s", st.VendorName)
	}

	return nil
}
```

**Step 5: Add provider and voice_id to Streaming model**

In `bin-tts-manager/models/streaming/streaming.go`, add these fields:

```go
Provider string `json:"provider,omitempty"` // e.g. "elevenlabs"
VoiceID  string `json:"voice_id,omitempty"` // provider-specific voice ID
```

Keep `Gender` for backwards compatibility with the internal AI talk flow.

**Step 6: Update ElevenLabs voice selection to check VoiceID first**

In `bin-tts-manager/pkg/streaminghandler/elevenlabs.go`, in the voice selection code (the `Init` or voice ID resolution function), add a check:

```go
// If VoiceID is explicitly set, use it directly
if st.VoiceID != "" {
	voiceID = st.VoiceID
} else {
	// Fall back to gender + language lookup
	voiceID = getVoiceID(...)
}
```

**Step 7: Verify it compiles and tests pass**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api/bin-tts-manager
go mod tidy && go mod vendor && go build ./...
go test ./pkg/streaminghandler/...
```

**Step 8: Commit**

```bash
git add bin-tts-manager/pkg/streaminghandler/ bin-tts-manager/models/streaming/
git commit -m "NOJIRA-Add-speaking-api

- bin-tts-manager: Add SayFlush to streamer interface and ElevenLabs handler
- bin-tts-manager: Add provider and voice_id fields to streaming model"
```

---

## Task 6: Speaking Handler (`bin-tts-manager`)

Create the speaking handler that orchestrates DB records and streaming sessions.

**Files:**
- Create: `bin-tts-manager/pkg/speakinghandler/main.go`
- Create: `bin-tts-manager/pkg/speakinghandler/speaking.go`
- Create: `bin-tts-manager/pkg/speakinghandler/speaking_test.go`

**Step 1: Create the handler interface**

Create `bin-tts-manager/pkg/speakinghandler/main.go`:

```go
package speakinghandler

import (
	"context"

	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/dbhandler"
	"monorepo/bin-tts-manager/pkg/streaminghandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// SpeakingHandler handles speaking session lifecycle
type SpeakingHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, referenceType streaming.ReferenceType, referenceID uuid.UUID, language, provider, voiceID string, direction streaming.Direction) (*speaking.Speaking, error)
	Get(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error)
	Gets(ctx context.Context, token string, size uint64, filters map[speaking.Field]any) ([]*speaking.Speaking, error)
	Say(ctx context.Context, id uuid.UUID, text string) (*speaking.Speaking, error)
	Flush(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error)
	Stop(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error)
	Delete(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error)
}

type speakingHandler struct {
	db               dbhandler.DBHandler
	streamingHandler streaminghandler.StreamingHandler
	podID            string
}

// NewSpeakingHandler creates a new SpeakingHandler
func NewSpeakingHandler(
	db dbhandler.DBHandler,
	streamingHandler streaminghandler.StreamingHandler,
	podID string,
) SpeakingHandler {
	return &speakingHandler{
		db:               db,
		streamingHandler: streamingHandler,
		podID:            podID,
	}
}
```

**Step 2: Implement the speaking operations**

Create `bin-tts-manager/pkg/speakinghandler/speaking.go`:

```go
package speakinghandler

import (
	"context"
	"fmt"

	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Create creates a new speaking session.
// 1. Check for existing active sessions on the same reference.
// 2. Create DB record with status "initiating".
// 3. Start a streaming session (AudioSocket + ElevenLabs).
// 4. Update DB status to "active".
func (h *speakingHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType streaming.ReferenceType,
	referenceID uuid.UUID,
	language, provider, voiceID string,
	direction streaming.Direction,
) (*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"customer_id":    customerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	// Check for existing active session on same reference
	filters := map[speaking.Field]any{
		speaking.FieldReferenceType: string(referenceType),
		speaking.FieldReferenceID:   referenceID,
		speaking.FieldDeleted:       false,
	}
	existing, err := h.db.SpeakingGets(ctx, "", 1, filters)
	if err != nil {
		log.Errorf("Could not check existing sessions. err: %v", err)
		return nil, fmt.Errorf("could not check existing sessions: %v", err)
	}
	for _, s := range existing {
		if s.Status == speaking.StatusActive || s.Status == speaking.StatusInitiating {
			return nil, fmt.Errorf("an active speaking session already exists for this reference. speaking_id: %s", s.ID)
		}
	}

	// Default provider
	if provider == "" {
		provider = "elevenlabs"
	}

	id := uuid.Must(uuid.NewV4())

	// Create DB record
	spk := &speaking.Speaking{
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Language:      language,
		Provider:      provider,
		VoiceID:       voiceID,
		Direction:     direction,
		Status:        speaking.StatusInitiating,
		PodID:         h.podID,
	}
	spk.ID = id
	spk.CustomerID = customerID

	if errCreate := h.db.SpeakingCreate(ctx, spk); errCreate != nil {
		log.Errorf("Could not create speaking record. err: %v", errCreate)
		return nil, fmt.Errorf("could not create speaking record: %v", errCreate)
	}
	log.WithField("speaking", spk).Debugf("Created speaking record. speaking_id: %s", id)

	// Start streaming session (AudioSocket + ElevenLabs)
	// Use speaking ID as streaming ID for 1:1 mapping
	_, errStart := h.streamingHandler.Start(
		ctx,
		customerID,
		uuid.Nil,         // no activeflow for external API
		referenceType,
		referenceID,
		language,
		streaming.GenderNeutral, // unused for speaking API, voice_id takes priority
		direction,
	)
	if errStart != nil {
		log.Errorf("Could not start streaming session. err: %v", errStart)
		// Mark speaking as stopped since streaming failed
		_ = h.db.SpeakingUpdate(ctx, id, map[speaking.Field]any{
			speaking.FieldStatus: speaking.StatusStopped,
		})
		return nil, fmt.Errorf("could not start streaming session: %v", errStart)
	}

	// Update status to active
	if errUpdate := h.db.SpeakingUpdate(ctx, id, map[speaking.Field]any{
		speaking.FieldStatus: speaking.StatusActive,
	}); errUpdate != nil {
		log.Errorf("Could not update speaking status. err: %v", errUpdate)
	}

	// Read back from DB for consistent response
	res, errGet := h.db.SpeakingGet(ctx, id)
	if errGet != nil {
		log.Errorf("Could not get speaking record. err: %v", errGet)
		return spk, nil
	}

	return res, nil
}

// Get returns a speaking session by ID.
func (h *speakingHandler) Get(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Get",
		"speaking_id": id,
	})

	res, err := h.db.SpeakingGet(ctx, id)
	if err != nil {
		log.Infof("Could not get speaking. err: %v", err)
		return nil, err
	}
	log.WithField("speaking", res).Debugf("Retrieved speaking. speaking_id: %s", id)

	return res, nil
}

// Gets returns a list of speaking sessions filtered by the given criteria.
func (h *speakingHandler) Gets(ctx context.Context, token string, size uint64, filters map[speaking.Field]any) ([]*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Gets",
	})

	res, err := h.db.SpeakingGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get speakings. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Say adds text to the speech queue of an active session.
func (h *speakingHandler) Say(ctx context.Context, id uuid.UUID, text string) (*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Say",
		"speaking_id": id,
	})

	// Get speaking from DB to verify it exists and is active
	spk, err := h.db.SpeakingGet(ctx, id)
	if err != nil {
		log.Infof("Could not get speaking. err: %v", err)
		return nil, fmt.Errorf("speaking not found: %v", err)
	}
	log.WithField("speaking", spk).Debugf("Retrieved speaking. speaking_id: %s", id)

	if spk.Status != speaking.StatusActive {
		return nil, fmt.Errorf("session is no longer active. status: %s", spk.Status)
	}

	// Send text via streaming handler
	if errSay := h.streamingHandler.SayAdd(ctx, id, uuid.Nil, text); errSay != nil {
		log.Errorf("Could not add text. err: %v", errSay)

		// Check if session is gone (pod death or connection lost)
		return nil, fmt.Errorf("could not add text: %v", errSay)
	}

	return spk, nil
}

// Flush clears queued messages and stops current playback. Session stays alive.
func (h *speakingHandler) Flush(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Flush",
		"speaking_id": id,
	})

	spk, err := h.db.SpeakingGet(ctx, id)
	if err != nil {
		log.Infof("Could not get speaking. err: %v", err)
		return nil, fmt.Errorf("speaking not found: %v", err)
	}
	log.WithField("speaking", spk).Debugf("Retrieved speaking. speaking_id: %s", id)

	if spk.Status != speaking.StatusActive {
		return nil, fmt.Errorf("session is no longer active. status: %s", spk.Status)
	}

	if errFlush := h.streamingHandler.SayFlush(ctx, id); errFlush != nil {
		log.Errorf("Could not flush. err: %v", errFlush)
		return nil, fmt.Errorf("could not flush: %v", errFlush)
	}

	return spk, nil
}

// Stop terminates the speaking session entirely.
func (h *speakingHandler) Stop(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Stop",
		"speaking_id": id,
	})

	spk, err := h.db.SpeakingGet(ctx, id)
	if err != nil {
		log.Infof("Could not get speaking. err: %v", err)
		return nil, fmt.Errorf("speaking not found: %v", err)
	}
	log.WithField("speaking", spk).Debugf("Retrieved speaking. speaking_id: %s", id)

	if spk.Status == speaking.StatusStopped {
		return spk, nil // already stopped, idempotent
	}

	// Stop the streaming session
	if _, errStop := h.streamingHandler.Stop(ctx, id); errStop != nil {
		log.Errorf("Could not stop streaming. err: %v", errStop)
		// Continue to update DB even if streaming stop fails
	}

	// Update DB status
	if errUpdate := h.db.SpeakingUpdate(ctx, id, map[speaking.Field]any{
		speaking.FieldStatus: speaking.StatusStopped,
	}); errUpdate != nil {
		log.Errorf("Could not update speaking status. err: %v", errUpdate)
		return nil, fmt.Errorf("could not update speaking status: %v", errUpdate)
	}

	// Re-read for consistent response
	res, errGet := h.db.SpeakingGet(ctx, id)
	if errGet != nil {
		log.Errorf("Could not get speaking after stop. err: %v", errGet)
		return spk, nil
	}

	return res, nil
}

// Delete soft-deletes a speaking record.
func (h *speakingHandler) Delete(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Delete",
		"speaking_id": id,
	})

	spk, err := h.db.SpeakingGet(ctx, id)
	if err != nil {
		log.Infof("Could not get speaking. err: %v", err)
		return nil, fmt.Errorf("speaking not found: %v", err)
	}

	// If still active, stop first
	if spk.Status == speaking.StatusActive || spk.Status == speaking.StatusInitiating {
		if _, errStop := h.Stop(ctx, id); errStop != nil {
			log.Errorf("Could not stop speaking before delete. err: %v", errStop)
		}
	}

	if errDelete := h.db.SpeakingDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete speaking. err: %v", errDelete)
		return nil, fmt.Errorf("could not delete speaking: %v", errDelete)
	}

	res, errGet := h.db.SpeakingGet(ctx, id)
	if errGet != nil {
		log.Errorf("Could not get speaking after delete. err: %v", errGet)
		return spk, nil
	}

	return res, nil
}
```

**Step 3: Add mock generation**

Add to `bin-tts-manager/pkg/speakinghandler/main.go`:

```go
//go:generate mockgen -package speakinghandler -destination mock_main.go -source main.go
```

**Step 4: Generate mocks and verify**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api/bin-tts-manager
go generate ./pkg/speakinghandler/...
go build ./...
go test ./pkg/speakinghandler/...
```

**Step 5: Commit**

```bash
git add bin-tts-manager/pkg/speakinghandler/
git commit -m "NOJIRA-Add-speaking-api

- bin-tts-manager: Add speakinghandler with Create, Say, Flush, Stop, Get, Gets, Delete"
```

---

## Task 7: Listen Handler — Add Speaking RPC Routes

Add RPC routes for speaking operations and clean up unused streaming routes.

**Files:**
- Modify: `bin-tts-manager/pkg/listenhandler/main.go`
- Create: `bin-tts-manager/pkg/listenhandler/v1_speakings.go`
- Create: `bin-tts-manager/pkg/listenhandler/models/request/speakings.go`

**Step 1: Add request models**

Create `bin-tts-manager/pkg/listenhandler/models/request/speakings.go`:

```go
package request

import (
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
)

// V1DataSpeakingsPost is the request for POST /v1/speakings
type V1DataSpeakingsPost struct {
	CustomerID    uuid.UUID               `json:"customer_id,omitempty"`
	ReferenceType streaming.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID               `json:"reference_id,omitempty"`
	Language      string                  `json:"language,omitempty"`
	Provider      string                  `json:"provider,omitempty"`
	VoiceID       string                  `json:"voice_id,omitempty"`
	Direction     streaming.Direction     `json:"direction,omitempty"`
}

// V1DataSpeakingsIDSayPost is the request for POST /v1/speakings/{id}/say
type V1DataSpeakingsIDSayPost struct {
	Text string `json:"text,omitempty"`
}

// V1DataSpeakingsGetFilters is the filter struct for GET /v1/speakings
type V1DataSpeakingsGetFilters struct {
	CustomerID    uuid.UUID `json:"customer_id,omitempty"`
	ReferenceType string    `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID `json:"reference_id,omitempty"`
	Status        string    `json:"status,omitempty"`
}
```

**Step 2: Implement speaking RPC handlers**

Create `bin-tts-manager/pkg/listenhandler/v1_speakings.go`:

```go
package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *listenHandler) v1SpeakingsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsPost",
	})

	var req request.V1DataSpeakingsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return nil, err
	}
	log.WithField("request", req).Debugf("Processing v1SpeakingsPost.")

	tmp, err := h.speakingHandler.Create(ctx, req.CustomerID, req.ReferenceType, req.ReferenceID, req.Language, req.Provider, req.VoiceID, req.Direction)
	if err != nil {
		log.Errorf("Could not create speaking. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1SpeakingsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsGet",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	pageSize := uint64(100)
	pageToken := q.Get(PageToken)

	filters := map[speaking.Field]any{
		speaking.FieldDeleted: false,
	}

	if v := q.Get("customer_id"); v != "" {
		filters[speaking.FieldCustomerID] = uuid.FromStringOrNil(v)
	}
	if v := q.Get("reference_type"); v != "" {
		filters[speaking.FieldReferenceType] = v
	}
	if v := q.Get("reference_id"); v != "" {
		filters[speaking.FieldReferenceID] = uuid.FromStringOrNil(v)
	}
	if v := q.Get("status"); v != "" {
		filters[speaking.FieldStatus] = v
	}

	tmp, err := h.speakingHandler.Gets(ctx, pageToken, pageSize, filters)
	if err != nil {
		log.Errorf("Could not get speakings. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1SpeakingsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsIDGet",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	speakingID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.speakingHandler.Get(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not get speaking. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1SpeakingsIDSayPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsIDSayPost",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	speakingID := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataSpeakingsIDSayPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return nil, err
	}
	log.WithField("request", req).Debugf("Processing v1SpeakingsIDSayPost. speaking_id: %s", speakingID)

	tmp, err := h.speakingHandler.Say(ctx, speakingID, req.Text)
	if err != nil {
		log.Errorf("Could not say. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1SpeakingsIDFlushPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsIDFlushPost",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	speakingID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.speakingHandler.Flush(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not flush. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1SpeakingsIDStopPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsIDStopPost",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	speakingID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.speakingHandler.Stop(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not stop. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1SpeakingsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsIDDelete",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	speakingID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.speakingHandler.Delete(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not delete speaking. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
```

**Step 3: Update listenHandler to include speakingHandler**

In `bin-tts-manager/pkg/listenhandler/main.go`:

1. Add `speakingHandler` to the struct:

```go
type listenHandler struct {
	sockHandler      sockhandler.SockHandler
	ttsHandler       ttshandler.TTSHandler
	streamingHandler streaminghandler.StreamingHandler
	speakingHandler  speakinghandler.SpeakingHandler
}
```

2. Add regex patterns for speaking routes:

```go
// speakings
resV1Speakings          = regexp.MustCompile("/v1/speakings$")
resV1SpeakingsID        = regexp.MustCompile("/v1/speakings/" + regUUID + "$")
resV1SpeakingsIDSay     = regexp.MustCompile("/v1/speakings/" + regUUID + "/say$")
resV1SpeakingsIDFlush   = regexp.MustCompile("/v1/speakings/" + regUUID + "/flush$")
resV1SpeakingsIDStop    = regexp.MustCompile("/v1/speakings/" + regUUID + "/stop$")
```

3. Add cases to `processRequest` switch:

```go
// /speakings POST (create)
case resV1Speakings.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
	requestType = "/speakings"
	response, err = h.v1SpeakingsPost(ctx, m)

// /speakings GET (list)
case resV1Speakings.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
	requestType = "/speakings"
	response, err = h.v1SpeakingsGet(ctx, m)

// /speakings/<id> GET
case resV1SpeakingsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
	requestType = "/speakings/<speaking-id>"
	response, err = h.v1SpeakingsIDGet(ctx, m)

// /speakings/<id> DELETE
case resV1SpeakingsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
	requestType = "/speakings/<speaking-id>"
	response, err = h.v1SpeakingsIDDelete(ctx, m)

// /speakings/<id>/say POST
case resV1SpeakingsIDSay.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
	requestType = "/speakings/<speaking-id>/say"
	response, err = h.v1SpeakingsIDSayPost(ctx, m)

// /speakings/<id>/flush POST
case resV1SpeakingsIDFlush.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
	requestType = "/speakings/<speaking-id>/flush"
	response, err = h.v1SpeakingsIDFlushPost(ctx, m)

// /speakings/<id>/stop POST
case resV1SpeakingsIDStop.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
	requestType = "/speakings/<speaking-id>/stop"
	response, err = h.v1SpeakingsIDStopPost(ctx, m)
```

4. Update `NewListenHandler` to accept `speakingHandler`:

```go
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	ttsHandler ttshandler.TTSHandler,
	streamingHandler streaminghandler.StreamingHandler,
	speakingHandler speakinghandler.SpeakingHandler,
) ListenHandler {
	h := &listenHandler{
		sockHandler:      sockHandler,
		ttsHandler:       ttsHandler,
		streamingHandler: streamingHandler,
		speakingHandler:  speakingHandler,
	}
	return h
}
```

**Step 4: Remove old unused streaming routes**

Delete the old streaming route regexes, switch cases, and handler functions from `main.go` and `v1_streamings.go`. Also delete `models/request/streamings.go`.

Note: Keep the file `v1_streamings.go` if the internal streaming flow still needs it (for AI talk service). Check if `v1StreamingsPost`, etc., are still used. Based on prior analysis, they are NOT called by any service — safe to delete.

**Step 5: Update main.go to wire speakingHandler into listenHandler**

In `bin-tts-manager/cmd/tts-manager/main.go`:

```go
speakingHandler := speakinghandler.NewSpeakingHandler(dbHandler, streamingHandler, podID)

listenHandler := listenhandler.NewListenHandler(sockHandler, ttsHandler, streamingHandler, speakingHandler)
```

**Step 6: Verify it compiles**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api/bin-tts-manager
go mod tidy && go mod vendor && go generate ./... && go build ./...
go test ./...
```

**Step 7: Commit**

```bash
git add bin-tts-manager/pkg/listenhandler/ bin-tts-manager/cmd/tts-manager/main.go
git commit -m "NOJIRA-Add-speaking-api

- bin-tts-manager: Add speaking RPC routes to listenHandler
- bin-tts-manager: Remove unused streaming RPC routes
- bin-tts-manager: Wire speakingHandler into main"
```

---

## Task 8: RPC Client Methods (`bin-common-handler`)

Create RPC client methods for speaking operations and clean up unused streaming ones.

**Files:**
- Create: `bin-common-handler/pkg/requesthandler/tts_speakings.go`
- Create: `bin-common-handler/pkg/requesthandler/tts_speakings_test.go`
- Delete: `bin-common-handler/pkg/requesthandler/tts_streamings.go`
- Delete: `bin-common-handler/pkg/requesthandler/tts_streamings_test.go`
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (RequestHandler interface)

**Step 1: Create speaking RPC methods**

Create `bin-common-handler/pkg/requesthandler/tts_speakings.go`:

```go
package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// TTSV1SpeakingCreate creates a speaking session.
func (r *requestHandler) TTSV1SpeakingCreate(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType tmstreaming.ReferenceType,
	referenceID uuid.UUID,
	language string,
	provider string,
	voiceID string,
	direction tmstreaming.Direction,
) (*tmspeaking.Speaking, error) {
	uri := "/v1/speakings"

	m, err := json.Marshal(request.V1DataSpeakingsPost{
		CustomerID:    customerID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Language:      language,
		Provider:      provider,
		VoiceID:       voiceID,
		Direction:     direction,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTTS(ctx, uri, sock.RequestMethodPost, "tts/speakings", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TTSV1SpeakingGet gets a speaking session by ID.
func (r *requestHandler) TTSV1SpeakingGet(ctx context.Context, speakingID uuid.UUID) (*tmspeaking.Speaking, error) {
	uri := fmt.Sprintf("/v1/speakings/%s", speakingID)

	tmp, err := r.sendRequestTTS(ctx, uri, sock.RequestMethodGet, "tts/speakings/<speaking-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TTSV1SpeakingGets lists speaking sessions.
func (r *requestHandler) TTSV1SpeakingGets(ctx context.Context, pageToken string, pageSize uint64, filters map[tmspeaking.Field]any) ([]*tmspeaking.Speaking, error) {
	uri := fmt.Sprintf("/v1/speakings?page_token=%s&page_size=%d", pageToken, pageSize)

	// Append filters as query params
	if v, ok := filters[tmspeaking.FieldCustomerID]; ok {
		uri += fmt.Sprintf("&customer_id=%s", v)
	}
	if v, ok := filters[tmspeaking.FieldReferenceType]; ok {
		uri += fmt.Sprintf("&reference_type=%s", v)
	}
	if v, ok := filters[tmspeaking.FieldReferenceID]; ok {
		uri += fmt.Sprintf("&reference_id=%s", v)
	}
	if v, ok := filters[tmspeaking.FieldStatus]; ok {
		uri += fmt.Sprintf("&status=%s", v)
	}

	tmp, err := r.sendRequestTTS(ctx, uri, sock.RequestMethodGet, "tts/speakings", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []*tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// TTSV1SpeakingSay sends text to a speaking session. Pod-targeted.
func (r *requestHandler) TTSV1SpeakingSay(ctx context.Context, podID string, speakingID uuid.UUID, text string) (*tmspeaking.Speaking, error) {
	uri := fmt.Sprintf("/v1/speakings/%s/say", speakingID)

	m, err := json.Marshal(request.V1DataSpeakingsIDSayPost{
		Text: text,
	})
	if err != nil {
		return nil, err
	}

	queueName := fmt.Sprintf("bin-manager.tts-manager.request.%s", podID)

	tmp, err := r.sendRequest(ctx, commonoutline.QueueName(queueName), uri, sock.RequestMethodPost, "tts/speakings/<speaking-id>/say", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TTSV1SpeakingFlush flushes a speaking session. Pod-targeted.
func (r *requestHandler) TTSV1SpeakingFlush(ctx context.Context, podID string, speakingID uuid.UUID) (*tmspeaking.Speaking, error) {
	uri := fmt.Sprintf("/v1/speakings/%s/flush", speakingID)

	queueName := fmt.Sprintf("bin-manager.tts-manager.request.%s", podID)

	tmp, err := r.sendRequest(ctx, commonoutline.QueueName(queueName), uri, sock.RequestMethodPost, "tts/speakings/<speaking-id>/flush", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TTSV1SpeakingStop stops a speaking session. Pod-targeted.
func (r *requestHandler) TTSV1SpeakingStop(ctx context.Context, podID string, speakingID uuid.UUID) (*tmspeaking.Speaking, error) {
	uri := fmt.Sprintf("/v1/speakings/%s/stop", speakingID)

	queueName := fmt.Sprintf("bin-manager.tts-manager.request.%s", podID)

	tmp, err := r.sendRequest(ctx, commonoutline.QueueName(queueName), uri, sock.RequestMethodPost, "tts/speakings/<speaking-id>/stop", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TTSV1SpeakingDelete soft-deletes a speaking session.
func (r *requestHandler) TTSV1SpeakingDelete(ctx context.Context, speakingID uuid.UUID) (*tmspeaking.Speaking, error) {
	uri := fmt.Sprintf("/v1/speakings/%s", speakingID)

	tmp, err := r.sendRequestTTS(ctx, uri, sock.RequestMethodDelete, "tts/speakings/<speaking-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**Step 2: Update RequestHandler interface**

In `bin-common-handler/pkg/requesthandler/main.go`, replace the streaming methods (lines ~1214-1229) with:

```go
// tts-manager speakings
TTSV1SpeakingCreate(ctx context.Context, customerID uuid.UUID, referenceType tmstreaming.ReferenceType, referenceID uuid.UUID, language string, provider string, voiceID string, direction tmstreaming.Direction) (*tmspeaking.Speaking, error)
TTSV1SpeakingGet(ctx context.Context, speakingID uuid.UUID) (*tmspeaking.Speaking, error)
TTSV1SpeakingGets(ctx context.Context, pageToken string, pageSize uint64, filters map[tmspeaking.Field]any) ([]*tmspeaking.Speaking, error)
TTSV1SpeakingSay(ctx context.Context, podID string, speakingID uuid.UUID, text string) (*tmspeaking.Speaking, error)
TTSV1SpeakingFlush(ctx context.Context, podID string, speakingID uuid.UUID) (*tmspeaking.Speaking, error)
TTSV1SpeakingStop(ctx context.Context, podID string, speakingID uuid.UUID) (*tmspeaking.Speaking, error)
TTSV1SpeakingDelete(ctx context.Context, speakingID uuid.UUID) (*tmspeaking.Speaking, error)
```

Add imports for `tmspeaking "monorepo/bin-tts-manager/models/speaking"`.

**Step 3: Delete old streaming files**

```bash
rm bin-common-handler/pkg/requesthandler/tts_streamings.go
rm bin-common-handler/pkg/requesthandler/tts_streamings_test.go
```

**Step 4: Regenerate mocks**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api/bin-common-handler
go generate ./pkg/requesthandler/...
```

**Step 5: Write tests for new RPC methods**

Create `bin-common-handler/pkg/requesthandler/tts_speakings_test.go` following the same test patterns from the deleted `tts_streamings_test.go`.

**Step 6: Run verification**

Since we changed `bin-common-handler`, we MUST verify ALL services:

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api
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

Fix any compilation errors in services that referenced the old `TTSV1Streaming*` methods. Based on prior analysis, no service calls them, so this should be clean.

**Step 7: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-Add-speaking-api

- bin-common-handler: Add TTSV1Speaking* RPC client methods with pod-targeted routing
- bin-common-handler: Remove unused TTSV1Streaming* RPC methods"
```

---

## Task 9: OpenAPI Spec (`bin-openapi-manager`)

Define the speaking resource schema and endpoint paths.

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (add schemas and paths)
- Create: `bin-openapi-manager/openapi/paths/speakings/main.yaml`
- Create: `bin-openapi-manager/openapi/paths/speakings/id.yaml`
- Create: `bin-openapi-manager/openapi/paths/speakings/id_say.yaml`
- Create: `bin-openapi-manager/openapi/paths/speakings/id_flush.yaml`
- Create: `bin-openapi-manager/openapi/paths/speakings/id_stop.yaml`

**Step 1: Add schemas to openapi.yaml**

In the `components.schemas` section of `openapi.yaml`, add:

```yaml
TtsManagerSpeaking:
  type: object
  properties:
    id:
      type: string
      description: Speaking session identifier
    customer_id:
      type: string
      description: Customer identifier
    reference_type:
      type: string
      description: Type of the referenced entity (call, confbridge)
    reference_id:
      type: string
      description: ID of the referenced entity
    language:
      type: string
      description: TTS language (e.g. en-US)
    provider:
      type: string
      description: TTS provider (elevenlabs)
    voice_id:
      type: string
      description: Provider-specific voice ID
    direction:
      type: string
      description: Audio injection direction (in, out, both)
    status:
      type: string
      description: Session status (initiating, active, stopped)
    pod_id:
      type: string
      description: Kubernetes pod hosting this session
    tm_create:
      type: string
      description: Creation timestamp
    tm_update:
      type: string
      description: Last update timestamp
    tm_delete:
      type: string
      description: Soft-delete timestamp
```

**Step 2: Add tag**

In the `tags` section:

```yaml
- name: Speaking
  description: Operations related to real-time streaming TTS sessions
```

**Step 3: Add path references**

In the `paths` section:

```yaml
/speakings:
  $ref: './paths/speakings/main.yaml'
/speakings/{id}:
  $ref: './paths/speakings/id.yaml'
/speakings/{id}/say:
  $ref: './paths/speakings/id_say.yaml'
/speakings/{id}/flush:
  $ref: './paths/speakings/id_flush.yaml'
/speakings/{id}/stop:
  $ref: './paths/speakings/id_stop.yaml'
```

**Step 4: Create path YAML files**

Create `bin-openapi-manager/openapi/paths/speakings/main.yaml`:

```yaml
get:
  summary: List speaking sessions
  description: Returns a list of speaking sessions for the authenticated customer.
  tags:
    - Speaking
  parameters:
    - name: page_size
      in: query
      schema:
        type: integer
    - name: page_token
      in: query
      schema:
        type: string
  responses:
    '200':
      description: A list of speaking sessions.
      content:
        application/json:
          schema:
            type: array
            items:
              $ref: '#/components/schemas/TtsManagerSpeaking'

post:
  summary: Create a speaking session
  description: Creates a new streaming TTS session on a call or conference.
  tags:
    - Speaking
  requestBody:
    content:
      application/json:
        schema:
          type: object
          required:
            - reference_type
            - reference_id
          properties:
            reference_type:
              type: string
              description: Type of the referenced entity (call, confbridge)
            reference_id:
              type: string
              description: ID of the referenced entity
            language:
              type: string
              description: TTS language (e.g. en-US)
            provider:
              type: string
              description: TTS provider. Defaults to elevenlabs.
            voice_id:
              type: string
              description: Provider-specific voice ID. If empty, uses default for language.
            direction:
              type: string
              description: Audio injection direction (in, out, both). Defaults to out.
  responses:
    '200':
      description: The created speaking session.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/TtsManagerSpeaking'
```

Create `bin-openapi-manager/openapi/paths/speakings/id.yaml`:

```yaml
get:
  summary: Get a speaking session
  description: Returns details of a speaking session.
  tags:
    - Speaking
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
  responses:
    '200':
      description: The speaking session.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/TtsManagerSpeaking'

delete:
  summary: Delete a speaking session
  description: Soft-deletes a speaking session. Stops the session if still active.
  tags:
    - Speaking
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
  responses:
    '200':
      description: The deleted speaking session.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/TtsManagerSpeaking'
```

Create `bin-openapi-manager/openapi/paths/speakings/id_say.yaml`:

```yaml
post:
  summary: Send text to a speaking session
  description: Adds text to the speech queue. Can be called multiple times.
  tags:
    - Speaking
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
  requestBody:
    content:
      application/json:
        schema:
          type: object
          required:
            - text
          properties:
            text:
              type: string
              description: Text to be spoken.
  responses:
    '200':
      description: The speaking session.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/TtsManagerSpeaking'
```

Create `bin-openapi-manager/openapi/paths/speakings/id_flush.yaml`:

```yaml
post:
  summary: Flush a speaking session
  description: Cancels current speech and clears all queued messages. Session stays open.
  tags:
    - Speaking
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
  responses:
    '200':
      description: The speaking session.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/TtsManagerSpeaking'
```

Create `bin-openapi-manager/openapi/paths/speakings/id_stop.yaml`:

```yaml
post:
  summary: Stop a speaking session
  description: Terminates the session. Closes AudioSocket and ElevenLabs connections.
  tags:
    - Speaking
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
  responses:
    '200':
      description: The stopped speaking session.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/TtsManagerSpeaking'
```

**Step 5: Regenerate OpenAPI types**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api/bin-openapi-manager
go generate ./...
```

**Step 6: Verify**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api/bin-openapi-manager
go mod tidy && go mod vendor && go build ./...
```

**Step 7: Commit**

```bash
git add bin-openapi-manager/openapi/
git commit -m "NOJIRA-Add-speaking-api

- bin-openapi-manager: Add TtsManagerSpeaking schema and 7 speaking endpoints"
```

---

## Task 10: API Manager Routes (`bin-api-manager`)

Add HTTP routes in the API manager that call the speaking RPC methods.

**Files:**
- Create: `bin-api-manager/server/speakings.go`
- Create: `bin-api-manager/pkg/servicehandler/speakings.go`
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (add interface methods)

**Step 1: Add servicehandler methods**

In `bin-api-manager/pkg/servicehandler/main.go`, add to the `ServiceHandler` interface:

```go
SpeakingCreate(ctx context.Context, a *amagent.Agent, referenceType, referenceID, language, provider, voiceID, direction string) (*tmspeaking.Speaking, error)
SpeakingGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*tmspeaking.Speaking, error)
SpeakingGets(ctx context.Context, a *amagent.Agent, pageSize uint64, pageToken string) ([]*tmspeaking.Speaking, error)
SpeakingSay(ctx context.Context, a *amagent.Agent, id uuid.UUID, text string) (*tmspeaking.Speaking, error)
SpeakingFlush(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*tmspeaking.Speaking, error)
SpeakingStop(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*tmspeaking.Speaking, error)
SpeakingDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*tmspeaking.Speaking, error)
```

**Step 2: Implement servicehandler speaking methods**

Create `bin-api-manager/pkg/servicehandler/speakings.go`:

```go
package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *serviceHandler) speakingGet(ctx context.Context, id uuid.UUID) (*tmspeaking.Speaking, error) {
	res, err := h.reqHandler.TTSV1SpeakingGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get speaking: %v", err)
	}
	return res, nil
}

func (h *serviceHandler) SpeakingCreate(
	ctx context.Context,
	a *amagent.Agent,
	referenceType, referenceID, language, provider, voiceID, direction string,
) (*tmspeaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingCreate",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	refID := uuid.FromStringOrNil(referenceID)

	res, err := h.reqHandler.TTSV1SpeakingCreate(
		ctx,
		a.CustomerID,
		tmstreaming.ReferenceType(referenceType),
		refID,
		language,
		provider,
		voiceID,
		tmstreaming.Direction(direction),
	)
	if err != nil {
		log.Errorf("Could not create speaking. err: %v", err)
		return nil, err
	}
	log.WithField("speaking", res).Debugf("Created speaking. speaking_id: %s", res.ID)

	return res, nil
}

func (h *serviceHandler) SpeakingGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*tmspeaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingGet",
		"speaking_id": id,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res, err := h.speakingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get speaking. err: %v", err)
		return nil, err
	}

	if res.CustomerID != a.CustomerID {
		return nil, fmt.Errorf("speaking does not belong to this customer")
	}

	return res, nil
}

func (h *serviceHandler) SpeakingGets(ctx context.Context, a *amagent.Agent, pageSize uint64, pageToken string) ([]*tmspeaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingGets",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	filters := map[tmspeaking.Field]any{
		tmspeaking.FieldCustomerID: a.CustomerID,
	}

	res, err := h.reqHandler.TTSV1SpeakingGets(ctx, pageToken, pageSize, filters)
	if err != nil {
		log.Errorf("Could not get speakings. err: %v", err)
		return nil, err
	}

	return res, nil
}

func (h *serviceHandler) SpeakingSay(ctx context.Context, a *amagent.Agent, id uuid.UUID, text string) (*tmspeaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingSay",
		"speaking_id": id,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// Get speaking to verify ownership and get pod_id
	spk, err := h.speakingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get speaking. err: %v", err)
		return nil, err
	}

	if spk.CustomerID != a.CustomerID {
		return nil, fmt.Errorf("speaking does not belong to this customer")
	}

	res, err := h.reqHandler.TTSV1SpeakingSay(ctx, spk.PodID, id, text)
	if err != nil {
		log.Errorf("Could not say. err: %v", err)
		return nil, err
	}

	return res, nil
}

func (h *serviceHandler) SpeakingFlush(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*tmspeaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingFlush",
		"speaking_id": id,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	spk, err := h.speakingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get speaking. err: %v", err)
		return nil, err
	}

	if spk.CustomerID != a.CustomerID {
		return nil, fmt.Errorf("speaking does not belong to this customer")
	}

	res, err := h.reqHandler.TTSV1SpeakingFlush(ctx, spk.PodID, id)
	if err != nil {
		log.Errorf("Could not flush. err: %v", err)
		return nil, err
	}

	return res, nil
}

func (h *serviceHandler) SpeakingStop(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*tmspeaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingStop",
		"speaking_id": id,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	spk, err := h.speakingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get speaking. err: %v", err)
		return nil, err
	}

	if spk.CustomerID != a.CustomerID {
		return nil, fmt.Errorf("speaking does not belong to this customer")
	}

	res, err := h.reqHandler.TTSV1SpeakingStop(ctx, spk.PodID, id)
	if err != nil {
		log.Errorf("Could not stop. err: %v", err)
		return nil, err
	}

	return res, nil
}

func (h *serviceHandler) SpeakingDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*tmspeaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingDelete",
		"speaking_id": id,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	spk, err := h.speakingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get speaking. err: %v", err)
		return nil, err
	}

	if spk.CustomerID != a.CustomerID {
		return nil, fmt.Errorf("speaking does not belong to this customer")
	}

	res, err := h.reqHandler.TTSV1SpeakingDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete speaking. err: %v", err)
		return nil, err
	}

	return res, nil
}
```

**Step 3: Add server route handlers**

Create `bin-api-manager/server/speakings.go`:

```go
package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostSpeakings(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{"func": "PostSpeakings"})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	var req openapi_server.PostSpeakingsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	referenceType := ""
	if req.ReferenceType != nil {
		referenceType = *req.ReferenceType
	}
	referenceID := ""
	if req.ReferenceId != nil {
		referenceID = *req.ReferenceId
	}
	language := ""
	if req.Language != nil {
		language = *req.Language
	}
	provider := ""
	if req.Provider != nil {
		provider = *req.Provider
	}
	voiceID := ""
	if req.VoiceId != nil {
		voiceID = *req.VoiceId
	}
	direction := ""
	if req.Direction != nil {
		direction = *req.Direction
	}

	res, err := h.serviceHandler.SpeakingCreate(c.Request.Context(), &a, referenceType, referenceID, language, provider, voiceID, direction)
	if err != nil {
		log.Errorf("Could not create speaking. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetSpeakings(c *gin.Context, params openapi_server.GetSpeakingsParams) {
	log := logrus.WithFields(logrus.Fields{"func": "GetSpeakings"})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	res, err := h.serviceHandler.SpeakingGets(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get speakings. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetSpeakingsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{"func": "GetSpeakingsId"})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	speakingID := uuid.FromStringOrNil(id)

	res, err := h.serviceHandler.SpeakingGet(c.Request.Context(), &a, speakingID)
	if err != nil {
		log.Errorf("Could not get speaking. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteSpeakingsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{"func": "DeleteSpeakingsId"})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	speakingID := uuid.FromStringOrNil(id)

	res, err := h.serviceHandler.SpeakingDelete(c.Request.Context(), &a, speakingID)
	if err != nil {
		log.Errorf("Could not delete speaking. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostSpeakingsIdSay(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{"func": "PostSpeakingsIdSay"})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	speakingID := uuid.FromStringOrNil(id)

	var req openapi_server.PostSpeakingsIdSayJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	text := ""
	if req.Text != nil {
		text = *req.Text
	}

	res, err := h.serviceHandler.SpeakingSay(c.Request.Context(), &a, speakingID, text)
	if err != nil {
		log.Errorf("Could not say. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostSpeakingsIdFlush(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{"func": "PostSpeakingsIdFlush"})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	speakingID := uuid.FromStringOrNil(id)

	res, err := h.serviceHandler.SpeakingFlush(c.Request.Context(), &a, speakingID)
	if err != nil {
		log.Errorf("Could not flush. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostSpeakingsIdStop(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{"func": "PostSpeakingsIdStop"})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	speakingID := uuid.FromStringOrNil(id)

	res, err := h.serviceHandler.SpeakingStop(c.Request.Context(), &a, speakingID)
	if err != nil {
		log.Errorf("Could not stop. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
```

**Step 4: Regenerate api-manager server code and verify**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api/bin-api-manager
go generate ./...
go mod tidy && go mod vendor && go build ./...
go test ./...
golangci-lint run -v --timeout 5m
```

The generated code in `gens/openapi_server/gen.go` will define the method signatures (`PostSpeakings`, `GetSpeakings`, etc.) that `server/speakings.go` must implement. If the names don't match exactly, adjust the server methods to match the generated interface.

**Step 5: Commit**

```bash
git add bin-api-manager/
git commit -m "NOJIRA-Add-speaking-api

- bin-api-manager: Add HTTP routes for speaking endpoints
- bin-api-manager: Add servicehandler methods for speaking CRUD and operations"
```

---

## Task 11: Full Verification

Run the full verification workflow across all affected services.

**Step 1: Verify all changed services individually**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-speaking-api

# bin-tts-manager
(cd bin-tts-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m)

# bin-openapi-manager
(cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m)

# bin-api-manager
(cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m)
```

**Step 2: Since bin-common-handler was changed, verify ALL services**

```bash
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

**Step 3: Fix any failures**

If any service fails because it references the removed `TTSV1Streaming*` methods, update that service to remove the references or replace them with the new `TTSV1Speaking*` methods.

**Step 4: Commit any fixes**

```bash
git add -A
git commit -m "NOJIRA-Add-speaking-api

- Fix verification issues across services after bin-common-handler changes"
```

---

## Dependency Order

```
Task 1 (migration) — no code deps, can start immediately
Task 2 (model) — no deps
Task 3 (config + main DB) — no deps
Task 4 (dbhandler) — depends on Task 2, 3
Task 5 (streamer flush) — no deps on speaking model
Task 6 (speakinghandler) — depends on Task 4, 5
Task 7 (listenhandler) — depends on Task 6
Task 8 (RPC client) — depends on Task 2, 7 (for request models)
Task 9 (OpenAPI) — no code deps, can happen in parallel
Task 10 (API manager) — depends on Task 8, 9
Task 11 (verification) — depends on all
```

Recommended execution order:

```
[Task 1] + [Task 2] + [Task 3] + [Task 5] in parallel
    → [Task 4]
    → [Task 6]
    → [Task 7] + [Task 9] in parallel
    → [Task 8]
    → [Task 10]
    → [Task 11]
```
