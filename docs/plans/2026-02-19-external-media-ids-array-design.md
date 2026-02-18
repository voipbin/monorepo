# Design: ExternalMediaID → ExternalMediaIDs Array

## Problem Statement

Currently, both Call and Confbridge models support only a single ExternalMediaID per resource. This prevents scenarios where multiple external media streams (e.g., transcription + TTS streaming simultaneously) need to run on the same call or confbridge.

## Approach

Replace the singular `ExternalMediaID uuid.UUID` field with `ExternalMediaIDs []uuid.UUID` array in both Call and Confbridge models. Use the existing JSON column pattern already established by `RecordingIDs`, `ChainedCallIDs`, and `Dialroutes`.

## Design Decisions

- **Array only, no singular field.** Unlike Recording (which has both `RecordingID` and `RecordingIDs`), external media uses only the array. There is no concept of "current" external media.
- **Configurable max limit** of 5 external medias per call/confbridge to prevent resource exhaustion.
- **Stop by specific ID.** Callers must specify which external media to stop.
- **Flow action `external_media_stop` stops ALL** external medias on the call (cleanup operation).
- **`externalMediaHandler.Stop()` removes from parent array.** Fixes existing inconsistency where external services stopping via `DELETE /v1/external-medias/<id>` left a stale reference in the call model.

## Affected Components

### 1. Models

**Call** (`bin-call-manager/models/call/call.go`):
```
Before: ExternalMediaID uuid.UUID `json:"external_media_id,omitempty" db:"external_media_id,uuid"`
After:  ExternalMediaIDs []uuid.UUID `json:"external_media_ids,omitempty" db:"external_media_ids,json"`
```

**Call Field** (`models/call/field.go`):
```
Before: FieldExternalMediaID Field = "external_media_id"
After:  FieldExternalMediaIDs Field = "external_media_ids"
```

**Confbridge** (`models/confbridge/main.go`): Same change.
**Confbridge Field** (`models/confbridge/field.go`): Same change.

### 2. Database Schema (Alembic Migration)

Tables: `call_calls`, `call_confbridges`

- Drop column `external_media_id binary(16)`
- Drop index `idx_call_calls_external_media_id`
- Add column `external_media_ids json DEFAULT '[]'`
- Update test SQL scripts in `scripts/database_scripts_test/`

### 3. DB Handler — New Operations

Following the `CallAddChainedCallID` / `CallRemoveChainedCallID` pattern using MySQL JSON functions:

- `CallAddExternalMediaID(ctx, callID, externalMediaID)` — `json_array_append`
- `CallRemoveExternalMediaID(ctx, callID, externalMediaID)` — `json_remove` + `json_search`
- `ConfbridgeAddExternalMediaID(ctx, confbridgeID, externalMediaID)` — same
- `ConfbridgeRemoveExternalMediaID(ctx, confbridgeID, externalMediaID)` — same

Remove:
- `CallSetExternalMediaID`
- `ConfbridgeSetExternalMediaID`

### 4. Call Handler

**ExternalMediaStart** (`pkg/callhandler/external_media.go`):
- Replace `if c.ExternalMediaID != uuid.Nil` guard with `if len(c.ExternalMediaIDs) >= defaultMaxExternalMediaPerCall`
- Call `db.CallAddExternalMediaID(ctx, callID, externalMediaID)` instead of `db.CallSetExternalMediaID`
- Return updated call

**ExternalMediaStop** (`pkg/callhandler/external_media.go`):
- Add `externalMediaID uuid.UUID` parameter
- Validate externalMediaID exists in `c.ExternalMediaIDs`
- Stop the specific external media
- Call `db.CallRemoveExternalMediaID(ctx, callID, externalMediaID)`

**UpdateExternalMediaID** → Remove. Replace with direct `db.CallAddExternalMediaID` / `db.CallRemoveExternalMediaID` calls.

### 5. Confbridge Handler — Same Changes as Call

### 6. External Media Handler

**Stop** (`pkg/externalmediahandler/stop.go`):
- After stopping the external media channel and deleting from cache, also remove the external media ID from the parent call/confbridge's `ExternalMediaIDs` array.
- Use `referenceType` + `referenceID` from the external media record to call the appropriate DB remove function.

### 7. Listen Handler

