# Pipecat Pod Ping Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `/v1/ping` per-pod RPC route to `bin-pipecat-manager`, a 1-second client method `PipecatV1Ping` in `bin-common-handler`, and a preflight ping in `bin-ai-manager` before the chat-message RPC at `pkg/aicallhandler/send.go:46` — cutting dead-pod stalls from 3s → 1s without DB schema changes.

**Architecture:** Process-level liveness ping routed through the existing per-pod RabbitMQ queue `bin-manager.pipecat-manager.request.<host_id>`. The ping is a normal `r.sendRequest()` call so it shares the existing per-target circuit breaker — five consecutive failures trip a 30s fast-fail window for free. Server returns `{host_id, timestamp}`; client verifies the echo. Old pods that 404 the route are treated as alive (rolling-deploy safe).

**Tech Stack:** Go 1.22, RabbitMQ via `amqp091-go`, `github.com/pkg/errors` v0.9.1 (Unwrap-aware), `go.uber.org/mock` for mockgen, logrus, Prometheus. Verification per service: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`.

**Reference:** [`docs/plans/2026-04-26-pipecat-pod-ping-design.md`](./2026-04-26-pipecat-pod-ping-design.md)

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design`
**Branch:** `NOJIRA-pipecat-pod-ping-design` (already created; design doc already committed at `cfa75eb52`)

---

## Branch sync precheck (run before starting Task 1)

**Step 0.1: Pull latest main into the worktree base for conflict detection**

Run from the worktree:
```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" || echo "no conflicts"
git log --oneline HEAD..origin/main | head -10
```
Expected: `no conflicts`. If conflicts, rebase onto `origin/main` and re-run; do not start implementation until clean.

---

## Task 1: Create `PingResult` model in bin-pipecat-manager

**Files:**
- Create: `bin-pipecat-manager/models/pipecatcall/ping.go`

**Step 1.1: Write the model file**

```go
package pipecatcall

import "time"

// PingResult is returned by the per-pod GET /v1/ping route.
// HostID is the responding pod's POD_IP, used by the caller to verify the
// queue is consumed by the expected pod (best-effort; does not detect Calico
// POD_IP recycle).
type PingResult struct {
	HostID    string    `json:"host_id"`
	Timestamp time.Time `json:"timestamp"`
}
```

**Step 1.2: Verify it compiles**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design/bin-pipecat-manager
go build ./models/pipecatcall/...
```
Expected: no output, exit 0.

---

## Task 2: Add `Ping` to `PipecatcallHandler` interface and implement it

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/main.go` (add to interface)
- Create: `bin-pipecat-manager/pkg/pipecatcallhandler/ping.go`
- Create: `bin-pipecat-manager/pkg/pipecatcallhandler/ping_test.go`

**Step 2.1: Write the failing test**

`bin-pipecat-manager/pkg/pipecatcallhandler/ping_test.go`:

```go
package pipecatcallhandler

import (
	"context"
	"testing"
	"time"
)

func Test_pipecatcallHandler_Ping(t *testing.T) {
	tests := []struct {
		name   string
		hostID string
	}{
		{"non-empty host id", "10.4.2.18"},
		{"empty host id (defensive)", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &pipecatcallHandler{hostID: tt.hostID}
			before := time.Now().UTC()

			res, err := h.Ping(context.Background())

			if err != nil {
				t.Fatalf("Ping returned error: %v", err)
			}
			if res == nil {
				t.Fatalf("Ping returned nil result")
			}
			if res.HostID != tt.hostID {
				t.Errorf("HostID = %q, want %q", res.HostID, tt.hostID)
			}
			if res.Timestamp.Before(before) {
				t.Errorf("Timestamp %v is before test start %v", res.Timestamp, before)
			}
			if time.Since(res.Timestamp) > 5*time.Second {
				t.Errorf("Timestamp %v is too old", res.Timestamp)
			}
		})
	}
}
```

