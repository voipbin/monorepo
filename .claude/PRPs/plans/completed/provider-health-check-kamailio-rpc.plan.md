# Phase 3 Plan: bin-common-handler RPC — KamailioV1ProviderHealthCheck

**Phase**: 3 of 7
**Status**: in-progress
**PRD**: `.claude/PRPs/prds/provider-health-check.prd.md`
**Depends on**: Phase 2 (voip-kamailio-proxy service, complete)
**Parallel with**: Phase 4 (Provider health fields in bin-route-manager)

---

## Goal

Add `KamailioV1ProviderHealthCheck(ctx, hostname string) (*KamailioProviderHealthResult, error)` to `bin-common-handler/pkg/requesthandler/` so that `bin-route-manager` (Phase 5) can call the `voip-kamailio-proxy` RabbitMQ listener without knowing the queue name or wire format.

---

## Files to Create / Modify

| Action | File | Notes |
|--------|------|-------|
| CREATE | `bin-common-handler/pkg/requesthandler/kamailio_providers.go` | New RPC method + result type |
| CREATE | `bin-common-handler/pkg/requesthandler/kamailio_providers_test.go` | Unit test (table-driven, gomock) |
| MODIFY | `bin-common-handler/pkg/requesthandler/main.go` | Add `KamailioV1ProviderHealthCheck` to `RequestHandler` interface |
| MODIFY | `bin-common-handler/models/outline/queuename.go` | Add `QueueNameKamailioRequest` constant |

**No other files change in Phase 3.** The mock is regenerated automatically by `go generate ./...` (it reads `main.go` via the `//go:generate` directive at the top of `main.go`).

---

## Codebase Patterns (Verified by Direct Reading)

### Pattern: `sendRequestRTPEngine` in `send_request.go` (lines 322-332)

The closest pattern for kamailio is `sendRequestRTPEngine`, which also targets a named single queue (not per-instance dynamic queue like asterisk). The kamailio proxy uses a static queue `kamailio.request` — same style as `sendRequestFlow`, `sendRequestRoute`, etc.

```go
// send_request.go (existing, do NOT modify)
func (r *requestHandler) sendRequestRTPEngine(ctx context.Context, rtpengineID, uri string, ...) (*sock.Response, error) {
    target := fmt.Sprintf("rtpengine.%s.request", rtpengineID)
    return r.sendRequest(ctx, commonoutline.QueueName(target), uri, method, resource, timeout, delayed, dataType, data)
}
```

For kamailio-proxy the queue is static (`kamailio.request`), so it follows the simpler pattern (no ID interpolation):

```go
// Analogous to sendRequestRoute (send_request.go lines 237-240)
func (r *requestHandler) sendRequestKamailio(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout int, delayed int, dataType string, data []byte) (*sock.Response, error) {
    return r.sendRequest(ctx, commonoutline.QueueNameKamailioRequest, uri, method, resource, timeout, delayed, dataType, data)
}
```

### Pattern: `RTPEngineV1CommandsSend` in `rtpengine_commands.go` (full file, 45 lines)

This is the closest public method pattern — it marshals a struct, calls `sendRequest*`, then unmarshals the response:

```go
func (r *requestHandler) RTPEngineV1CommandsSend(ctx context.Context, rtpengineID string, command map[string]interface{}) (map[string]interface{}, error) {
    uri := "/v1/commands"
    m, err := json.Marshal(command)
    if err != nil {
        return nil, err
    }
    tmp, err := r.sendRequestRTPEngine(ctx, rtpengineID, uri, sock.RequestMethodPost, "rtpengine/commands", requestTimeoutDefault, 0, ContentTypeJSON, m)
    if err != nil {
        return nil, err
    }
    var res map[string]interface{}
    if errParse := parseResponse(tmp, &res); errParse != nil {
        return nil, errParse
    }
    return res, nil
}
```

### Pattern: Inline request/response structs in `ast_proxy.go` (lines 14-19)

```go
type Data struct {
    Filenames []string `json:"filenames,omitempty"`
}
```

Inline structs are used when the type is only used inside a single method. For `KamailioProviderHealthResult` the PRD specifies it is defined at package scope (not inline) so it is accessible to callers — but it lives in `kamailio_providers.go`, not in a separate models package.

### Pattern: Timeout constants in `main.go` (lines 129-130)

```go
const requestTimeoutDefault int = 3000  // 3 seconds
const requestTimeoutLong int = 30000    // 30 seconds
```

