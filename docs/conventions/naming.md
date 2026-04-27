# Naming Conventions

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