**Step 2.2: Run the test — expect compile failure (Ping doesn't exist)**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design/bin-pipecat-manager
go test ./pkg/pipecatcallhandler/ -run Test_pipecatcallHandler_Ping -v
```
Expected: FAIL with "h.Ping undefined" or similar build error.

**Step 2.3: Add `Ping` to the interface**

In `bin-pipecat-manager/pkg/pipecatcallhandler/main.go`, inside the `PipecatcallHandler` interface block (after the `RunnerMemberSwitchedHandle` method, before the closing brace):

```go
	Ping(ctx context.Context) (*pipecatcall.PingResult, error)
```

**Step 2.4: Implement `Ping` in a new file**

`bin-pipecat-manager/pkg/pipecatcallhandler/ping.go`:

```go
package pipecatcallhandler

import (
	"context"
	"time"

	"monorepo/bin-pipecat-manager/models/pipecatcall"
)

// Ping returns this pod's identity for liveness verification by callers.
// Returning error keeps the door open for forward-compatible drain semantics
// (e.g., responding 503 during shutdown). v1 always returns nil.
func (h *pipecatcallHandler) Ping(ctx context.Context) (*pipecatcall.PingResult, error) {
	return &pipecatcall.PingResult{
		HostID:    h.hostID,
		Timestamp: time.Now().UTC(),
	}, nil
}
```

**Step 2.5: Run the test — expect pass**

```bash
go test ./pkg/pipecatcallhandler/ -run Test_pipecatcallHandler_Ping -v
```
Expected: PASS.

---

## Task 3: Regenerate the `pipecatcallhandler` mock

**Files:**
- Modify (regenerated): `bin-pipecat-manager/pkg/pipecatcallhandler/mock_main.go`

**Step 3.1: Regenerate the mock**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design/bin-pipecat-manager
go generate ./pkg/pipecatcallhandler/...
```
Expected: no output, exit 0.

**Step 3.2: Verify the regenerated mock compiles and existing tests still pass**

```bash
go test ./pkg/pipecatcallhandler/...
```
Expected: PASS for all package tests.

---

## Task 4: Add `/v1/ping` route to listenhandler

**Files:**
- Modify: `bin-pipecat-manager/pkg/listenhandler/main.go` (add regex + switch case)
- Create: `bin-pipecat-manager/pkg/listenhandler/v1_ping.go`
- Create: `bin-pipecat-manager/pkg/listenhandler/v1_ping_test.go`

**Step 4.1: Write the failing test**

`bin-pipecat-manager/pkg/listenhandler/v1_ping_test.go`:

```go
package listenhandler

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	mocksock "monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"
)

func Test_processV1PingGet(t *testing.T) {
	tests := []struct {
		name           string
		request        *sock.Request
		mockPing       *pipecatcall.PingResult
		expectedStatus int
	}{
		{
			name: "GET /v1/ping returns 200 with PingResult body",
			request: &sock.Request{
				URI:    "/v1/ping",
				Method: sock.RequestMethodGet,
			},
			mockPing:       &pipecatcall.PingResult{HostID: "10.4.2.18", Timestamp: time.Now().UTC()},
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockPCH := pipecatcallhandler.NewMockPipecatcallHandler(mc)
			mockPCH.EXPECT().Ping(gomock.Any()).Return(tt.mockPing, nil)

			h := &listenHandler{
				sockHandler:        mocksock.NewMockSockHandler(mc),
				pipecatcallHandler: mockPCH,
			}

			res, err := h.processV1PingGet(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != tt.expectedStatus {
				t.Errorf("StatusCode = %d, want %d", res.StatusCode, tt.expectedStatus)
			}
			if res.DataType != "application/json" {
				t.Errorf("DataType = %q, want application/json", res.DataType)
			}
			var got pipecatcall.PingResult
			if err := json.Unmarshal(res.Data, &got); err != nil {
				t.Fatalf("response body did not unmarshal as PingResult: %v", err)
			}
			if !reflect.DeepEqual(&got, tt.mockPing) {
				t.Errorf("body = %+v, want %+v", got, tt.mockPing)
			}
		})
	}
}
```

**Step 4.2: Run the test — expect compile failure (`processV1PingGet` undefined)**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design/bin-pipecat-manager
go test ./pkg/listenhandler/ -run Test_processV1PingGet -v
```
Expected: FAIL with "h.processV1PingGet undefined" or similar.

**Step 4.3: Add the route regex and switch case in `main.go`**

In `bin-pipecat-manager/pkg/listenhandler/main.go`:

(a) Add to the `var (...)` block that defines route regexes (after `regV1Messages`):

```go
	// ping
	regV1Ping = regexp.MustCompile(`/v1/ping$`)
```

(b) Add a switch case in `processRequest()` (place it after the `messages` block, before the `default:`):

```go
		////////////////////
		// ping
		////////////////////
		// GET /v1/ping
		case regV1Ping.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
			response, err = h.processV1PingGet(ctx, m)
			requestType = "/v1/ping"
```

**Step 4.4: Implement `processV1PingGet`**

`bin-pipecat-manager/pkg/listenhandler/v1_ping.go`:

```go
package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
)

