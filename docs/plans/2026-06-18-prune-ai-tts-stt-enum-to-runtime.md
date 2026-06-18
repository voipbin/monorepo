# Prune AI TTS/STT enum to runtime-supported providers

Date: 2026-06-18
Branch: NOJIRA-Prune-ai-tts-stt-enum-to-runtime
Repo: monorepo (backend)
Decision: pchero approved "현재 상황에 맞추자... 런타임으로 가자" (align enum to runtime,
do NOT add the missing runtime adapters - that is a separate future feature).

## Problem

The AI config layer accepts far more TTS/STT provider values than the pipecat
runtime actually implements. A customer setting an accepted-but-unimplemented
provider (e.g. tts_type "playht") passes config validation, then fails at runtime
with `ValueError: Unsupported TTS service` in bin-pipecat-manager. This is a
fail-late footgun.

## Runtime source of truth (verified 2026-06-18)

bin-pipecat-manager/scripts/pipecat/run.py:
- TTS factory `create_tts_service` (~L349-383): cartesia, elevenlabs, google. else ValueError.
- STT factory `create_stt_service` (~L386-410): deepgram, google. else ValueError.

Target enums after prune:
- TTSType: "" (none), cartesia, elevenlabs, google  (3 + none)
- STTType: "" (none), deepgram, google              (2 + none)

## Current drift across surfaces

| Surface | STT now | TTS now |
|---|---|---|
| bin-ai-manager models/ai/main.go | cartesia, deepgram, elevenlabs, google (4) | 21 |
| bin-openapi-manager openapi.yaml | "", cartesia, deepgram, elevenlabs (3, google MISSING) | 22 |
| bin-ai-manager aicallhandler mapDefaultTTSVoiceIDByTTSType | n/a | 21 entries |

Note: OpenAPI STT is missing `google` (a runtime-supported provider) and must ADD it
while removing cartesia/elevenlabs. Net STT target = deepgram, google.

## Scope: backend only (this PR)

### File 1: bin-ai-manager/models/ai/main.go
- TTSType const block (L201-226): keep TTSTypeNone, TTSTypeCartesia, TTSTypeElevenLabs,
  TTSTypeGoogle. Remove the other 18 (async, aws, azure, deepgram, fish, groq, hume,
  inworld, lmnt, minimax, neuphonic, nvidia-riva, openai, piper, playht, rime, sarvam,
  xtts). NOTE: Deepgram is a valid runtime STT but NOT a runtime TTS, so TTSTypeDeepgram
  is removed from TTS.
- validTTSTypes map (L228-237): reduce to the 4 kept values (none + 3).
- STTType const block (L259-265): keep STTTypeNone, STTTypeDeepgram, STTTypeGoogle.
  Remove STTTypeCartesia, STTTypeElevenLabs.
- validSTTTypes map (L267-271): reduce to none + deepgram + google.
- IsValid / ValidValues helpers: unchanged (map-driven, auto-correct).

### File 2: bin-ai-manager/pkg/aicallhandler/main.go
- mapDefaultTTSVoiceIDByTTSType (L94-117): remove the 18 dropped TTS entries, keep
  None, Cartesia, ElevenLabs, Google. defaultTTSType stays ElevenLabs (kept), defaultSTTType
  stays Deepgram (kept). Verify no other code indexes the removed keys.
- Check any switch/lookup on the removed types elsewhere in the package.

### File 3: bin-ai-manager test files (explicit list, from review round 1)
These reference removed consts and will FAIL to compile after the prune. Update each:
- models/ai/main_test.go: L54 `STTTypeCartesia` -> STTTypeDeepgram, plus L395,400,432,442,
  469-488,547,549 (TTS/STT cases referencing removed consts).
- models/aicall/main_test.go: L71 `STTTypeCartesia` -> STTTypeDeepgram.
- pkg/aihandler/chatbot_test.go: L242,259,285,298,315,329,347 `TTSTypeOpenAI` -> TTSTypeElevenLabs (or Google).
- pkg/dbhandler/ai_test.go: L50,69,358,377 `STTTypeElevenLabs` -> STTTypeDeepgram.
After edits, `go test ./...` compilation is the backstop that catches any straggler.

