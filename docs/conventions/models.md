# Model Definitions

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