**`processV1CallsIDExternalMediaDelete`** (`pkg/listenhandler/v1_calls.go`):
- Parse `external_media_id` from request body (JSON)
- Pass it to `callHandler.ExternalMediaStop(ctx, callID, externalMediaID)`

**`processV1ConfbridgesIDExternalMediaDelete`** (if exists): Same change.

### 8. Request Handler (bin-common-handler)

**`CallV1CallExternalMediaStop`** (`pkg/requesthandler/call_calls.go`):
- Add `externalMediaID uuid.UUID` parameter
- Send it in the request body as JSON

### 9. Action Handler

**`actionExecuteExternalMediaStart`** (`pkg/callhandler/action.go`):
- After start, log from `cc.ExternalMediaIDs` (last element) instead of `cc.ExternalMediaID`

**`actionExecuteExternalMediaStop`** (`pkg/callhandler/action.go`):
- Replace single `c.ExternalMediaID` check with iteration over `c.ExternalMediaIDs`
- Stop ALL external medias on the call (flow-driven cleanup)

### 10. Nil Slice Initialization

- `callGetFromRow`: Add `if res.ExternalMediaIDs == nil { res.ExternalMediaIDs = []uuid.UUID{} }`
- `confbridgeGetFromRow`: Same
- `callHandler.Create`: Initialize `ExternalMediaIDs: []uuid.UUID{}`
- `confbridgeHandler.Create`: Initialize `ExternalMediaIDs: []uuid.UUID{}`

### 11. Max Limit Constant

```go
const defaultMaxExternalMediaPerCall = 5
```

Defined in both `callhandler` and `confbridgehandler`.

## Services NOT Changed

These services use `CallV1ExternalMediaStop(ctx, externalMediaID)` which hits `DELETE /v1/external-medias/<id>` directly. They don't interact with the call's ExternalMediaID field:

- `bin-tts-manager`
- `bin-transcribe-manager`
- `bin-pipecat-manager`
- `bin-api-manager`

## Files Changed (Summary)

### bin-call-manager
- `models/call/call.go` — field change
- `models/call/field.go` — field constant change
- `models/confbridge/main.go` — field change
- `models/confbridge/field.go` — field constant change
- `pkg/dbhandler/main.go` — interface update
- `pkg/dbhandler/call.go` — add/remove operations, remove set operation
- `pkg/dbhandler/confbridge.go` — same
- `pkg/dbhandler/call_test.go` — update tests
- `pkg/dbhandler/confbridge_test.go` — update tests
- `pkg/callhandler/main.go` — interface update (ExternalMediaStop signature)
- `pkg/callhandler/external_media.go` — start/stop logic changes
- `pkg/callhandler/external_media_test.go` — update tests
- `pkg/callhandler/db.go` — replace UpdateExternalMediaID
- `pkg/callhandler/db_test.go` — update tests
- `pkg/callhandler/action.go` — action handler changes
- `pkg/callhandler/action_test.go` — update tests
- `pkg/confbridgehandler/main.go` — interface update
- `pkg/confbridgehandler/external_media.go` — same changes as call
- `pkg/confbridgehandler/external_media_test.go` — update tests
- `pkg/confbridgehandler/db.go` — replace UpdateExternalMediaID
- `pkg/confbridgehandler/db_test.go` — update tests
- `pkg/externalmediahandler/stop.go` — add parent array cleanup
- `pkg/listenhandler/v1_calls.go` — parse externalMediaID from body
- `pkg/listenhandler/v1_calls_test.go` — update tests
- `pkg/listenhandler/v1_confbridges.go` — same
- `pkg/listenhandler/v1_confbridge_test.go` — update tests
- `pkg/listenhandler/models/request/calls.go` — add field to delete request
- `scripts/database_scripts_test/table_calls.sql` — schema change
- `scripts/database_scripts_test/table_confbridges.sql` — schema change

### bin-common-handler
- `pkg/requesthandler/main.go` — interface update (CallV1CallExternalMediaStop signature)
- `pkg/requesthandler/call_calls.go` — add externalMediaID param
- `pkg/requesthandler/call_calls_test.go` — update tests
- `pkg/requesthandler/call_confbridge.go` — same
- `pkg/requesthandler/call_confbridge_test.go` — update tests

### bin-dbscheme-manager
- New Alembic migration for `call_calls` and `call_confbridges` tables

### OpenAPI (bin-openapi-manager)
- Update Call and Confbridge schemas: `external_media_id` → `external_media_ids` (array of UUIDs)
