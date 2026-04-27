# Error Handling

## 4.1 Sentinel Errors

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

## 4.2 Error Wrapping

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

## 4.3 Checking Sentinel Errors

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

## 4.4 Error Propagation Pattern

Wrap and propagate errors up the call stack. Log at a reasonable level where you have meaningful context — not at every layer.

**Inner/mid-level functions** — wrap and return, do NOT log:

```go
// CORRECT — wrap and propagate
cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
if err != nil {
    return nil, errors.Wrapf(err, "could not get customer info")
}

// WRONG — log then return at every layer (produces duplicate log lines)
cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
if err != nil {
    log.Errorf("Could not get customer info: %v", err)  // Duplicate log
    return nil, errors.Wrapf(err, "could not get customer info")
}
```

**Reasonable-level functions** — log where you have meaningful context to act on the error:

```go
// CORRECT — log at a reasonable level where the error is handled
func (h *listenHandler) processV1CallCreate(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    res, err := h.callHandler.CreateCallOutgoing(ctx, ...)
    if err != nil {
        log.Errorf("Could not create outgoing call: %v", err)
        return simpleResponse(400), nil
    }
    return res, nil
}
```

**Always log data retrieval and significant state changes** (per [Logging §5.3](logging.md)):

```go
// CORRECT — log important data retrieval regardless of level
cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
if err != nil {
    return nil, errors.Wrapf(err, "could not get customer info")
}
log.WithField("customer", cu).Debugf("Retrieved customer info. customer_id: %s", cu.ID)
```

---

## Common Error Scenarios

These patterns recur across `listenhandler/` and `subscribehandler/` packages. They translate business-logic errors into the appropriate `sock.Response` status codes (see [rpc.md §9.4](rpc.md) for the full code table).

### Validation Errors (400)

```go
// Request validation
if req.Name == "" {
    return simpleResponse(400), nil
}

// UUID parsing
id, err := uuid.FromString(idStr)
if err != nil {
    return simpleResponse(400), nil
}
```

### Not Found (404)

```go
res, err := h.dbHandler.Get(ctx, id)
if err != nil || res == nil {
    return simpleResponse(404), nil
}
```

### Conflict (409)

```go
// Check for duplicate
existing, _ := h.dbHandler.GetByName(ctx, req.Name)
if existing != nil {
    return simpleResponse(409), nil
}
```

### Permission Denied (403)

```go
// Check ownership
if res.CustomerID != customerID {
    return simpleResponse(403), nil
}
```

### Internal Error (500)

```go
res, err := h.callHandler.Create(ctx, &req)
if err != nil {
    log.Errorf("Could not create call: %v", err)
    return simpleResponse(500), nil
}
```

## Database Error Handling

```go
// Check for no rows from raw queries
res, err := h.db.Query(ctx, query)
if err == sql.ErrNoRows {
    return nil, nil  // Not found, not an error
}
if err != nil {
    return nil, errors.Wrapf(err, "database query failed")
}

// dbhandler functions return ErrNotFound (sentinel) instead of sql.ErrNoRows;
// see §4.3 for the comparison pattern.
```

Soft-deleted rows (rows with `tm_delete IS NOT NULL`) are filtered out automatically by `dbhandler` `WHERE` clauses — callers do not need to check `tm_delete` themselves.

## Event Handler Errors

For `subscribehandler` event processing, errors are logged but should never stop processing — return `nil` to acknowledge the message (see also [events.md §11.4](events.md)):

```go
func (h *subscribeHandler) processEvent(e *sock.Event) error {
    log := logrus.WithFields(logrus.Fields{
        "func": "processEvent",
        "type": e.Type,
    })

    if err := h.handleEvent(ctx, e); err != nil {
        log.Errorf("Could not process event: %v", err)
        // Return nil to acknowledge message (don't requeue)
        return nil
    }

    return nil
}
```

## Error Message Style

**Do:**
- Include relevant IDs: `"could not get call. id: %s"`
- Include status codes: `"wrong status. status: %d"`
- Use past tense for failures: `"could not create resource"`
- Match the canonical log format: `"Could not <action>: %v"` (see [logging.md §5.4](logging.md))

**Don't:**
- Don't expose internal details to API clients (return `simpleResponse(<code>)` rather than echoing the wrapped error)
- Don't log sensitive data (passwords, tokens, raw request bodies that may contain credentials)
- Don't use `panic` for expected errors — wrap and return

## Testing Error Paths

Cover error branches with table-driven tests (see [testing.md §13.1](testing.md)):

```go
func Test_Get_NotFound(t *testing.T) {
    mc := gomock.NewController(t)
    defer mc.Finish()

    mockDB := NewMockDBHandler(mc)
    mockDB.EXPECT().Get(gomock.Any(), testID).Return(nil, dbhandler.ErrNotFound)

    h := &handler{db: mockDB}
    res, err := h.Get(context.Background(), testID)

    if res != nil {
        t.Errorf("Wrong match. expect: nil, got: %v", res)
    }
    if err == nil {
        t.Errorf("Wrong match. expect: err, got: ok")
    }
}
```
