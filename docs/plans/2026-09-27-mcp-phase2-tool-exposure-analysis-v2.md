# Analysis v2: expose MCP tools to the LLM (PR B)

This is the current, normative analysis. It supersedes
`2026-09-27-mcp-phase2-tool-exposure-analysis-v1-superseded.md`, which grew to
1,214 lines by appending four rounds of corrections after a body that those
corrections refuted, so a builder reading its first page acquired a false causal
model of the defect. The corrections are folded in here. The evidence that
produced them (three live experiments and a token measurement) is preserved in
Appendix A, which is a non-normative decision log.

Rounds 1 through 4 each demolished part of the round before. Where a claim below
is the survivor of that, it is stated plainly rather than annotated.

## 0. Current state

**The defect.** The published API reference and the docs site state, in the
present tense with no beta marker, that a whitelisted MCP server's tools are
merged into an AI's tool list and presented to the LLM namespaced
`mcp_<8hex>_<tool_name>`. Verified in the served artefacts, not the sources:
`bin-api-manager/gens/openapi_redoc/openapi.json` under
`AIManagerAI.properties.mcp_server_ids.description` and
`paths./mcpservers.post.description`, and the built HTML
`docsdev/build/html/ai_struct_ai.html`, `ai_struct_mcpserver.html`,
`ai_overview.html`. No code path does it.

**The true cause, which v1 got wrong on its first page.** v1 blamed a
deny-by-default whitelist, claiming no MCP name could pass pipecat's filter.
That filter cannot see an MCP tool at all: `GetByNames`
(`bin-pipecat-manager/pkg/toolhandler/main.go:91-108`) filters `h.tools`, the
process-wide static catalogue fetched once via `AIV1ToolList`, and a per-AI
per-customer MCP tool is never a member of that slice, so it never reaches the
`allowed[t.Name]` check at line 100.

The real cause is that **there is no transport.** The merged list is built in
bin-ai-manager (`resolveTools`, `pkg/aicallhandler/mcp_tool.go:52`, which
namespaces correctly at `:90-105`) and the LLM tool list is assembled in a
different service (`bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:150`
for single AIs). The start RPC between them carries no tools channel:
`PipecatV1PipecatcallStart`
(`bin-common-handler/pkg/requesthandler/pipecat_pipecatcall.go:15-65`) and
`bin-pipecat-manager/models/pipecatcall/main.go` both return zero matches for
`Tools`. All three non-test callers of `resolveTools` discard the list, and two
of them are right to: `writeInsightSessionMetadata`
(`insight_session.go:225`) and `refreshMcpToolMap` (`mcp_tool.go:327`) exist
only to write the name-to-ref map.

**Second cause, proven by experiment.** Even with a transport, the MCP client
cannot read tools from a spec-conformant server. It POSTs `tools/list` directly
with `Accept: application/json` (`pkg/mcptoolhandler/client.go:125-138`), which
a reference server rejects with 406; with the header fixed it is rejected with
400 for having no session; and with the handshake added the body arrives
SSE-framed, which `json.Unmarshal` at `client.go:163` cannot parse. See
Appendix A.2.

**Ordering.** PR A (lifecycle: credential revocation on delete, ownership and
status gates) is merged as `c4336940e` and is a hard prerequisite, because PR B
is what makes `CallTool` reachable by a model. Its dispatch-time gates are
present in this tree and were verified: `toolHandleMcpCall`
(`mcp_tool.go:127-187`) trusts stored metadata only for the name-to-ref mapping
and then re-reads live state, and `refuseDeleted` (`client.go:197`) backstops
both `ListTools` (`client.go:215`) and `CallTool` (`client.go:241`).

**Blast radius.** Production has 5 AIs, all `type=normal`, and **zero** with a
non-empty `mcp_server_ids`. The 104 MCP server rows are all api-validator test
data. Nothing a customer relies on changes, which is why this can be fixed
properly rather than hastily, and why the one breaking payload change below is
safe.

## 1. Scope ledger

Correctness blockers. Each is a defect the moment the feature is reachable.