### File 3b: bin-api-manager cross-service test (from review round 2)
bin-api-manager imports `amai "monorepo/bin-ai-manager/models/ai"` and its tests
reference removed consts. Pruning bin-ai-manager breaks bin-api-manager test compilation.
- bin-api-manager/server/ais_test.go: 11 sites of `amai.STTTypeCartesia`
  (L76,109,142,175,208,241,599,633,667,701,735) -> `amai.STTTypeDeepgram`.
Repo-wide grep confirmed this is the ONLY cross-service reference to any removed const
(no other consumer references the removed TTS consts).

### File 4: bin-openapi-manager/openapi/openapi.yaml
- AIManagerAITTSType enum (L2347-2396): reduce to "", cartesia, elevenlabs, google +
  matching x-enum-varnames. CRITICAL: enum list and x-enum-varnames are matched by
  POSITION (1:1 by index) in oapi-codegen. Keep both lists the same length and order;
  a single missing line silently misaligns a varname to the wrong value.
- AIManagerAISTTType enum (L2398-2411): set to "", deepgram, google + x-enum-varnames
  (removes cartesia/elevenlabs, ADDS google).
- Update example values if they reference a removed provider (TTS example "elevenlabs"
  is kept-valid; STT example "deepgram" is kept-valid).

### File 5: bin-openapi-manager/gens/models/gen.go
- Regenerated via `go generate ./...` (DO NOT hand-edit). Commit the regenerated file.

### File 6: bin-api-manager generated artifacts (from review round 2 - MUST regenerate)
bin-api-manager generates its OWN artifacts from openapi.yaml; `go build` does NOT
regenerate them, so a build-only check leaves stale enum values in shipped code/docs.
Run `cd bin-api-manager && go generate ./...` and commit:
- bin-api-manager/gens/openapi_server/gen.go (oapi-codegen server types)
- bin-api-manager/gens/openapi_redoc/openapi.json and api.html (redocly, force-add per
  repo convention - these are gitignored-but-committed build artifacts).

## Migration / backward-compat consideration

Existing AI configs in the DB may already store a now-removed provider value (e.g. a
customer who set tts_type "openai"). Pruning the enum does NOT delete DB rows. Behavior
after prune:
- Reads: the stored string still round-trips (Go string type); IsValid() returns false
  but reads do not call IsValid().
- Runtime: those configs ALREADY fail at pipecat today (ValueError), so this PR does not
  regress them; it surfaces the rejection earlier (at config-update validation time).
- We are NOT writing a data migration. If audit shows live configs using removed values,
  raise separately. Likelihood low (runtime already broke them).

## Out of scope (separate follow-ups, NOT this PR)
- Adding the missing runtime adapters (aws, azure, playht, etc.) = NEW FEATURE, deferred.
- ToolName enum drift in openapi.yaml (missing create_call, get_resource, get_correlation,
  stop_service vs the 13 in models/tool/main.go) = separate PR.
- square-admin api.ts + constants.js TTS/STT dropdown alignment = PR B (frontend), after
  this PR merges (OpenAPI is source of truth).

## Verification plan
Per monorepo CLAUDE.md, run in EACH changed service dir:
1. bin-ai-manager: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
2. bin-openapi-manager: `go generate ./...` to regenerate gens/models/gen.go.
3. bin-api-manager: `go generate ./...` (NOT just build) to regenerate gens/openapi_server/gen.go
   and gens/openapi_redoc/{openapi.json,api.html}, then `go build ./...` and `go test ./...`.
   Commit the regenerated artifacts (force-add redoc per repo convention).
4. Grep guard (full removed-const set, must return 0 in NON-test code):
   `grep -rn "TTSTypeAsync\|TTSTypeAWS\|TTSTypeAzure\|TTSTypeDeepgram\|TTSTypeFish\|TTSTypeGroq\|TTSTypeHume\|TTSTypeInworld\|TTSTypeLMNT\|TTSTypeMiniMax\|TTSTypeNeuphonic\|TTSTypeNvidiaRiva\|TTSTypeOpenAI\|TTSTypePiper\|TTSTypePlayHT\|TTSTypeRime\|TTSTypeSarvam\|TTSTypeXTTS\|STTTypeCartesia\|STTTypeElevenLabs" bin-ai-manager --include=*.go | grep -v _test.go`

## Commit/PR
- Branch/commit title: NOJIRA-Prune-ai-tts-stt-enum-to-runtime
- Body bullets prefixed bin-ai-manager: / bin-openapi-manager:
- No AI attribution. Squash merge only, after explicit pchero instruction.
