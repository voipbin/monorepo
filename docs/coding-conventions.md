# VoIPbin Monorepo Coding Conventions

> **This is the authoritative source for all coding conventions in this monorepo.**
> For workflow rules (verification, git, deployment), see [CLAUDE.md](../CLAUDE.md).
> For architecture details, see [architecture-deep-dive.md](architecture-deep-dive.md).

## 1. Package Structure & File Organization

### 1.1 Standard Service Layout

Every `bin-*` service follows this directory structure:

```
bin-<service-name>/
  cmd/
    <service-name>/main.go              # Daemon entry point (Cobra command)
    <service-name>-control/main.go      # CLI tool (direct DB/cache, no RabbitMQ)
  internal/
    config/main.go                      # Viper/Cobra config, sync.Once singleton
  models/
    <entity>/
      <entity>.go                       # Core struct with db: and json: tags
      field.go                          # Field type string constants for map keys
      event.go                          # EventType string constants
      webhook.go                        # WebhookMessage + ConvertWebhookMessage()
  pkg/
    dbhandler/
      main.go                           # DBHandler interface + handler struct + ErrNotFound
      <entity>.go                       # Squirrel SQL operations per entity
      mock_main.go                      # Generated mock (via go:generate)
    cachehandler/
      main.go                           # CacheHandler interface + Redis implementation
    <domain>handler/
      main.go                           # Handler interface + struct + constructor
      <feature>.go                      # Business logic grouped by feature
      db.go                             # Private DB-layer wrappers (dbGet, dbCreate, etc.)
      event.go                          # Event handlers (EventXxx methods)
      mock_main.go                      # Generated mock
    listenhandler/
      main.go                           # Regex routing + prometheus + Run()
      v1_<resource>.go                  # Per-resource RPC request handlers
      models/request/                   # Request body structs
    subscribehandler/
      main.go                           # Event subscription + routing
  CLAUDE.md                             # Service-specific conventions
```

**Rationale:** Uniform layout across 30+ services makes navigation predictable and enables cross-service tooling.

### 1.2 Two Binaries Per Service

Every service provides two binaries:
- **Daemon** (`cmd/<service-name>/main.go`) — Long-running process consuming RabbitMQ RPC messages
- **Control CLI** (`cmd/<service-name>-control/main.go`) — Admin tool that bypasses RabbitMQ and accesses DB/cache directly

```go
// CORRECT — daemon binary
// bin-agent-manager/cmd/agent-manager/main.go
func main() {
    rootCmd := &cobra.Command{
        Use:  "agent-manager",
        RunE: runCommand,
    }
    // ...
}

// CORRECT — control CLI binary
// bin-agent-manager/cmd/agent-control/main.go
func main() {
    rootCmd := &cobra.Command{
        Use: "agent-control",
    }
    // subcommands for direct DB/cache operations
}
```

### 1.3 Model File Organization

Each entity in `models/<entity>/` has companion files:

| File | Purpose | Example |
|------|---------|---------|
| `<entity>.go` | Core struct with `db:` and `json:` tags | `models/agent/agent.go` |
| `field.go` | `Field` type + constants for type-safe update maps | `models/agent/field.go` |
| `event.go` | `EventType` constants for event publishing | `models/agent/event.go` |
| `webhook.go` | `WebhookMessage` struct + `ConvertWebhookMessage()` | `models/agent/webhook.go` |

```go
// CORRECT — all four files present for agent entity
models/agent/
  agent.go      // type Agent struct { ... }
  field.go      // type Field string; const FieldID Field = "id"
  event.go      // const EventTypeAgentCreated = "agent_created"
  webhook.go    // type WebhookMessage struct { ... }
```

**Wrong:**
```go
// WRONG — putting everything in one file
models/agent/agent.go  // contains Agent struct + Field type + events + webhook
```

### 1.4 Where Code Lives

| Code Type | Location | Example |
|-----------|----------|---------|
| Business logic | `pkg/<domain>handler/` | `pkg/agenthandler/agent.go` |
| Database operations | `pkg/dbhandler/` | `pkg/dbhandler/agent.go` |
| Cache operations | `pkg/cachehandler/` | `pkg/cachehandler/main.go` |
| RPC routing | `pkg/listenhandler/` | `pkg/listenhandler/v1_agents.go` |
| Event subscriptions | `pkg/subscribehandler/` | `pkg/subscribehandler/main.go` |
| Model definitions | `models/<entity>/` | `models/agent/agent.go` |
| Config | `internal/config/` | `internal/config/main.go` |
| Service entrypoint | `cmd/<service>/` | `cmd/agent-manager/main.go` |

**Wrong — business logic in dbhandler:**
```go
// WRONG — dbhandler should only do DB operations, not business logic
func (h *handler) AgentCreate(ctx context.Context, a *agent.Agent) error {
    // validation logic here  ← WRONG, belongs in agenthandler
    if a.Name == "" { return errors.New("name required") }
    // ...
}
```

---

## 2. Naming Conventions

### 2.1 Function Naming

| Operation | Pattern | Example |
|-----------|---------|---------|
| Single item by ID | `Get` | `AgentGet(ctx, id)` |
| Paginated collection | `List` | `AgentList(ctx, size, token, filters)` |
| Alternate lookups | `GetBy<Criteria>` | `GetByCustomerIDAndAddress(ctx, custID, addr)` |
| Create resource | `Create` | `AgentCreate(ctx, custID, username, ...)` |
| Full update | `Update` | `AgentUpdate(ctx, id, fields)` |
| Partial update | `Update<Field>` | `UpdateBasicInfo(ctx, id, name, detail)` |
| Delete resource | `Delete` | `AgentDelete(ctx, id)` |
| Force delete | `deleteForce` | `deleteForce(ctx, id)` (private, unconditional) |

