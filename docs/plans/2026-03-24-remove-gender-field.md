# Remove Gender Field

## Problem

The `gender` field on AIcall and TTS streaming is redundant. The TTS voice ID already determines voice characteristics (gender, accent, tone). Gender is threaded through 6 services as a parameter but provides no value beyond what the voice ID already encodes.

## Approach

Remove `gender` entirely from the platform — model structs, function signatures, request models, RPC interfaces, OpenAPI schema, database column, and documentation. No backward compatibility — clean removal.

## Changes by Service

### bin-ai-manager

- **models/aicall/main.go**: Remove `Gender` type definition, constants (`GenderNone`, `GenderMale`, `GenderFemale`, `GenderNeutral`), and `Gender` field from AIcall struct
- **models/aicall/webhook.go**: Remove `Gender` from WebhookMessage and ConvertWebhookMessage()
- **models/aicall/field.go**: Remove `FieldGender`
- **models/aicall/filters.go**: Remove `Gender` from FieldStruct
- **pkg/aicallhandler/main.go**: Remove `gender` param from `Start()` and `ServiceStart()` interfaces; remove `variableGender` constant
- **pkg/aicallhandler/start.go**: Remove `gender` param from all functions in the start chain
- **pkg/aicallhandler/service.go**: Remove `gender` param from ServiceStart and related functions
- **pkg/aicallhandler/db.go**: Remove `gender` param from `Create()` and `CreateByMessaging()`; remove `Gender: gender` from struct initialization
- **pkg/aicallhandler/chat.go**: Remove `variableGender` from setActiveflowVariables
- **pkg/listenhandler/models/request/aicalls.go**: Remove `Gender` from V1DataAIcallsPost
- **pkg/listenhandler/v1_aicalls.go**: Remove `req.Gender` from Start() call
- **All corresponding test files**: Update for removed field/param

### bin-common-handler

- **pkg/requesthandler/main.go**: Remove `gender` from `AIV1AIcallStart` and `AIV1ServiceTypeAIcallStart`; remove `gender` from `TTSV1StreamingStart`
- **pkg/requesthandler/ai_aicalls.go**: Remove `gender` param and `Gender: gender` from request data
- **pkg/requesthandler/ai_services.go**: Remove `gender` param and field
- **pkg/requesthandler/tts_streamings.go**: Remove `gender` param and field
- **Test files and mocks**: Update

### bin-api-manager

- **pkg/servicehandler/main.go**: Remove `gender` from `AIcallCreate` interface
- **pkg/servicehandler/aicall.go**: Remove `gender` from function signature and RPC call
- **server/aicalls.go**: Remove `amaicall.Gender(req.Gender)` and stop passing gender
- **Test files and mocks**: Update

### bin-flow-manager

- **models/action/option.go**: Remove `Gender` from `OptionAITalk`
- **pkg/activeflowhandler/actionhandle.go**: Remove `opt.Gender` from `AIV1ServiceTypeAIcallStart` call
- **Test files**: Update

### bin-tts-manager

- **models/streaming/streaming.go**: Remove `Gender` type, constants, and field from Streaming struct
- **pkg/listenhandler/models/request/streamings.go**: Remove `Gender` from request model
- **pkg/listenhandler/v1_streamings.go**: Remove `req.Gender` from Start() call
- **Test files**: Update

### bin-openapi-manager

- Remove `AIManagerAIcallGender` enum schema from openapi.yaml
- Remove `gender` property from `AIManagerAIcall` response schema
- Remove `gender` from POST `/aicalls` request body
- Remove `gender` from `FlowManagerActionOptionAITalk`
- Remove `gender` from TTS streaming request schema
- Regenerate types in bin-openapi-manager and bin-api-manager

### bin-dbscheme-manager

- New Alembic migration: `ALTER TABLE ai_aicalls DROP COLUMN gender`
- Downgrade: `ALTER TABLE ai_aicalls ADD COLUMN gender VARCHAR(255) NOT NULL DEFAULT ''`

### RST Documentation

- Remove gender references from: ai_overview.rst, flow_struct_action.rst, flow_advanced_patterns.rst, variable_variable.rst, quickstart_realtime.rst
- Rebuild HTML

## Deployment

Run Alembic migration first, then deploy all services together. The migration only drops a column, so it's safe to run before deployment (existing code writes to it but the column being absent won't break reads — the DB tag scan will just skip it).

## Testing

All existing tests updated to remove gender params. No new test logic needed — this is pure removal.
