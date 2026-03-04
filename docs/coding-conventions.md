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
```
