# get_resource LLM Tool Design

- Date: 2026-06-11
- Service: bin-ai-manager (toolhandler / aicallhandler)
- Status: v6 (rounds 1-3 CHANGES REQUESTED applied; round 4 APPROVE; post-approval design change: aicall history INCLUDED per pchero — fresh re-review round pending)

## 1. Problem Statement

The `get_correlation` tool returns the correlation graph for an activeflow: an ID
graph of related resources grouped by publisher (`call-manager: call <uuid>`,
`transcribe-manager: transcribe <uuid>`, ...). It deliberately stops at IDs.

There is no follow-up tool to retrieve the content of those resources. The
read tools today are `get_aicall_messages` (single-purpose: raw JSON dump of
one aicall's messages — it DOES accept an arbitrary aicall_id with a
customer-ownership check, but returns the unfiltered marshal including system
prompts, and covers no other resource type), `get_variables`, and
`search_knowledge` (RAG). An LLM that discovers a transcribe id via
`get_correlation` cannot answer "what was said in that call" because it has no
way to fetch the transcript. The chaining mechanism (run_llm=true sequential
tool calling) exists and works; the missing piece is the second link.

Goal: add a generic read tool `get_resource(resource_type, resource_id)` that
fetches a single VoIPBin resource by id and returns a curated, LLM-readable
text summary of its content.

## 2. Scope

### In scope (Phase 1)

One new LLM function tool in bin-ai-manager. Supported resource types, all of
which already have a single-resource Get RPC in bin-common-handler
requesthandler and embed `commonidentity.Identity` (direct `CustomerID`):

| resource_type | RPC | Source model |
|---|---|---|
| `call` | `CallV1CallGet` | bin-call-manager/models/call |
| `groupcall` | `CallV1GroupcallGet` | bin-call-manager/models/groupcall |
| `recording` | `CallV1RecordingGet` | bin-call-manager/models/recording |
| `transcribe` | `TranscribeV1TranscribeGet` + `TranscribeV1TranscriptList` | bin-transcribe-manager/models/transcribe, transcript |
| `summary` | `AIV1SummaryGet` | bin-ai-manager/models/summary |
| `aicall` | `AIV1AIcallGet` | bin-ai-manager/models/aicall |
| `conferencecall` | `ConferenceV1ConferencecallGet` | bin-conference-manager/models/conferencecall |
| `queuecall` | `QueueV1QueuecallGet` | bin-queue-manager/models/queuecall |

The `resource_type` values match each resource's event-type prefix as seen in
`get_correlation` output (`call_created` → `call`, `transcribe_done` →
`transcribe`, `aicall_status_progressing` → `aicall`, ...). NOTE (review round
1, code-verified): the correlation graph's per-resource `DataType` field is NOT
the model name — `notifyhandler.PublishEvent` stamps every event envelope with
`data_type = "application/json"`, and timeline materializes that verbatim. The
LLM therefore derives the type from the event-type prefix (which
`formatCorrelationSummary` prints per resource), not from `data_type`. The
chosen enum values (`call`, `groupcall`, `recording`, `transcribe`, `summary`,
`aicall`, `conferencecall`, `queuecall`) are exactly the event-type prefixes of
the eight publishers, verified against each `models/<type>/event.go`. The tool
description spells the mapping out. A follow-up note: `formatCorrelationSummary`
currently prints `r.DataType` per line, which renders as `application/json` —
fixed IN THIS PR (see Open Question 5 and Implementation Order step 5b).

### Out of scope (Phase 2+ or never)

- `conversation`, `campaigncall`, `message` (SMS), `email`: not part of the
  correlation graph today for SMS/transfer (no activeflow_id), or lower
  diagnostic value. Add per demand; the dispatch table makes each addition
  ~30 lines + tests.
- `confbridge`, `channel`, `externalmedia`: internal plumbing resources.
  Exposing them to an LLM has no customer-facing value and widens the surface.
- `activeflow` itself: flow internals (current action, stack) are execution
  state, not customer data. Revisit only with a concrete use case.
- Recording media download / playback URLs. Metadata only.
- List/search semantics ("all calls of last week"): this tool is strictly
  single-resource point lookup. Listing is a different feature with paging and
  cost concerns.
- No new RPC, no DB change, no OpenAPI change, no RST docs change (internal
  RPC-exposed tool only; tool definitions are auto-exposed via the existing
  `GET /v1/tools`).

## 3. Tool Definition

### models/tool/main.go

```go
ToolNameGetResource ToolName = "get_resource"
// + append to AllToolNames (both — TestAllToolNames hardcodes the set)
```

### models/message/tool.go