| # | Item | Where | Notes |
|---|---|---|---|
| B1 | MCP protocol client: dual `Accept`, `initialize` and session lifecycle, `MCP-Protocol-Version`, SSE `data:` parsing, session-expiry re-init | `pkg/mcptoolhandler/client.go:108-176` | Largest single item. See A.2 |
| B2 | `inputSchema` json tag, non-nil `Parameters` default | `pkg/mcptoolhandler/main.go:26`, `mcp_tool.go:97` | Two lines. Without it every session with one MCP tool dies. See A.3 |
| B3 | Redirect guard: re-validate on every hop, or refuse redirects | `pkg/mcpserverhandler/ssrf.go:95-112` | Credential leak, proven. See A.5 |
| B4 | Type-filter ai-manager's built-in resolver | `pkg/toolhandler/main.go:29`, caller `mcp_tool.go:60` | Fail-open today. See A.4 |
| B5 | Name policy: reject invalid charset and over-length, drop duplicates instead of last-write-wins, count caps | `mcp_tool.go:90-105` | An invalid name fails the whole completion, not one tool |
| B6 | Tool-result cap well under 64 KiB | `pkg/mcptoolhandler/client.go:267-274` | `ai_messages.content` is `TEXT`; also feeds the auditor, see B8 |
| B7 | Description cap, expressed in **tokens** not characters | `mcp_tool.go:96` | Reclassified from hardening. See A.6 |
| B8 | Auditor exposure: cap and mark remote tool text as untrusted | `pkg/aiaudithandler/main.go:428` | Second LLM consumer nobody knew about. See A.7 |
| B9 | Dispatch-time gates for AI type and AI deletion | `mcp_tool.go:266-279` | Three revocation vectors uncovered. See A.8 |
| B10 | Insight write gate in **two** locations | `pkg/aihandler/chatbot.go:155` and `mcpserver_validation.go:57` | One location is bypassable. See A.9 |
| B11 | Team symmetry: do not write `mcp_tool_map` for `AssistanceTypeTeam` | `start.go:1191`, `insight_session.go:232`, `mcp_tool.go:341` | Today unadvertised but dispatchable. See A.9 |
| B12 | Transport: per-AIcall callback RPC | new `AIV1AIcallToolList`, consumer `runner.go:150` | The feature turns on here |
| B13 | Parallel fan-out with an aggregate budget derived from the greeting budget | `mcp_tool.go:72-106` | Sequential 10s per server today |
| B14 | Tool-list cache with negative caching and working invalidation | `pkg/mcptoolhandler/`, `pkg/subscribehandler/main.go` | Reclassified to blocker. See A.10 |
| B15 | Registration-time `tools/list` probe | `POST /mcpservers`, OAuth complete | Turns discovery into lookup; the cheapest cure for shipping inert |
| B16 | Realtime voice writes `MetaKeyMcpToolMap` | `start.go:1127-1130` | Reconcile against B12, see open decision O3 |
| B17 | Metric label cardinality | `pkg/aicallhandler/tool.go:123` | Customer-controlled label today |
| B18 | Advertisement metric and MCP outcome metrics | `pkg/aicallhandler/main.go` | Without it the next inert release is a customer email |
| B19 | 401/403 forces a refresh bypassing the expiry check, then one retry | `pkg/mcpoauthhandler/access_token.go:46`, `client.go:159-161` | See A.11 |
| B20 | Single-flight per server | `resolveTools` | Restored after being dropped from v1's ledger |
| B21 | Re-enable `bin-pipecat-manager-test` | `.circleci/config_work.yml:543-545` | Verified green today, so it does not expand scope |
| B22 | Docs, OpenAPI, and the webhook projection change | `docsdev/source/*`, `openapi.yaml`, `models/aicall/webhook.go` | See section 4 |

Hardening, cuttable to a follow-up: retryable/terminal reason codes beyond the
401 case (B19 covers the one that matters).

**Out of scope, with a negative requirement rather than silence.** Insight AIs
(B10) and the team surface (B11). Both exclusions require code, because an
exclusion that leaves a half-wired path is worse than either including or
blocking it.

