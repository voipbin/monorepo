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