```go
FunctionCallNameGetResource FunctionCallName = "get_resource"
```

### pkg/toolhandler/definitions.go

```go
{
    Name:   tool.ToolNameGetResource,
    RunLLM: true,
    Description: `Retrieves the content of a single VoIPBin resource by its id and returns a readable summary.

Use this as the follow-up to get_correlation: get_correlation returns the ids and types of resources linked to an activeflow; get_resource fetches the actual content of one of them. An activeflow's reference is not always a call; do not assume the session is a phone call.

Supported resource types: call, groupcall, recording, transcribe, summary, aicall, conferencecall, queuecall. Derive the type from the event names shown by get_correlation: the type is the leading part of the event name (call_created means type call, transcribe_done means type transcribe, aicall_status_progressing means type aicall). Not every type get_correlation lists is retrievable here; unsupported types return an error listing the supported set. Transcript entries are retrieved via their parent transcribe id (type transcribe), not their own id. For transcribe, the response includes the transcript messages. For aicall, the response includes the session's conversation messages.

WHEN TO USE:
- You discovered a resource id (e.g. via get_correlation) and need its details
- A diagnostic question requires the content of a related resource (e.g. what was said in a transcribed call, why a call ended, how long a caller waited in a queue)

WHEN NOT TO USE:
- A raw, unfiltered JSON dump of an aicall's messages (use get_aicall_messages; get_resource returns a curated readable summary)
- Runtime variables (use get_variables)
- Knowledge-base questions (use search_knowledge)

ARGUMENTS:
- resource_type (required): one of the supported types above.
- resource_id (required): the resource id (UUID).

You can only retrieve resources owned by your own account; others return "Resource not found.". A wrong resource_type for a correct id also returns "Resource not found." — retry with the type matching the event prefix before concluding the resource is gone.

run_llm: Set true so you can reason about the resource content.`,
    Parameters: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "run_llm": map[string]any{
                "type":        "boolean",
                "description": "Set true to reason about the resource content.",
                "default":     true,
            },
            "resource_type": map[string]any{
                "type":        "string",
                "enum":        []string{"call", "groupcall", "recording", "transcribe", "summary", "aicall", "conferencecall", "queuecall"},
                "description": "The type of the resource to retrieve.",
            },
            "resource_id": map[string]any{
                "type":        "string",
                "description": "The resource id (UUID) to retrieve.",
            },
        },
        "required": []string{"resource_type", "resource_id"},
    },
},
```

Both arguments are required. Unlike `get_correlation` there is no own-session
default: the tool is meaningless without a concrete target, and an LLM that
wants "this session" should use `get_correlation` first.

## 4. Handler Design (pkg/aicallhandler/tool_resource.go)

### Dispatch entry

Add `message.FunctionCallNameGetResource: h.toolHandleGetResource` to the
`mapFunctions` map literal inside `ToolHandle` (a local literal rebuilt per
call, not a package-level map).

The existing `promAIcallToolExecuteTotal` metric auto-covers the new function
name label. No new metrics.

### Fetcher table

Two-stage fetcher per resource type. Stage 1 fetches ONLY the primary resource
(which carries `CustomerID`); stage 2 renders the summary and performs any
enrichment RPC (transcript list). Stage 2 runs strictly AFTER the ownership
check, so no foreign content (especially transcript text) is ever fetched or
rendered for a cross-customer id (review round 1, Critical 1):

```go
// resourceFetchResult carries the owner of a fetched resource and a render
// closure. ownerID is valid only when err == nil. render is invoked by the
// caller ONLY after ownership has been validated; any enrichment RPC
// (e.g. transcript list) happens inside render, never before.
type resourceFetchResult struct {
    ownerID uuid.UUID
    render  func(ctx context.Context) string
}

type resourceFetcher func(ctx context.Context, h *aicallHandler, id uuid.UUID) (*resourceFetchResult, error)

var mapResourceFetchers = map[string]resourceFetcher{
    "call":           fetchResourceCall,
    "groupcall":      fetchResourceGroupcall,
    "recording":      fetchResourceRecording,
    "transcribe":     fetchResourceTranscribe,
    "summary":        fetchResourceSummary,
    "aicall":         fetchResourceAIcall,
    "conferencecall": fetchResourceConferencecall,
    "queuecall":      fetchResourceQueuecall,
}
```

The JSON-schema `enum`, the unsupported-type error string, and the description's
supported-type list are all derived from (or asserted equal to) the sorted map
keys by a unit test, so Phase 2 additions cannot drift (review round 1, Low 9).
The error-string source is a single derived value:
`var supportedResourceTypes = strings.Join(sortedFetcherKeys(), ", ")`.

### Main handler flow