```go
// CORRECT
func (h *flowHandler) Get(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
func (h *flowHandler) List(ctx context.Context, token string, size uint64, filters map[flow.Field]any) ([]*flow.Flow, error)

// WRONG — never use "Gets" for collection retrieval
func (h *flowHandler) Gets(ctx context.Context, filters map[flow.Field]any) ([]*flow.Flow, error)
```

**Rationale:** `List` follows Go standard library conventions. `Gets` is non-idiomatic.

### 2.2 Event Handler Naming

Event handlers use the pattern: `Event<ServicePrefix><EventName>`

```go
// CORRECT
func (h *flowHandler) EventCMCallHangup(ctx context.Context, call *cmcall.Call) error
func (h *agentHandler) EventGroupcallCreated(ctx context.Context, gc *groupcall.Groupcall) error
func (h *billingHandler) EventCustomerDeleted(ctx context.Context, cust *customer.Customer) error

// WRONG — missing Event prefix or service prefix
func (h *flowHandler) HandleCallHangup(ctx context.Context, call *cmcall.Call) error
func (h *flowHandler) OnCallHangup(ctx context.Context, call *cmcall.Call) error
```

### 2.3 Private DB Helper Naming

Private methods that wrap DB operations use the `db` prefix:

```go
// CORRECT
func (h *agentHandler) dbCreate(ctx context.Context, ...) (*agent.Agent, error)
func (h *agentHandler) dbGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
func (h *agentHandler) dbDelete(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
func (h *agentHandler) dbList(ctx context.Context, ...) ([]*agent.Agent, error)
func (h *agentHandler) dbUpdateInfo(ctx context.Context, ...) (*agent.Agent, error)

// WRONG
func (h *agentHandler) createInDB(ctx context.Context, ...) (*agent.Agent, error)
func (h *agentHandler) getFromDatabase(ctx context.Context, ...) (*agent.Agent, error)
```

### 2.4 Type Naming

| Type | Naming Pattern | Example |
|------|---------------|---------|
| String enums | `Type` | `flow.Type`, `agent.Status`, `billing.ReferenceType` |
| Map key types | `Field` | `flow.Field`, `agent.Field` |
| Enum values | `Type<Value>` | `TypeFlow`, `TypeNone`, `StatusOffline` |
| Event types | `EventType<Name>` | `EventTypeFlowCreated`, `EventTypeAgentDeleted` |
| Table names | Unexported var | `var agentTable = "agent_agents"` |

```go
// CORRECT — field.go
type Field string

const (
    FieldID         Field = "id"
    FieldCustomerID Field = "customer_id"
    FieldUsername    Field = "username"
    FieldStatus     Field = "status"
    FieldTMDelete   Field = "tm_delete"
    FieldDeleted    Field = "deleted"  // filter-only sentinel: maps to "tm_delete IS NULL"
)

// CORRECT — event.go
const (
    EventTypeAgentCreated       = "agent_created"
    EventTypeAgentUpdated       = "agent_updated"
    EventTypeAgentDeleted       = "agent_deleted"
)

// CORRECT — table name
var agentTable = "agent_agents"
```

### 2.5 Import Aliases

Cross-service model imports use a 2-3 letter prefix derived from the service name + model:

```go
// CORRECT
import (
    cucustomer "monorepo/bin-customer-manager/models/customer"
    fmaction "monorepo/bin-flow-manager/models/action"
    fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
    commonaddress "monorepo/bin-common-handler/models/address"
    commonidentity "monorepo/bin-common-handler/models/identity"
    commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
    smpod "monorepo/bin-sentinel-manager/models/pod"
)

// WRONG — unclear or inconsistent aliases
import (
    cust "monorepo/bin-customer-manager/models/customer"        // too short
    customer "monorepo/bin-customer-manager/models/customer"    // conflicts with package name
    flowAction "monorepo/bin-flow-manager/models/action"        // camelCase not used
)
```

### 2.6 Test Variable Naming

| Variable | Naming | Example |
|----------|--------|---------|
| gomock controller | `mc` | `mc := gomock.NewController(t)` |
| Handler under test | `h` | `h := &flowHandler{...}` |
| Test case iterator | `tt` | `for _, tt := range tests` |
| Mock instances | `mock<Name>` | `mockDB`, `mockReq`, `mockNotify`, `mockUtil` |

---

## 3. Import Ordering

### 3.1 Five Groups

Imports are organized into five groups, separated by blank lines:

```go
import (
    // 1. Standard library
    "context"
    "database/sql"
    "fmt"
    "time"

    // 2. bin-common-handler (shared monorepo library)
    commonaddress "monorepo/bin-common-handler/models/address"
    commonidentity "monorepo/bin-common-handler/models/identity"
    "monorepo/bin-common-handler/pkg/notifyhandler"
    "monorepo/bin-common-handler/pkg/requesthandler"
    "monorepo/bin-common-handler/pkg/utilhandler"

    // 3. Cross-service models (other bin-* services)
    cucustomer "monorepo/bin-customer-manager/models/customer"
    fmaction "monorepo/bin-flow-manager/models/action"

    // 4. Third-party packages
    "github.com/Masterminds/squirrel"
    "github.com/gofrs/uuid"
    "github.com/pkg/errors"
    "github.com/sirupsen/logrus"
    gomock "go.uber.org/mock/gomock"

    // 5. Local service packages (same service)
    "monorepo/bin-agent-manager/models/agent"
    "monorepo/bin-agent-manager/pkg/cachehandler"
    "monorepo/bin-agent-manager/pkg/dbhandler"
)
```

