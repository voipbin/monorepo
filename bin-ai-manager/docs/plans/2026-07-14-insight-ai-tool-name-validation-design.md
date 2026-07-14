# VOIP-1257: Enforce tool_names restriction for ai.Type=insight

## Origin

VOIP-1234 introduced `ai.Type` (`normal` / `insight`) with a documented restricted
tool set for Insight AIs (`tool.AllInsightToolNames` in
`bin-ai-manager/models/tool/main.go`). VOIP-1257 (filed 2026-07-14, found during
square-admin VOIP-1234 follow-up work) confirms that neither the backend nor the
OpenAPI spec actually enforces this restriction: an admin can save an
`ai.Type=insight` AI with `tool_names=["connect_call", "send_email", ...]` (any
Normal-AI tool), and nothing stops a `Normal`-type AI from being given the two
Insight-only tools (`get_contact_interactions`, `get_conversation_content`)
either, once those are implemented.

CPO decision (2026-07-14, pchero approved): reject invalid combinations with a
400, rather than silently filtering `tool_names` on write. Silent filtering
would make the stored value diverge from what the client submitted, which
contradicts this codebase's "code is the single source of truth, write-through
not silent correction" convention, and would make a square-admin bug report
("I checked this tool but it disappeared") the normal outcome instead of an
immediate, explicit error.

## Current state (verified)

- `bin-ai-manager/models/tool/main.go`:
  - `AllToolNames` — the 15 Normal-AI tools (`connect_call`, `create_call`,
    `get_variables`, `get_aicall_messages`, `send_email`, `send_message`,
    `set_variables`, `stop_flow`, `stop_media`, `stop_service`,
    `search_knowledge`, `get_correlation`, `get_resource`, `describe_action`,
    `case_create`).
  - `AllInsightToolNames` — `get_contact_interactions`, `get_conversation_content`
    (both **not yet implemented**: no `toolDefinitions` entry, no execution
    handler; VOIP-1234 TODO). This list is currently unused outside its own
    declaration and doc comments referencing it.
  - `ToolNameAll = "all"` — a wildcard sentinel expanding to "every tool" at
    read time (`toolhandler.GetByNames`) and to "every conversation-safe tool"
    at `toolhandler.FilterToolsForConversation` time. It is not itself a
    per-type restriction concept.
- `bin-ai-manager/pkg/aihandler/chatbot.go`:
  - `Create()` validates `engineModel`, `aiType.IsValid()`, `ttsType`, `sttType`,
    `vadConfig` — but passes `toolNames` straight to `h.dbCreate(...)` with zero
    type-conditional check.
  - `Update()` resolves `aiType` (defaulting to the pre-update value when the
    caller omits `type`, never silently downgrading Insight → Normal on a
    partial PUT) — same zero-check pass-through for `toolNames`.
- `bin-openapi-manager/openapi/paths/ais/main.yaml` (and `id.yaml`, same shape):
  `tool_names` is a bare array of `AIManagerToolName` with no per-type
  constraint documented; `type` enum documents Insight's *intent* to restrict
  tools but not the actual mechanism.
- No other write path exists for `ai.ToolNames` — `bin-api-manager`'s
  `AICreate`/`AIUpdate` servicehandler pass `toolNames` through unchanged (see
  `bin-api-manager/pkg/servicehandler/ai.go`), so a single validation point in
  `bin-ai-manager` (the service of record for the `ai_ais` table) covers both
  the REST surface and any other RPC caller.

## Scope decision

Enforce the restriction at write time in `bin-ai-manager/pkg/aihandler`
(`Create` and `Update`), returning a typed `400 INVALID_ARGUMENT` via
`cerrors.InvalidArgument` (the pattern already used by `bin-agent-manager`,
`bin-queue-manager`, `bin-contact-manager` for this exact kind of validation
error; `bin-ai-manager`'s existing `chatbot.go` validations predate this
convention and use bare `fmt.Errorf`, which is an existing gap — see
"Out of scope" below).

### Validation rule

Given the AI's resolved `Type` (post-default-resolution, i.e. `Create`'s
`TypeNone → TypeNormal` fallback and `Update`'s "keep existing type if
omitted" resolution have already run):

