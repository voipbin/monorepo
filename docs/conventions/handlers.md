# Handler Architecture

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
