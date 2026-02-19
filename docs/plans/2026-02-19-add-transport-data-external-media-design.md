# Add transport_data to External Media

## Problem Statement

The Asterisk ARI `externalMedia` endpoint supports a `transport_data` parameter described as "Transport-specific data. For websocket this is appended to the dialstring." VoIPbin's external media stack currently passes `transport`, `encapsulation`, `connection_type`, and other fields to Asterisk, but does NOT pass `transport_data`. This field is needed for upcoming websocket transport support (chan_websocket integration).

Reference: https://docs.asterisk.org/Asterisk_23_Documentation/API_Documentation/Asterisk_REST_Interface/Channels_REST_API/#externalmedia

## Approach

Add a `TransportData string` field threaded through the entire external media stack — from API callers, through inter-service RPC, into call-manager's internal handler chain, down to the Asterisk ARI call. The field is optional (defaults to empty string), stored in the ExternalMedia model (Redis cache), and fully backwards compatible.

## Key Decisions

- **Full stack exposure**: The field is available at all layers (API, RPC, internal handlers, ARI) so any caller can set it.
- **Stored in model**: Persisted in the ExternalMedia struct (Redis). Useful for debugging and retrieving external media state.
- **No Alembic migration**: External media is Redis-only (cache), not MySQL. Adding a field to the struct handles persistence automatically via JSON serialization.
- **Existing callers pass `""`**: All 5 external services (bin-api-manager, bin-tts-manager, bin-transcribe-manager, bin-pipecat-manager) currently use UDP/TCP and will pass empty string. No behavior change.

## Files to Change

### Layer 1: ExternalMedia Model

| File | Change |
|------|--------|
| `bin-call-manager/models/externalmedia/main.go` | Add `TransportData string` field to `ExternalMedia` struct |
| `bin-call-manager/models/externalmedia/field.go` | Add `FieldTransportData Field = "transport_data"` constant |

`filters.go` does NOT need updating — `transport_data` is not a filterable field.

### Layer 2: Request Models

| File | Change |
|------|--------|
| `bin-call-manager/pkg/listenhandler/models/request/externalmedias.go` | Add `TransportData string` to `V1DataExternalMediasPost` |
| `bin-call-manager/pkg/listenhandler/models/request/calls.go` | Add `TransportData string` to `V1DataCallsIDExternalMediaPost` |
| `bin-call-manager/pkg/listenhandler/models/request/confbridge.go` | Add `TransportData string` to `V1DataConfbridgesIDExternalMediaPost` |

### Layer 3: call-manager Internal Handlers

**externalmediahandler:**

| File | Change |
|------|--------|
| `pkg/externalmediahandler/main.go` | Add `transportData string` param to `ExternalMediaHandler.Start()` interface |
| `pkg/externalmediahandler/start.go` | Thread `transportData` through `Start()`, `startReferenceTypeCall()`, `startReferenceTypeConfbridge()`, `startExternalMedia()` |
| `pkg/externalmediahandler/db.go` | Add `transportData string` param to `Create()`, set `TransportData` on struct |

**callhandler:**

| File | Change |
|------|--------|
| `pkg/callhandler/main.go` | Add `transportData string` to `CallHandler.ExternalMediaStart()` interface |
| `pkg/callhandler/external_media.go` | Thread through `ExternalMediaStart()` implementation |
| `pkg/callhandler/action.go` | Pass `option.TransportData` in `actionExecuteExternalMediaStart()` |

**confbridgehandler:**

| File | Change |
|------|--------|
| `pkg/confbridgehandler/main.go` | Add `transportData string` to `ConfbridgeHandler.ExternalMediaStart()` interface |
| `pkg/confbridgehandler/external_media.go` | Thread through `ExternalMediaStart()` implementation |

**channelhandler:**

| File | Change |
|------|--------|
| `pkg/channelhandler/main.go` | Add `transportData string` to `ChannelHandler.StartExternalMedia()` interface |
| `pkg/channelhandler/start.go` | Thread through `StartExternalMedia()` implementation |

**listenhandler:**

| File | Change |
|------|--------|
| `pkg/listenhandler/v1_external_medias.go` | Pass `req.TransportData` in `processV1ExternalMediasPost()` |
| `pkg/listenhandler/v1_calls.go` | Pass `req.TransportData` in `processV1CallsIDExternalMediaPost()` |
| `pkg/listenhandler/v1_confbridges.go` | Pass `req.TransportData` in `processV1ConfbridgesIDExternalMediaPost()` |

### Layer 4: bin-common-handler RequestHandler

**Interface changes (4 functions in `main.go`):**

| Interface Method | File |
|------------------|------|
| `AstChannelExternalMedia` | `ast_channel.go` |
| `CallV1ExternalMediaStart` | `call_externalmedias.go` |
| `CallV1CallExternalMediaStart` | `call_calls.go` |
| `CallV1ConfbridgeExternalMediaStart` | `call_confbridge.go` |

Each implementation adds `transportData string` parameter and includes it in the JSON request body.

### Layer 5: External Caller Services

All pass `""` for `transportData`:

| File | Call sites |
|------|-----------|
| `bin-api-manager/pkg/streamhandler/start.go` | 1 |
| `bin-tts-manager/pkg/streaminghandler/start.go` | 1 |
| `bin-transcribe-manager/pkg/streaminghandler/start.go` | 1 |
| `bin-pipecat-manager/pkg/pipecatcallhandler/start.go` | 2 |

Note: `CallV1CallExternalMediaStart` and `CallV1ConfbridgeExternalMediaStart` have no external callers — only `CallV1ExternalMediaStart` is used by external services.

### Layer 6: Flow Action Model

| File | Change |
|------|--------|
| `bin-flow-manager/models/action/option.go` | Add `TransportData string` to `OptionExternalMediaStart` |

### Layer 7: OpenAPI Spec

| File | Change |
|------|--------|
| `bin-openapi-manager/openapi/openapi.yaml` | Add `transport_data` field to `FlowManagerActionOptionExternalMediaStart` |

### Layer 8: Code Generation

Run `go generate ./...` for:
- `bin-common-handler` (regenerate `mock_main.go`)
- `bin-call-manager` (regenerate all handler mocks)
- `bin-openapi-manager` (regenerate OpenAPI types)
- `bin-api-manager` (regenerate server code from OpenAPI)

### Layer 9: Tests

Update mock expectations in all affected test files to include the new `transportData` parameter.

## Services Requiring Verification

`bin-common-handler`, `bin-call-manager`, `bin-api-manager`, `bin-tts-manager`, `bin-transcribe-manager`, `bin-pipecat-manager`, `bin-flow-manager`, `bin-openapi-manager`

## Pre-existing Issue (Not Fixed Here)

`ExternalMediaHandler.Start()` accepts `connectionType` as a parameter but does not thread it through to sub-functions. The `startExternalMedia()` function uses `defaultConnectionType` constant instead, silently dropping the caller's value. The `transportData` field in this design is properly threaded to avoid repeating this pattern.

## Backwards Compatibility

Fully backwards compatible:
- All existing callers pass `""` for `transportData`
- Empty string means Asterisk uses default behavior (no transport-specific data)
- The JSON field `transport_data` is `omitempty` — absent from wire format when empty
- Redis-stored ExternalMedia structs missing the field will deserialize with zero value (`""`)
