# External Media Type Field Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `Type` field (`TypeNormal`, `TypeWebsocket`) to the external media model and thread it through the entire creation flow across 7 services.

**Architecture:** The `Type` is added to the `ExternalMedia` struct in `bin-call-manager/models/externalmedia/`, then threaded through all handler interfaces and function signatures in bin-call-manager, through the shared RPC client in bin-common-handler, and to all 4 cross-service callers. When empty, it defaults to `TypeNormal`.

**Tech Stack:** Go, RabbitMQ RPC (via bin-common-handler requesthandler), Redis cache, gomock

---

### Task 1: Add Type to externalmedia model

**Files:**
- Modify: `bin-call-manager/models/externalmedia/main.go:8-31` (struct), `main.go:71-77` (add after Status type)
- Modify: `bin-call-manager/models/externalmedia/field.go:17` (add after FieldStatus)
- Modify: `bin-call-manager/models/externalmedia/filters.go:14` (add after Status)

**Step 1: Add Type type, constants, and struct field to main.go**

After the `Status` type block (line 77), add:

```go
// Type define
type Type string

// list of Type types
const (
	TypeNormal    Type = "normal"
	TypeWebsocket Type = "websocket"
)
```

In the `ExternalMedia` struct (after line 9 `ID`), add:

```go
	Type Type `json:"type"` // type of the external media
```

**Step 2: Add FieldType to field.go**

After `FieldStatus` (line 17), add:

```go
	FieldType Field = "type" // type of the external media
```

**Step 3: Add Type to FieldStruct in filters.go**

After `Status` (line 14), add:

```go
	Type Type `filter:"type"`
```

**Step 4: Run model tests**

Run: `cd bin-call-manager && go test ./models/externalmedia/... -v`
Expected: existing tests pass (they don't yet test the new field — that's next task)

---

### Task 2: Add tests for Type in externalmedia model

**Files:**
- Modify: `bin-call-manager/models/externalmedia/externalmedia_test.go:13-31` (struct test), add new TestTypeConstants
- Modify: `bin-call-manager/models/externalmedia/field_test.go:12-33` (add FieldType to table)

**Step 1: Update TestExternalMediaStruct to include Type**

In `externalmedia_test.go`, add `Type: TypeNormal,` to the struct literal (after `ID:` line 14):

```go
	e := ExternalMedia{
		ID:              id,
		Type:            TypeNormal,
		AsteriskID:      "asterisk-1",
```

Add assertion after the ID check (after line 35):

```go
	if e.Type != TypeNormal {
		t.Errorf("ExternalMedia.Type = %v, expected %v", e.Type, TypeNormal)
	}
```

**Step 2: Add TestTypeConstants**

After `TestStatusConstants` (after line 182), add:

```go
func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_normal", TypeNormal, "normal"},
		{"type_websocket", TypeWebsocket, "websocket"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
```

**Step 3: Add FieldType to field_test.go**

In the test table (after `{"field_status", FieldStatus, "status"},` line 20), add:

```go
		{"field_type", FieldType, "type"},
```

**Step 4: Run tests**

Run: `cd bin-call-manager && go test ./models/externalmedia/... -v`
Expected: ALL tests pass including new TestTypeConstants

---

### Task 3: Add Type to request structs

**Files:**
- Modify: `bin-call-manager/pkg/listenhandler/models/request/externalmedias.go:12-24`
- Modify: `bin-call-manager/pkg/listenhandler/models/request/calls.go:77-86`
- Modify: `bin-call-manager/pkg/listenhandler/models/request/confbridge.go:24-31`

**Step 1: Add Type to V1DataExternalMediasPost**

In `externalmedias.go`, add after `ID` field (line 13):

```go
	Type          externalmedia.Type          `json:"type,omitempty"`
```

**Step 2: Add Type to V1DataCallsIDExternalMediaPost**

In `calls.go`, add after `ExternalMediaID` field (line 78):

```go
	Type            externalmedia.Type         `json:"type,omitempty"`
```

**Step 3: Add Type to V1DataConfbridgesIDExternalMediaPost**