- **`Type = insight`**: every name in `toolNames` MUST be a member of
  `tool.AllInsightToolNames`. `tool.ToolNameAll` is REJECTED for Insight AIs —
  "all" is defined today as "all Normal tools" (see
  `toolhandler.FilterToolsForConversation`), so allowing it for Insight would
  either silently mean nothing (0 tools, since none of `AllToolNames` are
  Insight tools) or require redefining what "all" means per-type. Rejecting it
  outright with a clear error is simpler and avoids a second silent-surprise
  path in the same ticket that is trying to remove one.
- **`Type = normal`**: every name in `toolNames` MUST be a member of
  `tool.AllToolNames` OR be `tool.ToolNameAll`. Insight-only tool names
  (`get_contact_interactions`, `get_conversation_content`) are REJECTED for
  Normal AIs — this covers the ticket's "vice versa" requirement pre-emptively,
  ahead of those two tools actually being implemented, so the constraint does
  not need a second follow-up ticket later.
- An empty/nil `toolNames` is always valid for either type (an AI with no
  tools enabled).
- Any name that is neither a known `AllToolNames` member, a known
  `AllInsightToolNames` member, nor `ToolNameAll` is ALSO rejected by this same
  check (closes a pre-existing gap where a typo'd/unknown tool name was
  silently accepted and stored) — one unified "is this name allowed for this
  type" check rather than a separate unknown-name check plus a separate
  type-restriction check.

### Where the check lives

New exported function in `bin-ai-manager/models/ai` (not `models/tool`, to
avoid a circular import — `ai` already imports `tool`):

```go
// ValidateToolNames returns an error if any name in toolNames is not permitted
// for the given (already-resolved) Type.
func ValidateToolNames(t Type, toolNames []tool.ToolName) error
```

Returns a plain `error` (not a `*cerrors.VoipbinError`) so the function stays
testable without importing `cerrors` into a pure model-validation helper;
`chatbot.go`'s call site wraps it into `cerrors.InvalidArgument(...)` with the
offending name(s) in the message, consistent with how call sites elsewhere in
the monorepo construct their own `cerrors.VoipbinError` rather than have the
validated-against model package do it.

**Message format must name both halves, per this codebase's own convention.**
Every other `cerrors.InvalidArgument` call site checked (e.g.
`bin-queue-manager/pkg/queuehandler/create.go:77`:
`"unsupported routing_method %q: only %q is supported"`;
`bin-contact-manager/pkg/contacthandler/resolution.go:42`:
`"resolution_type must be %q or %q, got %q"`) states BOTH the offending value
AND what is actually valid, in one sentence, so the client can fix the
request without a second round trip. `ValidateToolNames`'s error message must
follow the same shape, e.g. for an Insight AI:
`"invalid tool_names for type=insight: \"send_email\" is not an Insight tool
(valid: get_contact_interactions, get_conversation_content)"` — not just
`"invalid tool names: send_email"`.

Call sites: `aihandler.Create()` right after the existing `aiType.IsValid()`
check; `aihandler.Update()` right after `aiType` resolution (both the
`TypeNone` fallback branches and the `IsValid()` check), before the
`promptChanged`/`promptCleared`/default switch — so an invalid combination is
rejected before any DB write or prompt-history side effect, on both the
create path and every update path (prompt-changed, prompt-cleared, and
prompt-unchanged branches all call through the same validated `toolNames`).

### OpenAPI spec

Update `tool_names` field description in both
`bin-openapi-manager/openapi/paths/ais/main.yaml` and `.../ais/id.yaml` to
state the constraint explicitly, e.g.: "List of tool names to enable for this
AI. For `type=insight` AIs, only Insight tool names are permitted (currently:
`get_contact_interactions`, `get_conversation_content`); `type=normal` AIs may
use any Normal tool name or `[\"all\"]`. Mismatched combinations are rejected
with a 400."

**Codegen follow-through is mandatory, not optional prose editing.**
`oapi-codegen` propagates a schema's `description:` verbatim into the
generated Go doc-comment (verified: the current description string appears
unchanged as the doc-comment above `ToolNames *[]AIManagerToolName` at
`bin-openapi-manager/gens/models/gen.go:7713-7714` and `:7758-7759`). Editing
the YAML description without regenerating leaves the committed `gen.go` doc-
comment stale/out of sync with the spec. Per `bin-openapi-manager/CLAUDE.md`'s
codegen pipeline and `bin-api-manager/CLAUDE.md`'s "Code generation (after
OpenAPI spec changes)" section, the required sequence after editing the two
YAML files is:
1. `cd bin-openapi-manager && go generate ./...` (regenerates
   `gens/models/gen.go`).
