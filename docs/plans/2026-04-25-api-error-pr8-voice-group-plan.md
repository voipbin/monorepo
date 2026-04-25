# PR 8 — Voice group handler migration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the speakings (TTS), transcribes (STT), and transcripts handlers to the canonical error envelope. 13 handlers, 42 sites across 3 files. **No 402, no 409, no 503** modifiers — verified upstream behavior (see "Modifier reachability checks" below).

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 7 (`NOJIRA-api-error-pr7-ai-group`, merged `bc9b2270a`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr8-voice-group`
**Branch:** `NOJIRA-api-error-pr8-voice-group` (branched from `origin/main` at `bc9b2270a`)

## Scope

| File | LoC | Sites | Handlers |
|---|---|---|---|
| `server/speakings.go` | 324 | 26 | 7 |
| `server/transcribes.go` | 209 | 14 | 5 |
| `server/transcripts.go` | 58 | 2 | 1 |

**Total: 42 sites across 13 handlers in 3 files.**

### Handler classification (per §6.1)

**Read (no path param) → 401, 500:**
- `GetSpeakings`, `GetTranscribes`, `GetTranscripts`

**Read (with resource ID) → 400, 401, 403, 404, 500:**
- `GetSpeakingsId`, `GetTranscribesId`

**Write (no resource ID) → 400, 401, 500:**
- `PostSpeakings` — creates a TTS speaking session for an active call.
- `PostTranscribes` — creates an STT transcription session for an active call.

**Write (with resource ID) → 400, 401, 403, 404, 500:**
- `DeleteSpeakingsId`, `DeleteTranscribesId`
- `PostSpeakingsIdFlush`, `PostSpeakingsIdSay`, `PostSpeakingsIdStop`
- `PostTranscribesIdStop`

## Modifier reachability checks

### 402 PaymentRequired — NOT WIRED

`grep -rE "insufficient|enough.balance|has not.*balance" bin-tts-manager/pkg/ bin-transcribe-manager/pkg/` returns zero matches. While TTS character usage and STT seconds are conceptually billable, neither manager currently emits a balance error. Per PR 6 Round 1 lesson, do NOT declare 402 on endpoints whose upstream managers don't actually emit `"insufficient"`-style errors. Defer 402 wiring to a future PR that also adds the balance pre-check.

### 409 Conflict — NOT WIRED

The `Stop` operations are **idempotent** in both managers:
- `bin-tts-manager/pkg/speakinghandler/speaking_test.go:743` — test case `"already stopped idempotent"` confirms tts-manager returns success on already-stopped sessions.
- `bin-transcribe-manager/pkg/transcribehandler/stop.go:30,87` — comments explicitly say "already stopped" and continue without error.

So `PostSpeakingsIdStop` and `PostTranscribesIdStop` do not surface 409 today. Do not declare 409. (Contrast with PR 3's `PostActiveflowsIdStop`, which DID surface state errors.)

### 503 ServiceUnavailable — NOT WIRED

Single-manager hops (api-manager → tts-manager or transcribe-manager). No fan-out justifying 503.

## Forward-dependency notes

- The remaining `bin-api-manager/server/` files after PR 8 still include: `accesskeys.go`, `agents.go`, `aggregated_events.go`, `calls.go` (partial), `campaigncalls.go`, `campaigns.go`, `conferencecalls.go`, `conferences.go`, `contacts.go`, `extensions.go`, `outdials.go`, `outplans.go`, `queuecalls.go`, `queues.go`, `service_agents_*.go` (multiple), `storage_files.go`, `tags.go`, `teams.go`, `timelines.go`, `timelines_sip.go`, `transcripts.go` (after PR 8 only `transcribes.go` and `speakings.go` will be done; transcripts is included in PR 8), `ws.go`. These are out-of-scope for PR 8 and require follow-up PRs.
- **Voice catalog completion:** PR 8 adds two new manager sections (`tts-manager`, `transcribe-manager`).

## Tasks

### Task 1: Plan doc + commit (this file)

### Task 2: Migrate `server/speakings.go` (7 handlers, 26 sites)

Path-UUID hardening for `GetSpeakingsId`, `DeleteSpeakingsId`, `PostSpeakingsIdFlush`, `PostSpeakingsIdSay`, `PostSpeakingsIdStop` (string IDs).

Sample tests in `server/speakings_test.go` (file already exists — append):
- `Test_speakingsPost_MissingAuthIdentity`
- `Test_speakingsPost_InvalidJSONBody`
- `Test_speakingsIDDelete_InvalidID`
- `Test_speakingsIDSayPost_InvalidJSONBody`

### Task 3: Migrate `server/transcribes.go` (5 handlers, 14 sites)

Path-UUID hardening for `GetTranscribesId`, `DeleteTranscribesId`, `PostTranscribesIdStop`.

Sample test in `server/transcribes_test.go`:
- `Test_transcribesIDStopPost_InvalidID`

### Task 4: Migrate `server/transcripts.go` (1 handler, 2 sites)

Trivial — `GetTranscripts` is read-no-id (401, 500).

### Task 5: OpenAPI path wiring

Files under `bin-openapi-manager/openapi/paths/`:

- `speakings/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `speakings/id.yaml` — GET, DELETE (400, 401, 403, 404, 500)
- `speakings/id_flush.yaml`, `speakings/id_say.yaml`, `speakings/id_stop.yaml` — POST (400, 401, 403, 404, 500)
- `transcribes/main.yaml` — GET (401, 500), POST (400, 401, 500)
- `transcribes/id.yaml` — GET, DELETE (400, 401, 403, 404, 500)
- `transcribes/id_stop.yaml` — POST (400, 401, 403, 404, 500)
- `transcripts/main.yaml` — GET (401, 500)

**No 402, no 409 anywhere.**

Reference yaml `$ref` style from PR 7's `ais/main.yaml` (no 402).

Regenerate `gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 6: RST catalog updates

Edit `bin-api-manager/docsdev/source/restful_api_errors.rst`:

1. **Add `tts-manager` section** (new):
   - `SPEAKING_NOT_FOUND` (404) — for `/speakings/{id}`.
   - Note: `POST /speakings/{id}/stop` is idempotent today; future state-transition typing may add 409 with reason `SPEAKING_STATE_INVALID`.

2. **Add `transcribe-manager` section** (new):
   - `TRANSCRIBE_NOT_FOUND` (404) — for `/transcribes/{id}`.
   - Note: `POST /transcribes/{id}/stop` is idempotent today; same future note applies.

3. **Update deferred-billing list** (in `billing-manager` INSUFFICIENT_BALANCE description, line ~162):
   - Add `POST /speakings` and `POST /transcribes` to the "Future endpoints (deferred)" list — both conceptually billing-sensitive (TTS character cost, STT second cost) but pending balance pre-check wiring upstream.

Match disclaimer style from PR 4-7.

Rebuild Sphinx HTML.

### Task 7: Full verification

Standard 5-step workflow.

### Task 8: Push + open PR

Conflict check, push, open PR.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr8-voice-group` on every commit.
- No AI attribution.
- `commonoutline.ServiceName*` constants.
- 3-group import layout in test files.
- Reuse `assertMissingAuthIdentity` and `assertErrorResponse` helpers.
- Path-UUID hardening pattern: `uuid.FromStringOrNil(id) == uuid.Nil`.
- `getAuthIdentity(c)` (NOT `commonmiddleware.AuthIdentityGet`).
- `abortWithError` takes `*cerrors.VoipbinError`.
- Standard message strings: "The request body is not valid JSON.", "The provided id is not a valid UUID.", "Authentication is required."
- Preserve existing `log.Errorf/Infof` site-specific lines.

## Success criteria

- All 8 tasks committed
- `go test -race ./...` green for both `bin-api-manager` and `bin-openapi-manager`
- `golangci-lint` 0 issues
- Loose `ServerInterface` signatures unchanged
- New tts-manager and transcribe-manager catalog sections
- Billing-manager deferred-list extended with `/speakings` and `/transcribes`
- All 42 sites converted
- Zero 402 / 409 declarations added