**Rationale:** Consistent grouping makes import blocks scannable and prevents merge conflicts.

**Wrong:**
```go
// WRONG — all mixed together
import (
    "context"
    "monorepo/bin-agent-manager/models/agent"
    "github.com/gofrs/uuid"
    "monorepo/bin-common-handler/pkg/requesthandler"
    "fmt"
)

---

## 4. Error Handling

### 4.1 Sentinel Errors

Define sentinel errors as package-level variables in `dbhandler/main.go`:

```go
// CORRECT — dbhandler/main.go
var ErrNotFound = errors.New("record not found")

// For services with more error types:
var (
    ErrNotFound            = errors.New("record not found")
    ErrInsufficientBalance = errors.New("insufficient balance")
    ErrDuplicateKey        = errors.New("duplicate key")
)
```

**Wrong:**
```go
// WRONG — sentinel errors defined in business handler
// pkg/agenthandler/main.go
var ErrNotFound = errors.New("not found")  // Should be in dbhandler
```

### 4.2 Error Wrapping

Use `fmt.Errorf` with `%w` or `errors.Wrap`/`errors.Wrapf` from `github.com/pkg/errors`:

```go
// CORRECT — wrapping with context
return nil, fmt.Errorf("could not get flow count: %w", err)
return nil, errors.Wrap(err, "could not create an agent")
return nil, errors.Wrapf(err, "could not update flow actions. flow_id: %s", id)

// CORRECT — creating new errors with context
return nil, errors.Errorf("agent is guest agent")
return nil, fmt.Errorf("resource limit exceeded")

// WRONG — returning raw errors without context
return nil, err  // No context about where or why it failed
```

### 4.3 Checking Sentinel Errors

Compare sentinel errors directly (not with `errors.Is` unless wrapping is involved):

```go
// CORRECT — direct comparison for dbhandler sentinel errors
ag, err := h.GetByCustomerIDAndAddress(ctx, a.CustomerID, &address)
if err != nil && err != dbhandler.ErrNotFound {
    return nil, errors.Wrap(err, "could not get agent info of the address")
}

// CORRECT — errors.Is when error might be wrapped
if errors.Is(err, dbhandler.ErrNotFound) {
    return nil, fmt.Errorf("resource not found")
}
```

### 4.4 Log-Then-Return Pattern

Always log the error before returning, especially at handler boundaries:

```go
// CORRECT — log then return
af, err := h.Get(ctx, activeflowID)
if err != nil {
    log.Errorf("Could not get activeflow info: %v", err)
    return errors.Wrapf(err, "could not get activeflow info")
}

// WRONG — returning without logging
af, err := h.Get(ctx, activeflowID)
if err != nil {
    return errors.Wrapf(err, "could not get activeflow info")  // No log = invisible in production
}
```

---

## 5. Logging

### 5.1 Function-Scoped Logger

**MANDATORY:** Create a function-scoped log variable as the first statement of every function:

```go
// CORRECT — multiple context fields
func (h *flowHandler) Get(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
    log := logrus.WithFields(logrus.Fields{
        "func": "Get",
        "id":   id,
    })
    // use log throughout the function
}

// CORRECT — single context field
func (h *handler) processRequest(m *sock.Request) (*sock.Response, error) {
    log := logrus.WithField("func", "processRequest")
    // ...
}

// WRONG — using package-level logger
func (h *flowHandler) Get(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
    logrus.Debugf("Getting flow %s", id)  // No function context
}
```

### 5.2 Log Levels

| Level | Use For | Example |
|-------|---------|---------|
| `Debug` | Routine operations, entry/progress | `log.Debug("Creating a new flow.")` |
| `Info` | Non-error notable events | `log.Infof("Could not get call: %v", err)` (not-found is not an error) |
| `Warn` | Safe-default fallbacks | `log.Warnf("Cache miss, falling back to DB")` |
| `Error` | All failures | `log.Errorf("Could not get channel: %v", err)` |

### 5.3 Structured Object Logging After Data Retrieval

**MANDATORY:** Add debug logs when retrieving data from other services or databases:

```go
// CORRECT — log the full object and key identifier after retrieval
call, err := h.callGet(ctx, callID)
if err != nil {
    log.Infof("Could not get call: %v", err)
    return nil, fmt.Errorf("call not found")
}
log.WithField("call", call).Debugf("Retrieved call info. call_id: %s", call.ID)

ch, err := h.reqHandler.CallV1ChannelGet(ctx, call.ChannelID)
if err != nil {
    log.Errorf("Could not get channel: %v", err)
    return nil, fmt.Errorf("no data available")
}
log.WithField("channel", ch).Debugf("Retrieved channel info. channel_id: %s", ch.ID)

// WRONG — no logging after retrieval
call, err := h.callGet(ctx, callID)
if err != nil {
    return nil, err  // Also missing: no log, no context
}
// silently continues without logging the retrieved object
```

### 5.4 Error Message Format

Use the consistent format `"Could not <action>: %v"` or `"Could not <action>. err: %v"`:

```go
// CORRECT
log.Errorf("Could not get flow info: %v", err)
log.Errorf("Could not get flow info. err: %v", err)

