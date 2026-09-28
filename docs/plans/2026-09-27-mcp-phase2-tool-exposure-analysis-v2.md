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

**Reachability, corrected in round 6 after five rounds asserted it wrongly.**
Only **discovery** is live. `resolveTools` runs on three real paths
(`start.go:1184`, `insight_session.go:225`, `mcp_tool.go:327`) and does issue real
outbound `tools/list` requests to a customer-hosted server, so everything on the
outbound request path is a live defect. **Dispatch is unreachable**, and the chain
was traced end to end: `toolHandleMcpCall` has one caller (`tool.go:142`),
`ToolHandle` has one caller (`listenhandler/v1_aicalls.go:321`), whose only
producer is pipecat's `RunnerToolHandle`, reachable only through a closure created
at registration time by `tool_register` (`scripts/pipecat/tools.py:101-116`),
which iterates the **built-in-only** static catalogue. An `mcp_`-named tool is
never in it, and this runner registers **no catch-all**: the docstring of
`register_missing_tool_logging` (`tools.py:29-31`) states that it never registers
a `None` catch-all handler, so pipecat drops the call and our `[missing_tool]`
ERROR fires. The name never crosses the wire. **`CallTool` has no live caller.**

That correction invalidates reasoning several earlier rounds relied on, including
A.9's claim that a team AIcall's stored map lets a model reach a real server. It
does not. Every dispatch-path finding is real as a **mechanism** and **dormant**
in effect, and must be stated that way rather than as a live leak.

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
| B2a | `inputSchema` json tag | `pkg/mcptoolhandler/main.go:26` | **Live inbound-parse defect.** `ListTools` runs today and silently drops the schema of every tool from every conformant server, because the tag reads `input_schema` and the wire field is `inputSchema`. Belongs to PR B1: it needs no consumer to be wrong, and PR B1's own acceptance gate asserts a non-nil schema |
| B2b | Non-nil `Parameters` default | `mcp_tool.go:97`, and note `models/tool/main.go:108` has no `omitempty` | The half that kills the python pipeline, which requires advertisement. PR B2. See A.3 |
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
| B23 | `tools/list` pagination: send `cursor`, follow `nextCursor` | `pkg/mcptoolhandler/client.go:42-44` | `toolsListResult` has only `Tools`; zero cursor handling. A server paginating silently truncates, and which tools vanish is server-chosen. Defeats B5's caps and B15's probe |
| B24 | Honour `isError`: a remote tool failure must not be reported to the model as a success | `pkg/mcptoolhandler/client.go:61`, `mcp_tool.go:185` | `IsError` is declared and **never read**; `CallTool` returns the error text as a normal result and `fillSuccess` labels it `success`. A hostile server's prose reaches the model under a success label |
| B25 | Cancel an in-flight `tools/call` when the call ends | `mcp_tool.go:127`, `listenhandler/main.go:279` | Dispatch runs under `context.Background()`, so a hangup mid-call leaves a side-effecting request running with no consumer for the result. This is the other half of the double-fire class |
| B26 | `clearListenState` must not lose `mcp_tool_map` | `pkg/aicallhandler/listen.go:539-546` | It copies keys from `c.Metadata`, the caller's **in-memory snapshot**, and writes the whole column, so a map written after `c` was fetched is lost. Dispatch then fails closed for the rest of that AIcall's life |
| B27 | Rollback: a global config disable. The per-customer flag is **not** viable as first assumed | `internal/config/main.go` | Nothing can turn this feature off today, and the precedent this was modelled on does not transfer. See A.15, A.19 |
| B28 | Cap and dedupe `mcp_server_ids` on the write path | `pkg/aihandler/mcpserver_validation.go:57-96`, `bin-openapi-manager/openapi/openapi.yaml` | **Live and unconditional, no feature involved.** One serial `McpServerGet` per submitted id with no cardinality cap and no dedupe, under a handler running on `context.Background()`, and the schema declares no `maxItems` or `uniqueItems` (verified: zero occurrences in the whole file). One authenticated PUT with N ids is N serial DB round trips today. It also uncaps the discovery fan-out, which makes B20 correctness-load-bearing rather than an optimisation |

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
| D16 | Aggregate fan-out budget | **6s aggregate, 5s per server**, defended rather than derived. The latency design targets a 3.0-3.5s greeting and reaches it by removing about 2.3s from a measured 5.8s, with every phase allocated and no slack line, so no number can be derived from it. B1 makes the current worst case worse (initialize plus tools/list is two round trips per server, so three servers is 60s under today's 10s per-request bound), and that is the number the budget exists to kill. Revisit against B18's histogram once it exists | SETTLED, replaces O4 |
| D17 | How the conformance test runs in CI | **No build tag.** Verified: all three `go test` invocations run `$(go list ./...)` with no `-tags`, and the repo has zero `//go:build` test files, so a tagged test would compile out and silently never run. Use a runtime skip gated on an environment variable, set only on the `bin-ai-manager-test` job, with one pip step to install the reference SDK there | SETTLED, replaces O5 |
| O6 | Where the per-AI count cap is applied, given a per-server cache | 3 servers at 32 each exceeds a 64 per-AI cap; the drop must be deterministic | **OPEN** |
| O7 | Is the square-admin picker change in PR B or PR C? | Converting an AI to Insight with a stored whitelist will 400 with the picker still showing the cause | **OPEN**, A.9 |
| D12 | Does the callback **replace** or **supplement** `GetByNames` at `runner.go:150`? | **Supplement.** Replacing it would bypass pipecat's AIType whitelist for the built-in half, which makes B4 a hard blocker rather than an independent fix | SETTLED, forced by A.4 |
| D13 | Failure posture when the callback fails | **Fail open to built-ins**, not closed to nothing. `runner.go:143-148` clears *all* tools including built-ins, which contradicts D2. A customer's MCP server being down must not remove `connect_call` | SETTLED, corrects A.12 |
| D14 | Is the usability gate inside or outside the cache? | **Outside.** Only the `tools/list` payload is cached; deleted, status and ownership are evaluated on every resolution, or disabling a server keeps advertising it for up to the TTL | SETTLED, forced by A.15 |
| D15 | Is the advertised list the same list the map derives from, after the per-AI cap? | **Yes, the map must be built post-cap.** Otherwise a replayed name dispatches a tool the cap excluded, and A.1 shows replay happens 3/3 | SETTLED |
| O8 | Does B15's probe result get **persisted**? | If yes, this needs a new column or table and bin-dbscheme-manager becomes a fifth service in scope. If it is only returned in the `POST /mcpservers` response, it does not | **OPEN**, and it changes the PR's service count |
| O9 | What bounds the transport RPC's response payload? | 64 tools with capped descriptions over RabbitMQ. B6 bounds tool results; nothing bounds the tool list | **OPEN** |

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

**The split criterion, stated once so it is not re-derived a fourth time.**
Earlier attempts justified the boundary twice and contradicted themselves both
times, because "the loop runs" admits pagination and "a truncated list feeds
nothing" excludes the fan-out bound. The criterion that actually separates the two
PRs is:

> **PR B1 is defects whose harm does not require a consumer of the tool list.**
> Wire behaviour, leaked credentials, latency on the session-start path, and load
> against a third party all hurt with nothing reading the result. **PR B2 is
> defects whose harm requires a consumer**, which does not exist until the
> transport lands.

Section 0 records that only discovery is live and that dispatch is unreachable
because pipecat drops an unregistered name before it crosses the wire. That holds
here too: nothing in PR B1 depends on dispatch, and every PR B2 item is a real
finding whose effect waits on advertisement.

PR B1, in order:

0. B21 re-enable the pipecat CI job. Green at HEAD today, so it costs nothing,
   and it must gate the work rather than follow it
1. B1, B2a, B3 the MCP client's protocol, the `inputSchema` tag, and the redirect
   guard, with the L1 and L2 conformance tests. Includes making the JSON-RPC
   request id monotonic (A.20), without which a multi-step session cannot match
   responses, and moving the status check ahead of the body read
2. B19 the 401/403 refresh-and-retry, bypassing the expiry check
3. B10 the Insight write gate in both locations
4. B28 cap and dedupe the whitelist on the write path, which bounds what the
   fan-out can be asked to do
5. B13, B20 the aggregate fan-out budget and single-flight, plus D14's rule that
   the usability gate stays outside any cache. B20 is correctness, not
   optimisation, until B28 lands

PR B2, in order, after PR B1 merges:

6. B2b, B23, B24, B6, B25 the parameters default, pagination, the error flag, the
   result cap, and dispatch cancellation
6. B4 the built-in resolver type filter, mandatory per D12
7. B5, B7 name policy and the token-based description cap
8. B14 the cache with negative caching
9. B9, B11, B26 the dispatch-time gates, team symmetry, and the metadata clobber,
   **before** anything is advertised
10. B22's webhook projection fix, ahead of the voice write
11. B16 voice metadata write, subject to O3
12. B17, B18 metric cardinality then MCP metrics
13. B27 the global rollback key, before the feature can be turned on
14. **B12 the transport. Advertisement turns on here.**
15. B15 registration-time probe, subject to O8
16. The python test cases
17. B22's docs and OpenAPI regeneration

Known weakness, stated rather than hidden. Every PR B2 item lacks a live
consumer until step 14, so most of them can only be tested against
`resolveTools`' first return value, which no production caller reads yet, or
against `CallTool`, which has no caller at all. That is the price of the honest
boundary: PR B1 is smaller than the first attempt claimed, and PR B2 is mostly
work whose effect cannot be observed end to end until its last step.

**The split is authorised, and round 6 re-derived its boundary.** The standing
rule is one PR per repository and the CEO granted an exception for this work. The
first boundary attempt used dispatch reachability, which the code refutes (see
section 0), so eight of the fourteen items originally placed in the first PR were
dead code by the split's own criterion. The boundary is **discovery**
reachability: does the item fix something that runs today, given that
`resolveTools` issues real outbound requests and nothing consumes its tool list.

**PR B1, the live outbound-path PR.** B1, B2a, B3, B19, B10, B13, B20, B21, B28,
plus B14's usability gate rule only.

Every item here fixes code that executes today. B1, B3 and B19 are the outbound
request itself: the protocol is wrong, the redirect leaks credentials, and a
vendor-revoked token is never refreshed. B10 is live because
`writeInsightSessionMetadata` reaches `resolveTools`, so an Insight AI already
performs MCP discovery. B13 and B20 move **into** this PR from the activation
half, because the unbounded sequential fan-out and the absence of single-flight
are properties of a loop that runs now: they are latent only in the sense that
every production AI has an empty whitelist, and any customer setting the field
makes them immediate. B14 contributes only the rule that the usability gate is
evaluated outside any cache (D14), which is a correctness constraint on this PR's
outbound path rather than the cache itself.

**PR B2, the activation PR.** B2b, B4, B5, B6, B7, B9, B11, B12, B15, B16, B17,
B18, B22, B23, B24, B25, B26, B27, and the rest of B14.

These are correct findings whose consequence is dormant until advertisement
exists. The non-nil parameters default matters when a tool is advertised; the tag
that makes the schema parse at all does not, which is why B2 was split. The resolver type filter
is harmless while the list is discarded. The result cap, the error flag, the
cancellation and the dispatch gates all sit on `CallTool`, which has no live
caller. The metadata clobber protects a map that only dispatch reads. Pagination's request is live but its harm is not. The stated reason for that was
wrong and is corrected here: a truncated list does **not** feed nothing, it is
persisted at all three map write sites and rides the aicall webhook. It is
harmless because nothing **reads** it, since `lookupMcpToolRef` is dispatch-only.
Filing them here is not a downgrade of their validity; it keeps the first PR
honest about what it fixes.

B27's rollback switches stay in PR B2 and must land before B12 within it.

The ordering inside each PR follows the numbered steps above. B27's rollback
switches belong to PR B2 rather than PR B1, because there is nothing to roll
back until advertisement exists, but they must land before B12 within that PR.

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
`resolveActiveAIForMcp` explicitly supports team AIcalls. v1 concluded from this that a model emitting a name it was never
given reaches a real server. **Round 6 refuted that**: pipecat drops an
unregistered name before it crosses the wire (section 0), so the hazard is
mechanism-only until advertisement exists. B11 is still right, because the moment
B12 lands the mechanism becomes live, but it belongs to the activation PR rather
than being presented as a live leak. Blocking `mcp_server_ids` on a member's AI would be the wrong
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
defended against the latency target rather than derived from it: the cited design
allocates every phase it lists and leaves no slack line, so v1's claim that a
number could be derived from it was unsupported.

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

## A.15 There is no way to turn this off (round 5)

Four rounds found defects in this feature up to and including round 5, and no
round asked how it gets disabled if the first real customer's server misbehaves.
Verified three ways, all negative:

- **Config is frozen at process start.** `LoadGlobalConfig` is `sync.Once`
  guarded (`internal/config/main.go:225`), with no reload, no SIGHUP and no
  watch.
- **Config arrives as container environment.** Both the Komodo compose fragment
  and the k8s manifest inject env, each at 2 replicas, so any flag change is a
  redeploy and a pod restart.
- **No flag framework exists.** A repo-wide search for feature-flag or
  kill-switch machinery across non-vendor Go returns one comment and no code.

**A precedent does exist and no round had found it.**
`bin-customer-manager/models/customer/metadata.go` defines a per-customer
`Metadata` with `RTPDebug bool` (key `rtp_debug`), updatable at runtime through
the customer-metadata endpoints and read live at call start
(`bin-call-manager/pkg/callhandler/start.go:612`). That is a shipped,
zero-deploy, per-customer runtime toggle, and it is the right model for an MCP
enable flag. Hence B27.

**Why "set status to inactive" is not sufficient on its own.** The operator path
works for new sessions: `resolveTools` (`mcp_tool.go:79`) checks
`mcpServerIsUsable` *before* `ListTools`, so a disabled server is skipped, which
is also why D14 must keep that gate outside the cache. But it is one server row
at a time across 104 rows, it does nothing about a defect in our own code (an SSE
parser bug, a fan-out eating the greeting budget), and on voice the tool list is
frozen for the length of the call (A.8). So after a flip, every live call keeps
seeing and calling the tools and keeps getting refusals from dispatch, which is
wasted turns and LLM spend for as long as that call lasts. The cache TTL is
irrelevant to this window; the per-call freeze is.

## A.16 Verified safe, do not re-litigate

Round 5 found that the rewrite dropped v1's re-litigation guard, which cost a
reviewer time re-deriving it. Restored:

- **A remote tool cannot shadow a built-in.** `mapFunctions` is consulted first
  (`tool.go:139`) and only a miss falls through to the `mcp_` prefix branch, so
  even a remote tool literally named `emit_info_card` arrives as
  `mcp_<hex>_emit_info_card` and the special case at `:176` compares against the
  bare constant.
- **Underscores in remote tool names are harmless.** `lookupMcpToolRef`
  (`mcp_tool.go:194-215`) is a whole-string map lookup, not a parse.
- **The 8-hex prefix may start with a digit, and that is fine.** The full name
  always begins with the literal `mcp_`, so it satisfies the provider rule
  requiring a leading letter or underscore. B5 must not add a needless check.
- **No non-pipecat engine handler can pick up the merged tool list** (A.7). Only
  the auditor's *content* path is exposed.
- **Nothing survives a pipecat pod restart mid-call** (A.12), so no design option
  needs to.

## A.17 Why "never sanitize silently" is the rule (restored from v1)

D5 rejects sanitizing an invalid remote tool name in favour of dropping it, and
the rewrite kept the rule while losing the evidence that produced it. The
contrast is with the python team path, which does the opposite:
`scripts/pipecat/team_flow.py` has a `_sanitize_function_name` that rewrites
disallowed characters and truncates to 64 characters with a warning, advertises
the sanitized name, and closes over the **raw** name for dispatch. Two remote
names that truncate to the same 64 characters therefore overwrite each other in
the registry with only a warning, and the collision is invisible. That is the
behaviour D5 exists to avoid. Teams are out of scope (D8), so the sanitizer is
not being changed, but the next builder needs to know a second python path exists
with the opposite policy.

## A.18 Staleness asymmetries, complete list (extended round 5)

A.8 named two. There are four, and the design must state all of them because
each has a different window:

1. **Voice fetches once per call.** Stale for the length of the call, up to hours.
2. **Chat, contact_case and task fetch per turn.** Stale for one turn.
3. **Contact_case reuse** refreshes only through the idle-session path, which is
   gated on assistance type, a positive idle threshold, the idle interval having
   elapsed, and the AI being Insight, so ordinary non-idle reuse carries a stale
   map indefinitely. Fail-closed direction.
4. **Team member switch** (`pipecat_message.go:66-82`) updates only
   `CurrentMemberID` and never refreshes the map, so dispatch validates the *new*
   member's whitelist against the *old* member's stored map. Fail-closed, and
   moot once B11 stops writing the map for team AIcalls.

## A.19 The kill switch precedent does not transfer (round 6)

A.15 proposed following the per-customer `rtp_debug` toggle. The precedent is real
and it is read live at call start, but round 6 found three reasons it cannot be
copied for this feature, all verified:

1. **bin-ai-manager cannot read customer metadata at all.** Every
   `reqHandler.*` call in `bin-ai-manager/pkg` was enumerated: **zero**
   `CustomerV1*` calls. The only `bin-customer-manager` imports are for sentinel
   ID constants. The RPC exists in common-handler, so it is callable, but adding it
   means a **new cross-service call on the session-start critical path**,
   uncached, per session, against a greeting budget that O4 already says has no
   slack. That is much larger than "one metadata field".
2. **The customer can re-enable themselves.** `CustomerSelfUpdateMetadata`
   (`bin-api-manager/pkg/servicehandler/customer.go:679-702`) requires only
   `PermissionCustomerAdmin` on the caller's own customer, so an operator
   disabling a misbehaving tenant can be reverted by that tenant with one PUT.
   For a safety control that is disqualifying, and the shared `Metadata` struct
   cannot express a ProjectSuperAdmin-only field.
3. **Metadata is replaced wholesale.** `UpdateMetadata` sets the whole field, so a
   PUT carrying only an MCP flag would silently clear `rtp_debug`. Adding a second
   field to that struct is a regression vector for the first.

So B27 reduces to a **global config key**, defaulting off for the first release,
read at `resolveTools` and at the B12 handler. The PR body must state plainly that
flipping it needs a redeploy and a pod restart, so nobody discovers that during an
incident. A per-tenant control, if it is wanted later, needs its own design: a
ProjectSuperAdmin-only field on a resource ai-manager already reads, not customer
metadata.

## A.20 Further items round 6 found

- **The JSON-RPC request id is hardcoded.** `doJSONRPCRequest` sends `ID: 1` for
  every request (`client.go:115`). Harmless for one-shot POSTs, but B1 introduces
  an `initialize` followed by `tools/list` on one session and a paginated loop
  issuing several requests, so responses become indistinguishable. B1 must make
  the id monotonic per session and assert the response id matches.
- **The response size cap is applied before the status check**
  (`client.go:152-160`), so a hostile server's 1 MiB error body is read in full
  and then truncated to 512 characters for the message. Bounded, wasteful, worth
  one line in B6.
- **`resolveTools` cannot distinguish a platform failure from a disabled server.**
  A `mcpServerHandler.Get` error is swallowed with a Warn and the loop continues
  (`mcp_tool.go:73-77`), so a DB outage produces a partial tool list shaped
  exactly like "the customer disabled it". D2 requires observability; B18 must
  separate the two outcomes.
- **B15's registration probe would call `ListTools` with no AI in hand**, so
  `mcpServerIsUsable` never runs for it and the only remaining guard is
  `refuseDeleted` on the transport. B15's design must state which checks it
  performs instead.
- **B22's webhook exposure is smaller than stated.** The key is written today, but
  `mcpToolMap` is empty for every production AI, so the webhook carries
  `"mcp_tool_map": {}`. That is an undocumented empty key, not a data leak. The
  projection change is still right; the urgency claimed for it was not.

## A.21 Round 7 additions

- **The protocol version header must echo the server, not a constant.** Round 7
  re-ran the reference server and found the header is **optional** but a wrong
  value is **fatal** (400, listing the versions it supports). So hardcoding our
  own constant is strictly more dangerous than omitting the header, and B1 must
  store the `protocolVersion` the server returns from `initialize` and echo that.
  The acceptance gate's "assert the header is present" would otherwise pass while
  the client is broken against any server older than us.
- **Session scope, stated because B14 and B20 make it sharp.** The session lives
  for one `ListTools` call, as a local variable, never stored and never cached.
  B20's single-flight shares it only because coalesced callers share one function
  execution, which is the only safe sharing: a cached session id would outlive the
  credential that opened it, which is the same error D14 forbids for the usability
  gate. PR B2 must not add session reuse without its own design.
- **Refuse redirects rather than re-validating each hop.** Both were left open.
  Re-validating means re-implementing Go's per-hop header-forwarding decision,
  which A.5 proved is exactly what gets this wrong. An MCP endpoint is a URL the
  customer registered and that is already pinned to https, so a server redirecting
  its own endpoint is misconfigured; refusal gives a clear error instead of a
  silent leak, and the redirect target must be logged redacted.
- **The two retries must not compose.** B1's re-init fires on 404 only and B19's
  refresh on 401 or 403 only, each with its own once-flag, so a server cannot be
  made to produce four requests. B19 retries on the **same** session, because a
  401 is an auth failure rather than a session failure.
- **Error truncation cuts UTF-8 mid-rune.** `truncateForError` (`client.go:180`)
  and `capErrText` (`mcp_tool.go:294`) slice bytes, so a multi-byte character
  becomes invalid UTF-8 in a log line. `client.go:159` runs on every non-2xx,
  which per A.2 is every conformant server today. One line each.