**Explicitly not needed:** the flow layer. `grep -rni mcp bin-flow-manager
--include=*.go` excluding vendor returns zero; MCP dispatch lives entirely in
`aicallhandler`. Recorded so it is not re-asked.

## 2. Decision register

| # | Decision | Value | Status |
|---|---|---|---|
| D1 | Transport shape | Per-AIcall callback RPC from pipecat into ai-manager, not a field on the start RPC | SETTLED, A.12 |
| D2 | Behaviour when the customer's server is down at session start | Start without those tools, never fail the call, but make it observable | SETTLED |
| D3 | Cache | Keyed on `mcp_server_id`, positive TTL 60s, negative TTL 10-15s, invalidated on write and delete | SETTLED; location and mechanism OPEN (O1) |
| D4 | Conversation-lifetime pin | Rejected. Replay of an unadvertised name is provably safe, and revocation must stay prompt | SETTLED, A.1 |
| D5 | Invalid remote tool name | Drop that tool, keep the rest, log once per server. Never sanitize silently | SETTLED |
| D6 | Duplicate namespaced name | Drop the second and log. Never last-write-wins | SETTLED |
| D7 | Insight AIs | Excluded, gated at two write locations plus a dispatch gate | SETTLED, A.9 |
| D8 | Teams | Excluded, enforced by not writing the tool map | SETTLED, A.9 |
| D9 | `mcp_tool_map` on the webhook | Remove from the projection, publish a documented `mcp_tool_status` summary instead | SETTLED, section 4 |
| D10 | `[missing_tool]` alerting for `mcp_` names | Suppress, or the first tool-set change pages an operator for correct behaviour | SETTLED, A.1 |
| D11 | Architecture: per-session discovery from a customer-hosted server | Right shape, wrong default. Keep late binding, add a cache and a registration-time probe | SETTLED, A.6 |
| O1 | Cache location and invalidation mechanism | ai-manager runs 2 replicas, so an in-process cache means a per-pod TTL and two customers' views. Redis via `pkg/cachehandler` is the alternative. Invalidation needs a subscription case that does not exist: `mcp_server_updated` is published (`models/mcpserver/event.go:6`) but ai-manager's `processEvent` (`pkg/subscribehandler/main.go:201-241`) has no case for it | **OPEN** |
| O2 | Caps, final numbers | A character cap does not bound tokens: measured 284 tok/tool for English prose at 1,024 chars, 622 for filler, and **2,069 for Korean**, which is most of the current market. Must be a token cap, and the per-AI count must be sized against the worst case | **OPEN**, A.6 |
| O3 | Does the transport RPC also write `MetaKeyMcpToolMap`? | If yes, B16's separate write is redundant and the staleness window shrinks. If no, both exist | **OPEN** |
| O4 | Aggregate fan-out budget, as a number | Must derive from the greeting budget (`docs/plans/2026-03-04-optimize-aicall-conversation-latency-design.md` budgets the initial greeting at 3.0-3.5s total), not from an RPC deadline | **OPEN**, A.13 |
| O5 | How the build-tagged conformance test runs in CI | `bin-ai-manager-test` invokes `go test ... $(go list ./...)` with no `-tags`, so a tagged test is compiled out and silently never runs | **OPEN**, A.14 |
| O6 | Where the per-AI count cap is applied, given a per-server cache | 3 servers at 32 each exceeds a 64 per-AI cap; the drop must be deterministic | **OPEN** |
| O7 | Is the square-admin picker change in PR B or PR C? | Converting an AI to Insight with a stored whitelist will 400 with the picker still showing the cause | **OPEN**, A.9 |

## 3. Risk register