// WRONG — inconsistent formats
log.Errorf("Error getting flow: %v", err)
log.Errorf("failed to get flow %v", err)
log.Errorf("GetFlow failed: %v", err)
```

### 5.5 Import Pattern

Always import logrus directly without alias:

```go
// CORRECT
import "github.com/sirupsen/logrus"

func (h *handler) Get(ctx context.Context, id uuid.UUID) {
    log := logrus.WithFields(logrus.Fields{"func": "Get", "id": id})
}

// WRONG — aliasing logrus
import log "github.com/sirupsen/logrus"  // Confusing: shadows log variable pattern
```

---

## 6. Model Definitions

### 6.1 Identity Embedding

All models with an ID and customer ownership embed `commonidentity.Identity`:

```go
// CORRECT
type Agent struct {
    commonidentity.Identity  // Provides ID uuid.UUID `db:"id,uuid"` and CustomerID uuid.UUID `db:"customer_id,uuid"`

    Username string `json:"username" db:"username"`
    // ...
}

// WRONG — defining ID fields manually
type Agent struct {
    ID         uuid.UUID `json:"id" db:"id,uuid"`
    CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`
    Username   string    `json:"username" db:"username"`
}
```

### 6.2 DB Tag Conventions

| Tag | Usage | Example |
|-----|-------|---------|
| `db:"column_name"` | Plain column | `Name string \`db:"name"\`` |
| `db:"column_name,uuid"` | UUID stored as BINARY(16) | `ID uuid.UUID \`db:"id,uuid"\`` |
| `db:"column_name,json"` | Slice/map/struct as JSON text | `TagIDs []uuid.UUID \`db:"tag_ids,json"\`` |
| `db:"-"` | Excluded from DB operations | `TempField string \`db:"-"\`` |

```go
// CORRECT — all tags present and correct
type Agent struct {
    commonidentity.Identity

    Username     string                  `json:"username" db:"username"`
    PasswordHash string                  `json:"-" db:"password_hash"`           // json:"-" hides from API
    Name         string                  `json:"name" db:"name"`
    Status       Status                  `json:"status" db:"status"`
    TagIDs       []uuid.UUID             `json:"tag_ids" db:"tag_ids,json"`      // JSON-serialized in DB
    Addresses    []commonaddress.Address `json:"addresses" db:"addresses,json"`  // JSON-serialized in DB

    TMCreate *time.Time `json:"tm_create,omitempty" db:"tm_create"`
    TMUpdate *time.Time `json:"tm_update,omitempty" db:"tm_update"`
    TMDelete *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`
}

// WRONG — missing ,uuid tag on UUID field
type Agent struct {
    ID uuid.UUID `db:"id"`  // BUG: queries will fail silently
}
```

### 6.3 Timestamp Fields

All models use pointer timestamps: `TMCreate`, `TMUpdate`, `TMDelete` as `*time.Time`:

```go
// CORRECT
TMCreate *time.Time `json:"tm_create,omitempty" db:"tm_create"`
TMUpdate *time.Time `json:"tm_update,omitempty" db:"tm_update"`
TMDelete *time.Time `json:"tm_delete,omitempty" db:"tm_delete"` // nil = active (soft delete)
```

### 6.4 Field Type Definition

Each model defines a `Field` type in `models/<entity>/field.go` for type-safe update maps:

```go
// CORRECT — field.go
package agent

type Field string

const (
    FieldID         Field = "id"
    FieldCustomerID Field = "customer_id"
    FieldUsername    Field = "username"
    FieldName       Field = "name"
    FieldStatus     Field = "status"
    FieldTMCreate   Field = "tm_create"
    FieldTMUpdate   Field = "tm_update"
    FieldTMDelete   Field = "tm_delete"
    FieldDeleted    Field = "deleted"  // Filter sentinel: maps to "tm_delete IS NULL"
)
```

### 6.5 Event Constants

Event types are defined in `models/<entity>/event.go`:

```go
// CORRECT — event.go
package agent

const (
    EventTypeAgentCreated       = "agent_created"
    EventTypeAgentUpdated       = "agent_updated"
    EventTypeAgentDeleted       = "agent_deleted"
    EventTypeAgentStatusUpdated = "agent_status_updated"
)
```

### 6.6 WebhookMessage Pattern

**MANDATORY:** All external-facing API responses use `WebhookMessage`, never the internal model struct.

```go
// CORRECT — webhook.go
type WebhookMessage struct {
    commonidentity.Identity

    Username   string                  `json:"username"`
    Name       string                  `json:"name"`
    Detail     string                  `json:"detail"`
    RingMethod RingMethod              `json:"ring_method"`
    Status     Status                  `json:"status"`
    Permission Permission              `json:"permission"`
    TagIDs     []uuid.UUID             `json:"tag_ids"`
    Addresses  []commonaddress.Address `json:"addresses"`
    TMCreate   *time.Time              `json:"tm_create,omitempty"`
    TMUpdate   *time.Time              `json:"tm_update,omitempty"`
    TMDelete   *time.Time              `json:"tm_delete,omitempty"`
    // NOTE: PasswordHash intentionally omitted — internal only
}

func (h *Agent) ConvertWebhookMessage() *WebhookMessage {
    return &WebhookMessage{
        Identity:   h.Identity,
        Username:   h.Username,
        Name:       h.Name,
        // ... all safe fields
    }
}

