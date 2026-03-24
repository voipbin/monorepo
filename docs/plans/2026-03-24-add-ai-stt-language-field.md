# Design: Add STT Language Field to AI Model

**Date:** 2026-03-24
**Branch:** NOJIRA-Add-AI-STT-language-field

## Problem Statement

The current AI system uses a single `Language` field on AICall, passed per-call and shared for both STT and TTS. This has two issues:

1. **No AI-level default** — language must be specified every time a call is created, rather than being part of the AI agent configuration.
2. **STT and TTS language are conflated** — they serve different purposes. STT needs an explicit language upfront (the engine must know what to listen for), while TTS language is implicit from the voice ID (a Korean voice speaks Korean).

## Approach

- Add `stt_language` to the **AI model** as the single source of truth for STT language.
- Rename `language` → `stt_language` on the **AICall model** (always derived from AI config, no per-call override).
- Remove `language` from all request models and RPC call chains.
- TTS language stays implicit from `tts_voice_id` — no new TTS field needed. Pass `""` for TTS language to Pipecat.
- No backward compatibility shims — clean rename.

## Data Flow

```
User creates/updates AI config
    → AI.STTLanguage = "ko-KR"   (stored in DB)

AICall creation (any path):
    → resolveAI() returns *ai.AI
    → Create()/CreateByMessaging() sets AIcall.STTLanguage = c.STTLanguage
    → No language param in the call chain

Pipecat call start:
    → sttLanguage = c.STTLanguage ("ko-KR")
    → ttsLanguage = ""  (voice ID handles it)

Activeflow variable:
    → "voipbin.aicall.stt_language" = c.STTLanguage
```

## TTS Language Rationale

TTS providers handle language through the voice ID:

- **Cartesia / ElevenLabs**: Accept empty language — infer from text or voice defaults.
- **Google**: Extracts language from voice name (e.g., `ko-KR-Chirp3-HD-Charon` → `ko-KR`). Ignores the language parameter when voice_id is set.

Passing `""` for TTS language is safe for all providers.

## Edge Cases

| Case | Behavior |
|------|----------|
| AI config with empty `stt_language` | Deepgram: auto-detect. Google: defaults to `en-US`. Safe. |
| `CreateByMessaging` (no audio) | Still sets `STTLanguage` from AI config for activeflow variable and API response. |
| `StartTask` (no STT/TTS) | Passes `STTTypeNone` and `""` to Pipecat — no change needed in `startPipecatcallTask`. |
| Team pipeline | All members share start member's `stt_language`. Pre-existing limitation. |
| Team member switch | Only `CurrentMemberID` updated. STT language stays from init. Pre-existing. |
| Existing AI records after migration | `stt_language` will be empty — safe per STT provider defaults. |
| Existing flows with `language` in `ai_talk` action | `OptionAITalk.Language` removed — field silently ignored on old saved flows during JSON unmarshal. |

## Changes by Service

### bin-ai-manager — Models

| File | Change |
|------|--------|
| `models/ai/main.go` | Add `STTLanguage string` field with `json:"stt_language,omitempty" db:"stt_language"` |
| `models/ai/field.go` | Add `FieldSTTLanguage Field = "stt_language"` |
| `models/ai/webhook.go` | Add `STTLanguage` to `WebhookMessage` and `ConvertWebhookMessage()` |
| `models/aicall/main.go` | Rename `Language` → `STTLanguage` (json: `stt_language`, db: `stt_language`) |
| `models/aicall/field.go` | Rename `FieldLanguage` → `FieldSTTLanguage`, value `"stt_language"` |
| `models/aicall/webhook.go` | Rename `Language` → `STTLanguage` in `WebhookMessage` and `ConvertWebhookMessage()` |

### bin-ai-manager — Handlers

| File | Change |
|------|--------|
| `pkg/aicallhandler/main.go` | Remove `language` from `Start()` and `ServiceStart()` interface. Rename `variableLanguage` → `variableSTTLanguage = "voipbin.aicall.stt_language"` |
| `pkg/aicallhandler/start.go` | Remove `language` param from all functions. Use `a.STTLanguage` from resolved AI config. In `startPipecatcall()`: STT language = `c.STTLanguage`, TTS language = `""` |
| `pkg/aicallhandler/service.go` | Remove `language` param from `ServiceStart()` and helpers |
| `pkg/aicallhandler/db.go` | Remove `language` param from `Create()` and `CreateByMessaging()`. Set `STTLanguage: c.STTLanguage` (AI config already passed as `c *ai.AI`) |
| `pkg/aicallhandler/chat.go` | Update `setActiveflowVariables()`: `variableSTTLanguage: cc.STTLanguage` |
| `pkg/listenhandler/models/request/aicalls.go` | Remove `Language` from `V1DataAIcallsPost` |
| `pkg/listenhandler/models/request/services.go` | Remove `Language` from `V1DataServicesTypeAIcallPost` |
| `pkg/listenhandler/models/request/ais.go` | Add `STTLanguage` to `V1DataAIsPost` and `V1DataAIsIDPut` |
| `pkg/listenhandler/v1_aicalls.go` | Remove `req.Language` from `Start()` call |
| `pkg/listenhandler/v1_services.go` | Remove `req.Language` from `ServiceStart()` call |
| `pkg/listenhandler/v1_ais.go` | Pass `req.STTLanguage` when creating/updating AI |
| `pkg/aicallhandler/mock_main.go` | Regenerate (`go generate`) |
| All `*_test.go` files | Update mock expectations, test fixtures, and assertions |

