# AI Call Terminate API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire up `POST /aicalls/{id}/terminate` in `bin-api-manager` so customers can terminate an in-progress AI call via the REST API.

**Architecture:** The OpenAPI spec and RabbitMQ request handler already exist — only `bin-api-manager` needs changes. Work follows the standard two-level handler pattern: HTTP server handler (`server/aicalls.go`) delegates to service handler (`pkg/servicehandler/aicall.go`), which authorises the request and forwards via `reqHandler.AIV1AIcallTerminate`. Permission model mirrors `AIcallDelete` exactly.

**Tech Stack:** Go 1.22, Gin, oapi-codegen, gomock, `bin-ai-manager` RabbitMQ RPC.

---

## File Map

| File | Action | Purpose |
|---|---|---|
| `bin-api-manager/gens/openapi_server/gen.go` | Regenerated (do not hand-edit) | Adds `PostAicallsIdTerminate` to `ServerInterface` and registers the Gin route |
| `bin-api-manager/pkg/servicehandler/main.go` | Modify | Add `AIcallTerminate` to `ServiceHandler` interface |
| `bin-api-manager/pkg/servicehandler/mock_main.go` | Regenerated (do not hand-edit) | Adds mock method for `AIcallTerminate` |
| `bin-api-manager/pkg/servicehandler/aicall.go` | Modify | Implement `AIcallTerminate` |
| `bin-api-manager/pkg/servicehandler/aicall_test.go` | Modify | Add `Test_AIcallTerminate` |
| `bin-api-manager/server/aicalls.go` | Modify | Implement `PostAicallsIdTerminate` HTTP handler |
| `bin-api-manager/server/aicalls_test.go` | Modify | Add `Test_aicallsIDTerminate` |
| `bin-api-manager/docs/routing.md` | Modify | Add two missing route rows |
| `bin-api-manager/docsdev/source/ai_overview.rst` | Modify | Document the terminate endpoint |
| `bin-api-manager/docsdev/build/` | Regenerated | Rebuilt HTML docs (force-added to git) |

**No changes to:**
- `bin-common-handler` — `AIV1AIcallTerminate` and its mock already exist
- `bin-openapi-manager` — spec and route registration already exist on `main`
- `bin-ai-manager` — terminate logic already exists

---

## Task 1: Regenerate the server interface

The `POST /aicalls/{id}/terminate` route is in the OpenAPI spec but not yet in
`gen.go`. This step is **required before all other tasks** — without it, the
`ServerInterface` is missing the method and the package won't compile.

**Files:**
- Regenerates: `bin-api-manager/gens/openapi_server/gen.go`

- [ ] **Step 1.1: Regenerate bin-openapi-manager (bundle refresh)**

```bash
cd /path/to/monorepo/bin-openapi-manager
go generate ./...
go build ./...
```

Expected: no errors. If `go generate` produces no diff in `gens/models/gen.go`,
that is expected — the spec was already there.

- [ ] **Step 1.2: Regenerate bin-api-manager server interface**

```bash
cd /path/to/monorepo/bin-api-manager
go generate ./...
```

Expected: `gens/openapi_server/gen.go` is updated.

- [ ] **Step 1.3: Confirm the new method exists in gen.go**

```bash
grep "PostAicallsIdTerminate" gens/openapi_server/gen.go
```

Expected output (two matches):
```
PostAicallsIdTerminate(c *gin.Context, id openapi_types.UUID)
router.POST(options.BaseURL+"/aicalls/:id/terminate", wrapper.PostAicallsIdTerminate)
```

If the method is not there, the spec or its registration in `openapi.yaml` may not
be on the current branch — stop and investigate.

---

## Task 2: Write the failing service handler test

