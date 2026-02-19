# Add Type Field to External Media

## Problem

External media currently has no way to distinguish between different channel technologies. We need a `Type` field to differentiate between normal external media (RTP-based via ARI ExternalMedia) and websocket-based external media (via `chan_websocket`, a new Asterisk channel technology).

This is the model-layer preparation. The actual `chan_websocket` integration will be a follow-up change.

## Approach

Add `Type` as a string type with two constants (`TypeNormal = "normal"`, `TypeWebsocket = "websocket"`) and thread it through the entire creation flow — from request structs through handler interfaces to the stored model.

When `Type` is empty, it defaults to `TypeNormal` to maintain backward compatibility.

## Files to Modify

### Layer 1: Model (bin-call-manager/models/externalmedia/)

- `main.go` — Add `Type` type, constants, and struct field
- `field.go` — Add `FieldType`
- `filters.go` — Add `Type` to `FieldStruct`
- `externalmedia_test.go` — Add tests for new type and constants
- `field_test.go` — Add `FieldType` to test table

### Layer 2: Request structs (bin-call-manager/pkg/listenhandler/models/request/)

- `externalmedias.go` — Add `Type` to `V1DataExternalMediasPost`
- `calls.go` — Add `Type` to `V1DataCallsIDExternalMediaPost`
- `confbridge.go` — Add `Type` to `V1DataConfbridgesIDExternalMediaPost`

### Layer 3: Handler interfaces + implementations (bin-call-manager/pkg/)

- `externalmediahandler/main.go` — Add `typ` param to `Start()` interface
- `externalmediahandler/start.go` — Thread `typ` through `Start()` → `startReferenceTypeCall()` / `startReferenceTypeConfbridge()` → `startExternalMedia()` → `Create()`; default to `TypeNormal` when empty
- `externalmediahandler/db.go` — Add `typ` param to `Create()`, set on struct

- `callhandler/main.go` — Add `typ` param to `ExternalMediaStart()` interface
- `callhandler/external_media.go` — Thread `typ` to `externalMediaHandler.Start()`

- `confbridgehandler/main.go` — Add `typ` param to `ExternalMediaStart()` interface
- `confbridgehandler/external_media.go` — Thread `typ` to `externalMediaHandler.Start()`

### Layer 4: Listen handler routing (bin-call-manager/pkg/listenhandler/)

- `v1_external_medias.go` — Pass `req.Type` to `externalMediaHandler.Start()`
- `v1_calls.go` — Pass `req.Type` to `callHandler.ExternalMediaStart()`
- `v1_confbridges.go` — Pass `req.Type` to `confbridgeHandler.ExternalMediaStart()`

### Layer 5: bin-common-handler (shared RPC client)

- `pkg/requesthandler/main.go` — Add `typ` param to `CallV1ExternalMediaStart()` interface
- `pkg/requesthandler/call_externalmedias.go` — Add `typ` param, set on request struct

### Layer 6: Cross-service callers of CallV1ExternalMediaStart

These services call the RPC client and need the new parameter (pass empty string or `TypeNormal`):

- `bin-tts-manager/pkg/streaminghandler/start.go`
- `bin-transcribe-manager/pkg/streaminghandler/start.go`
- `bin-api-manager/pkg/streamhandler/start.go`
- `bin-pipecat-manager/pkg/pipecatcallhandler/start.go`

### Layer 7: Flow manager action option

- `bin-flow-manager/models/action/option.go` — Add `Type` to `OptionExternalMediaStart`

### Layer 8: Mock regeneration

Run `go generate ./...` in:
- `bin-call-manager`
- `bin-common-handler`

### Layer 9: Tests

Update all affected test files for the new parameter.

## Default Behavior

- Empty `Type` defaults to `TypeNormal` in `startExternalMedia()`
- Existing callers that don't specify type get normal (current) behavior
- No behavior change for `TypeWebsocket` yet — it stores the value but follows the same code path as normal

## Not in Scope

- `chan_websocket` Asterisk channel creation logic
- OpenAPI spec changes (ExternalMedia is not exposed in the public API spec)
- Database migration (external media is stored in Redis cache)