### bin-common-handler

| File | Change |
|------|--------|
| `pkg/requesthandler/main.go` | Remove `language` from `AIV1AIcallStart()` and `AIV1ServiceTypeAIcallStart()` |
| `pkg/requesthandler/ai_aicalls.go` | Remove `language` param and from request body |
| `pkg/requesthandler/ai_services.go` | Remove `language` from `AIV1ServiceTypeAIcallStart()` only. `AIV1ServiceTypeSummaryStart` unchanged. |
| `pkg/requesthandler/mock_main.go` | Regenerate (`go generate`) |
| Test files | Update all affected tests |

### bin-api-manager

| File | Change |
|------|--------|
| `pkg/servicehandler/aicall.go` | Remove `language` from `AIV1AIcallStart()` call |
| `pkg/servicehandler/aicall_test.go` | Update mock expectations |
| Generated code | Regenerate from OpenAPI (`go generate ./...`) |

### bin-flow-manager

| File | Change |
|------|--------|
| `models/action/option.go` | Remove `Language` from `OptionAITalk` |
| `pkg/activeflowhandler/actionhandle.go` | Remove `opt.Language` from `AIV1ServiceTypeAIcallStart()` call |
| `pkg/activeflowhandler/actionhandle_test.go` | Update test expectations |

### bin-dbscheme-manager

New Alembic migration:
- `upgrade()`: ADD `stt_language VARCHAR(16) DEFAULT '' NOT NULL` to `ais` table. RENAME column `language` → `stt_language` in `aicalls` table.
- `downgrade()`: Reverse both operations.

### bin-openapi-manager

| File | Change |
|------|--------|
| `openapi/openapi.yaml` | Add `stt_language` to `AIManagerAI` schema. Rename `language` → `stt_language` in `AIManagerAIcall` schema. Remove `language` from `FlowManagerActionOptionAITalk`. |
| `openapi/paths/ais/main.yaml` | Add `stt_language` to POST request body |
| `openapi/paths/ais/id.yaml` | Add `stt_language` to PUT request body |
| `openapi/paths/aicalls/main.yaml` | Remove `language` from request body and `required` list |
| Service endpoint paths | Remove `language` from aicall service request (if in OpenAPI) |
| Regenerate types | `go generate ./...` |

### RST Docs (`bin-api-manager/docsdev/source/`)

| File | Change |
|------|--------|
| `ai_struct_ai.rst` | Add `stt_language` field documentation |
| `ai_tutorial.rst` | Update examples: add `stt_language` to AI config, remove `language` from flow actions |
| `ai_overview.rst` | Update multilingual support section |
| Rebuild HTML | `rm -rf build && sphinx build && git add -f build/` |

## Test Coverage

| Test Scenario | Location |
|------|--------|
| AI create with `stt_language` stored correctly | AI handler tests |
| AI update `stt_language` reflected in subsequent AIcalls | AI handler tests |
| AICall creation (realtime) copies `STTLanguage` from AI config | `db_test.go` |
| AICall creation (messaging) copies `STTLanguage` from AI config | `db_test.go` |
| AI with empty `stt_language` handled gracefully | `start_test.go` |
| Pipecat call: `sttLanguage` = `c.STTLanguage`, `ttsLanguage` = `""` | `start_test.go` |
| `startPipecatcallTask` still passes `""` for both (no regression) | `start_test.go` |
| Team AICall uses start member's AI `stt_language` | `start_test.go` |
| Activeflow variable uses `voipbin.aicall.stt_language` | `chat_test.go` |
| RPC `AIV1AIcallStart` no longer sends `language` | requesthandler tests |
| RPC `AIV1ServiceTypeAIcallStart` no longer sends `language` | requesthandler tests |
| API POST `/aicalls` no longer accepts `language` | servicehandler tests |
| Flow action `ai_talk` no longer passes `language` | actionhandle tests |

## Verification

Run full workflow for each affected service:

1. `bin-common-handler`
2. `bin-ai-manager`
3. `bin-openapi-manager`
4. `bin-api-manager`
5. `bin-flow-manager`

Each: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`

## Deploy Order

1. Alembic migration (add column, rename column) — must run first.
2. Code deploy — all services together (standard k8s rolling update).

## Known Limitations

- Team members all share the pipeline-level STT language from the start member's AI config.
- No per-utterance dynamic STT language switching.
- `ai-control` CLI may need `--stt_language` flag if it uses explicit flag definitions.