| Risk | Status |
|---|---|
| SSRF to internal networks | Mitigated by a dial-time `Control` hook (`ssrf.go:95-112`), which survives DNS rebinding |
| Credential leak via redirect | **NOT mitigated.** B3. Proven: a custom API-key header survives a cross-host redirect, and `Authorization` survives a same-host `https` to `http` downgrade. A.5 |
| Deleted, foreign or inactive server still callable | Mitigated by PR A, re-verified in this tree |
| AI flipped to Insight, or deleted, mid-call | **NOT mitigated.** B9. A.8 |
| Prompt injection via tool description | **NOT mitigated.** B7. The asymmetry is the argument: remote error text is capped at 200 chars for a log line (`mcp_tool.go:180`) while the description goes uncapped into the prompt |
| Prompt injection into the **auditor** | **NOT mitigated.** B8. A.7 |
| Tool output exceeding the 64 KiB column | **NOT mitigated.** B6 |
| Slow or hanging server delaying the greeting | **NOT mitigated.** B13, O4 |
| Metric cardinality via tool names | **NOT mitigated.** B17. Note `server_id` is also unbounded over time, so bucket to a constant |
| Concurrent discovery against one third-party server | **NOT mitigated.** B20 |
| Vendor-side token revocation producing a permanently silent tool list | **NOT mitigated.** B19. A.11 |
| Whitelisting grants all present and future tools | Accepted, requires one docs sentence and B15's probe to be discoverable |
| Duplicate side effects from inverted timeouts | **NOT mitigated.** The python tool timeout is a hardcoded 10s (`scripts/pipecat/tools.py:165`) and `McpToolCallTimeoutSeconds` defaults to 10, so the python side gives up first and the model retries. Needs the python value to become config before the invariant can be asserted |
| Model calling a tool it was never advertised | Accepted, graceful. Refused politely; D10 removes the false alert |

## 4. Customer-facing changes

Sentences that become true and need caveats: the `mcp_server_ids` bullet in
`ai_struct_ai.rst` and the same description in `openapi.yaml` (normal-type AIs
only, single-AI sessions only, the caps, fail-open when the server is down, and
that whitelisting grants present and future tools). A sentence that becomes
false: the `active` row of the status table in `ai_struct_mcpserver.rst`, which
says tools are callable from **any** AI referencing the server. A sentence that
is only true if B14 ships with invalidation: that disabling a server "stops
serving tools immediately." Ship the invalidation or change the sentence.

**Webhook.** `ConvertWebhookMessage` (`models/aicall/webhook.go:78`) copies
`Metadata` as a whole field with no key filtering, so the customer's server
UUIDs and remote tool names already ride the messaging webhook, and B16 would
extend that to every voice AIcall. Remove `mcp_tool_map` from the projection and
publish a documented `mcp_tool_status` summary instead. This is a breaking
payload change and the PR body must say so; the blast radius is nil because no
AI has a non-empty whitelist, so the key is empty in every webhook emitted
today.

## 5. Acceptance gate

L1, hermetic Go tests that **reject like a real server** rather than accepting
anything: 406 unless `Accept` carries both types, 400 unless a session id from a
prior `initialize` is present, `MCP-Protocol-Version` asserted, the result served
SSE-framed, `inputSchema` camelCase with a non-nil schema asserted. Every one
fails today. The reason the 406 was never caught is that all eight existing
`httptest` handlers answer a bare `tools/list`.

L2, integration against the reference MCP server as a subprocess, in both
default and stateless modes. This is the test that would have caught the inert
feature. **Blocked on O5**: as a build-tagged test it would never run.

L3, the pipecat half. Re-enable the commented-out job and add a python job
asserting an `mcp_`-named tool with a valid schema reaches `tool_register`, that
a null `parameters` does not take down the built-ins, and that a team AIcall
gets **no** tool map written (B11's negative test, which no level currently
owns).

L4, one manual end-to-end check recorded in the PR body: register the reference
server, whitelist it on a normal AI, place one call to a virtual or internal
number, and grep for the advertisement log line. An inert feature fails there,
and that step needs no model cooperation. Not automatable: the SSRF guard blocks
localhost, a real completion costs money, and a model cannot be made to choose a
tool deterministically.

Definition of done: L1 through L3 green in CI with O5 resolved, and L4's log
line quoted in the PR body.

## 6. Build order

The property that matters is that **the feature is unreachable until step 9**.
Every earlier step fixes code that is already shipped and already wrong, and
whose only consumer today discards the tool list.