```go
func (h *aicallHandler) toolHandleGetResource(ctx, c, tc) *messageContent {
    log := logrus.WithFields(logrus.Fields{
        "func":      "toolHandleGetResource",
        "aicall_id": c.ID,
    })
    res := newToolResult(tc.ID)

    var args struct {
        ResourceType string `json:"resource_type"`
        ResourceID   string `json:"resource_id"`
    }
    if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
        // Pass the unmarshal error through (sibling precedent:
        // toolHandleSearchKnowledge) so the LLM can self-correct its arguments.
        fillFailed(res, errors.Wrap(err, "invalid arguments"))
        return res
    }

    if args.ResourceType == "" {
        fillFailed(res, fmt.Errorf("resource_type is required. supported: %s", supportedResourceTypes))
        return res
    }
    fetcher, ok := mapResourceFetchers[args.ResourceType]
    if !ok {
        fillFailed(res, fmt.Errorf("unsupported resource_type: %s. supported: %s", args.ResourceType, supportedResourceTypes))
        return res
    }

    resourceID, err := uuid.FromString(args.ResourceID)
    if err != nil || resourceID == uuid.Nil {
        fillFailed(res, fmt.Errorf("invalid resource_id"))
        return res
    }

    summary, err := h.resolveResource(ctx, c.CustomerID, fetcher, resourceID)
    if err != nil {
        if stderrors.Is(err, ErrResourceNotAccessible) {
            // Single masking site for ALL not-accessible paths.
            // CRITICAL: this return is load-bearing (IDOR if fallen through).
            fillSuccess(res, args.ResourceType, resourceID.String(), msgResourceNotFound)
            return res
        }
        // Transient/infra failure: existence unknown, report honest tool failure.
        // CRITICAL: this return is load-bearing (nil deref if fallen through).
        log.Errorf("Resource lookup failed. err: %v", err) // operator signal; cause never reaches the LLM
        fillFailed(res, fmt.Errorf("resource lookup failed"))
        return res
    }

    fillSuccess(res, args.ResourceType, resourceID.String(), summary)
    return res
}
```

### Ownership/masking helper