The PRD specifies 10 seconds for kamailio RPC. This is neither `requestTimeoutDefault` (3s) nor `requestTimeoutLong` (30s). A dedicated constant is needed.

### Pattern: Test structure from `ast_proxy_test.go`

```go
func Test_AstProxyRecordingFileMove(t *testing.T) {
    tests := []struct { ... }{ { ... } }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()

            mockSock := sockhandler.NewMockSockHandler(mc)
            reqHandler := requestHandler{sock: mockSock}

            mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
            err := reqHandler.AstProxyRecordingFileMove(...)
        })
    }
}
```

---

## Wire Format (Verified from `voip-kamailio-proxy`)

**Queue**: `kamailio.request` (single static queue, `QueueCreate` type `"normal"`)

**Source**: `voip-kamailio-proxy/pkg/listenhandler/request/providers.go`

```go
// Request body (JSON)
type V1DataProvidersHealthPost struct {
    Hostname string `json:"hostname"`
}

// Response body (JSON)
type V1ResponseProvidersHealthPost struct {
    Status     string `json:"status"`      // "healthy" | "unhealthy"
    ResultCode string `json:"result_code"` // SIP code e.g. "200", "404", or "timeout"
}
```

**Route matched by kamailio-proxy**: `POST /v1/providers/health`

---

## Implementation Tasks

### Task 1: Add queue name constant

**MIRROR**: `queuename.go` follows one constant-per-service pattern. All existing entries use the `bin-manager.<service>.request` prefix for manager services. Kamailio-proxy is a proxy service, not a bin-manager service — it uses the same simple dot-separated format as `asterisk.*` queues (no `bin-manager.` prefix).

**The init.go** in voip-kamailio-proxy sets the listen queue to `kamailio.request` (confirmed in `cmd/kamailio-proxy/init.go`).

**IMPORTS**: No new imports — pure constant addition.

**GOTCHA**: Do NOT use `bin-manager.kamailio-proxy.request` — that would be inconsistent with the Asterisk proxy convention and inconsistent with what the voip-kamailio-proxy service actually listens on.

Edit `bin-common-handler/models/outline/queuename.go` — add after the `// asterisk` block:

```go
// kamailio
QueueNameKamailioRequest QueueName = "kamailio.request"
```

**VALIDATE**: `grep -n "kamailio" bin-common-handler/models/outline/queuename.go` should show the new constant.

---

### Task 2: Add `sendRequestKamailio` helper in `send_request.go`

**MIRROR**: Identical structure to `sendRequestRoute` (lines 237-240 in `send_request.go`) and all other `sendRequest*` helpers.

**IMPORTS**: No new imports needed — uses existing `commonoutline` import already present.

**GOTCHA**: The `data` parameter type is `[]byte` (not `json.RawMessage`) — match the existing `sendRequestRoute` signature, which uses `[]byte`. Some other helpers use `json.RawMessage`. Both are assignable, but be consistent with the local convention for the kamailio method.

Add at the bottom of `send_request.go`:

```go
// sendRequestKamailio sends a request to the kamailio-proxy and returns the response.
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestKamailio(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout int, delayed int, dataType string, data []byte) (*sock.Response, error) {
	return r.sendRequest(ctx, commonoutline.QueueNameKamailioRequest, uri, method, resource, timeout, delayed, dataType, data)
}
```

**VALIDATE**: File compiles — verified by `go build ./...` as part of the full verification workflow.

---

### Task 3: Add timeout constant in `main.go`

**MIRROR**: `requestTimeoutDefault` and `requestTimeoutLong` are both package-level `const int` values in milliseconds.

**GOTCHA**: 10 seconds = 10,000 milliseconds. The PRD specifies 10s to exceed the voip-kamailio-proxy's 5s SIP UDP read deadline plus network overhead.

Add after `requestTimeoutLong` in `main.go` (around line 130):

```go
const requestTimeoutKamailio int = 10000 // kamailio RPC timeout(10 sec) — exceeds 5s SIP UDP read deadline
```

**VALIDATE**: `grep requestTimeoutKamailio bin-common-handler/pkg/requesthandler/main.go` shows the constant.

---

### Task 4: Create `kamailio_providers.go`

**MIRROR**: Follows `rtpengine_commands.go` exactly — marshal request, call `sendRequest*`, unmarshal response with `parseResponse`. The result type is at package scope per the PRD decision ("inline in kamailio_providers.go" means defined in the file, not buried inside the method).

