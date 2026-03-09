# Add RTPEngine Proxy RequestHandler Methods - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add RTPEngine proxy RPC methods to bin-common-handler requesthandler so call-manager can send NG protocol commands to specific rtpengine-proxy instances.

**Architecture:** Follow the asterisk-proxy pattern — dynamic queue names (`rtpengine.<rtpengineID>.request`) with a `sendRequestRTPEngine()` helper and a single generic `RTPEngineV1CommandsSend` method that accepts any NG command as `map[string]interface{}`.

**Tech Stack:** Go, RabbitMQ RPC, gomock

---

## Problem

The `voip-rtpengine-proxy` service exists and handles RTPEngine NG protocol commands via RabbitMQ RPC, but there are no typed methods in `bin-common-handler/pkg/requesthandler` for other services (specifically `bin-call-manager`) to call it.

## Design Decisions

- Dynamic queue name (like asterisk-proxy), not a constant
- Generic `map[string]interface{}` for request/response (NG protocol is dynamic)
- Single method `RTPEngineV1CommandsSend` matching the proxy's single `/v1/commands` POST endpoint

---

### Task 1: Add `sendRequestRTPEngine` helper

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/send_request.go` (append after `sendRequestAst` at line ~104)

**Step 1: Add the send helper**

Add after the existing `sendRequestAst` function:

```go
// sendRequestRTPEngine sends a request to the rtpengine-proxy and returns the response.
// The queue name is dynamically constructed from the rtpengineID: rtpengine.<rtpengineID>.request
// timeout millisecond
// delayed millisecond
func (r *requestHandler) sendRequestRTPEngine(ctx context.Context, rtpengineID, uri string, method sock.RequestMethod, resource string, timeout int, delayed int, dataType string, data []byte) (*sock.Response, error) {

	// create target
	target := fmt.Sprintf("rtpengine.%s.request", rtpengineID)

	return r.sendRequest(ctx, commonoutline.QueueName(target), uri, method, resource, timeout, delayed, dataType, data)
}
```

**Step 2: Verify it compiles**

Run: `cd bin-common-handler && go build ./...`
Expected: Success (no new imports needed — `fmt` and `commonoutline` already imported)

---

### Task 2: Add `RTPEngineV1CommandsSend` to the interface

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (add to `RequestHandler` interface, after the Ast* methods block around line ~312)

**Step 1: Add the interface method**

Add in the `RequestHandler` interface, grouped with a comment:

```go
	// RTPEngine
	RTPEngineV1CommandsSend(ctx context.Context, rtpengineID string, command map[string]interface{}) (map[string]interface{}, error)
```

**Step 2: Verify it compiles (expect failure — method not implemented yet)**

Run: `cd bin-common-handler && go build ./...`
Expected: Build error — `requestHandler` does not implement `RTPEngineV1CommandsSend`

---

### Task 3: Implement `RTPEngineV1CommandsSend`

**Files:**
- Create: `bin-common-handler/pkg/requesthandler/rtpengine_commands.go`

**Step 1: Create the implementation file**

```go
package requesthandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
)

// RTPEngineV1CommandsSend sends an RTPEngine NG protocol command to a specific rtpengine-proxy instance.
//
// The command is sent as POST /v1/commands to the rtpengine-proxy identified by rtpengineID.
// The rtpengineID determines the target queue: rtpengine.<rtpengineID>.request
//
// The only required field in command is "call-id". Fields like "from-tag", "to-tag",
// and "via-branch" are optional.
//
// Request examples:
//
//	{"command": "start recording", "call-id": "abc123@sip-server"}
//	{"command": "stop recording", "call-id": "abc123@sip-server"}
//
// Response examples:
//
//	Success: {"result": "ok"}
//	Error:   {"result": "error", "error-reason": "Unknown call-id"}
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

**Step 2: Verify it compiles**

Run: `cd bin-common-handler && go build ./...`
Expected: Success

---

### Task 4: Write the test

**Files:**
- Create: `bin-common-handler/pkg/requesthandler/rtpengine_commands_test.go`

**Step 1: Write the test file**

```go
package requesthandler

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_RTPEngineV1CommandsSend(t *testing.T) {

	tests := []struct {
		name        string
		rtpengineID string
		command     map[string]interface{}

		expectQueue  string
		expectURI    string
		expectMethod sock.RequestMethod

		response    *sock.Response
		expectError bool
	}{
		{
			"start recording",
			"10.164.0.12",
			map[string]interface{}{
				"command": "start recording",
				"call-id": "abc123@sip-server",
			},

			"rtpengine.10.164.0.12.request",
			"/v1/commands",
			sock.RequestMethodPost,

			&sock.Response{StatusCode: 200, Data: []byte(`{"result":"ok"}`)},
			false,
		},
		{
			"stop recording",
			"10.164.0.12",
			map[string]interface{}{
				"command": "stop recording",
				"call-id": "abc123@sip-server",
			},

			"rtpengine.10.164.0.12.request",
			"/v1/commands",
			sock.RequestMethodPost,

			&sock.Response{StatusCode: 200, Data: []byte(`{"result":"ok"}`)},
			false,
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

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				gomock.Any(),
			).Return(tt.response, nil)

			res, err := reqHandler.RTPEngineV1CommandsSend(context.Background(), tt.rtpengineID, tt.command)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res["result"] != "ok" {
				t.Errorf("Wrong result. expect: ok, got: %v", res["result"])
			}
		})
	}
}
```

**Step 2: Run test to verify it passes**

Run: `cd bin-common-handler && go test ./pkg/requesthandler/ -run Test_RTPEngineV1CommandsSend -v`
Expected: PASS

---

### Task 5: Regenerate mocks and run full verification

**Files:**
- Regenerated: `bin-common-handler/pkg/requesthandler/mock_main.go`

**Step 1: Run full verification workflow**

```bash
cd bin-common-handler && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass. `mock_main.go` will be updated with the new `RTPEngineV1CommandsSend` mock method.

---

### Task 6: Commit

**Step 1: Stage and commit**

```bash
git add bin-common-handler/pkg/requesthandler/send_request.go \
        bin-common-handler/pkg/requesthandler/main.go \
        bin-common-handler/pkg/requesthandler/rtpengine_commands.go \
        bin-common-handler/pkg/requesthandler/rtpengine_commands_test.go \
        bin-common-handler/pkg/requesthandler/mock_main.go \
        docs/plans/2026-03-09-add-rtpengine-requesthandler-methods-design.md

git commit -m "NOJIRA-Add-rtpengine-requesthandler-methods

Add RTPEngine proxy RPC methods to bin-common-handler requesthandler,
following the asterisk-proxy pattern with dynamic queue names.

- bin-common-handler: Add sendRequestRTPEngine() helper for dynamic queue routing
- bin-common-handler: Add RTPEngineV1CommandsSend() to RequestHandler interface
- bin-common-handler: Add implementation and tests for RTPEngine command sending
- docs: Add design document for rtpengine requesthandler integration"
```