Write the test before the implementation so it compiles-fails first, then passes
after implementation. The `AIcallTerminate` interface method does not yet exist, so
the test will fail at compile time — that is the expected red state.

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/aicall_test.go`

- [ ] **Step 2.1: Add Test_AIcallTerminate to `aicall_test.go`**

Append after `Test_AIcallDelete`:

```go
func Test_AIcallTerminate(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	aicallID   := uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a")

	aicall := &amaicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
			CustomerID: customerID,
		},
	}
	wantRes := &amaicall.WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
			CustomerID: customerID,
		},
	}

	tests := []struct {
		name    string
		agent   *auth.AuthIdentity
		wantErr bool

		setupMocks func(mockReq *requesthandler.MockRequestHandler)
	}{
		{
			name: "normal - agent with CustomerAdmin",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			wantErr: false,
			setupMocks: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(aicall, nil)
				mockReq.EXPECT().AIV1AIcallTerminate(gomock.Any(), aicallID).Return(aicall, nil)
			},
		},
		{
			name: "aicallGet failure",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			wantErr: true,
			setupMocks: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(nil, fmt.Errorf("not found"))
			},
		},
		{
			name: "agent permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerObserver, // insufficient
			}),
			wantErr: true,
			setupMocks: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(aicall, nil)
			},
		},
		{
			name: "direct token - resource type not allowed",
			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID:           customerID,
				AllowedResourceTypes: []string{"call"}, // "aicall" not in list
			}),
			wantErr: true,
			setupMocks: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(aicall, nil)
			},
		},
		{
			name: "direct token - customer ID mismatch",
			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID:           uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000000"), // different
				AllowedResourceTypes: []string{"aicall"},
			}),
			wantErr: true,
			setupMocks: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(aicall, nil)
			},
		},
		{
			name: "RPC failure",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			wantErr: true,
			setupMocks: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(aicall, nil)
				mockReq.EXPECT().AIV1AIcallTerminate(gomock.Any(), aicallID).Return(nil, fmt.Errorf("rpc error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			tt.setupMocks(mockReq)

			res, err := h.AIcallTerminate(ctx, tt.agent, aicallID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, wantRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", wantRes, res)
			}
		})
	}
}
```

- [ ] **Step 2.2: Confirm test fails to compile (red)**

```bash
cd /path/to/monorepo/bin-api-manager
go test ./pkg/servicehandler/... 2>&1 | head -20
```

Expected: compile error — `h.AIcallTerminate undefined` (or similar).

---

## Task 3: Add the interface method and regenerate mock

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go`
- Regenerates: `bin-api-manager/pkg/servicehandler/mock_main.go`

- [ ] **Step 3.1: Add `AIcallTerminate` to the `ServiceHandler` interface**

In `pkg/servicehandler/main.go`, find the comment `// aicall participant handlers`
(around line 320) and insert **before** it:

```go
	AIcallTerminate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error)
```

The block should now read:

```go
	AIcallDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error)
	AIcallTerminate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error)

	// aicall participant handlers
```

- [ ] **Step 3.2: Regenerate mock**

```bash
cd /path/to/monorepo/bin-api-manager
go generate ./...
```

Expected: `pkg/servicehandler/mock_main.go` gains an `AIcallTerminate` mock method.

- [ ] **Step 3.3: Confirm mock method exists**

```bash
grep "AIcallTerminate" pkg/servicehandler/mock_main.go
```

Expected: at least two lines (the `EXPECT().AIcallTerminate` registration).

---