**IMPORTS**:
- `"context"` — always needed
- `"encoding/json"` — for marshal/unmarshal
- `"monorepo/bin-common-handler/models/sock"` — for `sock.RequestMethodPost`

**GOTCHA**: `KamailioProviderHealthResult` field names must use `json:"..."` tags that match the wire response from `voip-kamailio-proxy`:
- `Status` → `json:"status"`
- `ResultCode` → `json:"result_code"`

If the tags are wrong, `parseResponse` will unmarshal successfully but leave both fields as empty strings — silent data loss.

```go
package requesthandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
)

// KamailioProviderHealthResult is the result of a provider health check via kamailio-proxy.
type KamailioProviderHealthResult struct {
	Status     string `json:"status"`      // "healthy" | "unhealthy"
	ResultCode string `json:"result_code"` // SIP response code e.g. "200", "404", or "timeout"
}

// KamailioV1ProviderHealthCheck sends a SIP OPTIONS health check request to voip-kamailio-proxy
// via RabbitMQ RPC and returns the result.
//
// The kamailio-proxy sends a SIP OPTIONS packet via Go UDP to <hostname>:5060 and returns:
//   - Status "healthy" if any SIP response is received
//   - Status "unhealthy" if no response within the 5s SIP timeout
//   - ResultCode: the SIP response code (e.g. "200", "404") or "timeout"
//
// RPC timeout is 10s — longer than the 5s SIP read deadline in voip-kamailio-proxy,
// accounting for network overhead between GKE and the Kamailio VM.
func (r *requestHandler) KamailioV1ProviderHealthCheck(ctx context.Context, hostname string) (*KamailioProviderHealthResult, error) {
	uri := "/v1/providers/health"

	type Data struct {
		Hostname string `json:"hostname"`
	}

	m, err := json.Marshal(Data{Hostname: hostname})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestKamailio(ctx, uri, sock.RequestMethodPost, "kamailio/providers/health", requestTimeoutKamailio, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res KamailioProviderHealthResult
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**VALIDATE**:
1. `grep -n "KamailioV1ProviderHealthCheck\|KamailioProviderHealthResult" bin-common-handler/pkg/requesthandler/kamailio_providers.go` shows both symbols.
2. `go build ./...` in `bin-common-handler` passes.

---

### Task 5: Add to `RequestHandler` interface in `main.go`

**MIRROR**: Interface entries are grouped by target service, with a `// <service>` comment header. Add a new `// kamailio` group following the existing pattern. The RTPEngine entry is a useful reference since it's also a proxy (not a manager service).

**GOTCHA**: The interface in `main.go` is used by `//go:generate mockgen` to regenerate `mock_main.go`. If you add the method to the implementation file but NOT to the interface, the mock will be stale and callers that depend on the interface (e.g., route-manager in Phase 5) will get a compile error when they try to call `KamailioV1ProviderHealthCheck` on the interface type.

Find the `// RTPEngine` block in `main.go` and add a new group after it:

```go
// kamailio-proxy
KamailioV1ProviderHealthCheck(ctx context.Context, hostname string) (*KamailioProviderHealthResult, error)
```

**VALIDATE**: `grep -n "KamailioV1ProviderHealthCheck" bin-common-handler/pkg/requesthandler/main.go` shows the entry.

---

### Task 6: Create `kamailio_providers_test.go`

**MIRROR**: Follows `ast_proxy_test.go` (table-driven, gomock, `requestHandler{sock: mockSock}`, `mockSock.EXPECT().RequestPublish(...)`).

**IMPORTS**:
- `"context"`
- `"testing"`
- `"monorepo/bin-common-handler/models/sock"`
- `"monorepo/bin-common-handler/pkg/sockhandler"`
- `"go.uber.org/mock/gomock"`

**GOTCHA**: The test verifies the exact queue name (`"kamailio.request"`), URI (`"/v1/providers/health"`), method (`sock.RequestMethodPost`), data type (`ContentTypeJSON`), and data payload (`{"hostname":"sip.example.com"}`). Any mismatch between the implementation and test will catch bugs in the wire format.

The test response data must use the correct JSON field names (`"status"` and `"result_code"`) matching the wire format, not the Go field names (`Status`, `ResultCode`).