1. B1, B2, B3 with L1 and L2 conformance tests
2. B6 tool-result cap
3. B4 built-in resolver type filter, a security fix on its own
4. B5, B7 name policy and the token-based description cap
5. B13, B14, B20 fan-out budget, cache with negative caching, single-flight
6. B10, B11, B9 the scope gates and the dispatch-time gates, **before** anything
   is advertised
7. B16 voice metadata write, subject to O3
8. B17, B18 metric cardinality then MCP metrics
9. **B12 the transport. The feature turns on here.**
10. B15 registration-time probe
11. B21 re-enable the pipecat job, add the python cases
12. B22 docs, OpenAPI regeneration, webhook projection

Known weakness, stated rather than hidden: steps 4 and 5 have no consumer until
step 9, so their only test is against `resolveTools`' first return value, which
no production caller reads yet. They are verifiable, but against a contract that
does not exist until step 9.

## 7. What the design document must still specify

Concrete signatures and their listenhandler wiring for B12
(`AIV1AIcallToolList` in `bin-common-handler/pkg/requesthandler/ai_aicalls.go`,
a route registration and dispatch case in
`bin-ai-manager/pkg/listenhandler/main.go`, a handler method, mocks). The
cache's location, encoding and invalidation path (O1). The final cap numbers as
tokens (O2). The error taxonomy enumerated. Metric names and labels. Config keys
and defaults, remembering that a new key needs four edits in
`internal/config/main.go` and that the python timeout is currently a literal.
The test matrix as a matrix: {single-AI} times {call, conversation,
contact_case, task} times {ok, down, 401, nil schema, over-long name, duplicate
name, over cap}. And whether the RAG filter immediately after `runner.go:150`
applies to the merged list or moves.

---

# Appendix A: decision log (non-normative)

Four review rounds, eight reviewers. Each round found what the previous missed:
five items, then three, then four, then four. What follows is the evidence, kept
because it is the most valuable content in this file and because several
conclusions are counter-intuitive.

## A.1 Replaying an unadvertised tool name is safe (round 3, live provider calls)

The worry was that a stored `tool_calls` pair whose function is no longer
advertised would corrupt the conversation. Live calls with `tools=[get_weather]`
and a history containing a completed `mcp_deadbeef_lookup_order` pair:

| provider | result |
|---|---|
| openai gpt-4o-mini, gpt-4o, gpt-5-mini, gpt-4.1-mini | 200, answered normally |
| google gemini-2.5-flash | 200 |
| x-ai grok-4.3 | 200 |
| same history with no `tools` array at all | 200 everywhere |

Control proving the harness detects rejection: an **orphaned** `role:tool` with
no preceding call returns 400 from OpenAI. So providers police pairing, not
membership in the current tool array, and D4 follows.

Two residues. First, the replayed pair is a strong few-shot prompt: gemini
emitted the unadvertised function 3/3 times, which lands on pipecat's
missing-function handler, returns a polite refusal, and fires an **alerted**
counter, hence D10. Second, the replayed content can be the failure JSON written
by `fillFailed`, which is a complete pair and therefore survives the filter,
teaching the model that a tool exists and returns errors.

Closed as structurally impossible, so it is not re-asked: a partially-answered
multi-`tool_calls` message. `filter_valid_messages` is all-or-nothing per
assistant message, and `tool.go:88` writes exactly one `ToolCall` per message,
which is the only non-test construction site.

## A.2 The MCP client cannot talk to a conformant server (round 2, local experiment)

Rounds 1 and 2 recorded this as unproven because both hardcoded vendors reject
an unauthenticated probe before protocol validation. That was the wrong
experiment; the right one needs no credential. Against the reference `mcp`
python SDK over Streamable HTTP, sending exactly what `doJSONRPCRequest` sends:

| request | result |
|---|---|
| `Accept: application/json`, bare `tools/list` (current client) | **406** "Client must accept both application/json and text/event-stream" |
| dual `Accept`, no `initialize` | **400** "Missing session ID" |
| `initialize` then `tools/list` with session and version headers | **200** |