2. `cd bin-api-manager && go generate ./...` (regenerates any dependent
   server code).
3. Commit the YAML changes together with both regenerated diffs — do not
   ship a YAML-only diff.

### RST docs

`bin-api-manager/docsdev/source/ai_struct_ai.rst` already documents `type`'s
restriction intent (line 48, quoted in the ticket) — add one sentence to the
`tool_names` field's own doc entry (not just the `type` entry) cross-referencing
the same constraint, since a docs reader skimming `tool_names` directly should
not have to also read the `type` entry to learn this.

## Out of scope (explicitly, so this ticket doesn't scope-creep)

- **square-admin UI wiring** (auto-narrow/disable non-permitted tool
  checkboxes when Insight is selected): ticket explicitly calls this out as "a
  distinct follow-up now that the enforcement gap is confirmed" — this design
  covers only the backend + spec enforcement. A UI-side ticket should be filed
  separately once this backend PR merges, so square-admin can surface the new
  400 as a client-side validation instead of a raw API error.
- **Retrofitting `chatbot.go`'s pre-existing bare `fmt.Errorf` validations**
  (invalid `engineModel`, invalid `ttsType`/`sttType`, invalid `vadConfig`) to
  the typed `cerrors.InvalidArgument` pattern. These currently fall through
  `errorResponse()`'s default branch to a bare 500
  (`bin-ai-manager/pkg/listenhandler/main.go`'s `errorResponse`: only a typed
  `*cerrors.VoipbinError` or `dbhandler.ErrNotFound` gets special-cased; a
  plain `fmt.Errorf` falls to `http.StatusInternalServerError`). This is a
  pre-existing bug affecting 4 other fields, not something VOIP-1257 introduced
  or is scoped to fix — flagging here so it doesn't get "discovered" mid-review
  and silently expand this PR's diff. Worth its own ticket.
- **Existing AIs already violating the rule** (created/updated before this
  change ships): no backfill/migration is required to make old rows conform.
  However, this is NOT purely a data-hygiene concern — see the paragraph
  below on customer-visible impact, which requires a pre-launch audit step
  (not a "nice to have").
