# Database Patterns

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