## Task 4: Implement `AIcallTerminate` in the service handler

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/aicall.go`

- [ ] **Step 4.1: Append `AIcallTerminate` after `AIcallDelete`**

At the end of `pkg/servicehandler/aicall.go`, append:

```go
// AIcallTerminate terminates the aicall.
func (h *serviceHandler) AIcallTerminate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "AIcallTerminate",
		"aicall_id": id,
	})
	log.Debugf("Terminating aicall.")

	c, err := h.aicallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get aicall info")
	}

	switch {
	case a.IsAgent() || a.IsAccesskey():
		if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, serviceerrors.ErrPermissionDenied
		}
	case a.IsDirect():
		if !a.HasAllowedResourceType("aicall") {
			return nil, fmt.Errorf("%w: direct token does not allow this resource type", serviceerrors.ErrPermissionDenied)
		}
		if c.CustomerID != a.CustomerID {
			return nil, fmt.Errorf("%w: resource not in token scope", serviceerrors.ErrPermissionDenied)
		}
	}

	tmp, err := h.reqHandler.AIV1AIcallTerminate(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not terminate the aicall")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
```

- [ ] **Step 4.2: Run the service handler tests — expect all to pass**

```bash
cd /path/to/monorepo/bin-api-manager
go test ./pkg/servicehandler/... -run Test_AIcallTerminate -v
```

Expected: `--- PASS: Test_AIcallTerminate` with all 6 subtests passing.

- [ ] **Step 4.3: Run the full servicehandler package to check for regressions**

```bash
go test ./pkg/servicehandler/... -v 2>&1 | tail -20
```

Expected: `ok  	monorepo/bin-api-manager/pkg/servicehandler`

---

## Task 5: Write the failing server handler test

**Files:**
- Modify: `bin-api-manager/server/aicalls_test.go`

- [ ] **Step 5.1: Add `Test_aicallsIDTerminate` to `server/aicalls_test.go`**

Append after `Test_aicallsIDDelete_InvalidID`:

```go
func Test_aicallsIDTerminate(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		// happy path
		responseAIcall *amaicall.WebhookMessage
		expectAIcallID uuid.UUID
		expectRes      string

		// error path
		svcErr  error
		wantCode int
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/aicalls/c1a95988-5382-4769-98a9-b404823a64bf/terminate",

			responseAIcall: &amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c1a95988-5382-4769-98a9-b404823a64bf"),
				},
			},
			expectAIcallID: uuid.FromStringOrNil("c1a95988-5382-4769-98a9-b404823a64bf"),
			expectRes:      `{"id":"c1a95988-5382-4769-98a9-b404823a64bf","customer_id":"00000000-0000-0000-0000-000000000000","assistance_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","current_member_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
			wantCode: http.StatusOK,
		},
		{
			name: "permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/aicalls/c1a95988-5382-4769-98a9-b404823a64bf/terminate",

			expectAIcallID: uuid.FromStringOrNil("c1a95988-5382-4769-98a9-b404823a64bf"),
			svcErr:         serviceerrors.ErrPermissionDenied,
			wantCode:       http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest(http.MethodPost, tt.reqQuery, nil)

			if tt.svcErr != nil {
				mockSvc.EXPECT().AIcallTerminate(req.Context(), tt.agent, tt.expectAIcallID).Return(nil, tt.svcErr)
			} else {
				mockSvc.EXPECT().AIcallTerminate(req.Context(), tt.agent, tt.expectAIcallID).Return(tt.responseAIcall, nil)
			}

			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Wrong status. expect: %d, got: %d", tt.wantCode, w.Code)
			}

			if tt.wantCode == http.StatusOK && w.Body.String() != tt.expectRes {
				t.Errorf("Wrong body.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}
```

Add the missing import to the `import` block at the top of `aicalls_test.go`:

```go
"monorepo/bin-api-manager/pkg/serviceerrors"
```

- [ ] **Step 5.2: Confirm test fails to compile (red)**

```bash
cd /path/to/monorepo/bin-api-manager
go test ./server/... 2>&1 | head -20
```

Expected: compile error — `h.PostAicallsIdTerminate undefined` or
`AIcallTerminate` not in `MockServiceHandler` interface. Both are expected —
the implementation is not written yet.

---

## Task 6: Implement `PostAicallsIdTerminate` in the server

**Files:**
- Modify: `bin-api-manager/server/aicalls.go`

- [ ] **Step 6.1: Append `PostAicallsIdTerminate` handler at the end of `server/aicalls.go`**

```go
func (h *server) PostAicallsIdTerminate(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAicallsIdTerminate",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{"auth": a})

	target := uuid.UUID(id)
	res, err := h.serviceHandler.AIcallTerminate(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not terminate the aicall. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
```

Note: `openapi_types.UUID` is already imported in `server/aicalls.go`. No new
imports are needed.

- [ ] **Step 6.2: Run the server tests — expect all to pass**

```bash
cd /path/to/monorepo/bin-api-manager
go test ./server/... -run Test_aicallsIDTerminate -v
```

Expected: `--- PASS: Test_aicallsIDTerminate` with both subtests passing.

- [ ] **Step 6.3: Run the full server package to check for regressions**

```bash
go test ./server/... -v 2>&1 | tail -20
```

Expected: `ok  	monorepo/bin-api-manager/server`

---

## Task 7: Update docs

**Files:**
- Modify: `bin-api-manager/docs/routing.md`
- Modify: `bin-api-manager/docsdev/source/ai_overview.rst`
- Regenerates: `bin-api-manager/docsdev/build/`

- [ ] **Step 7.1: Update `docs/routing.md`**

First check which rows are already present:

```bash
grep "aicalls.*participants\|aicalls.*terminate" docs/routing.md
```

In the aicalls section (after `| DELETE | /aicalls/:id | ...`), add the rows
that are missing. The full set should be:

```
| GET    | `/aicalls/:id/participants` | bin-ai-manager | List AI call participants |
| POST   | `/aicalls/:id/terminate`   | bin-ai-manager | Terminate AI call session |
```

Only add rows that do not already exist.