Against a server configured stateless with json responses the current request
does succeed, so the client works only against that minority configuration.
After the handshake the body arrives as `event: message\r\ndata: {...}`, which a
JSON parser rejects at the first character, so B1 includes an SSE parser.
`notifications/initialized` turned out not to be load-bearing. The live payload
also confirmed the field is `inputSchema` and that real servers send
`outputSchema`, which the current struct drops.

## A.3 The schema tag kills the whole call, not just tool use (round 2, executed both sides)

Go side: a spec payload leaves `InputSchema` nil, and the re-marshalled
`tool.Tool` emits `"parameters": null` with the **key present**, because
`tool.Tool.Parameters` has no `omitempty` (`models/tool/main.go:108`). Python
side: `tools.py:75` uses `tool.get("parameters", default)`, whose default never
fires because the key exists, then `run.py:434-444` calls `.get()` on `None`.
The `AttributeError` propagates out of `init_llm` through `asyncio.gather`, so
there is no STT, no LLM and no TTS. A list containing a built-in **and** one MCP
tool still dies. It fires on every session with one MCP tool present, not
occasionally.

## A.4 The built-in resolver is fail-open today (round 2, executed)

The most likely place for the analysis to be overstating itself, tested
deliberately and it held:

    ValidateToolNames(TypeInsight, ["all"]) = <nil>   legal, storable
    ai-manager GetByNames(["all"])          = 25 tools
    outside AllowedToolNames(Insight)       = 17
      connect_call send_email send_message stop_media stop_service stop_flow
      set_variables get_variables get_aicall_messages search_knowledge
      get_correlation get_resource create_call describe_action case_create
      list_queues join_queue

The explicit-name form is blocked, but `ToolNameAll` is exempted on purpose by a
`continue` at `models/ai/tool_validation.go:81-83` because it is meant to be
expanded at runtime against `AllowedToolNames`. ai-manager's resolver is the one
expander that never performs that expansion. Harmless only because the list is
discarded, hence B4.

## A.5 The redirect credential leak (round 4, executed)

`NewSSRFGuardedClient` sets no `CheckRedirect`, so Go's default of up to ten
redirects applies, and `ValidateURL`'s https-only check runs only on the stored
URL at create and update time, never on a redirect target. Two leaks confirmed
by running real redirects:

- **Different hostname:** `Authorization` is stripped, but the custom API-key
  header (default `X-API-Key`) is **not** in Go's sensitive list and arrives
  intact at the third-party host.
- **Same hostname:** Go compares `URL.Host`, which is identical for
  `https://x/mcp` and `http://x/mcp`, so `Authorization` survives an
  https-to-http downgrade and the customer's bearer token goes in cleartext.

The dial-time control hook still blocks private IPs, so this is credential
exfiltration to a public host rather than classic SSRF. B1 makes redirects more
likely by adding round trips, which is why B3 sits in the same step.

## A.6 Caps, and the architecture question nobody had asked (rounds 3 and 4)

Round 2 justified its caps with roughly 256 tokens per tool. Measured with
tiktoken `o200k_base` on a realistic tool at a 1,024-character description cap:
**622 to 637** tokens for filler text, so 64 tools is about 40,000 tokens
injected on **every turn**, not 16,000. Round 4 then showed the cap itself is
the wrong unit: the same 1,024 characters is **284** tokens of ordinary English
prose and **2,069** tokens of Korean, which would be about 132,000 tokens at 64
tools. Hence O2 and B7 as a token cap. v1 listing the description cap as
cuttable hardening was wrong in a way its own arithmetic forbids.

On the architecture: registration-time-only storage was considered and rejected,
because it would make the "immediately" promise false in the other direction,
needs a new table and a refresh scheduler, and argues against MCP's own dynamic
`tools/list` model. Late binding is right. What was wrong is the default of
fetching at session start on the greeting path. The correction is D11: keep late
binding, make the cache a blocker rather than cuttable hardening so discovery
becomes lookup, add a registration-time probe (B15) so a customer with a broken
server learns at registration instead of never, and give the session-start read
its own short deadline distinct from the call timeout.