```go
package requesthandler

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_KamailioV1ProviderHealthCheck(t *testing.T) {
	tests := []struct {
		name     string
		hostname string

		expectTarget  string
		expectRequest *sock.Request

		response      *sock.Response
		expectResult  *KamailioProviderHealthResult
		expectErrNil  bool
	}{
		{
			name:     "healthy provider",
			hostname: "sip.example.com",

			expectTarget: "kamailio.request",
			expectRequest: &sock.Request{
				URI:      "/v1/providers/health",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"hostname":"sip.example.com"}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"status":"healthy","result_code":"200"}`),
			},
			expectResult: &KamailioProviderHealthResult{
				Status:     "healthy",
				ResultCode: "200",
			},
			expectErrNil: true,
		},
		{
			name:     "unhealthy provider - timeout",
			hostname: "sip.unreachable.com",

			expectTarget: "kamailio.request",
			expectRequest: &sock.Request{
				URI:      "/v1/providers/health",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"hostname":"sip.unreachable.com"}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"status":"unhealthy","result_code":"timeout"}`),
			},
			expectResult: &KamailioProviderHealthResult{
				Status:     "unhealthy",
				ResultCode: "timeout",
			},
			expectErrNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.KamailioV1ProviderHealthCheck(context.Background(), tt.hostname)
			if tt.expectErrNil && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil {
				t.Fatal("Expected non-nil result")
			}
			resJSON, _ := json.Marshal(res)
			expectJSON, _ := json.Marshal(tt.expectResult)
			if string(resJSON) != string(expectJSON) {
				t.Errorf("Wrong match.\n  expect: %s\n  got:    %s", expectJSON, resJSON)
			}
		})
	}
}
```

**VALIDATE**: `go test ./pkg/requesthandler/... -run Test_KamailioV1ProviderHealthCheck -v` passes.

---

### Task 7: Regenerate mock

**MIRROR**: The `//go:generate` directive at the top of `main.go` regenerates `mock_main.go` from the interface. Always run after modifying the interface.

```bash
cd bin-common-handler && go generate ./...
```

This updates `mock_main.go` to include `KamailioV1ProviderHealthCheck` in the mock. Without this step, Phase 5 (route-manager) will get a compile error when it uses the mock in tests.

**VALIDATE**: `grep KamailioV1ProviderHealthCheck bin-common-handler/pkg/requesthandler/mock_main.go` shows the generated mock method.

---

## Full Verification Workflow

Run from the worktree root:

```bash
cd bin-common-handler && \
  go mod tidy && \
  go mod vendor && \
  go generate ./... && \
  go test ./... && \
  golangci-lint run -v --timeout 5m
```

All 5 steps must pass before committing.

**Expected test output**: `ok  monorepo/bin-common-handler/pkg/requesthandler` (no failures).

**Lint**: No new lint errors. The new file introduces no unused variables or missing error checks.

---

## Acceptance Criteria

- [ ] `KamailioProviderHealthResult` struct is defined at package scope in `kamailio_providers.go` with correct `json:"status"` and `json:"result_code"` tags
- [ ] `KamailioV1ProviderHealthCheck` is implemented on `*requestHandler` with RPC timeout `requestTimeoutKamailio` (10,000 ms)
- [ ] `KamailioV1ProviderHealthCheck` is in the `RequestHandler` interface in `main.go`
- [ ] `QueueNameKamailioRequest = "kamailio.request"` is in `queuename.go`
- [ ] `go generate ./...` regenerates `mock_main.go` with the new method
- [ ] `go test ./...` passes in `bin-common-handler` with the new test file covering healthy and unhealthy cases
- [ ] `golangci-lint run -v --timeout 5m` passes with no new errors
- [ ] Wire format matches: request JSON `{"hostname":"<value>"}` sent to `kamailio.request`, response JSON `{"status":"...","result_code":"..."}` unmarshalled into `KamailioProviderHealthResult`
- [ ] No changes outside `bin-common-handler/` in this phase

---

## Risk Notes

- **Mock staleness**: If `go generate ./...` is skipped, `mock_main.go` won't include the new method. Phase 5 (route-manager tests) will fail to compile. Mitigation: always run generate as step 3 in the verification workflow.
- **Queue name mismatch**: If the constant value doesn't match what `voip-kamailio-proxy` actually listens on (`kamailio.request` per `init.go`), RPC calls will time out silently. Mitigation: test verifies the exact target string `"kamailio.request"`.
- **JSON tag mismatch**: If `result_code` tag is written as `resultCode` or `result-code`, `parseResponse` will succeed but `ResultCode` will be empty. Mitigation: test asserts on the exact decoded value.

---

*Plan created: 2026-04-20*
