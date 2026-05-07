# Database Patterns

## 7.0 Table Naming Convention

**MANDATORY:** Every MySQL table name must be prefixed with its owning service's domain abbreviation:

| Service | Prefix | Example |
|---------|--------|---------|
| `bin-call-manager` | `call_` | `call_calls`, `call_groupcalls`, `call_outbound_configs` |
| `bin-flow-manager` | `flow_` | `flow_flows`, `flow_activeflows` |
| `bin-customer-manager` | `customer_` | `customer_customers` |
| `bin-agent-manager` | `agent_` | `agent_agents` |
| `bin-billing-manager` | `billing_` | `billing_accounts`, `billing_billings` |
| `bin-conference-manager` | `conference_` | `conference_conferences` |
| `bin-campaign-manager` | `campaign_` | `campaign_campaigns` |
| `bin-number-manager` | `number_` | `number_numbers` |
| `bin-registrar-manager` | `registrar_` | `registrar_trunks` |

Format: `<domain>_<plural-entity>` — always lowercase, words separated by underscores.

```python
# CORRECT
CREATE TABLE call_outbound_configs (...)         # call-manager owns this table

# WRONG — missing service prefix
CREATE TABLE outbound_configs (...)

# WRONG — use short prefix, not full service name
CREATE TABLE call_manager_outbound_configs (...)
```

The Go constant in the matching `pkg/dbhandler/` file must use the full prefixed name:

```go
const outboundConfigTable = "call_outbound_configs"  // CORRECT
const outboundConfigTable = "outbound_configs"        // WRONG — missing prefix
```

When adding a new service, derive the prefix from the service name (e.g. `bin-rag-manager` → `rag_`) and add it to the table above.

**Exception — `bin-call-manager`:** This service uses direct SQL (not Squirrel) per its own `CLAUDE.md`. The §7.0 prefix rule still applies; the §7.1 Squirrel rule does not.

---

## 7.1 Squirrel Query Builder (Mandatory)

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

## 7.2 CRUD Operations

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

## 7.3 Empty Slice Initialization

**MANDATORY:** List functions must initialize result slices as empty, never nil:

```go
// CORRECT — empty slice
res := []*agent.Agent{}

// WRONG — nil slice serializes to null in JSON instead of []
var res []*agent.Agent
```

## 7.4 Cache-Aside Pattern

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

## 7.5 Cursor-Based Pagination

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

## 7.6 Filter Application

Use `commondatabasehandler.ApplyFields()` for type-safe filter maps:

```go
// CORRECT
sb, _ = commondatabasehandler.ApplyFields(sb, filters)
// Handles: uuid → bytes, "deleted: false" → tm_delete IS NULL, etc.
```

## 7.7 DB Operations Location

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

## Checklist

Use this checklist when adding or modifying database models and operations:

### Model Definition

- [ ] All `uuid.UUID` fields have `,uuid` db tag (see §6.2 in [models.md](models.md))
- [ ] Slice/map fields stored as JSON have `,json` db tag
- [ ] Soft-delete field present: `TMDelete *time.Time \`db:"tm_delete"\`` (nil = active)
- [ ] All persisted fields have appropriate `db:"column_name"` tags
- [ ] JSON tags match API expectations and the `WebhookMessage` variant

### Database Operations

- [ ] Table name uses the owning service's domain prefix (`call_`, `flow_`, `agent_`, …) (§7.0)
- [ ] Queries use the squirrel query builder, not raw SQL (§7.1)
- [ ] INSERT/UPDATE go through `commondatabasehandler.PrepareFields` (§7.2)
- [ ] SELECT uses `commondatabasehandler.GetDBFields` + `ScanRow` — never manual `rows.Scan` (§7.2)
- [ ] List/Gets functions initialize `res := []*Type{}` (empty, never nil) (§7.3)
- [ ] All DB code lives in `pkg/dbhandler/`; business handlers receive the `DBHandler` interface only (§7.7)

## UUID Tag Gotcha

`commondatabasehandler.PrepareFields()` and `ApplyFields()` use the `,uuid` flag on `db:` tags to convert `uuid.UUID` values to MySQL `BINARY(16)`. Without the flag, the UUID is sent as a string and never matches any row.

**Symptoms of a missing `,uuid` tag:**

- `GET` with filters returns `[]` even though data exists
- `POST` works but `GET` by ID fails
- No errors in logs — just empty results

```go
// CORRECT
type Call struct {
    ID         uuid.UUID `json:"id" db:"id,uuid"`
    CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`
    // ...
}

// WRONG — silent failures
type Call struct {
    ID         uuid.UUID `json:"id" db:"id"`           // BUG: queries will fail
    CustomerID uuid.UUID `json:"customer_id" db:"customer_id"` // BUG
}
```

## Transaction Pattern

When a multi-step write must be atomic, wrap it in a transaction:

```go
func (h *dbHandler) CreateWithRelated(ctx context.Context, model *Model) error {
    tx, err := h.db.BeginTx(ctx, nil)
    if err != nil {
        return errors.Wrap(err, "could not begin transaction")
    }
    defer tx.Rollback()

    // Insert main record
    if err := h.insertModel(ctx, tx, model); err != nil {
        return err
    }

    // Insert related records
    if err := h.insertRelated(ctx, tx, model.ID, model.Related); err != nil {
        return err
    }

    return tx.Commit()
}
```

## Debugging Database Issues

### Log the generated query

```go
sql, args, _ := query.ToSql()
log.WithFields(logrus.Fields{
    "sql":  sql,
    "args": args,
}).Debug("Executing query")
```

### Verify UUID conversion

```go
id := uuid.FromStringOrNil("...")
log.Debugf("UUID bytes: %x", id.Bytes())
```

### Confirm the soft-delete filter is applied

`commondatabasehandler.ApplyFields` adds `tm_delete IS NULL` automatically when the `deleted: false` filter sentinel is present — verify by inspecting the rendered SQL.

## Pagination Defaults

Most services define page-size constants and clamp incoming sizes:

```go
const (
    DefaultPageSize = 100
    MaxPageSize     = 1000
)

if size <= 0 || size > MaxPageSize {
    size = DefaultPageSize
}
```

The cursor token is the previous page's `tm_create` value (see §7.5).

## Legacy Soft-Delete Variant

A handful of older services still use a sentinel `tm_delete` string instead of a nullable timestamp:

```go
// Legacy convention — used in a small number of older services
const DefaultTimeStamp = "9999-01-01 00:00:00.000000"

query := sq.Select("*").
    From("call_calls").
    Where(sq.Eq{"tm_delete": databasehandler.DefaultTimeStamp})
```

New code should follow the canonical `*time.Time` / nil-as-active pattern documented in §6.3 and §7.2. The legacy sentinel pattern is documented here only because some `dbhandler/` packages still use it; do not introduce it in new tables.
