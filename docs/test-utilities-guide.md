# Test Utilities Guide

> **Quick Reference:** How to use testing utilities from `bin-common-handler` and write effective tests.

## Overview

The monorepo provides comprehensive testing utilities in `bin-common-handler`. This guide shows how to use them effectively to improve test coverage across services.

## Mock Generation

### Standard Pattern

Every handler interface uses mockgen:

```go
// In main.go
//go:generate mockgen -package packagename -destination ./mock_main.go -source main.go -build_flags=-mod=mod

type Handler interface {
    Get(ctx context.Context, id uuid.UUID) (*Model, error)
    Create(ctx context.Context, req *CreateRequest) (*Model, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

### Generating Mocks

```bash
# Generate all mocks in a service
cd bin-<service-name>
go generate ./...

# Generate mocks for specific package
go generate ./pkg/callhandler
```

### Using Mocks

```go
func TestGet(t *testing.T) {
    mc := gomock.NewController(t)
    defer mc.Finish()

    mockDB := NewMockDBHandler(mc)
    mockCache := NewMockCacheHandler(mc)

    // Set expectations
    mockDB.EXPECT().
        Get(gomock.Any(), testID).
        Return(expectedResult, nil)

    h := NewHandler(mockDB, mockCache)
    result, err := h.Get(context.Background(), testID)

    assert.NoError(t, err)
    assert.Equal(t, expectedResult, result)
}
```

## Table-Driven Tests

### Standard Pattern

```go
func Test_List(t *testing.T) {
    tests := []struct {
        name       string
        filters    map[string]any
        dbResponse []*Model
        dbError    error
        expectRes  []*Model
        expectErr  bool
    }{
        {
            name:       "empty filters returns all",
            filters:    map[string]any{},
            dbResponse: []*Model{{ID: uuid1}, {ID: uuid2}},
            expectRes:  []*Model{{ID: uuid1}, {ID: uuid2}},
        },
        {
            name:       "with customer_id filter",
            filters:    map[string]any{"customer_id": customerID},
            dbResponse: []*Model{{ID: uuid1}},
            expectRes:  []*Model{{ID: uuid1}},
        },
        {
            name:      "database error",
            filters:   map[string]any{},
            dbError:   errors.New("connection failed"),
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()

            mockDB := NewMockDBHandler(mc)
            mockDB.EXPECT().
                List(gomock.Any(), tt.filters).
                Return(tt.dbResponse, tt.dbError)

            h := NewHandler(mockDB)
            result, err := h.List(context.Background(), tt.filters)

            if tt.expectErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.expectRes, result)
        })
    }
}
```

## Database Handler Testing

### Using PrepareFields and ApplyFields

The `databasehandler` package in `bin-common-handler` provides utilities for dynamic field handling:

```go
// From bin-common-handler/pkg/databasehandler/mapping.go

// PrepareFields extracts non-zero fields from a struct for INSERT/UPDATE
fields, values := databasehandler.PrepareFields(model)

// ApplyFields adds WHERE conditions based on filters
query = databasehandler.ApplyFields(query, filters)
```

### Testing Database Operations

```go
func Test_DBHandler_Get(t *testing.T) {
    // Setup test database (or mock)
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    // Set expectations
    rows := sqlmock.NewRows([]string{"id", "name", "customer_id"}).
        AddRow(testID.Bytes(), "test", customerID.Bytes())

    mock.ExpectQuery("SELECT").
        WithArgs(testID.Bytes()).
        WillReturnRows(rows)

    h := NewDBHandler(db)
    result, err := h.Get(context.Background(), testID)

    assert.NoError(t, err)
    assert.Equal(t, "test", result.Name)
}
```

## RequestHandler Test Examples

The `bin-common-handler/pkg/requesthandler/` contains 40+ test files showing how to test RPC calls:

### Pattern from requesthandler Tests

```go
// From bin-common-handler/pkg/requesthandler/call_test.go
func Test_CallV1CallGet(t *testing.T) {
    tests := []struct {
        name         string
        callID       uuid.UUID
        mockResponse *sock.Response
        mockError    error
        expectRes    *call.Call
        expectErr    bool
    }{
        {
            name:   "success",
            callID: testCallID,
            mockResponse: &sock.Response{
                StatusCode: 200,
                Data:       expectedCall,
            },
            expectRes: expectedCall,
        },
        {
            name:   "not found",
            callID: testCallID,
            mockResponse: &sock.Response{
                StatusCode: 404,
            },
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()

            mockSock := sockhandler.NewMockSockHandler(mc)
            mockSock.EXPECT().
                RequestPublish(gomock.Any(), gomock.Any()).
                Return(tt.mockResponse, tt.mockError)

            h := NewRequestHandler(mockSock)
            result, err := h.CallV1CallGet(context.Background(), tt.callID)

            if tt.expectErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.expectRes, result)
        })
    }
}
```

## NotifyHandler Testing

### Testing Event Publishing

```go
func Test_PublishEvent(t *testing.T) {
    mc := gomock.NewController(t)
    defer mc.Finish()

    mockSock := sockhandler.NewMockSockHandler(mc)
    mockSock.EXPECT().
        Publish(
            gomock.Any(),                              // context
            string(outline.QueueNameCallEvent),        // queue
            gomock.Any(),                              // event data
        ).
        Return(nil)

    h := NewNotifyHandler(mockSock)
    err := h.PublishEvent(context.Background(), outline.QueueNameCallEvent, "call.created", callData)

    assert.NoError(t, err)
}
```

## UUID Testing Helpers

```go
// Create test UUIDs
var (
    testID       = uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001")
    testID2      = uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002")
    customerID   = uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440010")
)

// Or generate fresh UUIDs
func generateTestID() uuid.UUID {
    return uuid.Must(uuid.NewV4())
}
```

## Context Testing

Always pass context in tests:

```go
func TestWithContext(t *testing.T) {
    ctx := context.Background()

    // Or with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    result, err := h.Get(ctx, testID)
    // ...
}
```

## Common Test Assertions

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Basic assertions
assert.NoError(t, err)
assert.Error(t, err)
assert.Nil(t, result)
assert.NotNil(t, result)
assert.Equal(t, expected, actual)
assert.True(t, condition)

// Require stops test on failure (use for setup)
require.NoError(t, err)
```

## Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
go tool cover -func=coverage.out

# Run specific test
go test -v ./pkg/callhandler -run Test_Get

# Run tests matching pattern
go test -v ./... -run "Test_.*Create"

# Clear test cache (important after bin-common-handler changes)
go clean -testcache
go test ./...
```

## Test File Organization

```
pkg/callhandler/
├── main.go           # Interface definition + go:generate
├── mock_main.go      # Generated mocks
├── call.go           # Implementation
├── call_test.go      # Tests for call.go
├── db.go             # Database operations
└── db_test.go        # Tests for db.go
```

## Checklist for New Tests

- [ ] Use table-driven tests for multiple scenarios
- [ ] Test success path first
- [ ] Test error paths (not found, validation, db errors)
- [ ] Mock external dependencies
- [ ] Use meaningful test names
- [ ] Pass context.Background() to all handler calls
- [ ] Use gomock.NewController with defer mc.Finish()
- [ ] Test edge cases (nil, empty, max values)

## See Also

- [Code Quality Standards](code-quality-standards.md) - Testing patterns
- [Development Guide](development-guide.md) - Build and test commands
- `bin-common-handler/pkg/requesthandler/*_test.go` - 40+ test examples
- `bin-common-handler/pkg/databasehandler/mapping_test.go` - DB utility tests