func (h *Agent) CreateWebhookEvent() ([]byte, error) {
    e := h.ConvertWebhookMessage()
    return json.Marshal(e)
}
```

**Compound result structs** must also have WebhookMessage variants:
```go
// CORRECT — compound result with webhook variant
type SignupResult struct {
    Customer *Customer
    Token    string
}

type SignupResultWebhookMessage struct {
    Customer *WebhookMessage  // Uses WebhookMessage, not internal Customer
    Token    string
}

func (h *SignupResult) ConvertWebhookMessage() *SignupResultWebhookMessage {
    return &SignupResultWebhookMessage{
        Customer: h.Customer.ConvertWebhookMessage(),
        Token:    h.Token,
    }
}
```

---

## 7. Database Patterns

### 7.1 Squirrel Query Builder (Mandatory)

All SQL queries MUST use the squirrel query builder. Raw SQL strings are forbidden.

```go
// CORRECT — squirrel
query, args, _ := squirrel.Select(fields...).
    From(agentTable).
    Where(squirrel.Eq{string(agent.FieldID): id.Bytes()}).
    PlaceholderFormat(squirrel.Question).
    ToSql()

// WRONG — raw SQL
query := "SELECT * FROM agent_agents WHERE id = ?"
```

**Exception:** Computed expressions that squirrel cannot express (e.g., `cost_per_unit * ?`). Document WHY with a comment.

### 7.2 CRUD Operations

**INSERT:**
```go
// CORRECT
a.TMCreate = h.utilHandler.TimeNow()
fields, _ := commondatabasehandler.PrepareFields(a)
sb := squirrel.Insert(agentTable).SetMap(fields).PlaceholderFormat(squirrel.Question)
query, args, _ := sb.ToSql()
_, err := h.db.ExecContext(ctx, query, args...)
```

**SELECT:**
```go
// CORRECT — using GetDBFields + ScanRow
fields := commondatabasehandler.GetDBFields(&agent.Agent{})
query, args, _ := squirrel.Select(fields...).
    From(agentTable).
    Where(squirrel.Eq{string(agent.FieldID): id.Bytes()}).
    PlaceholderFormat(squirrel.Question).ToSql()
row, err := h.db.QueryContext(ctx, query, args...)
res := &agent.Agent{}
if err := commondatabasehandler.ScanRow(row, res); err != nil { ... }

// WRONG — manual rows.Scan
row.Scan(&m.ID, &m.CustomerID, &m.Name)  // FORBIDDEN
```

**UPDATE:**
```go
// CORRECT
fields[agent.FieldTMUpdate] = h.utilHandler.TimeNow()
tmpFields, _ := commondatabasehandler.PrepareFields(fields)
q := squirrel.Update(agentTable).SetMap(tmpFields).Where(squirrel.Eq{"id": id.Bytes()})
```

**DELETE (soft):**
```go
// CORRECT — soft delete by setting TMDelete
now := h.utilHandler.TimeNow()
return h.agentUpdate(ctx, id, map[agent.Field]any{
    agent.FieldTMDelete: now,
    agent.FieldTMUpdate: now,
})
```

### 7.3 Empty Slice Initialization

**MANDATORY:** List functions must initialize result slices as empty, never nil:

```go
// CORRECT — empty slice
res := []*agent.Agent{}

// WRONG — nil slice serializes to null in JSON instead of []
var res []*agent.Agent
```

### 7.4 Cache-Aside Pattern

DB reads: cache-first, fallback to DB. Mutations: write-through.

```go
// CORRECT — cache-aside read
func (h *handler) AgentGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
    if res, err := h.agentGetFromCache(ctx, id); err == nil {
        return res, nil  // cache hit
    }
    return h.agentGetFromDB(ctx, id)  // cache miss → DB → set cache
}

// CORRECT — write-through on mutation
func (h *handler) AgentCreate(ctx context.Context, a *agent.Agent) error {
    // ... insert to DB ...
    _ = h.agentUpdateToCache(ctx, a.ID)  // update cache after DB write
    return nil
}
```

### 7.5 Cursor-Based Pagination

Pagination uses `TMCreate` timestamp as cursor token:

```go
// CORRECT
if token == "" {
    token = h.utilHandler.TimeGetCurTime()
}
sb := squirrel.Select(fields...).From(agentTable).
    Where(squirrel.Lt{string(agent.FieldTMCreate): token}).
    OrderBy(string(agent.FieldTMCreate) + " DESC").
    Limit(size)
```

### 7.6 Filter Application

Use `commondatabasehandler.ApplyFields()` for type-safe filter maps:

```go
// CORRECT
sb, _ = commondatabasehandler.ApplyFields(sb, filters)
// Handles: uuid → bytes, "deleted: false" → tm_delete IS NULL, etc.
```

### 7.7 DB Operations Location

All database operations MUST live in `pkg/dbhandler/`. Business logic handlers receive `DBHandler` interface only.

```go
// CORRECT — business handler uses interface
type agentHandler struct {
    db dbhandler.DBHandler  // interface, not *sql.DB
}

// WRONG — business handler accessing DB directly
type agentHandler struct {
    db *sql.DB  // FORBIDDEN outside dbhandler
}
```

---

## 8. Handler Architecture

### 8.1 Interface-Driven Pattern

Every handler package defines its interface in `main.go` with a `//go:generate` directive:

```go
// CORRECT — main.go
package flowhandler

//go:generate mockgen -package flowhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

type FlowHandler interface {
    Get(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
    List(ctx context.Context, token string, size uint64, filters map[flow.Field]any) ([]*flow.Flow, error)
    Create(ctx context.Context, customerID uuid.UUID, ...) (*flow.Flow, error)
    Update(ctx context.Context, id uuid.UUID, fields map[flow.Field]any) (*flow.Flow, error)
    Delete(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
}
```

