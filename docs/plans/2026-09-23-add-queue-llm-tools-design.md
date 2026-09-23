# Add `list_queues` and `join_queue` LLM tools (VOIP-1540)

Status: Draft (v1)

## 1. Problem statement

`bin-ai-manager`'s LLM function-calling tool set has no way for an AI agent
running inside a live call to (a) discover which Queues the customer account
has configured, or (b) place the current call into one. The only existing
path into a Queue is the Flow action `queue_join`
(`bin-flow-manager/models/action` `TypeQueueJoin` /
`OptionQueueJoin{QueueID}`), authored ahead of time by a human into a Flow —
it is not reachable from an AI session deciding at runtime.

`connect_call` (transfer to a person/department/phone number) is the closest
existing analog but is architecturally the wrong target: it resolves an
address (`commonaddress.Address{Type,Target}`), not a Queue id, and Queue
routing (wait/service timeout, tag-based agent matching, conference bridge)
is a materially different mechanism from a direct extension/SIP/PSTN
transfer.

## 2. Goals

1. `list_queues` — a read-only LLM tool that returns the calling customer's
   Queues (id, name, detail) so the LLM can decide, based on the
   conversation, which Queue best matches the caller's need.
2. `join_queue` — an LLM tool that adds a `queue_join` action to the current
   call's activeflow (mirroring `toolHandleConnect`'s `queue_join`
   equivalent) and terminates the AIcall, handing the call off to Queue
   routing exactly as a human-authored Flow's `queue_join` action would.

## 3. Non-goals

- No new Queue CRUD (create/update/delete) exposed to the LLM. Read + join
  only.
- No tag-based pre-filtering or agent-availability preview exposed to the
  LLM (`list_queues` returns id/name/detail only, not
  `wait_queuecall_ids`/counts — see §7 field selection rationale).
- No `conversation`/`api`/other `ReferenceType` support for `join_queue` in
  this phase. Queue routing's underlying mechanism (conference bridge +
  call-manager) is call-specific; a conversation-type join is a distinct,
  unscoped design question deferred to a follow-up ticket if ever needed.
- No change to `bin-queue-manager`, `bin-flow-manager`, or
  `bin-common-handler`'s `QueueV1QueueList` — it already exists and already
  supports a `customer_id` filter.

## 4. Decisions locked (2026-09-23, confirmed with 대표님)

| # | Decision | Rationale |
|---|---|---|
| 1 | `list_queues` purpose: let the LLM self-judge, mid-call, which Queue to route to | 대표님 confirmed |
| 2 | `join_queue` scope: `ReferenceTypeCall` only, mirrors `connect_call`'s shape (add action to activeflow, then terminate the AIcall) | 대표님 confirmed |
| 3 | Tool names: `list_queues`, `join_queue` (new naming, not `get_queue_list`/`connect_queue`) | 대표님 confirmed |
| 4 | Process: full design-first review loop (design 2+, PR 3+) | 대표님 confirmed |

## 5. Existing precedent (code-verified)

### 5.1 `connect_call` — the closest sibling pattern for `join_queue`

`bin-ai-manager/pkg/aicallhandler/tool.go` `toolHandleConnect` (lines
279-326):

```go
func (h *aicallHandler) toolHandleConnect(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	res := newToolResult(tool.ID)
	if c.ReferenceType != aicall.ReferenceTypeCall {
		fillFailed(res, fmt.Errorf("connect_call is only supported for call reference type"))
		return res
	}

	var tmpOpt fmaction.OptionConnect
	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &tmpOpt); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}

	opt := fmaction.ConvertOption(tmpOpt)
	actions := []fmaction.Action{{Type: fmaction.TypeConnect, Option: opt}}

	af, err := h.reqHandler.FlowV1ActiveflowAddActions(ctx, c.ActiveflowID, actions)
	if err != nil {
		fillFailed(res, err)
		return res
	}
	fillSuccess(res, "activeflow", af.ID.String(), "Added connect action successfully.")

	go func() {
		tmp, err := h.reqHandler.AIV1AIcallTerminate(context.Background(), c.ID)
		...
	}()

	return res
}
```

