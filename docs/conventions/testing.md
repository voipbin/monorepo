# Testing

## 13.1 Table-Driven Tests

All tests use anonymous struct slices with `name` as the first field:

```go
// CORRECT
func Test_Create(t *testing.T) {
    tests := []struct {
        name string
        // inputs
        customerID uuid.UUID
        // mock returns
        responseAgent *agent.Agent
        // expected
        expectErr bool
    }{
        {
            name:          "normal",
            customerID:    uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
            responseAgent: &agent.Agent{...},
            expectErr:     false,
        },
        {
            name:      "limit reached",
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()

            mockDB := dbhandler.NewMockDBHandler(mc)
            h := &agentHandler{
                db: mockDB,
            }

            ctx := context.Background()

            // Set expectations
            mockDB.EXPECT().AgentCreate(ctx, gomock.Any()).Return(nil)

            // Execute
            res, err := h.Create(ctx, tt.customerID, ...)

            // Assert
            if tt.expectErr {
                if err == nil {
                    t.Errorf("Wrong match. expect: err, got: ok")
                }
                return
            }
            if err != nil {
                t.Errorf("Wrong match. expect: ok, got: %v", err)
            }
        })
    }
}
```

## 13.2 Mock Generation

Mocks are generated via `//go:generate` and live in the same package as the interface:

```go
// In main.go of any handler package:
//go:generate mockgen -package flowhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

// Generated file: mock_main.go (co-located, NOT in separate mocks/ directory)
```

Run `go generate ./...` to regenerate all mocks.

## 13.3 Test Structure Conventions

Tests instantiate the private struct directly, not via the public constructor:

```go
// CORRECT — direct struct instantiation for testing
h := &flowHandler{
    db:            mockDB,
    reqHandler:    mockReq,
    notifyHandler: mockNotify,
}

// WRONG — using constructor (hides dependencies)
h := NewFlowHandler(mockDB, mockReq, mockNotify)
```

## 13.4 UUID Values in Tests

Use hardcoded real-looking UUID strings:

```go
// CORRECT
customerID := uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071")
agentID := uuid.FromStringOrNil("841c5fa2-f0c2-11ee-834f-53b2b00ec88d")

// WRONG — using uuid.Nil or uuid.Must(uuid.NewV4())
customerID := uuid.Nil
agentID := uuid.Must(uuid.NewV4())  // non-deterministic
```

## 13.5 Assertion Pattern

Use `t.Errorf` with the consistent format "Wrong match":

```go
// CORRECT
if err != nil {
    t.Errorf("Wrong match. expect: ok, got: %v", err)
}
if !reflect.DeepEqual(res, tt.responseAgent) {
    t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAgent, res)
}

// WRONG — using t.Fatal (stops test early)
if err != nil {
    t.Fatalf("unexpected error: %v", err)
}
```

## 13.6 Test Function Naming

Use `Test_<MethodName>`:

```go
// CORRECT
func Test_Create(t *testing.T) { ... }
func Test_Delete(t *testing.T) { ... }
func Test_FlowGet(t *testing.T) { ... }

// WRONG
func TestCreate(t *testing.T) { ... }  // Missing underscore
func TestHandler_Create(t *testing.T) { ... }  // Extra prefix
```

## 13.7 gomock.Any() Usage

`gomock.Any()` is a MATCHER — it can only be used in `EXPECT()`, never in `Return()`:

```go
// CORRECT
mockDB.EXPECT().AgentGet(ctx, gomock.Any()).Return(tt.responseAgent, nil)

// WRONG — gomock.Any() in Return() causes runtime panic
mockDB.EXPECT().AgentGet(ctx, id).Return(gomock.Any(), nil)  // PANIC
```

## 13.8 Never Use gomock.Any() For Parsed Request Payloads

`gomock.Any()` is the right matcher for opaque values you don't care about (a context, a randomly generated ID a previous call returned). It is the wrong matcher for **anything the test parsed out of an inbound RPC body**.

A listener handler typically does:

```go
var req request.V1DataXxxPut
if err := json.Unmarshal(m.Data, &req); err != nil { ... }
domainReq := &xxx.UpdateRequest{Name: req.Name, ...}
result, err := h.xxxHandler.Update(ctx, id, domainReq)
```

If the test asserts only `Update(gomock.Any(), tt.expectID, gomock.Any())`, it never validates that `m.Data` actually deserialized into the expected struct. A wire-format mismatch between the client and listener (e.g., a missing `request` wrapper, a renamed JSON tag) produces a zero-valued struct, and the test passes anyway. This is exactly the failure mode that produced the Listener Wire-Format Mismatch incident — see [`../workflows/common-gotchas.md`](../workflows/common-gotchas.md).

```go
// WRONG — third arg is the parsed body; gomock.Any() bypasses the assertion
mockOutboundConfig.EXPECT().
    Update(gomock.Any(), tt.expectID, gomock.Any()).
    Return(tt.responseConfig, nil)

// CORRECT — pass the expected struct; gomock falls back to reflect.DeepEqual
mockOutboundConfig.EXPECT().
    Update(gomock.Any(), tt.expectID, tt.expectReq).
    Return(tt.responseConfig, nil)
```

The `tt.expectReq` should be the **exact** `*UpdateRequest` (or equivalent) that the listener should construct from `m.Data`. Setting up `expectReq` per row in the table-driven test is the smallest unit that catches a wire-format regression.

When the value crosses pointer or slice boundaries, build it once with a helper closure so the test stays readable:

```go
expectReq: func() *outboundconfig.UpdateRequest {
    n := "updated"
    w := []string{"us", "kr"}
    return &outboundconfig.UpdateRequest{Name: &n, DestinationWhitelist: &w}
}(),
```

`gomock.Any()` is still appropriate for `ctx` and for genuinely opaque return values from the same mock.

## 13.9 RequestHandler Tests Must Assert The Marshaled Wire Shape

Every `bin-common-handler/pkg/requesthandler/<service>_<resource>.go` typed method that constructs a request body MUST have a paired test that:

1. Mocks `sockHandler.RequestPublish` with the **exact** `sock.Request` (URI, Method, DataType, **and `Data`** byte string).
2. Asserts the marshaled JSON shape — including which top-level keys must and must not be present.

```go
expectRequest: &sock.Request{
    URI:      "/v1/outbound_configs/c1234567-f7f5-11ef-92b3-0be9c3b04574",
    Method:   sock.RequestMethodPut,
    DataType: "application/json",
    Data:     []byte(`{"name":"my-config","detail":"detail-text","destination_whitelist":["us","kr"],"codecs":"PCMU,PCMA"}`),
},
// ...
mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
```

When the convention requires the body to be flat (no `request` wrapper), add an explicit guard so a future refactor that re-introduces the wrapper fails the test, not production:

```go
if strings.Contains(string(tt.expectRequest.Data), `"request":`) {
    t.Fatalf("expected flat wire format, but expectRequest.Data contains a `request` wrapper: %s", tt.expectRequest.Data)
}
```

Reference implementations: `bin-common-handler/pkg/requesthandler/call_calls_test.go`, `call_recordings_test.go`, `call_outbound_configs_test.go`.

---

## Test Utilities

The monorepo provides comprehensive testing utilities in `bin-common-handler` and per-service `dbhandler/`/`cachehandler/` mocks. This section is a quick reference for using them effectively.

### Regenerating Mocks

```bash
# Regenerate every mock in a service
cd bin-<service-name>
go generate ./...

# Regenerate mocks for a single package
go generate ./pkg/callhandler
```

After changing any handler interface, regenerate mocks before running tests — `go test` will fail with mismatched signatures otherwise.

### RequestHandler Test Pattern

Tests for code that calls other services via `requesthandler` mock the underlying `sockHandler.RequestPublish` rather than `requesthandler` itself, so the typed RPC method is exercised:

```go
// Reference: bin-common-handler/pkg/requesthandler/*_test.go (40+ examples)
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
            name:         "success",
            callID:       testCallID,
            mockResponse: &sock.Response{StatusCode: 200, Data: expectedCall},
            expectRes:    expectedCall,
        },
        {
            name:         "not found",
            callID:       testCallID,
            mockResponse: &sock.Response{StatusCode: 404},
            expectErr:    true,
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
            res, err := h.CallV1CallGet(context.Background(), tt.callID)

            if tt.expectErr {
                if err == nil {
                    t.Errorf("Wrong match. expect: err, got: ok")
                }
                return
            }
            if err != nil {
                t.Errorf("Wrong match. expect: ok, got: %v", err)
            }
            if !reflect.DeepEqual(tt.expectRes, res) {
                t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
            }
        })
    }
}
```

### NotifyHandler Test Pattern

Event publishing is tested by asserting on the underlying `sockHandler.Publish` call:

```go
mc := gomock.NewController(t)
defer mc.Finish()

mockSock := sockhandler.NewMockSockHandler(mc)
mockSock.EXPECT().
    Publish(
        gomock.Any(),                       // context
        string(outline.QueueNameCallEvent), // queue
        gomock.Any(),                       // event payload
    ).
    Return(nil)

h := NewNotifyHandler(mockSock)
err := h.PublishEvent(context.Background(), outline.QueueNameCallEvent, "call.created", callData)
if err != nil {
    t.Errorf("Wrong match. expect: ok, got: %v", err)
}
```

### Context with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

res, err := h.Get(ctx, testID)
```

### Test File Organization

Tests live alongside the code they cover; mocks live in the same package:

```
pkg/callhandler/
├── main.go           # Interface definition + go:generate
├── mock_main.go      # Generated mocks
├── call.go           # Implementation
├── call_test.go      # Tests for call.go
├── db.go             # Private DB-layer wrappers
└── db_test.go        # Tests for db.go
```

### Running Tests

```bash
# All tests
go test ./...

# Verbose
go test -v ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
go tool cover -func=coverage.out

# Specific test in a package
go test -v ./pkg/callhandler -run Test_Get

# Pattern match across the whole service
go test -v ./... -run "Test_.*Create"

# Clear test cache (important after bin-common-handler changes)
go clean -testcache
go test ./...
```

### Checklist for New Tests

- [ ] Use table-driven tests for multiple scenarios (§13.1)
- [ ] Test the success path first, then error paths
- [ ] Test not-found, validation, and DB-error branches
- [ ] Mock all external dependencies (DB, cache, request handler, notify handler)
- [ ] Use descriptive `name` values in each table row
- [ ] Pass `context.Background()` (or a derived context) to every handler call
- [ ] Always pair `gomock.NewController(t)` with `defer mc.Finish()`
- [ ] Cover edge cases (nil, empty slice, max values)
- [ ] Use hardcoded UUID strings for determinism (§13.4)
- [ ] Assert with `t.Errorf` and the "Wrong match" format (§13.5)