### 8.2 Private Struct + Public Interface

The implementing struct is private (lowercase). The constructor returns the interface:

```go
// CORRECT
type flowHandler struct {
    util          utilhandler.UtilHandler
    db            dbhandler.DBHandler
    reqHandler    requesthandler.RequestHandler
    notifyHandler notifyhandler.NotifyHandler
}

func NewFlowHandler(db dbhandler.DBHandler, req requesthandler.RequestHandler, ...) FlowHandler {
    h := &flowHandler{
        db:         db,
        reqHandler: req,
        // ...
    }
    return h
}

// WRONG — returning concrete type
func NewFlowHandler(...) *flowHandler {  // Returns struct, not interface
    return &flowHandler{...}
}
```

### 8.3 Two-Layer Split

Public methods handle business logic (validation, permissions, events). Private `db*` methods handle DB + cache + notify:

```go
// CORRECT — public method: validation + event publishing
func (h *agentHandler) Create(ctx context.Context, customerID uuid.UUID, ...) (*agent.Agent, error) {
    // 1. Validate inputs
    // 2. Check resource limits
    // 3. Call private db helper
    res, err := h.dbCreate(ctx, customerID, ...)
    // 4. Publish event
    h.notifyHandler.PublishWebhookEvent(ctx, customerID, agent.EventTypeAgentCreated, res)
    return res, nil
}

// CORRECT — private db helper: DB + cache + re-fetch
func (h *agentHandler) dbCreate(ctx context.Context, customerID uuid.UUID, ...) (*agent.Agent, error) {
    a := &agent.Agent{...}
    if err := h.db.AgentCreate(ctx, a); err != nil { return nil, err }
    res, err := h.Get(ctx, a.ID)  // re-fetch for DB-populated fields
    return res, nil
}
```

### 8.4 Get-After-Write Pattern

**MANDATORY:** Every Create/Update/Delete re-fetches the resource after the DB operation, then publishes the event:

```go
// CORRECT — universal pattern
if err := h.db.FlowCreate(ctx, f); err != nil { ... }
res, err := h.Get(ctx, id)   // re-fetch from DB/cache
if err != nil { ... }
h.notifyHandler.PublishEvent(ctx, flow.EventTypeFlowCreated, res)  // publish with fresh data
return res, nil
```

**Rationale:** DB may populate default values, timestamps, or trigger-generated fields. The published event must reflect the actual DB state.

---

## 9. Inter-Service Communication

### 9.1 RabbitMQ RPC via RequestHandler

All inter-service calls go through `requesthandler.RequestHandler` typed methods. Never call services directly:

```go
// CORRECT — typed RPC call
agent, err := h.reqHandler.AgentV1AgentGet(ctx, agentID)

// WRONG — constructing raw RPC requests
req := &sock.Request{URI: "/v1/agents/" + id.String(), Method: "GET"}
resp, err := h.sockHandler.RequestPublish(ctx, "bin-manager.agent-manager.request", req)
```

### 9.2 ListenHandler Routing

Incoming RPC requests are routed by regex matching on URI + method:

```go
// CORRECT — regex routing pattern
var (
    regV1Agents    = regexp.MustCompile("/v1/agents$")
    regV1AgentsGet = regexp.MustCompile(`/v1/agents\?(.*)$`)
    regV1AgentsID  = regexp.MustCompile("/v1/agents/" + regUUID + "$")
)

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
    ctx := context.Background()  // fresh context per request
    switch {
    case regV1AgentsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
        return h.processV1AgentsGet(ctx, m)
    case regV1Agents.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
        return h.processV1AgentsPost(ctx, m)
    }
}
```

### 9.3 Queue Naming

Services use three queues:
```
bin-manager.<service-name>.request    # RPC requests
bin-manager.<service-name>.event      # Published events
bin-manager.<service-name>.subscribe  # Event subscriptions
bin-manager.delay                     # Shared delayed message queue
```

### 9.4 Response Status Codes

Use HTTP-style status codes in `sock.Response`:

```go
// CORRECT
return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
return &sock.Response{StatusCode: 404}, nil
return &sock.Response{StatusCode: 500}, nil
```

---

## 10. API & External Interfaces

### 10.1 Atomic API Responses

API endpoints return single resource types without combining data from other services:

```go
// CORRECT — single resource
func (h *serviceHandler) BillingGet(ctx context.Context, ...) (*bmbilling.WebhookMessage, error) {
    // Returns just the billing record
}

// WRONG — combined resources
func (h *serviceHandler) BillingGet(ctx context.Context, ...) (*BillingWithAccount, error) {
    // Returns billing + account + call details
}
```

**Exceptions:** Pagination metadata (`next_page_token`) and atomic operations that create multiple resources in one transaction.

### 10.2 Two-Level ServiceHandler

In `bin-api-manager/pkg/servicehandler/`, private helpers return internal structs; public methods return `*WebhookMessage`:

```go
// CORRECT — private: internal struct for permission checks
func (h *serviceHandler) agentGet(ctx context.Context, id uuid.UUID) (*amagent.Agent, error) {
    res, err := h.reqHandler.AgentV1AgentGet(ctx, id)
    log.WithField("agent", res).Debug("Received result.")
    return res, nil
}

// CORRECT — public: WebhookMessage for API response, with permission check
func (h *serviceHandler) AgentGet(ctx context.Context, a *amagent.Agent, agentID uuid.UUID) (*amagent.WebhookMessage, error) {
    tmp, err := h.agentGet(ctx, agentID)
    if a.ID != agentID && !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
        return nil, fmt.Errorf("user has no permission")
    }
    return tmp.ConvertWebhookMessage(), nil
}
```