In `confbridge.go`, add after `ExternalMediaID` field (line 25):

```go
	Type            externalmedia.Type `json:"type,omitempty"`
```

**Step 4: Verify build**

Run: `cd bin-call-manager && go build ./...`
Expected: builds successfully

---

### Task 4: Thread Type through externalmediahandler

**Files:**
- Modify: `bin-call-manager/pkg/externalmediahandler/main.go:30-42` (Start interface)
- Modify: `bin-call-manager/pkg/externalmediahandler/start.go:19-31` (Start impl), `start.go:59-69` (startReferenceTypeCall), `start.go:150-158` (startReferenceTypeConfbridge), `start.go:207-221` (startExternalMedia)
- Modify: `bin-call-manager/pkg/externalmediahandler/db.go:13-31` (Create)

**Step 1: Add typ param to Start() in main.go interface**

In `main.go`, add `typ externalmedia.Type` after `id uuid.UUID` (line 32):

```go
	Start(
		ctx context.Context,
		id uuid.UUID,
		typ externalmedia.Type,
		referenceType externalmedia.ReferenceType,
```

**Step 2: Add typ param to Start() implementation in start.go**

Update the `Start` function signature (line 19-31):

```go
func (h *externalMediaHandler) Start(
	ctx context.Context,
	id uuid.UUID,
	typ externalmedia.Type,
	referenceType externalmedia.ReferenceType,
```

Add default after the `id` nil check (after line 43):

```go
	if typ == "" {
		typ = externalmedia.TypeNormal
	}
```

Pass `typ` to both `startReferenceTypeCall` and `startReferenceTypeConfbridge`:

```go
	case externalmedia.ReferenceTypeCall:
		return h.startReferenceTypeCall(ctx, id, typ, referenceID, externalHost, encapsulation, transport, format, directionListen, directionSpeak)

	case externalmedia.ReferenceTypeConfbridge:
		return h.startReferenceTypeConfbridge(ctx, id, typ, referenceID, externalHost, encapsulation, transport, format)
```

**Step 3: Add typ param to startReferenceTypeCall**

Update signature (line 59):

```go
func (h *externalMediaHandler) startReferenceTypeCall(
	ctx context.Context,
	id uuid.UUID,
	typ externalmedia.Type,
	callID uuid.UUID,
```

Pass `typ` to `startExternalMedia` call (line 126):

```go
	res, err := h.startExternalMedia(
		ctx,
		id,
		typ,
		ch.AsteriskID,
```

**Step 4: Add typ param to startReferenceTypeConfbridge**

Update signature (line 150):

```go
func (h *externalMediaHandler) startReferenceTypeConfbridge(
	ctx context.Context,
	id uuid.UUID,
	typ externalmedia.Type,
	confbridgeID uuid.UUID,
```

Pass `typ` to `startExternalMedia` call (line 183):

```go
	res, err := h.startExternalMedia(
		ctx,
		id,
		typ,
		br.AsteriskID,
```

**Step 5: Add typ param to startExternalMedia**

Update signature (line 207):

```go
func (h *externalMediaHandler) startExternalMedia(
	ctx context.Context,
	id uuid.UUID,
	typ externalmedia.Type,
	asteriskID string,
```

Pass `typ` to `h.Create` call (line 268):

```go
	em, err := h.Create(
		ctx,
		id,
		typ,
		asteriskID,
```

**Step 6: Add typ param to Create in db.go**

Update signature (line 13):

```go
func (h *externalMediaHandler) Create(
	ctx context.Context,
	id uuid.UUID,
	typ externalmedia.Type,
	asteriskID string,
```

Set Type on the struct (after `ID: id,` line 36):

```go
	extMedia := &externalmedia.ExternalMedia{
		ID:   id,
		Type: typ,
```

**Step 7: Verify build**

Run: `cd bin-call-manager && go build ./...`
Expected: build errors from callers not yet updated (callhandler, confbridgehandler, listenhandler) — that's expected, we fix them next

---