// processV1PingGet handles GET /v1/ping by returning the pod's PingResult.
// This is the per-pod liveness probe used by ai-manager preflight.
func (h *listenHandler) processV1PingGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1PingGet",
		"request": m,
	})

	res, err := h.pipecatcallHandler.Ping(ctx)
	if err != nil {
		log.Errorf("Could not get ping result. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		log.Errorf("Could not marshal ping result. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
```

**Step 4.5: Run the test — expect pass**

```bash
go test ./pkg/listenhandler/ -run Test_processV1PingGet -v
```
Expected: PASS.

**Step 4.6: Run the full listenhandler package tests to verify no regressions**

```bash
go test ./pkg/listenhandler/...
```
Expected: PASS.

---

## Task 5: Verify and commit `bin-pipecat-manager` changes

**Step 5.1: Run the full verification workflow for bin-pipecat-manager**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design/bin-pipecat-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: all 5 steps succeed; lint clean. **Do not skip any step**, even if "trivial" — `go mod tidy` may update `go.sum` even when no new imports are added. Vendor directory is gitignored — do **not** commit it.

**Step 5.2: Stage and review the diff**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design
git status
git diff --stat bin-pipecat-manager/
git diff bin-pipecat-manager/
```
Expected files changed:
- `bin-pipecat-manager/models/pipecatcall/ping.go` (new)
- `bin-pipecat-manager/pkg/pipecatcallhandler/main.go` (interface +1 line)
- `bin-pipecat-manager/pkg/pipecatcallhandler/ping.go` (new)
- `bin-pipecat-manager/pkg/pipecatcallhandler/ping_test.go` (new)
- `bin-pipecat-manager/pkg/pipecatcallhandler/mock_main.go` (regenerated)
- `bin-pipecat-manager/pkg/listenhandler/main.go` (regex + switch case)
- `bin-pipecat-manager/pkg/listenhandler/v1_ping.go` (new)
- `bin-pipecat-manager/pkg/listenhandler/v1_ping_test.go` (new)
- `bin-pipecat-manager/go.mod`, `bin-pipecat-manager/go.sum` (possibly updated by `go mod tidy`)

**Step 5.3: Commit (checkpoint commit on the feature branch — final squash on merge)**

```bash
git add bin-pipecat-manager/
git status  # confirm no vendor/ files staged
git commit -m "$(cat <<'EOF'
NOJIRA-pipecat-pod-ping-design: add ping endpoint

- bin-pipecat-manager: Add Ping method to PipecatcallHandler interface returning PingResult{host_id, timestamp}
- bin-pipecat-manager: Add models/pipecatcall/ping.go with PingResult struct
- bin-pipecat-manager: Add listenhandler route GET /v1/ping wired to PipecatcallHandler.Ping
- bin-pipecat-manager: Regenerate pipecatcallhandler mock for new interface method
EOF
)"
```

---

## Task 6: Add `PipecatV1Ping` client method in bin-common-handler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (add to interface)
- Create: `bin-common-handler/pkg/requesthandler/pipecat_ping.go`

**Step 6.1: Add `PipecatV1Ping` to the `RequestHandler` interface**

In `bin-common-handler/pkg/requesthandler/main.go`, find the existing pipecat method block (around line 1061-1090; look for `PipecatV1MessageSend` and `PipecatV1PipecatcallTerminateWithDelay`) and add at the end of that block:

```go
	// pipecat-manager ping
	PipecatV1Ping(ctx context.Context, hostID string) error
```

**Step 6.2: Implement `PipecatV1Ping`**

`bin-common-handler/pkg/requesthandler/pipecat_ping.go`:

```go
package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	outline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

const requestTimeoutPipecatPing = 1000 // 1s — sub-second decisive liveness probe

// PipecatV1Ping issues a sub-second liveness probe against the per-pod queue
// for hostID. Returns nil if the pod responded with a matching host_id (the
// live case), or an error otherwise (timeout, circuit open, mismatched
// host_id from a queue-name collision, etc.).
//
// IMPORTANT: do not add status-code checks here. A 404 from an old pipecat
// pod that predates this route is a valid "alive" signal — the pod responded.
// The only "dead" signal is err != nil, including ctx.DeadlineExceeded and
// circuitbreakerhandler.ErrCircuitOpen.
func (r *requestHandler) PipecatV1Ping(ctx context.Context, hostID string) error {
	queueName := fmt.Sprintf("%s.%s", outline.QueueNamePipecatRequest, hostID)
	res, err := r.sendRequest(
		ctx,
		outline.QueueName(queueName),
		"/v1/ping",
		sock.RequestMethodGet,
		"pipecat/ping",
		requestTimeoutPipecatPing,
		0,
		ContentTypeNone,
		nil,
	)
	if err != nil {
		return err
	}

	// Best-effort host_id echo verification. Note: Calico POD_IP recycle gives
	// a matching IP, so this check does NOT cover that case (see design §4.6).
	// For old pods the body is empty (404 simpleResponse) → skip the check.
	if res != nil && res.StatusCode == 200 && len(res.Data) > 0 {
		var pr pmpipecatcall.PingResult
		if errParse := json.Unmarshal(res.Data, &pr); errParse == nil {
			if pr.HostID != "" && pr.HostID != hostID {
				return fmt.Errorf("ping host_id mismatch: requested %s, got %s", hostID, pr.HostID)
			}
		}
	}
	return nil
}
```

**Step 6.3: Verify it compiles**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design/bin-common-handler
go build ./pkg/requesthandler/...
```
Expected: exit 0.

---

## Task 7: Write `pipecat_ping_test.go` (table-driven)

**Files:**
- Create: `bin-common-handler/pkg/requesthandler/pipecat_ping_test.go`

**Step 7.1: Open `bin-common-handler/pkg/requesthandler/pipecat_message_test.go` for the test fixture pattern reference**

Read the file end-to-end. The new ping test follows the same structure: gomock controller, mock `sockhandler`, table cases.

**Step 7.2: Write the test file**

`bin-common-handler/pkg/requesthandler/pipecat_ping_test.go`:

```go
package requesthandler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

func Test_PipecatV1Ping(t *testing.T) {
	tests := []struct {
		name           string
		hostID         string
		expectQueue    string
		mockResponse   *sock.Response
		mockErr        error
		expectErr      bool
		expectErrSubstr string
	}{
		{
			name:        "alive pod returns matching host_id",
			hostID:      "10.4.2.18",
			expectQueue: "bin-manager.pipecat-manager.request.10.4.2.18",
			mockResponse: func() *sock.Response {
				body, _ := json.Marshal(pmpipecatcall.PingResult{HostID: "10.4.2.18", Timestamp: time.Now().UTC()})
				return &sock.Response{StatusCode: 200, DataType: "application/json", Data: body}
			}(),
			mockErr:   nil,
			expectErr: false,
		},
		{
			name:        "host_id echo mismatch returns error",
			hostID:      "10.4.2.18",
			expectQueue: "bin-manager.pipecat-manager.request.10.4.2.18",
			mockResponse: func() *sock.Response {
				body, _ := json.Marshal(pmpipecatcall.PingResult{HostID: "10.4.2.99", Timestamp: time.Now().UTC()})
				return &sock.Response{StatusCode: 200, DataType: "application/json", Data: body}
			}(),
			mockErr:         nil,
			expectErr:       true,
			expectErrSubstr: "host_id mismatch",
		},
		{
			name:        "old pod 404 with empty body treated as alive",
			hostID:      "10.4.2.18",
			expectQueue: "bin-manager.pipecat-manager.request.10.4.2.18",
			mockResponse: &sock.Response{StatusCode: 404},
			mockErr:     nil,
			expectErr:   false,
		},
		{
			name:        "200 with empty body treated as alive (no echo check possible)",
			hostID:      "10.4.2.18",
			expectQueue: "bin-manager.pipecat-manager.request.10.4.2.18",
			mockResponse: &sock.Response{StatusCode: 200},
			mockErr:     nil,
			expectErr:   false,
		},
		{
			name:            "timeout error from sendRequest is propagated",
			hostID:          "10.4.2.18",
			expectQueue:     "bin-manager.pipecat-manager.request.10.4.2.18",
			mockResponse:    nil,
			mockErr:         context.DeadlineExceeded,
			expectErr:       true,
			expectErrSubstr: "deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			r := &requestHandler{
				sock:      mockSock,
				publisher: commonoutline.ServiceNameAIManager,
			}
			initPrometheusOnce(t)

			mockSock.EXPECT().
				RequestPublish(gomock.Any(), tt.expectQueue, gomock.Any()).
				Return(tt.mockResponse, tt.mockErr)

			err := r.PipecatV1Ping(context.Background(), tt.hostID)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.expectErrSubstr != "" && !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), tt.expectErrSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.expectErrSubstr)
				}
			} else {
				if err != nil {
					t.Fatalf("expected nil, got: %v", err)
				}
			}
		})
	}
}
```

**Note:** `initPrometheusOnce(t)` is a placeholder — check whether existing tests in this package use a helper to avoid duplicate Prometheus registration. Search:
```bash
grep -n "initPrometheus\|prometheus.MustRegister" bin-common-handler/pkg/requesthandler/*_test.go bin-common-handler/pkg/requesthandler/main_test.go 2>/dev/null
```
If existing tests use `r := newRequestHandlerForTest(...)` or similar, mirror that. If they construct `requestHandler{}` directly, drop the `initPrometheusOnce(t)` call. The exact wiring depends on the package's test-setup convention; verify before pasting blindly.

**Step 7.3: Run the test — expect pass**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design/bin-common-handler
go test ./pkg/requesthandler/ -run Test_PipecatV1Ping -v
```
Expected: PASS for all 5 subtests.

---

## Task 8: Regenerate `requesthandler` mock

**Files:**
- Modify (regenerated): `bin-common-handler/pkg/requesthandler/mock_main.go`

**Step 8.1: Regenerate**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design/bin-common-handler
go generate ./pkg/requesthandler/...
```
Expected: exit 0.

**Step 8.2: Confirm `MockRequestHandler` now exposes `PipecatV1Ping`**

```bash
grep -n "PipecatV1Ping" pkg/requesthandler/mock_main.go | head -5
```
Expected: 4–6 matches (mocked method body + recorder + comment lines).

---

## Task 9: Verify and commit `bin-common-handler` changes

**Step 9.1: Full verification workflow**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design/bin-common-handler
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: all 5 steps succeed.

**Step 9.2: Stage and commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design
git status
git add bin-common-handler/
git commit -m "$(cat <<'EOF'
NOJIRA-pipecat-pod-ping-design: add PipecatV1Ping client method

- bin-common-handler: Add PipecatV1Ping(ctx, hostID) to RequestHandler interface with 1s timeout
- bin-common-handler: Add pipecat_ping.go implementation with best-effort host_id echo verification
- bin-common-handler: Add pipecat_ping_test.go covering alive, mismatch, old-pod 404, empty body, and timeout cases
- bin-common-handler: Regenerate requesthandler mock for new interface method
EOF
)"
```

---

## Task 10: Add `pingPipecatHost` helper in bin-ai-manager

**Files:**
- Create: `bin-ai-manager/pkg/aicallhandler/ping.go`

**Step 10.1: Verify pkg/errors v0.9.1 is pinned (Unwrap support)**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design/bin-ai-manager
grep "pkg/errors" go.mod
```
Expected: `github.com/pkg/errors v0.9.1`. If older, the `errors.Is` chain through `sendRequest`'s wrapping will not work; halt and bump dependency.

**Step 10.2: Implement the helper**

`bin-ai-manager/pkg/aicallhandler/ping.go`:

```go
package aicallhandler

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/circuitbreakerhandler"
)

// pingPipecatHost runs a ~1s preflight against hostID. Returns true if the
// pod is reachable and owns this host_id; false if the pod is unreachable
// (timeout) or the breaker is open. Broker/transport errors return false
// and are logged distinctly so an outage is not misclassified as pod death.
//
// Note: relies on errors.Is unwrapping through pkg/errors v0.9.0+ wrappers
// applied by sendRequest. Verified pkg/errors v0.9.1 is pinned in
// bin-common-handler/go.mod and bin-ai-manager/go.mod.
func (h *aicallHandler) pingPipecatHost(ctx context.Context, hostID string) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":    "pingPipecatHost",
		"host_id": hostID,
	})
	if hostID == "" {
		return false
	}
	cctx, cancel := context.WithTimeout(ctx, 1100*time.Millisecond)
	defer cancel()
	err := h.reqHandler.PipecatV1Ping(cctx, hostID)
	if err == nil {
		log.Debug("Pipecat host ping succeeded.")
		return true
	}
	switch {
	case errors.Is(err, circuitbreakerhandler.ErrCircuitOpen):
		log.Debug("Pipecat host ping skipped: circuit breaker open.")
	case errors.Is(err, context.DeadlineExceeded):
		log.Info("Pipecat host ping timed out; treating as dead.")
	default:
		log.Warnf("Pipecat ping failed with unexpected error; skipping per-pod RPC. err: %v", err)
	}
	return false
}
```

**Step 10.3: Verify it compiles**

```bash
go build ./pkg/aicallhandler/...
```
Expected: exit 0. (May fail because `bin-ai-manager`'s vendor doesn't yet have the new `PipecatV1Ping` method — Step 10.4 fixes that.)

**Step 10.4: Re-vendor to pick up the new common-handler method**

```bash
go mod tidy && go mod vendor
go build ./pkg/aicallhandler/...
```
Expected: exit 0.

---

## Task 11: Write `ping_test.go` for the helper

**Files:**
- Create: `bin-ai-manager/pkg/aicallhandler/ping_test.go`

**Step 11.1: Read an existing aicallhandler test for the fixture pattern**

```bash
head -80 /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design/bin-ai-manager/pkg/aicallhandler/send_test.go
```
Use the constructor pattern from the existing test (likely `newAIcallHandlerForTest(t, mc)` or a direct struct literal — match it).

**Step 11.2: Write the test**

`bin-ai-manager/pkg/aicallhandler/ping_test.go`:

```go
package aicallhandler

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/circuitbreakerhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
)

func Test_aicallHandler_pingPipecatHost(t *testing.T) {
	tests := []struct {
		name        string
		hostID      string
		mockErr     error
		expectCall  bool
		expectAlive bool
	}{
		{
			name:        "empty hostID returns false without RPC",
			hostID:      "",
			mockErr:     nil,
			expectCall:  false,
			expectAlive: false,
		},
		{
			name:        "alive pod returns true",
			hostID:      "10.4.2.18",
			mockErr:     nil,
			expectCall:  true,
			expectAlive: true,
		},
		{
			name:        "deadline exceeded returns false",
			hostID:      "10.4.2.18",
			mockErr:     errors.Wrap(context.DeadlineExceeded, "could not send the request"),
			expectCall:  true,
			expectAlive: false,
		},
		{
			name:        "circuit open returns false",
			hostID:      "10.4.2.18",
			mockErr:     errors.Wrap(circuitbreakerhandler.ErrCircuitOpen, "could not send the request"),
			expectCall:  true,
			expectAlive: false,
		},
		{
			name:        "unexpected error returns false",
			hostID:      "10.4.2.18",
			mockErr:     fmt.Errorf("broker connection refused"),
			expectCall:  true,
			expectAlive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &aicallHandler{reqHandler: mockReq}

			if tt.expectCall {
				mockReq.EXPECT().PipecatV1Ping(gomock.Any(), tt.hostID).Return(tt.mockErr)
			}

			got := h.pingPipecatHost(context.Background(), tt.hostID)
			if got != tt.expectAlive {
				t.Errorf("pingPipecatHost = %v, want %v", got, tt.expectAlive)
			}
		})
	}
}
```

**Note:** This file imports both stdlib `errors` and `github.com/pkg/errors`. The stdlib import is needed by `errors.Is` paths in the helper under test (already imported in `ping.go`). For the test, only `errors.Wrap` from pkg/errors is used. Resolve the alias collision by importing pkg/errors as `pkgerrors`:
```go
	"errors"

	pkgerrors "github.com/pkg/errors"
```
Then change `errors.Wrap(...)` to `pkgerrors.Wrap(...)` in the table cases. Update accordingly.

**Step 11.3: Run the test**

```bash
go test ./pkg/aicallhandler/ -run Test_aicallHandler_pingPipecatHost -v
```
Expected: PASS for all 5 subtests.

---

## Task 12: Gate `send.go:46` with the preflight ping

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/send.go` (around line 46)
- Modify: `bin-ai-manager/pkg/aicallhandler/send_test.go` (extend `SendReferenceTypeCall` cases)

**Step 12.1: Read the current `SendReferenceTypeCall` to confirm line numbers**

```bash
grep -n "PipecatV1MessageSend\|SendReferenceTypeCall\|PipecatV1PipecatcallGet" /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design/bin-ai-manager/pkg/aicallhandler/send.go | head -10
```
Confirm `PipecatV1MessageSend` is still on the line near 46 (line numbers may shift after the package gains `ping.go`).

**Step 12.2: Insert the preflight before `PipecatV1MessageSend`**

In `bin-ai-manager/pkg/aicallhandler/send.go`, find this block in `SendReferenceTypeCall`:

```go
	res, err := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the message. aicall_id: %s", c.ID)
	}
	log.WithField("message", res).Debugf("Created the message to the ai. aicall_id: %s, message_id: %s", c.ID, res.ID)

	tmp, err := h.reqHandler.PipecatV1MessageSend(ctx, pc.HostID, pc.ID, res.ID.String(), messageText, runImmediately, audioResponse)
```

Add a preflight check between the `log.Debugf` after message create and the `PipecatV1MessageSend` call:

```go
	res, err := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the message. aicall_id: %s", c.ID)
	}
	log.WithField("message", res).Debugf("Created the message to the ai. aicall_id: %s, message_id: %s", c.ID, res.ID)

	if !h.pingPipecatHost(ctx, pc.HostID) {
		return nil, errors.Errorf("pipecat pod for this aicall is no longer reachable. host_id: %s, pipecatcall_id: %s", pc.HostID, pc.ID)
	}

	tmp, err := h.reqHandler.PipecatV1MessageSend(ctx, pc.HostID, pc.ID, res.ID.String(), messageText, runImmediately, audioResponse)
```

**Step 12.3: Update existing `send_test.go` cases**

Find existing `SendReferenceTypeCall` tests in `send_test.go`. For every test case that goes through the success path, add an `EXPECT().PipecatV1Ping(...).Return(nil)` expectation **before** the existing `PipecatV1MessageSend` expectation:

```go
mockReq.EXPECT().PipecatV1Ping(gomock.Any(), tt.pipecatcall.HostID).Return(nil)
mockReq.EXPECT().PipecatV1MessageSend(gomock.Any(), tt.pipecatcall.HostID, tt.pipecatcall.ID, gomock.Any(), tt.messageText, tt.runImmediately, tt.audioResponse).Return(tt.expectMessage, nil)
```

If the existing tests use `.AnyTimes()` or wildcards that already accept any RPC, the failure mode is "method called but no expectation"; gomock will error. Add explicit `EXPECT().PipecatV1Ping(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()` to fixtures that need to remain permissive.

**Step 12.4: Add a new test case for the dead-pod path**

Add a new entry to the `SendReferenceTypeCall` table or a new `t.Run` covering:
- Setup: `pc.HostID = "10.4.2.18"`.
- Mock: `mockReq.EXPECT().PipecatV1Ping(gomock.Any(), "10.4.2.18").Return(context.DeadlineExceeded)`.
- Mock: assert `PipecatV1MessageSend` is **never** called by NOT registering an expectation; gomock will fail if it is called.
- Assert: `Send` returns non-nil error containing "no longer reachable".
- Assert: `messageHandler.Create` IS called (the message row is persisted before the preflight).

**Step 12.5: Run the test suite**

```bash
go test ./pkg/aicallhandler/...
```
Expected: PASS, including the new dead-pod case.

---

## Task 13: Verify and commit `bin-ai-manager` changes

**Step 13.1: Full verification workflow**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design/bin-ai-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: all 5 steps succeed.

**Step 13.2: Stage and commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design
git status
git add bin-ai-manager/
git commit -m "$(cat <<'EOF'
NOJIRA-pipecat-pod-ping-design: gate chat MessageSend with preflight ping

- bin-ai-manager: Add aicallhandler.pingPipecatHost helper distinguishing ErrCircuitOpen, DeadlineExceeded, and other errors
- bin-ai-manager: Gate PipecatV1MessageSend in SendReferenceTypeCall (send.go:46) with the preflight, returning a "no longer reachable" error on ping failure
- bin-ai-manager: Add ping_test.go covering alive, dead, circuit-open, broker-error, and empty-hostID cases
- bin-ai-manager: Update send_test.go fixtures and add a dead-pod regression case
EOF
)"
```

---

## Task 14: Conflict precheck against latest main

**Step 14.1: Re-run conflict check**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" || echo "no conflicts"
git log --oneline HEAD..origin/main | head
```
If conflicts exist: rebase on `origin/main`, resolve, and re-run the per-service verification workflows for any service whose files were touched by the merge before continuing.

---

## Task 15: Code review and fix loop

Per saved feedback memory: ALWAYS run code review after finishing work; fix HIGH+ severity issues before commit/PR.

**Step 15.1: Run a code review on the full diff**

Dispatch the `code-reviewer` agent (or run `/code-review` if available) against the diff range:
```bash
git diff origin/main...HEAD
```
Capture HIGH and CRITICAL findings.

**Step 15.2: Apply fixes for HIGH+ findings, re-verify each touched service, and amend or add commits.** Do not proceed to PR creation until reviewer reports no HIGH+ findings.

---

## Task 16: Push branch and open PR

**Step 16.1: Push the branch**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design
git push -u origin NOJIRA-pipecat-pod-ping-design
```

**Step 16.2: Create the PR**

PR title MUST exactly match the branch name. Body uses the project-prefix bullet format with no markdown headers, no test plan section, no AI attribution. Use `gh pr create`:

```bash
gh pr create --title "NOJIRA-pipecat-pod-ping-design" --body "$(cat <<'EOF'
Add a per-pod liveness preflight ping in bin-pipecat-manager, called from
bin-ai-manager before the chat-message RPC at pkg/aicallhandler/send.go to
skip the 3-second RabbitMQ timeout when the stored pipecatcall.HostID
points to a now-dead pod. v1 reuses the existing per-target circuit
breaker; no new caches, no DB schema change.

- docs: Add design doc 2026-04-26-pipecat-pod-ping-design.md and implementation plan 2026-04-26-pipecat-pod-ping-plan.md
- bin-pipecat-manager: Add Ping method to PipecatcallHandler interface returning PingResult{host_id, timestamp}
- bin-pipecat-manager: Add models/pipecatcall/ping.go with PingResult struct
- bin-pipecat-manager: Add listenhandler route GET /v1/ping wired to PipecatcallHandler.Ping
- bin-pipecat-manager: Regenerate pipecatcallhandler mock for new interface method
- bin-common-handler: Add PipecatV1Ping(ctx, hostID) to RequestHandler interface with 1s timeout
- bin-common-handler: Add pipecat_ping.go implementation with best-effort host_id echo verification
- bin-common-handler: Add pipecat_ping_test.go covering alive, mismatch, old-pod 404, empty body, and timeout cases
- bin-common-handler: Regenerate requesthandler mock for new interface method
- bin-ai-manager: Add aicallhandler.pingPipecatHost helper distinguishing ErrCircuitOpen, DeadlineExceeded, and other errors
- bin-ai-manager: Gate PipecatV1MessageSend in SendReferenceTypeCall with the preflight, returning a "no longer reachable" error on ping failure
- bin-ai-manager: Add ping_test.go covering alive, dead, circuit-open, broker-error, and empty-hostID cases
- bin-ai-manager: Update send_test.go fixtures and add a dead-pod regression case
EOF
)"
```

**Step 16.3: Wait for explicit user authorization before merging.** Per saved feedback memory and CLAUDE.md, NEVER merge without explicit user request, and when authorized use squash merge: `gh pr merge <pr-number> --squash --delete-branch`.

---

## Post-merge cleanup

```bash
cd /home/pchero/gitvoipbin/monorepo
git pull origin main
git worktree remove /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-pipecat-pod-ping-design
```

## Rollout deploy order (operations team)

Per design doc §5: deploy `bin-pipecat-manager` to 100% first, then `bin-ai-manager`. Old pipecat pods 404 the route → safely treated as alive.