Mirrors the `resolveCorrelation` pattern (PR #975): error-returning helper,
sentinel collapses every "cannot see" path to one masking site.

```go
// msgResourceNotFound is the single masked response for every path where the
// caller is not allowed to see the resource: absent, cross-customer, or any
// state in between. All such paths MUST return this byte-identical string so
// the tool is not an existence oracle.
const msgResourceNotFound = "Resource not found."

// ErrResourceNotAccessible signals that the resource is absent or not owned by
// the caller. The caller masks it. Transient failures are returned as wrapped
// ordinary errors and must NOT be masked (existence is unknown; masking would
// make the LLM assert a false "not found").
var ErrResourceNotAccessible = stderrors.New("resource not accessible")

func (h *aicallHandler) resolveResource(ctx context.Context, callerCustomerID uuid.UUID, fetcher resourceFetcher, resourceID uuid.UUID) (string, error) {
    log := logrus.WithFields(logrus.Fields{
        "func":        "resolveResource",
        "customer_id": callerCustomerID,
        "resource_id": resourceID,
    })

    r, err := fetcher(ctx, h, resourceID)
    if err != nil {
        if stderrors.Is(err, requesthandler.ErrNotFound) {
            // Absent. Mask.
            return "", ErrResourceNotAccessible
        }
        // Transient: honest failure, no masking.
        return "", errors.Wrap(err, "could not fetch resource")
    }

    if r.ownerID != callerCustomerID || r.ownerID == uuid.Nil {
        // ownerID == uuid.Nil is treated as not-accessible (fail closed): a
        // row with unset customer_id must never match any caller.
        log.Warnf("Cross-customer resource access blocked. resource_owner: %s", r.ownerID)
        return "", ErrResourceNotAccessible
    }

    // Ownership validated: only now render (and run any enrichment RPC).
    return r.render(ctx), nil
}
```

### Error mapping contract (masked vs honest)

Exactly one requesthandler sentinel maps to the masked path:
`requesthandler.ErrNotFound` (upstream 404) → masked. Everything else
(`ErrInternalServerError`, timeouts, circuit-breaker, and any other 4xx such as
400) → honest `fillFailed`. Relationship to the resolveCorrelation precedent,
stated precisely so future edits don't "fix" the split in the wrong direction:
the TWO-TIER contract (not-accessible → masked sentinel; transient → honest
fillFailed) is IDENTICAL to the precedent. The only difference is which errors
land in the "not-accessible" tier: get_correlation additionally masks failures
of its SECOND rpc (the FlowV1ActiveflowGet ownership lookup) because by that
point the first RPC had already established existence, so an honest error
would leak it. Here there is no second lookup — the single fetch IS the
existence+ownership resolution, so a non-404 error carries no existence
information (the resource was never resolved) and is reported honestly.
Masking non-404 errors would convert infra blips into false "not found"
assertions by the LLM. All eight target managers map a missing row
(`dbhandler.ErrNotFound`)
to RPC 404 — verified in each service's `pkg/listenhandler/main.go` error
mapping (call, queue, transcribe, ai, conference; groupcall/recording ride
call-manager's listenhandler). Implementation re-verifies this per RPC as a
blocking step (Implementation Order step 0) and the per-type absent tests in
section 10 lock it as a regression guard.

Notes:

- `requesthandler.ErrNotFound` is the canned sentinel the shared RPC client
  returns for upstream 404 (`HttpStatusErrorMap`). Managers migrated to the
  typed VoipbinError envelope still funnel through `errors.Is` matching; the
  implementation must verify per target manager that a missing resource
  surfaces as `ErrNotFound` (all eight Get RPCs are plain 404 paths today).
- Soft-delete behavior follows each manager's RPC GET semantics; both outcomes
  are acceptable for masking (a 404 on a deleted row masks identically to
  absent; a returned deleted row still passes ownership). No per-manager
  matrix is asserted in this design (review round 1, Medium 6).
- The primary fetch happens BEFORE the ownership check by necessity (the
  resource carries its own customer_id). It is an internal RPC, not observable
  by the caller, and its content is never rendered for foreign owners; the
  render closure (including the transcript enrichment RPC) runs only after
  ownership passes.

## 5. Security

### IDOR

Every supported model embeds `commonidentity.Identity`, so ownership is a
direct field comparison: `resource.CustomerID != c.CustomerID` → reject. No
detour via flow-manager (unlike get_correlation, where timeline had no
customer_id). One comparison, one place (`resolveResource`).

### Existence oracle

All "cannot see" outcomes collapse to the byte-identical
`msgResourceNotFound` via the single sentinel:

| Path | Response |
|---|---|
| Resource absent (ErrNotFound) | masked `Resource not found.` |
| Resource exists, other customer | masked `Resource not found.` (same bytes) |
| Transient RPC failure | honest `fillFailed("resource lookup failed")` — NOT masked |

Transient failures are deliberately not masked: existence is unknown, and
masking would make the LLM confidently assert "that resource does not exist"
during an infra blip. This matches the resolveCorrelation two-tier error
contract. The residual timing difference (cross-customer does one successful
RPC before rejecting) is not measurable through the LLM boundary;
accept-with-note.

### tool_names:["all"] reality

Adding the definition exposes the tool to every AI configured with the `all`
wildcard. As with get_correlation, exposure is not the control; the ownership
check is. Do NOT add to `ConversationSafeTools` (unwired utility; adding would
auto-expose later).

### Masking-invariant regression test

A dedicated test asserts that the reject paths (absent, cross-customer)
produce `reflect.DeepEqual`-identical `messageContent`, so future edits cannot
split the masked responses apart and recreate the oracle.

## 6. Output Format (per-type curated summaries)

Raw `json.Marshal` of the domain structs is prohibited: the models carry
internal fields (channel_id, bridge_id, asterisk_id, dialroutes, host_id,
filenames, confbridge_id...) that are infrastructure detail, token waste, and
mild information leakage. Each fetcher renders a short labeled-line summary of
customer-meaningful fields only. Timestamps render RFC3339; nil timestamps and
zero UUIDs are omitted.

- **call**: status, direction, source/destination (target only), created,
  ringing/progressing/hangup timestamps, hangup_by, hangup_reason,
  recording_ids count, groupcall_id (if set).
- **groupcall**: status, ring_method, answer_method, source, destination count,
  answered call id, call_ids.
- **recording**: status, format, reference_type/reference_id, recording_name,
  tm_start, tm_end. No filenames (storage paths are internal).
- **transcribe**: status, language, direction, reference_type/reference_id,
  provider, followed by the transcript body via `TranscribeV1TranscriptList`
  (filters `FieldTranscribeID` + `FieldDeleted: false` — matching the
  bin-api-manager servicehandler precedent, which passes
  `{"transcribe_id": ..., "deleted": "false"}`; exact value encoding to be
  verified during implementation). The dbhandler seeds the empty page token
  with "now" and orders by `tm_create DESC` (code-verified), so a single page
  is the MOST RECENT transcripts. Page-cap detection: request `page_size 101`;
  if 101 rows return, more exist — render the most recent 100. The fetcher
  reverses the page to chronological order before rendering. Line format
  `[in 00:00:03] hello ...` using Direction (enum `in`/`out`/`both` — render
  the literal value) and the TMTranscript offset. TMTranscript encodes an
  offset from the zero time (`0001-01-01 00:00:00` = transcribe start, per the
  model comment): normalize via `t.UTC()` then render `t.Sub(time.Time{})`
  formatted `HH:MM:SS` (hours may exceed 24; render total hours; verify the
  scan location against a real row during implementation). Nil TMTranscript
  renders `[in --:--:--]`. Empty transcript list renders `(no transcripts)`.
  Same-millisecond `tm_create` ties have no tiebreaker in the query —
  rendering order within a tie is nondeterministic; tests must not depend on
  tie order. Truncation: both the page cap and the 4000-rune cap keep the MOST
  RECENT lines and drop the oldest; the marker template is
  `...(earlier transcripts omitted; showing the most recent N)` (degenerate
  single-line variant defined under cap mechanics below) — never
  "first N", which would invert which end was cut. Transcript list failure
  after a successful, ownership-validated transcribe fetch degrades
  gracefully: metadata + a `(transcripts unavailable)` line, not a tool
  failure.
- **summary**: status, language, reference_type/reference_id, content. The
  whole-message cap governs: content gets whatever budget remains after the
  metadata lines; marker `...(truncated)`.
- **aicall**: status, reference_type/reference_id, engine model, tts/stt type,
  tm_create/tm_end, followed by the CONVERSATION HISTORY (settled: include —
  pchero, 2026-06-11; transcribe/summary are different resources, and for
  chat/conversation-referenced aicalls the ai_messages table is the only
  record of what was said). Mechanics mirror the transcribe body: messages
  fetched inside the render closure (post-ownership) via the LOCAL
  `h.messageHandler.List` (same call shape as `toolHandleGetAIcallMessages`:
  filter `FieldAIcallID`, page_size 101 for page-cap detection — messages are
  local to bin-ai-manager, no RPC). Rendered roles: `user` and `assistant`
  content as `[user] ...` / `[assistant] ...` lines; an assistant message
  whose ToolCalls is non-empty renders a compact `[assistant called <name>]`
  line per call. EXCLUDED from rendering: `system` (substituted prompt
  snapshot — prompt engineering exposure has no place in a readable summary;
  the full raw dump including system is already available via
  get_aicall_messages, which validates the same ownership), `notification`,
  `tool` result bodies (bulk; the assistant's visible reply reflects them),
  and empty-content rows. Ordering/truncation identical to transcripts:
  most recent kept, chronological render, marker
  `...(earlier messages omitted; showing the most recent N)`.
- **conferencecall**: status, conference_id, reference_type/reference_id,
  timestamps.
- **queuecall**: status, queue_id, routing_method, duration_waiting,
  duration_service, service_agent_id, tm_create/tm_service/tm_end.

All copy uses domain-correct terminology: "activeflow" not "call flow"; no
phrasing that assumes the session is a phone call (multi-channel platform;
recurring de-bias rule).

The global cap `maxResourceSummaryRunes = 4000` (const, rune-counted as the
name says; code change to raise) applies to the WHOLE rendered message of
every type — metadata + body + marker together can never exceed it. Cap
mechanics, uniform across types:
- Truncation is whole-line for line-structured bodies (transcripts): drop
  oldest lines until the remainder + marker fits. The marker is placed at the
  TOP of the transcript block (where the omitted earlier lines were), before
  the chronological lines.
- Degenerate case: if even the newest single line cannot fit the remaining
  budget, hard-truncate that line mid-text and still show it; the transcript
  marker then reads `...(earlier transcripts omitted; showing the most
  recent 1, truncated)`.
- Free-text bodies (summary content) hard-truncate mid-text with marker
  `...(truncated)`.
- Any OTHER type whose rendered message exceeds the cap (e.g. a groupcall
  with hundreds of call_ids) hard-truncates with the same generic
  `...(truncated)` marker appended at the cut.

Note on self-RPC: `AIV1SummaryGet`/`AIV1AIcallGet` are bin-ai-manager calling
itself over RabbitMQ. Deliberate: one uniform fetcher shape, circuit breaker
included, no special-casing of local resources.

## 7. Chaining / UX

- `run_llm` defaults true: the tool exists to feed content back into the LLM.
- Expected chain: `get_correlation` (1 call) → `get_resource` (1 call per
  resource of interest). Depth 2 is acceptable: this is a diagnostic/back-office
  pattern, predominantly text channels. Voice sessions pay ~1 extra LLM+RPC
  round trip only when the AI actually chooses to drill down.
- `get_correlation`'s description gains one sentence pointing at
  `get_resource` as the follow-up ("Use get_resource to fetch the content of a
  discovered resource."), closing the loop for the LLM. Same PR.

## 8. Affected Services

| Service | Change | Phase |
|---|---|---|
| bin-ai-manager | tool constant + AllToolNames, FunctionCallName, definitions.go entry (+1 sentence on get_correlation), dispatch entry, `tool_resource.go` (handler + fetchers + formatters), `formatCorrelationSummary` label fix in tool.go (OQ5), tests | 1 |
| bin-common-handler | none (all 8 RPCs exist) | - |
| others | none | - |

Single-service PR. No schema, no migration, no OpenAPI, no RST.

## 9. Implementation Order

0. BLOCKING verification: per target RPC, confirm a missing resource surfaces
   as `requesthandler.ErrNotFound` (trace each manager's listenhandler
   error mapping; design-time check found all five listenhandlers map
   `dbhandler.ErrNotFound` → 404). Confirm `FieldTranscribeID` filter and
   `tm_create DESC` ordering of `TranscribeV1TranscriptList`. Confirm the
   `FieldDeleted` value encoding (string "false" per the bin-api-manager
   precedent vs typed bool). Confirm the dbhandler accepts `page_size 101`
   (if a max page size < 101 exists, the page-cap detection scheme breaks —
   fall back to page_size = max, render max-1 rows + marker when len == max,
   keeping the marker truthful).
1. `models/tool/main.go`: `ToolNameGetResource` + `AllToolNames` (fix
   `TestAllToolNames` expected set).
2. `models/message/tool.go`: `FunctionCallNameGetResource`.
3. `pkg/aicallhandler/tool_resource.go`: sentinel, masking const,
   `resolveResource`, fetcher table, per-type formatters.
4. Dispatch entry: add the key to the `mapFunctions` map literal inside
   `ToolHandle` (pkg/aicallhandler/tool.go — it is a local literal rebuilt per
   call, not a package-level map).
5. `pkg/toolhandler/definitions.go`: new entry + get_correlation cross-link
   sentence.
5b. OQ5 fix (settled: DERIVE, not drop): in `formatCorrelationSummary`
   (`pkg/aicallhandler/tool.go` — same file as the correlation tool handler,
   NOT definitions.go), replace the per-line `r.DataType` label with a label
   derived from the resource's event types: take the first entry of the
   already-sorted `r.EventTypes` slice and cut at the first underscore
   (`call_created` → `call`, `aicall_status_progressing` → `aicall`,
   `conferencecall_joined` → `conferencecall`). Scope of the rule's
   correctness (round-4 verification): it is correct for every event type
   that can actually MATERIALIZE as a correlation row — timeline requires a
   top-level `id` (resource_id materialization) AND an `activeflow_id` in
   the payload, which excludes the known non-conforming names
   (`team_member_switched` has no top-level id;
   `call.outbound_whitelist_rejected` publishes `call_id` not `id`;
   `transcript_created` carries no activeflow_id). Watch item: a future
   cross-prefix event carrying both fields would mislabel — verify none
   exists at implementation time. If `EventTypes`
   is empty, render the neutral label `resource`. Add a unit test for the
   derivation (incl. empty-events fallback) in the existing correlation test
   file.
6. Tests (section 10).
7. bin-ai-manager code docs: add the tool to `docs/domain.md` tool list (and
   architecture.md if the tool table lives there). No RST (internal tool).
8. Full verification workflow in bin-ai-manager.

## 10. Test Plan (pkg/aicallhandler/tool_resource_test.go)

Table-driven, gomock requesthandler. Cases:

1. Success per resource type (8 cases; assert curated summary content, that
   internal fields like channel_id/filenames never appear in the output, and
   at least one case exercising nil-timestamp/zero-UUID omission).
2. transcribe success including transcript lines (chronological order after
   DESC reversal); transcript list error → metadata + `(transcripts
   unavailable)`, still success; empty transcript list → `(no transcripts)`;
   nil TMTranscript line rendering. Cross-customer transcribe: assert the
   transcript list RPC is NEVER called (strict gomock, no EXPECT registered —
   locks the post-ownership enrichment contract).
2b. aicall success including conversation lines: user/assistant rendering,
   compact `[assistant called <name>]` tool-call markers, system/notification/
   tool-role and empty-content rows excluded from output (assert prompt text
   never appears); message list error → metadata + `(messages unavailable)`,
   still success; empty list → `(no messages)`. Cross-customer aicall: assert
   the message list is NEVER fetched (mock messageHandler, no EXPECT — same
   post-ownership contract as transcribe).
3. Summary/transcript truncation at the 4000-rune cap with markers.
4. Cross-customer → masked `msgResourceNotFound` (fillSuccess, not failure).
5. Absent (RPC returns `requesthandler.ErrNotFound`) → masked, PER RESOURCE
   TYPE (8 cases — guards the per-manager 404 assumption), byte-identical
   to case 4 using the same fixed `tc.ID` AND the same fixed
   resource_type/resource_id per compared pair (messageContent echoes both, so
   DeepEqual requires identical inputs across the absent/cross-customer pair)
   (`reflect.DeepEqual` masking-invariant assertion).
6. Transient RPC error → `fillFailed("resource lookup failed")`, NOT masked.
7. Unsupported resource_type → fillFailed listing supported types; empty
   resource_type → dedicated `resource_type is required` message.
8. Invalid/empty resource_id, malformed JSON args → fillFailed.
9. Regression: cross-customer path emits no summary fragment (no IDOR
   fall-through), transient path does not panic.
10. Schema/enum drift guard: JSON-schema enum in definitions.go ==
    sorted `mapResourceFetchers` keys. Mechanics: export a
    `SupportedResourceTypes() []string` accessor from aicallhandler (sorted
    map keys; also the source of the handler's error strings), and place the
    equality test in toolhandler where the schema is visible. The description
    prose is NOT string-asserted (prose churns); the enum + error-string legs
    are.
11. OQ5 derivation: `formatCorrelationSummary` label = first-underscore cut of
    `EventTypes[0]`; empty EventTypes → `resource` (in the existing
    correlation test file).

## 11. Open Questions

| # | Question | Recommendation | Owner |
|---|---|---|---|
| 1 | Should aicall summaries include conversation history? | YES (settled — pchero, 2026-06-11: "적극적으로 포함"; transcribe/summary are different resources, and chat/conversation aicalls have no transcribe). Curated rendering: user/assistant lines + compact tool-call markers; system/notification/tool bodies excluded. Note: `get_aicall_messages` already exposes the full raw dump (incl. system) cross-session with the same ownership check, so this adds no new data surface. | settled |
| 2 | Add `conversation` / `campaigncall` in Phase 1? | No. Correlation coverage for them is unverified (event payload activeflow_id); add in Phase 2 after verifying. | CPO |
| 3 | Per-AI gating beyond tool_names? | No. Ownership check is the control, consistent with get_correlation precedent. | settled |
| 4 | PII: resource content (transcripts, summaries) flows to the customer's own LLM engine. | No new exposure: the same engine already receives the live conversation. No action. | settled |
| 5 | `formatCorrelationSummary` prints `r.DataType` per line, which is the envelope content-type (`application/json`), not the model name — it is the exact line the LLM reads to pick `resource_type`. | Fix IN THIS PR: derive the label via first-underscore cut of `EventTypes[0]` (empty → `resource`); see Implementation Order step 5b + test case 11. | settled (round 3) |
| 6 | Phase 1 carries 8 types; the motivating chain needs ~5 (call, transcribe, recording, summary, queuecall). Trim to reduce PR size? | Keep 8 per pchero's explicit scope direction (aicall, conferencecall included). Each fetcher is small; architecture identical either way. | settled (pchero, 2026-06-11) |

## Review Summary

### Round 1 (delegate_task adversarial reviewer, CHANGES REQUESTED → v2)

Critical/High fixes applied:
- C1: fetchers restructured to two-stage (fetch owner-bearing resource →
  ownership check → render closure runs enrichment RPC). Foreign transcript
  content is never fetched; locked by a no-EXPECT gomock test.
- H2: per-manager 404 mapping promoted to BLOCKING implementation step 0;
  absent-path masking test now per resource type (8 cases).
- H3: truncation marker fixed to "first N shown" (M unknowable from a single
  page) — (superseded by Round 2 H1: marker direction was itself wrong);
  transcript ordering verified in code (`tm_create DESC`) with explicit
  reversal; `FieldTranscribeID` existence verified.

Medium/Low fixes applied: TMTranscript offset rendering spec (zero-time
subtraction, nil and empty-list rendering, direction enum); data_type
misconception corrected (envelope data_type is `application/json`, not the
model name — resource_type now defined via event-type prefix, tool description
updated, correlation summary cosmetic issue logged as OQ5); soft-delete claim
reworded to per-manager semantics; whole-message cap semantics; generic
truncation markers; supported-type list drift guard (test); empty
resource_type dedicated error; dispatch wiring described as local map literal;
fixed tc.ID in masking-invariant test; nil-timestamp omission test; code-docs
(domain.md) step added; self-RPC note; wrong-type-retry hint in description.

### Round 2 (fresh delegate_task adversarial reviewer, CHANGES REQUESTED → v3)

Architecture, security contract, and test plan confirmed sound. Findings were
confined to Section 6 transcript semantics + description copy:
- H1: truncation marker inverted which end was cut — the single DESC page is
  the MOST RECENT 100, not the first. Marker fixed to
  `...(earlier transcripts omitted; showing the most recent N)`; page-cap
  detection specified (page_size 101 disambiguation).
- M2: transcript list now filters `FieldDeleted: false` (bin-api-manager
  servicehandler precedent) so customer-deleted transcripts never render.
- M3: prefix-derivation rule defined in the description ("leading part of the
  event name") + transcript-vs-transcribe near-miss closed ("transcript
  entries are retrieved via their parent transcribe id").
- L4: OQ5 (correlation summary's `application/json` DataType label) upgraded
  to fix-in-this-PR.
- L5: arg unmarshal error now passed through (sibling precedent) for LLM
  self-correction.
- L6/L7: tm_create tie nondeterminism noted (tests must not depend on tie
  order); TMTranscript normalized via t.UTC() with implementation-time
  verification against a real row.

### Round 3 (fresh delegate_task adversarial reviewer, CHANGES REQUESTED → v4)

Architecture/security again confirmed sound; findings were doc-coherence and
spec-completeness:
- H1: OQ5 fully specified — settled to DERIVE (first-underscore cut of
  EventTypes[0], empty → `resource`), located in pkg/aicallhandler/tool.go
  (NOT definitions.go), new Implementation Order step 5b, new test case 11,
  OQ5 marked settled, Section 2 cross-reference updated.
- M2: Sections 4/5 precedent characterization reconciled — the two-tier
  contract is IDENTICAL to resolveCorrelation; the only difference is the
  second-RPC (ownership lookup) failures that get_correlation additionally
  masks, which this design has no equivalent of.
- M3: cap mechanics unified — const renamed `maxResourceSummaryRunes`,
  whole-message budget governs all types, whole-line truncation + top-of-block
  marker for transcripts, degenerate single-line case defined, generic
  `...(truncated)` marker for all other types, summary content budget =
  remainder after metadata.
- L4-L10: OQ5 owner settled; step-0 expanded (FieldDeleted encoding,
  page_size 101 acceptance + fallback); masking-invariant test pins
  resource_type/resource_id per compared pair; transient path logs the cause
  (operator signal, never reaches the LLM); Round-1 H3 changelog annotated
  superseded; ownerID==uuid.Nil fail-closed guard; drift-guard test mechanics
  specified (exported SupportedResourceTypes(), prose leg excluded).

### Round 4 (fresh delegate_task adversarial reviewer, APPROVE with notes → v5)

Verdict APPROVE. Code-verified the OQ5 derivation against every publisher's
event constants and timeline's materialization pipeline; security/masking
contract confirmed with no constructible oracle or IDOR path. Notes applied:
- M1: "first-underscore cut correct for every published event type" claim
  scoped to events that can materialize as correlation rows (top-level `id` +
  `activeflow_id` preconditions; known non-conforming names listed); future
  cross-prefix event = watch item.
- L2: page-cap fallback disambiguated (render max-1 + marker when len == max).
- L3: `supportedResourceTypes` declaration added to the sketch.
- L4: "the single marker" → "the marker template" (degenerate variant
  cross-referenced).

### Post-approval design change (pchero, 2026-06-11 → v6)

OQ1 reversed from "metadata only" to "include conversation history" by pchero:
transcribe/summary are DIFFERENT resources; for chat/conversation-referenced
aicalls the ai_messages table is the only record of the dialogue. Changes:
- aicall summary now renders user/assistant lines + compact tool-call markers
  after the metadata block; system (prompt snapshot), notification, tool
  bodies, and empty-content rows excluded.
- Fetch is local `h.messageHandler.List` (no RPC), inside the render closure
  (post-ownership), page-cap/truncation/ordering identical to transcripts.
- Precedent note: `get_aicall_messages` already exposes the full raw dump
  (including system) for any own-customer aicall_id with the same ownership
  check — so this adds no new cross-session data surface; it is a curated
  subset of an existing one.
- Tool description, Problem Statement, WHEN-NOT-TO-USE, and test plan (new
  case 2b) updated accordingly.
Per the mandatory re-review rule, a fresh review round on the changed doc is
required before implementation.

## 12. Checklist deltas vs the standard template

DB schema / webhook events / flow variables / RabbitMQ action / Prometheus
(new) / OpenAPI: N/A — internal tool addition reusing existing RPCs; existing
`promAIcallToolExecuteTotal` covers execution counts by function label.