### Task 5: Thread Type through callhandler and confbridgehandler

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/main.go:119-130` (ExternalMediaStart interface)
- Modify: `bin-call-manager/pkg/callhandler/external_media.go:23-34` (ExternalMediaStart impl)
- Modify: `bin-call-manager/pkg/confbridgehandler/main.go:72-81` (ExternalMediaStart interface)
- Modify: `bin-call-manager/pkg/confbridgehandler/external_media.go:17-26` (ExternalMediaStart impl)

**Step 1: Add typ param to CallHandler.ExternalMediaStart interface**

In `callhandler/main.go` (line 119):

```go
	ExternalMediaStart(
		ctx context.Context,
		id uuid.UUID,
		typ externalmedia.Type,
		externalMediaID uuid.UUID,
```

**Step 2: Add typ param to callHandler.ExternalMediaStart implementation**

In `callhandler/external_media.go` (line 23):

```go
func (h *callHandler) ExternalMediaStart(
	ctx context.Context,
	id uuid.UUID,
	typ externalmedia.Type,
	externalMediaID uuid.UUID,
```

Pass `typ` to `externalMediaHandler.Start` (line 53):

```go
	tmp, err := h.externalMediaHandler.Start(
		ctx,
		externalMediaID,
		typ,
		externalmedia.ReferenceTypeCall,
```

**Step 3: Add typ param to ConfbridgeHandler.ExternalMediaStart interface**

In `confbridgehandler/main.go` (line 72):

```go
	ExternalMediaStart(
		ctx context.Context,
		id uuid.UUID,
		typ externalmedia.Type,
		externalMediaID uuid.UUID,
```

**Step 4: Add typ param to confbridgeHandler.ExternalMediaStart implementation**

In `confbridgehandler/external_media.go` (line 17):

```go
func (h *confbridgeHandler) ExternalMediaStart(
	ctx context.Context,
	id uuid.UUID,
	typ externalmedia.Type,
	externalMediaID uuid.UUID,
```

Pass `typ` to `externalMediaHandler.Start` (line 45):

```go
	tmp, err := h.externalMediaHandler.Start(
		ctx,
		externalMediaID,
		typ,
		externalmedia.ReferenceTypeConfbridge,
```

**Step 5: Verify build**

Run: `cd bin-call-manager && go build ./...`
Expected: build errors from listenhandler callers — fixed next

---

### Task 6: Thread Type through listenhandler routes

**Files:**
- Modify: `bin-call-manager/pkg/listenhandler/v1_external_medias.go:85-97` (processV1ExternalMediasPost)
- Modify: `bin-call-manager/pkg/listenhandler/v1_calls.go:483-493` (processV1CallsIDExternalMediaPost)
- Modify: `bin-call-manager/pkg/listenhandler/v1_confbridges.go:225-234` (processV1ConfbridgesIDExternalMediaPost)

**Step 1: Pass Type in processV1ExternalMediasPost**

In `v1_external_medias.go` (line 85), add `req.Type` after `req.ID`:

```go
	tmp, err := h.externalMediaHandler.Start(
		ctx,
		req.ID,
		req.Type,
		req.ReferenceType,
```

**Step 2: Pass Type in processV1CallsIDExternalMediaPost**

In `v1_calls.go` (line 483), add `req.Type` after `id`:

```go
	tmp, err := h.callHandler.ExternalMediaStart(
		ctx,
		id,
		req.Type,
		req.ExternalMediaID,
```

**Step 3: Pass Type in processV1ConfbridgesIDExternalMediaPost**

In `v1_confbridges.go` (line 225), add `req.Type` after `id`:

```go
	tmp, err := h.confbridgeHandler.ExternalMediaStart(
		ctx,
		id,
		req.Type,
		req.ExternalMediaID,
```

**Step 4: Regenerate mocks for bin-call-manager**

Run: `cd bin-call-manager && go generate ./...`
Expected: mocks regenerated successfully

**Step 5: Verify build**

Run: `cd bin-call-manager && go build ./...`
Expected: builds successfully (all internal callers updated)

---

### Task 7: Thread Type through bin-common-handler requesthandler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go:522-534` (interface)
- Modify: `bin-common-handler/pkg/requesthandler/call_externalmedias.go:63-74` (implementation)

**Step 1: Add typ param to CallV1ExternalMediaStart interface**

In `main.go` (line 522):

```go
	CallV1ExternalMediaStart(
		ctx context.Context,
		externalMediaID uuid.UUID,
		typ cmexternalmedia.Type,
		referenceType cmexternalmedia.ReferenceType,
```

**Step 2: Add typ param to CallV1ExternalMediaStart implementation**

In `call_externalmedias.go` (line 63):

```go
func (r *requestHandler) CallV1ExternalMediaStart(
	ctx context.Context,
	externalMediaID uuid.UUID,
	typ cmexternalmedia.Type,
	referenceType cmexternalmedia.ReferenceType,
```

Set `Type` on the request struct (after `ID:` line 79):

```go
	reqData := &cmrequest.V1DataExternalMediasPost{
		ID:              externalMediaID,
		Type:            typ,
		ReferenceType:   referenceType,
```

**Step 3: Regenerate mocks for bin-common-handler**

Run: `cd bin-common-handler && go generate ./...`
Expected: mocks regenerated (this will break cross-service callers until they're updated)

**Step 4: Verify build**

Run: `cd bin-common-handler && go build ./...`
Expected: builds successfully

---

### Task 8: Update cross-service callers

**Files:**
- Modify: `bin-tts-manager/pkg/streaminghandler/start.go:91-103`
- Modify: `bin-transcribe-manager/pkg/streaminghandler/start.go:37-49`
- Modify: `bin-api-manager/pkg/streamhandler/start.go:37-49`
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/start.go:93-105` and `start.go:131-143`

All callers pass `externalmedia.TypeNormal` (or `cmexternalmedia.TypeNormal`) as the new `typ` parameter after `externalMediaID`.

**Step 1: Update bin-tts-manager**

In `bin-tts-manager/pkg/streaminghandler/start.go` (line 91), add `externalmedia.TypeNormal` after `st.ID`:

```go
	em, err := h.requestHandler.CallV1ExternalMediaStart(
		ctx,
		st.ID,
		externalmedia.TypeNormal,
		externalmedia.ReferenceType(st.ReferenceType),
```

**Step 2: Update bin-transcribe-manager**

In `bin-transcribe-manager/pkg/streaminghandler/start.go` (line 37), add `externalmedia.TypeNormal` after `res.ID`:

```go
	em, err := h.reqHandler.CallV1ExternalMediaStart(
		ctx,
		res.ID,
		externalmedia.TypeNormal,
		externalmedia.ReferenceType(referenceType),
```

**Step 3: Update bin-api-manager**

In `bin-api-manager/pkg/streamhandler/start.go` (line 37), add `cmexternalmedia.TypeNormal` after `tmp.ID`:

```go
	em, err := h.reqHandler.CallV1ExternalMediaStart(
		ctx,
		tmp.ID,
		cmexternalmedia.TypeNormal,
		referenceType,
```

**Step 4: Update bin-pipecat-manager (two call sites)**

In `bin-pipecat-manager/pkg/pipecatcallhandler/start.go`, update `startReferenceTypeCall` (line 93):

```go
	em, err := h.requestHandler.CallV1ExternalMediaStart(
		ctx,
		pc.ID,
		cmexternalmedia.TypeNormal,
		cmexternalmedia.ReferenceTypeCall,
```

And `startReferenceTypeAIcall` (line 131):

```go
		em, err := h.requestHandler.CallV1ExternalMediaStart(
			ctx,
			pc.ID,
			cmexternalmedia.TypeNormal,
			cmexternalmedia.ReferenceTypeCall,
```

**Step 5: Verify each service builds**

Run each in parallel or sequence:
```bash
cd bin-tts-manager && go build ./...
cd bin-transcribe-manager && go build ./...
cd bin-api-manager && go build ./...
cd bin-pipecat-manager && go build ./...
```
Expected: all build successfully

---

### Task 9: Add Type to flow-manager action option

**Files:**
- Modify: `bin-flow-manager/models/action/option.go:211-220`

**Step 1: Add Type field to OptionExternalMediaStart**

After `ExternalHost` (line 212), add:

```go
	Type            string `json:"type,omitempty"`             // type. default: normal (normal, websocket)
```

**Step 2: Verify build**

Run: `cd bin-flow-manager && go build ./...`
Expected: builds successfully

---

### Task 10: Update tests across all services

**Files:**
- Test files in `bin-call-manager/pkg/externalmediahandler/` (start_test.go, db_test.go, stop_test.go)
- Test files in `bin-call-manager/pkg/listenhandler/` (v1_calls_test.go, v1_confbridge_test.go, v1_external_medias_test.go)
- Test files in `bin-call-manager/pkg/callhandler/` (external_media_test.go, action_test.go)
- Test files in `bin-call-manager/pkg/confbridgehandler/` (external_media_test.go)
- Test files in `bin-common-handler/pkg/requesthandler/` (call_externalmedias_test.go)
- Test files in cross-service callers (start_test.go in tts, transcribe, api-manager)

**Step 1: Fix all test compilation errors**

Every test that calls `Start()`, `ExternalMediaStart()`, `Create()`, or `CallV1ExternalMediaStart()` needs the new `typ` parameter added. Pass `externalmedia.TypeNormal` in all existing tests.

Search for all affected test call sites:

```bash
grep -rn "\.Start(" bin-call-manager/pkg/externalmediahandler/*_test.go
grep -rn "\.ExternalMediaStart(" bin-call-manager/pkg/listenhandler/*_test.go bin-call-manager/pkg/callhandler/*_test.go bin-call-manager/pkg/confbridgehandler/*_test.go
grep -rn "CallV1ExternalMediaStart(" bin-common-handler/pkg/requesthandler/*_test.go bin-tts-manager/pkg/streaminghandler/*_test.go bin-transcribe-manager/pkg/streaminghandler/*_test.go bin-api-manager/pkg/streamhandler/*_test.go
```

For each call site, add the `typ` parameter (or `gomock.Any()` for mock expectations).

**Step 2: Run tests**

Run: `cd bin-call-manager && go test ./... -count=1`
Expected: all tests pass

---

### Task 11: Full verification workflow

**Step 1: Run verification for bin-common-handler**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass

**Step 2: Run verification for bin-call-manager**

```bash
cd bin-call-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass

**Step 3: Run verification for cross-service callers**

```bash
cd bin-tts-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-transcribe-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-pipecat-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass

**Step 4: Run verification for bin-flow-manager**

```bash
cd bin-flow-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass

---

### Task 12: Commit

**Step 1: Stage and commit**

```bash
git add -A
git commit -m "NOJIRA-add-external-media-type-field

Add Type field to external media model with TypeNormal and TypeWebsocket
constants. Thread the type through all handler interfaces and the shared
RPC client. All existing callers pass TypeNormal. This prepares for
chan_websocket integration in a follow-up change.

- bin-call-manager: Add Type type, constants, and field to externalmedia model
- bin-call-manager: Thread typ param through externalmediahandler Start/Create
- bin-call-manager: Thread typ param through callhandler/confbridgehandler ExternalMediaStart
- bin-call-manager: Thread typ param through listenhandler request structs and routes
- bin-common-handler: Add typ param to CallV1ExternalMediaStart RPC client
- bin-tts-manager: Pass TypeNormal to CallV1ExternalMediaStart
- bin-transcribe-manager: Pass TypeNormal to CallV1ExternalMediaStart
- bin-api-manager: Pass TypeNormal to CallV1ExternalMediaStart
- bin-pipecat-manager: Pass TypeNormal to CallV1ExternalMediaStart
- bin-flow-manager: Add Type field to OptionExternalMediaStart"
```

**Step 2: Push and create PR**

```bash
git push -u origin NOJIRA-add-external-media-type-field
```