The admin UI supports this reading: `mcpservers_detail.js` shows name, detail,
URL, status and auth type but **no tool list, no tool count, no test-connection
and no last-fetch**, so a customer cannot see what they granted. Surfacing the
probe result is PR C's work, but PR B must return the data or PR C cannot be
built.

## A.7 The auditor is a second LLM consumer of remote text (round 4)

`buildTranscript` (`pkg/aiaudithandler/main.go:428`) emits
`[role]: <tool_result> content` with only a per-message character cut, and feeds
it to a Gemini evaluation whose score is customer-visible. So a hostile MCP
server's tool output becomes untrusted text inside an evaluator prompt,
undelimited and unmarked. Auto-audit is per-AI.

Confirmed in the same pass, and worth recording: no non-pipecat engine handler
can pick up the merged tool **list**. The analysis and summary handlers build
their own completion requests with no `Tools` field, and the one place that sets
`Tools` (`engine_openai_handler/streaming_send.go:50`) is dead code whose
package-level list names tools that no longer exist. The content path is the
only exposure.

## A.8 Revocation is not already safe (round 4)

v1 concluded that no mid-call refresh was needed because dispatch re-checks live
state. True for the vectors that were checked, false for three that were not.
`mcpServerIsUsable` (`mcp_tool.go:266-279`) takes the AI as a parameter and
inspects only `a.CustomerID`: it never reads `a.Type` and never reads the AI's
deletion state, and `aiGetFromDB` has no `tm_delete IS NULL` filter on the
single-row read. So an AI flipped to Insight mid-call, or deleted mid-call,
keeps dispatching MCP tools for the rest of that call, and a vendor-side token
revocation is invisible (A.11). Note the belt-and-braces suggested earlier, a
Type skip inside `resolveTools`, does **not** close this, because that is the
advertising path and this is dispatch. Hence B9.

This matters more on voice, where the tool list is fetched **once per call**:
`SendReferenceTypeCall` (`send.go:65-90`) fetches the existing pipecatcall and
sends a message with no tool rebuild, while everything else mints a fresh
pipecatcall per turn (`send.go:118-138`). v1's claim of a per-turn refetch was
right for chat and wrong for the flagship surface, and the two have opposite
staleness problems.

Related, same class: the 8-hex prefix derives from the immutable server id, so a
mid-call **URL** change keeps every namespaced name and the stored map identical
while pointing live calls at a different host. A URL change must be treated as a
revocation event.

## A.9 The two scope exclusions needed code, not prose (rounds 3 and 4)

**Insight.** The validation block is guarded on the field being present
(`listenhandler/v1_ais.go:268`), so a PUT sending only `{"type":"insight"}`
never reaches it, and `Update` resolves the effective type
(`chatbot.go:145-151`) while validating only `ToolNames` (`chatbot.go:155`)
despite already holding `preUpdateAI`. Adding a type parameter to the validator
therefore does not close the bypass; a second gate belongs at `chatbot.go:155`.
The rule should be to reject the transition while a non-empty stored whitelist
exists, **with an error naming the remedy**, because square-admin clears
`selectedTools` when the type flips to insight but leaves `mcpServerIds`
untouched and always submits it, so the customer would otherwise get a generic
400 with the picker still showing the cause (O7).

**Teams.** Advertising is clean: `resolveTeamForPython` has no MCP reference, so
a team AIcall would advertise nothing. Dispatch is not: the map is written
unconditionally (`start.go:1184`, refreshed for the current member at
`start.go:369-375`), and `tool.go:141` routes any `mcp_`-prefixed name to
`toolHandleMcpCall` with **no** team check anywhere on that path, while
`resolveActiveAIForMcp` explicitly supports team AIcalls. So a model emitting a
name it was never given reaches a real server that is genuinely whitelisted,
active and owned, which is exactly the case PR A's gates cannot catch. Hence
B11: make the exclusion symmetric by not writing the map at all for team
AIcalls. Blocking `mcp_server_ids` on a member's AI would be the wrong
instrument, since the same AI may be used standalone.

