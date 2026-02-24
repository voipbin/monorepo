# AIcall Schema Cleanup Design

## Problem

The AIcall model has drifted between Go code, database, and OpenAPI spec:

- `transcribe_id` still present in OpenAPI but already removed from Go struct and DB
- `AIEngineType` is deprecated and unused (comment: "currently not in used")
- WebhookMessage does not expose `AIEngineData`, `AITTSType`, `AITTSVoiceID`, `AISTTType`
- OpenAPI field names don't match Go JSON tags (`engine_type`/`engine_model` vs `ai_engine_type`/`ai_engine_model`)

## Changes

### 1. AIcall Struct (bin-ai-manager/models/aicall/main.go)

Remove the `AIEngineType` field. All other fields remain unchanged.

### 2. WebhookMessage (bin-ai-manager/models/aicall/webhook.go)

Remove `AIEngineType` and `AIEngineModel`. Add fields to match what the AIcall struct exposes:

- `AIEngineModel ai.EngineModel` (json: `ai_engine_model`)
- `AIEngineData map[string]any` (json: `ai_engine_data`)
- `AITTSType ai.TTSType` (json: `ai_tts_type`)
- `AITTSVoiceID string` (json: `ai_tts_voice_id`)
- `AISTTType ai.STTType` (json: `ai_stt_type`)

Update `ConvertWebhookMessage()` to populate all new fields.

### 3. Alembic Migration (bin-dbscheme-manager)

Create migration to drop the `ai_engine_type` column from `ai_aicalls` table.

Note: `transcribe_id` was already dropped in migration `bad27b40fe8e`.

### 4. OpenAPI Schema (bin-openapi-manager/openapi/openapi.yaml)

Update `AIManagerAIcall` schema:

- Remove `transcribe_id` property
- Remove `engine_type` property
- Rename `engine_model` to `ai_engine_model` (match Go JSON tag)
- Add `ai_engine_data` (type: object, additionalProperties: true)
- Add `ai_tts_type` (type: string)
- Add `ai_tts_voice_id` (type: string)
- Add `ai_stt_type` (type: string)

### 5. Code Reference Updates

Update all references in bin-ai-manager that use `aicall.AIEngineType` — remove or replace with `aicall.AIEngineModel` as appropriate.

### 6. Regeneration

- `bin-openapi-manager`: `go generate ./...`
- `bin-api-manager`: `go generate ./...`

## Not Changed

- `AIID` field — references parent AI entity
- Parent `AI` struct (models/ai/main.go) — separate concern
- `PipecatcallID` — internal, not exposed in WebhookMessage
- DB `transcribe_id` — already dropped in migration bad27b40fe8e
- Field names `AIEngineModel`, `AIEngineData`, `AITTSType`, `AITTSVoiceID`, `AISTTType` — keep current naming