`join_queue` follows the exact same shape, substituting
`fmaction.TypeQueueJoin` / `fmaction.OptionQueueJoin{QueueID: uuid.UUID}`
for `fmaction.TypeConnect` / `OptionConnect`, and MUST go through
`fmaction.ConvertOption(opt)` exactly as `toolHandleConnect` does
(`opt := fmaction.ConvertOption(tmpOpt)`, tool.go:298). This is required
by `Action.Option`'s field type, not by struct shape: `Action.Option` is
declared `map[string]any` (`bin-flow-manager/models/action/action.go:14`),
so a bare `fmaction.OptionQueueJoin{...}` struct cannot be assigned to it
directly — every action-adding call site in the codebase (`toolHandleConnect`,
`service.go`'s action-add paths) routes through `ConvertOption`'s
JSON-marshal-then-unmarshal-to-map conversion with zero exceptions. (Note:
an earlier draft of this design incorrectly stated `OptionQueueJoin`
"requires no ConvertOption step... a single-field struct, no nested address
conversion needed" — that reasoning was wrong; `ConvertOption` is mandatory
for every `Option*` struct regardless of field count or nesting, because it
is dictated by `Action.Option`'s map type, not by the option struct's
shape.)

### 5.2 `QueueV1QueueList` — already exists, no new RPC needed

`bin-common-handler/pkg/requesthandler/queue_queue.go:32`:

```go
func (r *requestHandler) QueueV1QueueList(ctx context.Context, pageToken string, pageSize uint64, filters map[qmqueue.Field]any) ([]qmqueue.Queue, error)
```

`qmqueue.Field` (`bin-queue-manager/models/queue/field.go`) includes
`FieldCustomerID = "customer_id"` and `FieldDeleted = "deleted"` (filter-only,
not a struct field) — the standard soft-delete exclusion filter used
elsewhere in the codebase (`message.FieldDeleted:false` pattern in
`toolHandleGetAIcallMessages`'s sibling tools). `list_queues` calls this
directly with `filters = {FieldCustomerID: c.CustomerID, FieldDeleted: false}`
— ownership scoping is a filter param on the existing RPC, not a
post-hoc check.

### 5.3 `Queue` domain fields (`bin-queue-manager/models/queue/queue.go`)

```go
type Queue struct {
	commonidentity.Identity // ID, CustomerID
	Name   string
	Detail string
	RoutingMethod RoutingMethod
	TagIDs []uuid.UUID
	...
	WaitQueuecallIDs, ServiceQueuecallIDs []uuid.UUID
	TotalIncomingCount, TotalServicedCount, TotalAbandonedCount int
	...
}
```

## 6. Tool 1: `list_queues`

### 6.1 Registration (7 touch points — this tool needs only the AI-manager-side
set, no new Flow action or `bin-ai-manager/pkg/actioncatalog` entry since it
introduces no Flow action type)

(Note: the "registration checklist" referenced informally during design
review is a Hermes-side skill reference note, not a file in this repo — no
`references/flow-action-and-ai-tool-registration-checklist.md` exists under
`monorepo/`. The checklist content below was independently re-derived and
verified against the real touch points in this codebase; do not cite that
path as an in-repo document.)

1. `bin-ai-manager/models/tool/main.go`: add `ToolNameListQueues ToolName =
   "list_queues"` and append to `AllToolNames`. NOT added to
   `AllInsightToolNames` (Insight AIs are Case-panel-scoped, not call-control
   — no legitimate reason for an Insight AI to enumerate Queues).
2. `bin-ai-manager/models/message/tool.go`: add
   `FunctionCallNameListQueues FunctionCallName = "list_queues"`.
3. `bin-ai-manager/pkg/toolhandler/definitions.go`: add the tool definition
   (schema below).
4. `bin-ai-manager/pkg/aicallhandler/tool.go`: add
   `message.FunctionCallNameListQueues: h.toolHandleListQueues` to
   `mapFunctions`.
5. `bin-ai-manager/pkg/toolhandler/main_test.go` `TestAllToolNames`: this
   test hardcodes an `expectedNames []tool.ToolName` slice and asserts
   `len(tool.AllToolNames) == len(expectedNames)` (main_test.go:137-159,
   verified). Both `ToolNameListQueues` and `ToolNameJoinQueue` MUST be
   added to `expectedNames` in the same commit, or this test fails on
   count mismatch.
6. `bin-ai-manager/pkg/toolhandler/whitelist.go` `ConversationSafeTools`:
   this is a separate, explicit per-tool whitelist gating which tools a
   `ReferenceTypeConversation` AIcall may use (verified: a `map[tool.
   ToolName]bool` literal, currently 9 entries, documented as "not yet
   wired to the pipecat session payload" per its own comment). Neither
   `list_queues` nor `join_queue` is added to this whitelist in this
   design — both are call-control-adjacent (feed into or perform a
   call-specific hand-off) and Queue routing itself is call-only (§3),
   so extending conversation-tool access is out of scope here, matching
   how `create_call`/`get_resource`/`get_correlation`/`connect_call` are
   also absent from this whitelist today. This is a deliberate scope
   statement, not a silent omission.
7. `bin-ai-manager/docs/domain.md`: add both tools to the "LLM Tools
   (Function Calling)" table (also listed in §11).
8. **`bin-openapi-manager/openapi/openapi.yaml`**: the `AIManagerToolName`
   schema (lines ~2964-3017, verified) is a closed `enum` + parallel
   `x-enum-varnames` list, currently 24 entries (`all` + 23 named tools) —
   this is what a customer sets via the public `tool_names` field on
   `POST/PUT /v1/ais`. Add `list_queues`/`join_queue` to BOTH the `enum:`
   list and `x-enum-varnames:` in lockstep (they are positionally paired;
   drift between the two breaks codegen). Without this, a customer cannot
   actually enable either tool via the public API except through the
   `["all"]` wildcard, and `POST /v1/ais {"tool_names":["list_queues"]}`
   would be rejected as an invalid enum value.
9. Regenerate `bin-openapi-manager/gens/models/gen.go` (`go generate
   ./...` in `bin-openapi-manager`) and then `bin-api-manager/gens/
   openapi_server/gen.go` + `bin-api-manager/gens/openapi_redoc/*` (same
   command in `bin-api-manager`, run AFTER openapi-manager's regen per
   that service's own CLAUDE.md ordering rule).
10. `bin-api-manager/docsdev/source/ai_struct_tool.rst`: add both tools to
    the Normal-tool summary table (currently 15 entries) and add their own
    per-tool prose sections, plus the `run_llm` defaults table. Clean
    Sphinx rebuild (`rm -rf build && python3 -m sphinx -M html source
    build`) and `git add -f docsdev/build/` per root CLAUDE.md's mandatory
    RST-sync rule — this is the same class of doc-sync step the sibling
    `get_call_transcript` and `emit_info_card` tool designs both required.

### 6.2 Handler

```go
func (h *aicallHandler) toolHandleListQueues(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	res := newToolResult(tool.ID)

	tmp, err := h.reqHandler.QueueV1QueueList(ctx, "", 100, map[qmqueue.Field]any{
		qmqueue.FieldCustomerID: c.CustomerID,
		qmqueue.FieldDeleted:    false,
	})
	if err != nil {
		fillFailed(res, err)
		return res
	}

	type queueSummary struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Detail string `json:"detail"`
	}
	out := make([]queueSummary, 0, len(tmp))
	for _, q := range tmp {
		out = append(out, queueSummary{ID: q.ID.String(), Name: q.Name, Detail: q.Detail})
	}

	body, errMarshal := json.Marshal(out)
	if errMarshal != nil {
		fillFailed(res, errMarshal)
		return res
	}

	fillSuccess(res, "queue", "", string(body))
	return res
}
```

- Ownership: enforced entirely by the `FieldCustomerID` filter param — the
  RPC never returns another customer's Queues, so there is no separate
  post-fetch ownership check to write (contrast with `get_correlation`/
  `get_resource`, which need one because their underlying RPCs are NOT
  customer-scoped). `c.CustomerID` is the aicall's own field, always
  trustworthy (never LLM-supplied).
- `resource_id` in `fillSuccess` is left empty (`""`) — the result is a
  collection, not a single resource; this mirrors
  `toolHandleGetAIcallMessages`'s use of a single id vs. a list shape only
  loosely, so the convention here is: single-resource tools populate
  `resource_id`, list tools leave it empty and rely on `resource_type` +
  the JSON body. (Precedent: no existing list-shaped tool in this codebase
  populates a meaningful single id either — `search_knowledge` uses the
  RAG id, which is a single owning resource, not a list. `list_queues` is
  the first pure list tool; leaving `resource_id` empty is the natural
  reading of the existing struct, not a new convention.)
- Page size 100, no pagination exposed to the LLM. A customer with over 100
  live Queues is far outside any current usage pattern; if this becomes
  real, add `page_token`/`page_size` params in a follow-up rather than
  guessing at need now (minimal-change bias).
- Empty result (`len(tmp)==0`) is NOT an error — `fillSuccess` with an empty
  JSON array `[]`. The LLM's tool description tells it how to read this.

### 6.3 Tool definition (`toolhandler/definitions.go`)

```go
{
	Name:   tool.ToolNameListQueues,
	RunLLM: true,
	Description: `Lists the Queues configured for this account, so you can decide which one best matches the caller's need before routing them with join_queue.

WHEN TO USE:
- Before calling join_queue, to see what Queues exist and pick the right one (e.g. "sales", "support", "billing").
- User asks what departments/queues are available.

WHEN NOT TO USE:
- You already know the exact queue_id to use (e.g. from a prior list_queues call in this same session).

Each queue entry has: id (UUID, pass this to join_queue), name, detail. An empty list means no Queues are configured for this account.

run_llm: Always set true — you should choose a queue and act (or tell the user none is available) based on the result.`,
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"run_llm": map[string]any{
				"type":        "boolean",
				"description": "Always set true to act on the queue list.",
				"default":     true,
			},
		},
		"required": []string{},
	},
},
```

## 7. Tool 2: `join_queue`

### 7.1 Registration (10 touch points, same file set as §6.1 items 1-10,
plus the `TestAllToolNames` update from §6.1 item 5. `ConversationSafeTools`
(§6.1 item 6) applies identically — `join_queue` is also excluded, same
reasoning. The OpenAPI enum (§6.1 item 8) and RST doc (§6.1 item 10) entries
are shared edits covering BOTH tools in the same PR, not per-tool
duplicated work.)

1. `models/tool/main.go`: `ToolNameJoinQueue ToolName = "join_queue"`,
   append to `AllToolNames`.
2. `models/message/tool.go`: `FunctionCallNameJoinQueue FunctionCallName =
   "join_queue"`.
3. `toolhandler/definitions.go`: tool definition (schema below).
4. `aicallhandler/tool.go`: `message.FunctionCallNameJoinQueue:
   h.toolHandleJoinQueue` in `mapFunctions`.

### 7.2 Handler

```go
func (h *aicallHandler) toolHandleJoinQueue(ctx context.Context, c *aicall.AIcall, tool *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleJoinQueue",
		"aicall_id": c.ID,
	})

	res := newToolResult(tool.ID)
	if c.ReferenceType != aicall.ReferenceTypeCall {
		fillFailed(res, fmt.Errorf("join_queue is only supported for call reference type"))
		return res
	}

	var args struct {
		QueueID uuid.UUID `json:"queue_id"`
	}
	if errUnmarshal := json.Unmarshal([]byte(tool.Function.Arguments), &args); errUnmarshal != nil {
		fillFailed(res, errUnmarshal)
		return res
	}
	if args.QueueID == uuid.Nil {
		fillFailed(res, fmt.Errorf("queue_id is required"))
		return res
	}

	// SECURITY: ownership (IDOR prevention). Mirrors create_call's flow_id
	// ownership check (errCouldNotResolveFlow) -- both not-found and
	// cross-customer collapse to the same byte-identical masked error so
	// the tool is not a Queue-existence oracle. Each branch logs its real
	// cause at Warnf/Errorf (server-side observability), matching
	// toolHandleCreateCall's own Warnf("Flow does not belong to the
	// customer...") — only the LLM-facing message is masked, not the log.
	q, errGet := h.reqHandler.QueueV1QueueGet(ctx, args.QueueID)
	if errGet != nil || q == nil {
		log.Errorf("Could not get the queue. queue_id: %s, err: %v", args.QueueID, errGet)
		fillFailed(res, errQueueNotResolvable)
		return res
	}
	if q.CustomerID != c.CustomerID {
		log.Warnf("Queue does not belong to the customer. queue_id: %s, queue_customer_id: %s, customer_id: %s", args.QueueID, q.CustomerID, c.CustomerID)
		fillFailed(res, errQueueNotResolvable)
		return res
	}

	opt := fmaction.OptionQueueJoin{QueueID: args.QueueID}
	actions := []fmaction.Action{{Type: fmaction.TypeQueueJoin, Option: fmaction.ConvertOption(opt)}}

	af, err := h.reqHandler.FlowV1ActiveflowAddActions(ctx, c.ActiveflowID, actions)
	if err != nil {
		fillFailed(res, err)
		return res
	}
	fillSuccess(res, "activeflow", af.ID.String(), "Added queue_join action successfully.")

	go func() {
		tmp, errTerm := h.reqHandler.AIV1AIcallTerminate(context.Background(), c.ID)
		if errTerm != nil {
			log.Errorf("Could not terminate the aicall after sending the tool actions. err: %v", errTerm)
			return
		}
		log.WithField("aicall", tmp).Debugf("Terminating the aicall after sending the tool actions. aicall_id: %s", c.ID)
	}()

	return res
}

var errQueueNotResolvable = stderrors.New("could not resolve queue")
```

- **Ownership check is new, load-bearing security logic** — `connect_call`
  needs none (it addresses by `commonaddress.Address`, not an internal id),
  but `join_queue` takes a caller/LLM-supplied `queue_id` UUID, and
  `QueueV1QueueGet` is NOT customer-scoped (`bin-common-handler/pkg/
  requesthandler/queue_queue.go:56` takes only `queueID`, no customer
  filter) — exactly the same shape as `create_call`'s `flow_id` ownership
  check (`toolHandleCreateCall`, `errCouldNotResolveFlow`). Both outcomes
  (not-found, cross-customer) collapse to one masked sentinel so the tool
  cannot be used to probe for the existence of another customer's Queue
  ids.
- No `run_llm`-gated pre-check against `list_queues` — the LLM is expected
  to call `list_queues` first per its own description, but `join_queue`
  does not hard-require it; a caller that already knows a valid `queue_id`
  (e.g. from a prior turn or a system prompt) may call `join_queue`
  directly.
- Mirrors `toolHandleConnect` exactly for the activeflow-add + terminate
  sequence: `FlowV1ActiveflowAddActions` then a `go func()` calling
  `AIV1AIcallTerminate` (fire-and-forget, matching the existing pattern's
  own comment: "this will connect the call right away").

### 7.3 Tool definition

```go
{
	Name:   tool.ToolNameJoinQueue,
	RunLLM: false,
	Description: `Places the CURRENT call into a Queue for routing to a human agent. This ends your (the AI's) participation in the call.

WHEN TO USE:
- You have determined (e.g. via list_queues) which department/queue matches the caller's need, and the caller should now wait for a human agent.
- User asks to be connected to a queue/department and a matching queue_id is known.

WHEN NOT TO USE:
- You have not yet identified which queue is appropriate -- call list_queues first.
- User wants to be transferred directly to a person, extension, or phone number (use connect_call instead; that is a direct transfer, not Queue-based routing with hold/wait handling).

DIFFERS FROM connect_call:
- join_queue = places the call into a Queue (hold music, wait timeout, agent matching by tag, then bridged when an agent is available)
- connect_call = direct transfer/bridge to a specific endpoint (person, extension, phone number), no wait/queue mechanics

ARGUMENTS:
- queue_id (required): the UUID of the queue to join, from list_queues' id field. Must belong to your own account; an id from a different account is rejected.

run_llm: Set false (default) -- the caller is being handed off to the queue's own wait experience (e.g. hold music/announcement), not a spoken confirmation from you.`,
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"run_llm": map[string]any{
				"type":        "boolean",
				"description": "Set true to speak a brief confirmation before handing off. Set false (default) to hand off silently.",
				"default":     false,
			},
			"queue_id": map[string]any{
				"type":        "string",
				"description": "UUID of the queue to join, as returned by list_queues' id field.",
			},
		},
		"required": []string{"queue_id"},
	},
},
```

- `RunLLM: false` — chosen deliberately opposite to `connect_call`'s
  `run_llm` default of `false` too (connect_call's static field default is
  unset/false, its schema default is `false`) — consistent with the
  "silent hand-off unless the LLM explicitly wants to narrate" convention
  already established by `connect_call`, `stop_media`.

## 8. Cross-tool table (business-outcome parity check — see §6.1's
registration-touch-point enumeration; this table checks that the same
business condition maps to the equivalent treatment on each surface)

| Outcome | `list_queues` | `join_queue` |
|---|---|---|
| Success | `fillSuccess("queue", "", <json array>)` | `fillSuccess("activeflow", af.ID, "Added queue_join action successfully.")` |
| Wrong reference type | N/A — list works for any reference type (read-only, no call-specific side effect) | `fillFailed("join_queue is only supported for call reference type")` |
| No queues configured | `fillSuccess("queue", "", "[]")` — NOT an error | N/A |
| Invalid/missing queue_id | N/A | `fillFailed("queue_id is required")` |
| Queue not found / not owned | N/A | `fillFailed(errQueueNotResolvable)` — masked, byte-identical for both cases |
| Downstream RPC failure | `fillFailed(err)` | `fillFailed(err)` |

`list_queues` has no reference-type restriction: unlike `join_queue`
(which mutates a call-specific activeflow), listing Queues is pure read and
has no dependency on what kind of session the AIcall is running in. This is
a deliberate, stated scope choice, not an oversight (per the "enumerate the
full value set" checklist item) — `c.ReferenceType` is not consulted at
all in `toolHandleListQueues`.

## 9. Security & IDOR summary

- `list_queues`: ownership enforced by `FieldCustomerID` filter on a
  customer-scoped RPC — no oracle risk (the RPC itself cannot return
  foreign data).
- `join_queue`: ownership enforced by an explicit `QueueV1QueueGet` +
  `CustomerID` comparison before any activeflow mutation, masked
  byte-identically for not-found and cross-customer, mirroring
  `create_call`'s `errCouldNotResolveFlow` pattern exactly.

## 10. Testing plan

- `models/message/tool_test.go`: extend `TestFunctionCallNameConstants`
  with `FunctionCallNameListQueues` / `FunctionCallNameJoinQueue`.
- `pkg/aicallhandler/tool_test.go` (or a new `tool_queue_test.go` file,
  following the existing per-feature split seen in `tool_resource_test.go`,
  `tool_case_create_test.go`):
  - `toolHandleListQueues`: success with N queues, success with 0 queues,
    RPC error.
  - `toolHandleJoinQueue`: success path (activeflow add + terminate
    called), wrong reference type, missing queue_id, queue not found,
    queue owned by a different customer (assert byte-identical error
    message to not-found), `FlowV1ActiveflowAddActions` error,
    `QueueV1QueueGet` transient error.
  - Use `gomock` strict expectations (no `.AnyTimes()` on
    `AIV1AIcallTerminate` unless the go func's async nature requires a
    sync primitive — follow `toolHandleConnect`'s existing test pattern in
    `tool_test.go` for the goroutine-termination assertion style).
- `pkg/toolhandler/definitions_test.go` (if one exists) or an
  `AllToolNames`/definitions parity test: confirm both new
  `ToolName`s have a matching entry in `toolDefinitions`.

## 11. Affected files

| File | Change |
|---|---|
| `bin-ai-manager/models/tool/main.go` | Add `ToolNameListQueues`, `ToolNameJoinQueue`; append both to `AllToolNames` |
| `bin-ai-manager/models/message/tool.go` | Add `FunctionCallNameListQueues`, `FunctionCallNameJoinQueue` |
| `bin-ai-manager/models/message/tool_test.go` | Extend constant test table |
| `bin-ai-manager/pkg/toolhandler/definitions.go` | Add 2 tool definitions |
| `bin-ai-manager/pkg/aicallhandler/tool.go` | Add 2 dispatch entries to `mapFunctions`, 2 new handler funcs, `errQueueNotResolvable` sentinel, `qmqueue` import |
| `bin-ai-manager/pkg/aicallhandler/tool_queue_test.go` (new) | Unit tests for both handlers |
| `bin-ai-manager/pkg/toolhandler/main_test.go` | Add `ToolNameListQueues`, `ToolNameJoinQueue` to `TestAllToolNames`'s `expectedNames` |
| `bin-ai-manager/docs/domain.md` | Add both tools to the "LLM Tools (Function Calling)" table |
| `bin-openapi-manager/openapi/openapi.yaml` | Add `list_queues`/`join_queue` to `AIManagerToolName` `enum` + `x-enum-varnames` (lockstep) |
| `bin-openapi-manager/gens/models/gen.go` | Regenerate (`go generate ./...`) |
| `bin-api-manager/gens/openapi_server/gen.go`, `gens/openapi_redoc/*` | Regenerate (`go generate ./...`, after openapi-manager's regen) |
| `bin-api-manager/docsdev/source/ai_struct_tool.rst` | Add both tools to summary table, per-tool sections, `run_llm` defaults table; clean Sphinx rebuild + force-add `docsdev/build/` |

No changes to `bin-flow-manager`, `bin-queue-manager`, or
`bin-common-handler` — `queue_join`/`OptionQueueJoin` and
`QueueV1QueueList`/`QueueV1QueueGet` already exist and are reused as-is.

## 12. Open questions

| # | Question | Recommendation |
|---|---|---|
| 1 | Should `list_queues` expose live wait/service counts (`WaitQueuecallIDs` length, `TotalIncomingCount`, etc.) so the LLM can judge queue load before routing? | Defer to Phase 2 — id/name/detail is sufficient for tool-selection-by-purpose; load-aware routing is a distinct, higher-cost feature (would need to explain queue-depth semantics to the LLM) that should be its own design if requested |
| 2 | Should `join_queue` be exposed on `ReferenceTypeConversation` (chat handed to a human queue)? | Out of scope per §3; Queue routing's conference-bridge mechanism is call-specific today — a chat-queue handoff would need its own design, not a copy-paste reference-type widening |
| 3 | Insight AI access to `list_queues`? | No — Insight AIs are Case-panel read-only assistants with no call-control surface; `list_queues` is call-control-adjacent (feeds into `join_queue`) and not added to `AllInsightToolNames` |

## 13. Review summary

### Iter-1 review response (2026-09-23)

Independent subagent review (file+terminal access, verified against real
source). Verdict: CHANGES_REQUESTED, 3 actionable items. All addressed in
this revision:

1. **Registration checklist incompleteness** (originally §6.1/§7.1/§11) —
   `bin-ai-manager/pkg/toolhandler/main_test.go::TestAllToolNames` (hardcoded
   count-and-membership assertion) and `bin-ai-manager/pkg/toolhandler/
   whitelist.go::ConversationSafeTools` (separate per-tool whitelist for
   `ReferenceTypeConversation`) were both verified as real, independent
   touch points. Fixed: §6.1 now enumerates 7 touch points (was 5), §7.1
   references the same set, §11's affected-files table adds
   `main_test.go`. `ConversationSafeTools` is explicitly scoped OUT (both
   tools excluded, same reasoning as `create_call`/`connect_call`'s
   existing absence from that whitelist) rather than left unaddressed.
2. **Nonexistent in-repo citation** (`references/flow-action-and-ai-tool-
   registration-checklist.md`) — verified zero hits under `monorepo/`;
   that path is a Hermes skill reference note, not a repo document. Fixed:
   §6.1 and §8 no longer cite it as an in-repo path; a note clarifies its
   actual nature and states the checklist content here was independently
   re-derived and verified.
3. **Missing observability on the IDOR-masking path** (§7.2) — the design
   previously masked both not-found and cross-customer cases with zero
   server-side logging, diverging from `toolHandleCreateCall`'s sibling
   pattern (`log.Warnf("Flow does not belong to the customer...")`). Fixed:
   §7.2 now logs `Errorf`/`Warnf` on each branch before returning the
   masked `errQueueNotResolvable`, distinguishing the real cause in logs
   only — the LLM-facing message stays byte-identical for both cases.

### Iter-2 review response (2026-09-23)

Independent subagent review, round 2 (file+terminal access). Confirmed all
3 iter-1 fixes landed correctly. Verdict: CHANGES_REQUESTED, 2 NEW
actionable items, both real defects:

1. **§7.2 code did not compile** — `Action.Option` is declared
   `map[string]any` (`bin-flow-manager/models/action/action.go:14`,
   verified), so assigning a bare `fmaction.OptionQueueJoin{...}` struct to
   it directly is a Go compile error. `toolHandleConnect` and every other
   action-adding call site route through `fmaction.ConvertOption(...)`
   first, with zero exceptions. §5.1's rationale claiming `OptionQueueJoin`
   "requires no ConvertOption step" was also factually wrong (conflated
   "single-field, no nesting" with "no conversion needed" — the conversion
   is dictated by the map-typed field, not by struct shape). Fixed: §5.1's
   rationale corrected, §7.2's handler code now calls
   `fmaction.ConvertOption(opt)`.
2. **§6.1/§7.1/§11 omitted the OpenAPI/RST touch points** — verified
   `bin-openapi-manager/openapi/openapi.yaml`'s `AIManagerToolName` enum +
   `x-enum-varnames` (currently 24 entries) is the closed enum a customer's
   `tool_names` field validates against; without adding both new tool names
   there, a customer cannot enable either tool except via the `["all"]`
   wildcard. This is the same class of gap the sibling `get_call_transcript`
   and `emit_info_card` tool designs both had to close. Fixed: §6.1 now
   enumerates 3 more touch points (openapi.yaml enum edit, two codegen
   regenerations, `ai_struct_tool.rst` update per root CLAUDE.md's mandatory
   RST-sync rule) — 10 touch points total (was 7). §7.1 and §11 updated to
   match.

### Iter-3 review response (2026-09-23)

Independent subagent review, round 3 (fresh reviewer, file+terminal access,
re-verified every claim from scratch rather than trusting prior rounds).
Confirmed all 5 prior fixes (iter-1's 3 + iter-2's 2) landed correctly.
Verdict: CHANGES_REQUESTED, 1 NEW actionable item — a real defect
introduced by iter-1's own fix:

1. **§7.2's `toolHandleJoinQueue` used `log.Errorf`/`log.Warnf` (added by
   the iter-1 IDOR-observability fix) without ever declaring a `log`
   variable in the function** — `bin-ai-manager/pkg/aicallhandler` has no
   package-level `log`; every sibling handler (`toolHandleConnect`,
   `toolHandleCreateCall`) declares its own `log := logrus.WithFields(...)`
   at the top of the function. This would not compile (`undefined: log`).
   Fixed: §7.2's code sample now opens with the same
   `log := logrus.WithFields(logrus.Fields{"func": "toolHandleJoinQueue",
   "aicall_id": c.ID})` pattern every sibling handler in this file uses.

This is the loop's 3rd round; 2 CONSECUTIVE APPROVED rounds are still
required to close (round 3 was CHANGES_REQUESTED, so round 4 begins the
consecutive count at zero).