- [ ] **Step 7.2: Add terminate section to `docsdev/source/ai_overview.rst`**

Find the "AI Participants" subsection (around line 801) in `ai_overview.rst` and
add a new subsection after it following the same structural pattern (subsection
heading, URL, path parameter description, response note, no query parameters
table):

```rst
Terminate AI Call
-----------------

Terminate an in-progress AI call session.

**Request:**

.. code-block::

   POST /aicalls/{id}/terminate

**Path parameter:**

- ``id`` — UUID of the AI call to terminate.

**Response:** Returns the terminated :ref:`AIcall <aicall_struct_aicall>` object.
```

- [ ] **Step 7.3: Rebuild RST docs**

```bash
cd /path/to/monorepo/bin-api-manager/docsdev
rm -rf build
python3 -m sphinx -M html source build
```

Expected: `Build succeeded.`

- [ ] **Step 7.4: Force-add the build output**

```bash
cd /path/to/monorepo/bin-api-manager
git add -f docsdev/build/
```

(The root `.gitignore` excludes `build/` so force-add is required.)

---

## Task 8: Full verification and commit

- [ ] **Step 8.1: Run the full verification workflow**

```bash
cd /path/to/monorepo/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all 5 steps pass with no errors. If lint fails, fix the issues before
committing.

- [ ] **Step 8.2: Verify bin-openapi-manager builds cleanly**

```bash
cd /path/to/monorepo/bin-openapi-manager
go build ./...
```

Expected: no errors.

- [ ] **Step 8.3: Stage all changed files**

```bash
cd /path/to/monorepo/bin-api-manager
git add \
  gens/openapi_server/gen.go \
  pkg/servicehandler/main.go \
  pkg/servicehandler/mock_main.go \
  pkg/servicehandler/aicall.go \
  pkg/servicehandler/aicall_test.go \
  server/aicalls.go \
  server/aicalls_test.go \
  docs/routing.md \
  docsdev/source/ai_overview.rst
git add -f docsdev/build/
```

If `bin-openapi-manager/gens/models/gen.go` has a diff, stage it too:

```bash
cd /path/to/monorepo
git diff --stat bin-openapi-manager/gens/
# If non-empty:
git add bin-openapi-manager/gens/
```

- [ ] **Step 8.4: Commit**

```bash
git commit -m "$(cat <<'EOF'
NOJIRA-Add-aicall-terminate-api

- bin-api-manager: Add PostAicallsIdTerminate HTTP handler
- bin-api-manager: Add AIcallTerminate service handler with agent/direct auth
- bin-api-manager: Add Test_AIcallTerminate (6 cases) and Test_aicallsIDTerminate (2 cases)
- bin-api-manager: Update routing.md and ai_overview.rst with terminate endpoint
EOF
)"
```

---

## Task 9: api-validator test

Add a safe integration test that confirms the endpoint is reachable without
triggering live state change.

**Files:**
- Modify or create: `~/gitvoipbin/monorepo-monitoring/api-validator/` (follow
  existing patterns in that directory for the aicall resource)

- [ ] **Step 9.1: Find the existing aicall api-validator test file**

```bash
ls ~/gitvoipbin/monorepo-monitoring/api-validator/ | grep -i aicall
```

- [ ] **Step 9.2: Add a terminate test case**

Add a test that calls `POST /aicalls/{non-existent-uuid}/terminate` with a valid
auth token and asserts the response contains `RESOURCE_NOT_FOUND` in the error
body (HTTP 404). Follow the existing test pattern in that directory.

Example pattern (adapt to the actual test framework used):

```go
// POST /aicalls/{non-existent-id}/terminate → 404 RESOURCE_NOT_FOUND
resp := doPost(t, "/aicalls/00000000-0000-0000-0000-000000000099/terminate", nil)
assertErrorCode(t, resp, "RESOURCE_NOT_FOUND")
```

- [ ] **Step 9.3: Run the api-validator to verify**

Follow the api-validator README for how to run tests locally.

---

## Post-Implementation Follow-up (separate PR)

File a ticket/issue to fix the `c.ClientIP` (missing parentheses) bug in the
older handlers in `server/aicalls.go` — `PostAicalls`, `GetAicalls`,
`GetAicallsId`, and `DeleteAicallsId` all log the method value (a hex pointer)
instead of the actual client IP string. The new `PostAicallsIdTerminate` handler
correctly uses `c.ClientIP()`. The existing bug does not block this PR.
