# Testing

### 13.1 Table-Driven Tests

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

### 13.2 Mock Generation

Mocks are generated via `//go:generate` and live in the same package as the interface:

```go
// In main.go of any handler package:
//go:generate mockgen -package flowhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

// Generated file: mock_main.go (co-located, NOT in separate mocks/ directory)
```

Run `go generate ./...` to regenerate all mocks.

### 13.3 Test Structure Conventions

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

### 13.4 UUID Values in Tests

Use hardcoded real-looking UUID strings:

```go
// CORRECT
customerID := uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071")
agentID := uuid.FromStringOrNil("841c5fa2-f0c2-11ee-834f-53b2b00ec88d")

// WRONG — using uuid.Nil or uuid.Must(uuid.NewV4())
customerID := uuid.Nil
agentID := uuid.Must(uuid.NewV4())  // non-deterministic
```

### 13.5 Assertion Pattern

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

### 13.6 Test Function Naming

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

### 13.7 gomock.Any() Usage

`gomock.Any()` is a MATCHER — it can only be used in `EXPECT()`, never in `Return()`:

```go
// CORRECT
mockDB.EXPECT().AgentGet(ctx, gomock.Any()).Return(tt.responseAgent, nil)

// WRONG — gomock.Any() in Return() causes runtime panic
mockDB.EXPECT().AgentGet(ctx, id).Return(gomock.Any(), nil)  // PANIC
```

---