### 10.3 Filters from Request Body

Filters are parsed from the request body, not URL query parameters:

```go
// CORRECT — filters in request body
type V1DataAgentsGet struct {
    PageSize  uint64                `json:"page_size"`
    PageToken string                `json:"page_token"`
    Filters   map[agent.Field]any   `json:"filters"`
}

// For detailed filter parsing patterns, see docs/common-workflows.md
```

### 10.4 OpenAPI Schema Sync

When modifying API-facing structs, update the OpenAPI schema to match `WebhookMessage` fields (not internal struct). See the [verification workflow](../CLAUDE.md#critical-before-committing-changes) for the required regeneration steps.

---

## 11. Event Publishing

### 11.1 PublishWebhookEvent

Use `notifyHandler.PublishWebhookEvent()` for both internal event and customer webhook:

```go
// CORRECT — fires both internal event + customer webhook
h.notifyHandler.PublishWebhookEvent(ctx, agent.CustomerID, agent.EventTypeAgentCreated, agent)

// CORRECT — internal event only (no customer webhook)
h.notifyHandler.PublishEvent(ctx, agent.EventTypeAgentStatusUpdated, agent)
```

### 11.2 Fire-and-Forget

Events are published asynchronously via goroutines. Do not wait for event delivery:

```go
// This is how PublishWebhookEvent works internally:
go h.PublishEvent(ctx, eventType, data)      // async
go h.PublishWebhook(ctx, customerID, eventType, data)  // async
```

### 11.3 Delayed Events

Use `EventPublishWithDelay` for events that should fire after a delay:

```go
// CORRECT — delayed event via RabbitMQ x-delayed-message plugin
h.notifyHandler.PublishDelayedEvent(ctx, eventType, data, delaySeconds)
```

### 11.4 Event Handler Return Values

Event handlers must return `nil` to acknowledge the message. Returning an error requeues the message:

```go
// CORRECT — return nil to acknowledge, even on handled errors
func (h *handler) EventCMCallHangup(ctx context.Context, call *cmcall.Call) error {
    if err := h.processHangup(ctx, call); err != nil {
        log.Errorf("Could not process hangup: %v", err)
        return nil  // Acknowledge — don't requeue on business logic errors
    }
    return nil
}
```

---

## 12. Configuration

### 12.1 Cobra + Viper + sync.Once

Every service uses the same configuration pattern:

```go
// CORRECT — internal/config/main.go
package config

var (
    globalConfig Config
    once         sync.Once
)

type Config struct {
    RabbitMQAddress         string
    DatabaseDSN             string
    RedisAddress            string
    RedisPassword           string
    RedisDatabase           int
    PrometheusEndpoint      string
    PrometheusListenAddress string
    // service-specific fields...
}

func Bootstrap(cmd *cobra.Command) error {
    initLog()
    return bindConfig(cmd)
}

func LoadGlobalConfig() {
    once.Do(func() {
        globalConfig = Config{
            DatabaseDSN: viper.GetString("database_dsn"),
            // ...
        }
    })
}

func Get() *Config { return &globalConfig }
```

### 12.2 Environment Variable Binding

Each config field maps to a CLI flag and an environment variable:

```go
// CORRECT
f := cmd.PersistentFlags()
f.String("database_dsn", "", "Database connection string")
viper.BindPFlag("database_dsn", f.Lookup("database_dsn"))
viper.BindEnv("database_dsn", "DATABASE_DSN")
```

### 12.3 Logging Initialization

All services set logrus to debug level with joonix formatter:

```go
// CORRECT
func initLog() {
    logrus.SetFormatter(joonix.NewFormatter())
    logrus.SetLevel(logrus.DebugLevel)
}
```

---

## 13. Testing

### 13.1 Table-Driven Tests

All tests use anonymous struct slices with `name` as the first field:

```go
// CORRECT
func Test_Create(t *testing.T) {
    tests := []struct {
        name string
        // inputs
        customerID uuid.UUID
        // mock returns
        responseAgent *agent.Agent
        // expected
        expectErr bool
    }{
        {
            name:          "normal",
            customerID:    uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
            responseAgent: &agent.Agent{...},
            expectErr:     false,
        },
        {
            name:      "limit reached",
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()

            mockDB := dbhandler.NewMockDBHandler(mc)
            h := &agentHandler{
                db: mockDB,
            }

            ctx := context.Background()

            // Set expectations
            mockDB.EXPECT().AgentCreate(ctx, gomock.Any()).Return(nil)

            // Execute
            res, err := h.Create(ctx, tt.customerID, ...)

            // Assert
            if tt.expectErr {
                if err == nil {
                    t.Errorf("Wrong match. expect: err, got: ok")
                }
                return
            }
            if err != nil {
                t.Errorf("Wrong match. expect: ok, got: %v", err)
            }
        })
    }
}
```

### 13.2 Mock Generation

Mocks are generated via `//go:generate` and live in the same package as the interface:

```go
// In main.go of any handler package:
//go:generate mockgen -package flowhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

// Generated file: mock_main.go (co-located, NOT in separate mocks/ directory)
```

Run `go generate ./...` to regenerate all mocks.

### 13.3 Test Structure Conventions

Tests instantiate the private struct directly, not via the public constructor:

```go
// CORRECT — direct struct instantiation for testing
h := &flowHandler{
    db:            mockDB,
    reqHandler:    mockReq,
    notifyHandler: mockNotify,
}

// WRONG — using constructor (hides dependencies)
h := NewFlowHandler(mockDB, mockReq, mockNotify)
```

### 13.4 UUID Values in Tests

Use hardcoded real-looking UUID strings:

```go
// CORRECT
customerID := uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071")
agentID := uuid.FromStringOrNil("841c5fa2-f0c2-11ee-834f-53b2b00ec88d")

// WRONG — using uuid.Nil or uuid.Must(uuid.NewV4())
customerID := uuid.Nil
agentID := uuid.Must(uuid.NewV4())  // non-deterministic
```

### 13.5 Assertion Pattern

Use `t.Errorf` with the consistent format "Wrong match":

```go
// CORRECT
if err != nil {
    t.Errorf("Wrong match. expect: ok, got: %v", err)
}
if !reflect.DeepEqual(res, tt.responseAgent) {
    t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAgent, res)
}

// WRONG — using t.Fatal (stops test early)
if err != nil {
    t.Fatalf("unexpected error: %v", err)
}
```

### 13.6 Test Function Naming

Use `Test_<MethodName>`:

```go
// CORRECT
func Test_Create(t *testing.T) { ... }
func Test_Delete(t *testing.T) { ... }
func Test_FlowGet(t *testing.T) { ... }

// WRONG
func TestCreate(t *testing.T) { ... }  // Missing underscore
func TestHandler_Create(t *testing.T) { ... }  // Extra prefix
```

### 13.7 gomock.Any() Usage

`gomock.Any()` is a MATCHER — it can only be used in `EXPECT()`, never in `Return()`:

```go
// CORRECT
mockDB.EXPECT().AgentGet(ctx, gomock.Any()).Return(tt.responseAgent, nil)

// WRONG — gomock.Any() in Return() causes runtime panic
mockDB.EXPECT().AgentGet(ctx, id).Return(gomock.Any(), nil)  // PANIC
```

---

## 14. Prometheus Metrics

### 14.1 Service Metrics Registration

Register metrics via `init()` in the `metricshandler` package:

```go
// CORRECT — pkg/metricshandler/main.go
var (
    CallCreateTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "call_manager",
            Name:      "call_create_total",
            Help:      "Total number of call create operations",
        },
        []string{"status"},
    )
)

func init() {
    prometheus.MustRegister(CallCreateTotal)
}
```

### 14.2 Avoid Name Collisions

The shared `requesthandler` auto-registers these metrics per service:
- `<namespace>_request_process_time`
- `<namespace>_event_publish_total`

**NEVER reuse these names** in service-level `metricshandler`:

```go
// WRONG — collides with requesthandler's event_publish_total → PANIC at startup
EventPublishTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: "agent_manager",
        Name:      "event_publish_total",  // Already registered!
    },
    []string{"type"},
)

// CORRECT — use unique name
ServiceEventTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: "agent_manager",
        Name:      "service_event_total",  // Unique
    },
    []string{"type"},
)
```

**Before adding metrics:** Check `bin-common-handler/pkg/requesthandler/main.go` `initPrometheus()` for existing names.

---

## 15. Security

### 15.1 XSS Prevention

Never inject user input into HTML via `fmt.Sprintf`:

```go
// WRONG — XSS vulnerability
html := fmt.Sprintf("<h1>Welcome %s</h1>", userInput)

// CORRECT — validate input format strictly first
if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(token) {
    return fmt.Errorf("invalid token format")
}
// Only use validated input in templates
```

### 15.2 Token Generation

Use `crypto/rand` for all token generation:

```go
// CORRECT
import "crypto/rand"

b := make([]byte, 32)
rand.Read(b)
token := hex.EncodeToString(b)  // 64 hex chars

// WRONG — predictable tokens
import "math/rand"
token := fmt.Sprintf("%d", rand.Int63())
```

### 15.3 Username Enumeration Prevention

Password-forgot endpoints always return 200 regardless of user existence:

```go
// CORRECT
func (h *serviceHandler) AuthPasswordForgot(ctx context.Context, email string) error {
    // Always return nil — don't leak whether user exists
    return nil
}
```

### 15.4 Guest Agent Protection

Check for the guest agent UUID in all mutation operations:

```go
// CORRECT — check before mutation
const guestAgentID = "d819c626-0284-4df8-99d6-d03e1c6fba88"

func (h *agentHandler) Delete(ctx context.Context, id uuid.UUID) error {
    if id.String() == guestAgentID {
        return errors.New("cannot delete guest agent")
    }
    // ...
}
```

### 15.5 Validation at System Boundaries

Validate at service entry points (API layer, RPC handlers). Trust internal code:

```go
// CORRECT — validate at boundary
func (h *listenHandler) processV1AgentsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    var req request.V1DataAgentsPost
    if err := json.Unmarshal(m.Data, &req); err != nil {
        return simpleResponse(400), nil  // Validate input here
    }
    // Internal handler trusts the parsed input
    res, err := h.agentHandler.Create(ctx, req.CustomerID, ...)
}
```

### 15.6 No Secrets in Code

Never commit secrets, API keys, or credentials:

```go
// WRONG
const apiKey = "sk-1234567890abcdef"

// CORRECT — use environment variables
apiKey := viper.GetString("api_key")
```
