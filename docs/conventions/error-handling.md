# Error Handling

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

### 4.4 Error Propagation Pattern

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