- **Backward compatibility for existing customers (real production risk,
  CPO-acknowledged).** VoIPBin is a live production CPaaS. A customer who
  already has a saved `ai.Type=insight` AI with non-Insight `tool_names`
  (legally created today, since no validation exists yet) will get a new 400
  on their very next `PUT /ais/{id}` call that echoes `tool_names` back
  unchanged — e.g. a client that always resends the full current state when
  changing just `name` or `detail` (a common pattern, and arguably forced by
  this same PUT endpoint's full-overwrite semantics documented above). This
  is a real behavior change for existing customers, distinct from "no
  backfill of DB rows."
  **Decision (CPO): ship the audit query as a REQUIRED pre-launch step, not
  an optional follow-up.** Before this PR is deployed to production, run a
  read-only query (`SELECT id, customer_id, tool_names FROM ai_ais WHERE type
  = 'insight' AND deleted_at IS NULL` and filter for tool_names not in
  `AllInsightToolNames`) against the production DB to enumerate any
  already-violating rows. If any exist, contact the affected customer(s)
  proactively (`support@voipbin.net` outreach) before the enforcement ships,
  rather than letting them discover it via a surprise 400. If the audit
  returns zero rows (likely, since Insight AIs are a very recent, low-adoption
  feature per VOIP-1234), this step is a quick confirmation, not a blocker.
  This audit must run as part of the PR's rollout checklist, not be deferred
  indefinitely as originally implied.
- **`tool_names` omission-on-PUT causing a silent wipe, and PUT's overall
  full-overwrite semantics.** Traced during round-2 design review:
  `V1DataAIsIDPut.ToolNames` (`bin-ai-manager/pkg/listenhandler/models/request/ais.go:61`)
  is a plain (non-pointer) slice with `json:"tool_names,omitempty"`. If a PUT
  body omits `tool_names`, it unmarshals to `nil`, which
  `aihandler.Update` passes straight through to `buildUpdateFields`
  (`db.go:210-245`), which unconditionally sets
  `ai.FieldToolNames: toolNames` in the field map handed to `AIUpdate` on
  **every** branch (`promptChanged` at `chatbot.go:158`, `promptCleared` at
  `chatbot.go:182`, default at `db.go:196`) — there is no "only include if the
  caller supplied it" merge. So a PUT that omits `tool_names` today already
  silently clears an AI's tool list to empty, independent of this ticket.
  This is **not unique to `tool_names`** — `name`, `detail`, `engine_model`,
  `parameter`, `engine_key`, `tts_type`, `tts_voice_id`, `stt_type`,
  `stt_language`, `vad_config`, `smart_turn_enabled`, and
  `auto_aicall_audit_enabled` are all plain (non-pointer, `omitempty`) fields
  in `V1DataAIsIDPut` with the identical unconditional-overwrite treatment in
  `buildUpdateFields`. Only `type` (`preUpdateAI.Type` fallback,
  `chatbot.go:139-141`) and `init_prompt` (the `promptChanged`/`promptCleared`
  switch) get "omit = keep existing" handling. In other words, `PUT
  /ais/{id}` is, and has always been, a full-overwrite endpoint for nearly
  every field, not a merge-patch — this predates VOIP-1257 and is not
  something this ticket introduces or worsens.
  Since `ValidateToolNames` treats nil/empty as valid for either `Type`
  (per the rule above), this new check will not catch or warn about a
  client silently wiping an Insight AI's tools via an incomplete PUT body —
  it will simply pass. This is a real, pre-existing, endpoint-wide
  full-overwrite-semantics risk, but it is **orthogonal to and out of scope
  for VOIP-1257** (fixing it would mean redesigning `PUT /ais/{id}` toward
  partial-PATCH semantics for every field, a materially larger change).
  Flagging it here so it is a documented, deliberate exclusion rather than a
  gap discovered later. Recommend a separate ETC/VOIP ticket if pchero wants
  `PUT /ais/{id}` to adopt partial-update (pointer-field) semantics platform-
  wide; VOIP-1257 does not attempt that.
- **Blast radius (verified, round-2 review): no other in-repo call site
  constructs an `ai.AI{Type: ai.TypeInsight, ToolNames: ...}` combination.**
  Grepped the full worktree (excluding `vendor/`, `.worktrees/`, `mock_*`).
  The only non-write `ai.TypeInsight` literal usages are
  `bin-ai-manager/pkg/teamhandler/handler_test.go:229,265`
  (`&ai.AI{Type: ai.TypeInsight}`, no `ToolNames` set — nil, stays valid) and
  `bin-ai-manager/pkg/teamhandler/handler.go:215` (a read, not a write). No
  seed script, fixture, or dbhandler test helper needs updating alongside
  this change.

## Testing plan

- `bin-ai-manager/models/ai` unit tests for `ValidateToolNames`: table-driven,
  covering — normal+normal-tools (ok), normal+all (ok), normal+insight-tool
  (reject), normal+unknown-name (reject), insight+insight-tools (ok),
  insight+all (reject), insight+normal-tool (reject), insight+unknown-name
  (reject), nil/empty toolNames for both types (ok), duplicate tool names in
  the same list (e.g. `["connect_call","connect_call"]` for Normal — ok, no
  dedup required by the rule), `["all","get_contact_interactions"]` for
  Insight (reject — `all` alone is invalid for Insight, so the mixed list is
  also invalid; guards against an implementation that special-cases `all`
  membership and short-circuits before checking the rest of the list), and
  `["all","bogus_tool"]` for Normal (reject — `all` is valid for Normal but
  the unknown name is not; guards against a loop that returns success on the
  first valid element instead of checking every element).
- `bin-ai-manager/pkg/aihandler` existing `chatbot_test.go` table-driven tests:
  add cases asserting `Create`/`Update` return an error (checked via
  `errors.As` to `*cerrors.VoipbinError` with `Status == cerrors.StatusInvalidArgument`)
  when an invalid type/tool_names combination is submitted, and that valid
  combinations still succeed (regression guard against the new check being
  overly strict).