## A.10 The cache's invalidation mechanism does not exist (round 4)

`mcp_server_updated` is published (`models/mcpserver/event.go:6`) but
ai-manager's `processEvent` (`pkg/subscribehandler/main.go:201-241`) has no case
for its own publisher, so the invalidation the published "immediately" promise
depends on requires either a new subscription case or a shared cache. Combined
with two ai-manager replicas, this is O1 rather than a settled design.

## A.11 OAuth refresh is sound but blind to a 401 (rounds 3 and 4)

`GetValidAccessToken` (`mcpoauthhandler/access_token.go:37-113`) is complete:
expiry margin, refresh-token decrypt, vendor token endpoint, rotation
persistence, per-server backoff. The gap is above it. Refresh is driven only by
the stored expiry, tokens with no recorded expiry take an early return at
`:46` and are therefore **never** refreshed, and `doJSONRPCRequest:159-161`
collapses every non-2xx into one error. So a vendor-side revocation yields a 401
that never triggers a refresh, and with PR B that is a permanently silent tool
list because `resolveTools` skips the server with a Warn.

The naive fix is a load amplifier: for the nil-expiry case a plain retry calls
`GetValidAccessToken` again, gets the same early return, and issues a second
doomed request to the customer's server on every tool call. Hence B19's precise
form: on 401 or 403, force a refresh **bypassing** the expiry check, retry once,
and record the outcome in the same per-server backoff.

## A.12 Why the callback transport (round 2, verified round 3)

Adding a `Tools` field to the start RPC means adding it to
`pipecatcall.Pipecatcall`, and `prepareFieldsFromStruct`
(`bin-common-handler/pkg/databasehandler/mapping.go:270-314`) maps every
exported `db`-tagged field into the insert, so it would name a nonexistent
column and break every create unless tagged out or migrated. The callback needs
neither. It also reuses an established failure posture
(`runner.go:143-148` already fails closed to no tools with a metric) and
resolves the list one hop from its consumer.

A constraint that turned out not to exist: nothing survives a pipecat pod
restart mid-call. `runnerStartScript` runs once inside `RunnerStart` with no
re-entry, and a restart kills the session, so no option needed to survive one.

Rejected: the python runner fetching tools itself (no credentials, no RabbitMQ,
would put customer secrets or a new auth hop in the python process), and a
per-AIcall overlay on the static catalogue cache (a single process-wide slice
under one mutex, populated at startup).

## A.13 The transport moves the latency problem, it does not fix it (round 4)

v1 claimed the callback "fixes" the timeout defect as a side effect. Two
separate claims were being conflated. The nested-budget framing was itself
wrong: on the voice path `startAIcallByRealtime` and `startPipecatcall` are
sequential siblings (`start.go:271-284`), not nested. And relocating the fan-out
into `RunnerStart`'s goroutine does take it off the RPC deadlines, but the
caller is already connected by then, so an unbounded sequential loop becomes
**audible dead air** rather than an RPC timeout. Three servers at 10s each is
three times the entire greeting budget. Hence O4: the budget must be a number
derived from the latency design, not from an RPC default.

One genuine side benefit the earlier rounds missed: an in-flight `tools/list` is
uncancellable today because the listen handler builds its own
`context.Background()`, and `RunnerStart`'s goroutine **is** bounded by the
session context, so the callback does fix that.

Also corrected: the timeout defect is latent rather than live. The discovery
loop iterates `a.McpServerIDs`, which is empty for every AI in production, so it
is unreachable today and becomes the common case only with PR B.

## A.14 The acceptance gate's own gate (round 4)

`bin-ai-manager-test` runs `go test -coverprofile cp.out -v $(go list ./...)`
with no `-tags`, so a build-tagged L2 test compiles out and never executes,
which would ship a test that looks present and does nothing, the exact failure
class this analysis exists to punish. Python availability is not the blocker,
since another job pip-installs freely. Hence O5. Separately verified: the
commented-out pipecat job passes both `go test ./...` and `golangci-lint run` at
HEAD today, so re-enabling it does not silently expand this PR.
