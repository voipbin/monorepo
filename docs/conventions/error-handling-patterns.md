# Error Handling Patterns

> **Quick Reference:** Standard error handling patterns used across all services in the monorepo.

## Overview

This document describes the three error handling layers in the monorepo:

1. **RabbitMQ RPC Layer** - Status codes in `sock.Response`
2. **Service Logic Layer** - Error wrapping and propagation
3. **API Gateway Layer** - HTTP responses in bin-api-manager

## Layer 1: RabbitMQ RPC Responses

All internal service communication uses `sock.Response` with HTTP-style status codes.

### Standard Status Codes

| Code | Meaning | When to Use |
|------|---------|-------------|
| 200 | OK | Successful operation |
| 201 | Created | Resource successfully created |
| 204 | No Content | Successful delete or update with no body |
| 400 | Bad Request | Invalid request data, validation failure |
| 401 | Unauthorized | Missing or invalid authentication |
| 403 | Forbidden | Valid auth but insufficient permissions |
| 404 | Not Found | Resource does not exist |
| 409 | Conflict | Resource already exists or state conflict |
| 500 | Internal Error | Unexpected server error |

### Response Pattern

```go
// bin-common-handler/models/sock/message.go
type Response struct {
    StatusCode int         `json:"status_code"`
    DataType   string      `json:"data_type"`
    Data       interface{} `json:"data"`
}

// Simple response helper (used in listenhandler)
func simpleResponse(code int) *sock.Response {
    return &sock.Response{
        StatusCode: code,
    }
}
```

### ListenHandler Examples

```go
// From bin-call-manager/pkg/listenhandler/

// 404 - Resource not found
func (h *listenHandler) processV1CallsID(m *sock.Request) (*sock.Response, error) {
    res, err := h.callHandler.Get(ctx, callID)
    if err != nil {
        return simpleResponse(404), nil  // Call not found
    }
    return &sock.Response{StatusCode: 200, Data: res}, nil
}

// 400 - Invalid request
func (h *listenHandler) processV1Calls(m *sock.Request) (*sock.Response, error) {
    var req call.CreateRequest
    if err := json.Unmarshal(m.Data, &req); err != nil {
        return simpleResponse(400), nil  // Bad request format
    }
    // ...
}

// 500 - Internal error
func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
    res, err := h.callHandler.Create(ctx, &req)
    if err != nil {
        log.Errorf("Failed to create call: %v", err)
        return simpleResponse(500), nil  // Internal error
    }
    return &sock.Response{StatusCode: 200, Data: res}, nil
}
```

## Layer 2: Service Logic Errors

Business logic handlers use Go's standard error handling with wrapping.

### Error Wrapping Pattern

Use `errors.Wrapf` from `github.com/pkg/errors` for context:

```go
import "github.com/pkg/errors"

func (h *callHandler) Get(ctx context.Context, id uuid.UUID) (*call.Call, error) {
    res, err := h.dbHandler.CallGet(ctx, id)
    if err != nil {
        return nil, errors.Wrapf(err, "could not get call. id: %s", id)
    }
    return res, nil
}
```

### Error Checking Patterns

```go
// Check for specific error conditions
if res == nil {
    return nil, fmt.Errorf("call not found. id: %s", id)
}

// Check response status from other services
resp, err := h.reqHandler.FlowV1FlowGet(ctx, flowID)
if err != nil {
    return nil, errors.Wrapf(err, "could not get flow info")
}
if resp.StatusCode != 200 {
    return nil, fmt.Errorf("wrong status from flow-manager. status: %d", resp.StatusCode)
}
```

### Logging Before Return

Always log errors with context before returning:

```go
func (h *handler) DoSomething(ctx context.Context, id uuid.UUID) error {
    log := logrus.WithFields(logrus.Fields{
        "func": "DoSomething",
        "id":   id,
    })

    res, err := h.dbHandler.Get(ctx, id)
    if err != nil {
        log.Errorf("Could not get resource: %v", err)
        return errors.Wrapf(err, "could not get resource")
    }

    return nil
}
```

## Layer 3: API Gateway Responses

The bin-api-manager translates internal responses to HTTP responses.

### Gin Framework Pattern

```go
// From bin-api-manager/lib/service/

// Success response
c.JSON(200, res.Data)

// Error response
c.AbortWithStatus(400)
c.AbortWithStatusJSON(404, gin.H{"error": "resource not found"})

// Authentication failure
c.AbortWithStatus(401)
```

### Response Translation

```go
func (h *handler) handleGet(c *gin.Context) {
    // Make RPC call
    resp, err := h.reqHandler.CallV1CallGet(ctx, callID)
    if err != nil {
        log.Errorf("RPC call failed: %v", err)
        c.AbortWithStatus(500)
        return
    }

    // Translate status code
    if resp.StatusCode != 200 {
        c.AbortWithStatus(resp.StatusCode)
        return
    }

    c.JSON(200, resp.Data)
}
```

## Common Error Scenarios

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

## Error Messages

### Do's

- Include relevant IDs: `"could not get call. id: %s"`
- Include status codes: `"wrong status. status: %d"`
- Use past tense for failures: `"could not create resource"`

### Don'ts

- Don't expose internal details to API clients
- Don't log sensitive data (passwords, tokens)
- Don't use panic for expected errors

## Database Error Handling

```go
// Check for no rows
res, err := h.db.Query(ctx, query)
if err == sql.ErrNoRows {
    return nil, nil  // Not found, not an error
}
if err != nil {
    return nil, errors.Wrapf(err, "database query failed")
}

// Soft delete check (active records have tm_delete = "9999-01-01...")
// Handled automatically by dbhandler WHERE clauses
```

## Event Handler Errors

For SubscribeHandler event processing, errors are logged but don't stop processing:

```go
func (h *subscribeHandler) processEvent(e *sock.Event) error {
    log := logrus.WithFields(logrus.Fields{
        "func":  "processEvent",
        "type":  e.Type,
    })

    if err := h.handleEvent(ctx, e); err != nil {
        log.Errorf("Failed to process event: %v", err)
        // Return nil to acknowledge message (don't requeue)
        return nil
    }

    return nil
}
```

## Testing Error Paths

```go
func Test_Get_NotFound(t *testing.T) {
    mc := gomock.NewController(t)
    defer mc.Finish()

    mockDB := NewMockDBHandler(mc)
    mockDB.EXPECT().Get(gomock.Any(), testID).Return(nil, sql.ErrNoRows)

    h := NewHandler(mockDB)
    res, err := h.Get(context.Background(), testID)

    assert.Nil(t, res)
    assert.Error(t, err)
}
```

## See Also

- [Code Quality Standards](code-quality-standards.md) - Logging patterns
- [Common Workflows](common-workflows.md) - Adding endpoints
- `bin-common-handler/models/sock/message.go` - Response structure
