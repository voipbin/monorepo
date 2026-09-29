# Draft design for five MCP items deferred from PR B1 to PR B2

Status: **draft, input to PR B2, not approved.** Written during PR B1 and taken through
nine design-review rounds (the ninth approved by both reviewers, with its minor points
applied afterwards); a tenth round was stopped when the CEO decided to defer these items
(see section A.28 of `2026-09-27-mcp-phase2-tool-exposure-analysis-v2.md`). The text still
speaks of landing "in PR B1" and cites in-place amendments to D16, D18 and A.27 that were
not kept; read both as proposals for PR B2. Re-review against the code as it stands when
PR B2 starts; file:line references are to `532095f6d`.

## Design (drafted as A.28 during PR B1)

Round 14 found that section 6 places B10, B13, B19, B20 and B28 in PR B1 and the
branch implemented none of them, with a deferral record only for part of B13 and
D18. The CEO chose to implement all five in PR B1 as planned. This section is their
implementation design, written against the code at `532095f6d` and revised after
design review rounds 1 to 9; where it departs from section 6, the ledger or an earlier
appendix it says so and why.

### B28, whitelist cap and duplicates (D18, D20)

`ValidateMcpServerIDs` rejects, before any database read, a submitted list longer
than `ai.MaxMcpServerIDs = 8` with D18's reason `TOO_MANY_MCP_SERVER_IDS`, and a list
naming an id twice with the new reason `DUPLICATE_MCP_SERVER_ID` (D18 settles
rejection of duplicates but names no reason; this one follows D18's shape). Both are
properties of the submitted list, so D21's stored-id exemption does not apply, with one
exception: **a submitted list identical to the stored one** (the same ids, in the same
order, with the same repetitions) **skips both checks**, so an AI written before the cap
stays saveable from square-admin, which submits the stored list in stored order
(`ais_detail.js:242,383-385`). This does not rely on a production count; self-hosted
installs may hold such rows. The ownership check (D25) and D21's deleted-id exemption
still run on such a list as today. The over-cap error message names the maximum, so a
customer trimming a longer stored list knows to reach eight in one step.

Discovery defends itself against such rows too: `discoverMcpTools` skips a repeated
id and stops after eight distinct ids, logging both.

OpenAPI gains `maxItems: 8` on the two **request** definitions of `mcp_server_ids`
(`paths/ais/main.yaml:93`, `paths/ais/id.yaml:133`), documentation only per D20. The
response schema (`AIManagerAI`, `openapi.yaml:2233`) does not, because a stored list over
the cap stays valid and a validating client would reject the GET. None gains
`uniqueItems`, which third-party generators map to a set type, a breaking type change for
customers generating clients from the public spec; the descriptions state the rule
instead. The RST error table gains both reasons.

### B10, the Insight gate (D7, D19, A.9)

A shared helper `ai.EffectiveType(submitted, stored Type) Type` returns the type a
write will leave: the submitted type, else the stored one, else `TypeNormal`.
`Create` (stored is `TypeNone`) and `Update` use it in place of their inline
defaulting, and so do the gates below, so the gates cannot drift from what is written.

The rule, following A.9's own wording, is to refuse **giving** an Insight AI a
whitelist, not to refuse saving one that already has it: an Insight AI must not become
unsaveable, which is D21's principle. Concretely, a write is refused when the effective
type is Insight and the whitelist the AI will hold is non-empty, **unless** the AI was
already Insight and that whitelist is identical to the stored one. Such a leftover
whitelist is inert for new sessions (below), so keeping it saveable costs nothing.

- **Edge gate, in the listen handlers, before anything is written.** POST refuses a
  non-empty submitted list when the effective type is Insight. PUT reads the stored
  AI when the request carries `mcp_server_ids` **or** a `type` (today it reads only for
  the former) and applies the rule, taking the whitelist the AI will hold as the
  submitted list when present, else the stored one. This closes A.9's bypass, a PUT
  carrying only `{"type":"insight"}` onto an AI holding a whitelist, and it accepts the
  one request that clears the list and switches to Insight together, because the
  whitelist it checks is the submitted empty one. `aihandler.Update` has two callers:
  `v1_ais.go:281` and `cmd/ai-control/main.go:358`, which passes `TypeNone` and never
  touches the whitelist, so it cannot create the state. The listen handler is therefore
  where A.9's second gate binds, and `Update` does not gain a list parameter.
- **Chokepoint (D19).** `UpdateMcpServerIDs` applies the same rule. Its signature
  changes from `(ctx, id, ids)` to `(ctx, a *ai.AI, ids)`: both callers hold the AI that
  `Create` or `Update` just returned, whose type is the effective one and whose
  whitelist is still the stored one (`Update` does not write it), so the gate reuses it
  rather than reading the row again. "Was already Insight" is not visible there after
  `Update` has run, so the chokepoint allows an identical list on an Insight AI; the
  edge gate is what refuses a transition carrying an unchanged list.
- All gates use the new reason `MCP_SERVER_IDS_NOT_ALLOWED_FOR_INSIGHT`, with a message
  naming the remedy (submit `mcp_server_ids: []`). The release notes say that an
  Insight AI may keep a whitelist it already holds, that it has no effect, and how to
  clear it.
- **Discovery.** `discoverMcpTools` returns nothing for an Insight AI, so
  `writeInsightSessionMetadata` stops performing discovery. Every public statement of
  where discovery runs is corrected in the same change, both those that name Insight
  and those that say "every AI session except realtime voice calls":
  `openapi.yaml:2245-2247` and `:2292-2293`, `ai_struct_ai.rst:69`,
  `ai_struct_mcpserver.rst:31`, `:95` and `:104`, and `ai_overview.rst:692`. The
  implementation greps for both phrasings rather than trusting this list. The two
  request descriptions (`paths/ais/main.yaml:100-102`, `paths/ais/id.yaml:140-142`)
  say nothing about where discovery runs; they gain new text for the cap, the
  Insight rule and whitelist order.

**What the gates do not close.** Two concurrent PUTs can interleave so that the AI
ends Insight with a new non-empty whitelist: one changes the type while the stored list
is empty, the other writes a list after reading the old type. So can a failure between
`Update` and `UpdateMcpServerIDs`. For **new sessions** such a whitelist is inert:
discovery returns nothing for Insight, so no tool map is written, and dispatch resolves
a tool only through that map. It is **not** inert for an aicall already running with a
tool map written while the AI was normal: `toolHandleMcpCall` re-checks the whitelist
and `mcpServerIsUsable`, neither of which reads the AI's type. That is A.8's mid-call
vector; it stays open until B9's dispatch-time gates in PR B2, it is unreachable while
nothing is advertised (section 0), and the risk register's "NOT mitigated" row stands.
So D7's dispatch gate is met in PR B1 for new sessions only. A conditional write would
close the race and is not worth it for this window.

For every production AI, square-admin submits `mcp_server_ids: []`, which passes every
gate. It does show the picker for Insight AIs and submits whatever is selected, so
switching an AI that holds servers to Insight from the UI is now a 400, and square-admin
discards the error text on a failed save (`ais_detail.js:444-446`). Surfacing the reason
and hiding the picker for Insight are made blockers of O7 in PR C, and so is the over-cap
case: a self-hosted AI holding more than eight servers stays saveable unchanged, but
trimming it one server at a time from the picker fails at every step until eight, with
the reason hidden.

### B19, a rejected OAuth token (A.11)

**Trigger: 401 only.** A.11 and the ledger say "401/403". That is corrected here: in
MCP authorization a 403 means insufficient scope, which a refresh cannot fix, the
reference SDKs answer a rejected `Host` or `Origin` with 403, and GitHub answers its
secondary rate limit with 403. Refreshing on those would spend refresh tokens for
nothing and multiply A.27's cross-pod reuse hazard. A 403 is refreshed only when its
`WWW-Authenticate` header carries `error="invalid_token"`; any other 403 is final, as
today. Bearer and API-key servers have nothing to refresh and fail as before.

A 401 from `initialize` or `notifications/initialized` triggers it, and so does a 401
on the method when the method is `tools/list`, which has no side effects. A 401 on
`tools/call` is final: that it was refused before the tool ran is an assumption about a
server the customer controls, a retry could run a side-effecting tool twice, and
`initialize` already carried the same token on the same call, so a genuinely rejected
token shows up there. The session `DELETE` never triggers it: the close is best effort.

**`McpOAuthHandler.ForceRefreshAccessToken(ctx, serverID, rejected string)`.** Round
2 showed that any give-up record held in process memory fails across the two replicas:
each pod's forced rotation makes the other pod's record stale, so a server answering 401
to everything would rotate the grant about twice a minute forever. The bound therefore
lives in the row.

1. **Re-read the row** (`McpServerGet`). If it is deleted or no longer `auth_type =
   oauth`, fail without contacting the vendor. Otherwise decrypt the stored access
   token; if it is not
   `rejected`, another caller or pod has replaced it: return the stored token without
   contacting the vendor, as is, even if it has itself expired: the retry then fails or
   succeeds on it, and an expiry-driven refresh handles the next call. This narrows cross-pod reuse of a spent refresh token; it does
   not close it, because two pods can both re-read before either stores its result.
   A.27's lease closes it, and that lease must cover this path. When `rejected` came
   from a retained exchange result whose write failed (A.27), the row still holds the
   older token, so step 1 returns that one; the retry then fails as the call would have
   anyway, without a vendor call.
2. If the row has no refresh token, fail with "reconnect required" (A.24).
3. **Refuse to force a refresh of recently written tokens.** The row's age is its
   `tm_update`, or its `tm_create` when `tm_update` is NULL: an insert leaves
   `tm_update` NULL (`dbhandler/mcpserver.go:27`), and a first OAuth connect is an insert
   (`complete.go:186`), so a just-connected server is exactly the case this step
   exists for. If that age is under `mcpForcedRefreshMinAge = 30 minutes`, fail without
   contacting the vendor, logged with its own reason ("forced refresh refused: the
   token was written less than 30 minutes ago"). A token written that recently was
   issued recently, by a connect, a reconnect or a refresh, and a server rejecting a
   freshly issued token will not accept the next one either. Every update writes
   `tm_update` (`dbhandler/mcpserver.go:152,192`), so an unrelated edit can only delay a
   forced refresh, never cause one; that call fails as it does today.

   **What this bounds, stated exactly.** It bounds **stored** rotations to one per server
   per 30 minutes, durably, across pods and restarts. It does not make the vendor see
   only one: both pods can pass this step on the same old row before either write
   lands, so up to one forced exchange per pod per window (two in production, four an
   hour), and the second presents a refresh token the first already spent (A.27's
   hazard, closed only by A.27's lease). And a forced refresh that **fails** writes
   nothing, so the age does not move: a server whose refresh token is dead is then held
   back by the per-process failure backoff. That is not what today does for a revoked
   grant: today a revoked but unexpired access token causes no vendor call until it
   expires, while a forced refresh calls at once. So a forced refresh that the vendor
   answers with `invalid_grant` (the refresh token is dead) backs the server off for
   `refreshBackoffDeadGrant = 10 minutes` rather than 60 seconds, per process. The
   vendor's answer is read from a typed error: `postTokenRequest` (`complete.go`) today
   flattens every vendor error into a string, so it returns a `vendorTokenError` carrying
   the vendor's `error` code and HTTP status instead, whose `Error()` text is today's
   message unchanged, so the authorization-code exchange (`Complete`) behaves as before.
   A dead grant is `invalid_grant` (Linear, RFC 6749) or `bad_refresh_token` (GitHub,
   which answers it with HTTP 200). `invalid_client` and `unauthorized_client` (VoIPBin's
   own registration rejected) cannot succeed without an operator fix either, and get the
   same ten minutes. The ten minutes apply to **any** exchange answered with one of these,
   forced or expiry-driven: `exchangeAndStore` is shared and does not know
   which it serves, and the longer backoff only removes calls that cannot succeed. At
   most
   one doomed vendor call per pod per ten minutes until the customer reconnects. Other
   failures keep the 60 seconds (`access_token.go:35,216`). The backoff map is the
   existing one, which briefly blocks an expiry-driven refresh as well; with a dead
   refresh token that refresh cannot succeed either. A durable failure record belongs
   with A.27's reconnect-required status. A customer with many OAuth servers
   multiplies both. Each server is its own grant, so no other customer's grant is
   touched, but every exchange uses VoIPBin's single client registration per vendor
   (`vendors.go`), and a vendor may rate-limit per registration: one customer with many
   dead grants can spend a limit other customers share. A per-customer cap on OAuth
   servers, or a process-wide limit on vendor exchanges, is added to A.27's gates on PR
   B2's activation.
4. Otherwise run the existing exchange machinery (coalescing by server and
   refresh-token digest, detached exchange, conditional write, failure backoff) on the
   **re-read** row, skipping only the expiry check. The existing failure backoff is
   checked, as today, only when a new exchange would start, so joining an exchange in
   flight is never refused. No separate forced-start map is needed: step 3 is the rate
   limit.

"Reconnect required" is visible only in ai-manager's logs today: `mcpserver.Status` has
`active` and `disabled` only. A customer-visible reconnect-required status is added to
A.27's gates on PR B2's activation.

**Capturing the 401.** `httpStatusError` gains the response's `WWW-Authenticate` header
(bounded), which `request` and `notify` both fill, so the 403 `invalid_token` case can be
read. The host check below runs inside `ForceRefreshAccessToken`, on the re-read row, where
the vendor catalog lives; `mcptoolhandler` does not duplicate it. A forced refresh is
attempted only when the row's URL host is the vendor's own
MCP server host (`vendors[m.OAuthVendor].MCPServerURL`, `vendors.go`), compared as
`url.Parse(...).Hostname()` case-insensitively with the effective port equal (443 when
absent), and without normalising a trailing dot, so an unusual spelling fails closed. A
first connect stores exactly the vendor URL (`complete.go:173`), so an unedited row matches: an OAuth row's URL
stays editable, and a 401 from any other host says nothing about whether the vendor's
token is valid, so it must not spend the vendor grant. `openSession` today closes the
session on any failure other than a 404
(`session.go:192-195`); it skips the close on a 401 or a qualifying 403 as well, since a
`DELETE` carrying the rejected token would be rejected too. That skip applies to every auth
type, including bearer and API-key servers that never retry, for the same reason.

**In `doJSONRPCRequest`.** On a qualifying 401 the call runs one forced refresh and
opens one replacement session with the new token. During discovery the whole retry
must fit what is left of the server's 1.5 seconds, which a vendor round trip often will
not; the exchange is detached and still stores the new token, so the next session start
uses it. Any failure of the forced refresh,
including steps 2 and 3, fails the call with the original 401. The rejected session is
not closed:
its token was just refused, so a `DELETE` would be too, and the server expires it.
Requirement 11 asks that the 404 re-initialisation and this retry cannot compose into
four requests: **a call opens at most one replacement session, for either cause**, so a
404 after an auth retry, or a 401 after a 404 re-initialisation, is final. A.21 put the
retry on the same session; a new session is used because the rejected session was
opened under the rejected token. The retry runs inside the call's deadline.

### B13, the aggregate budget (D16 amended), sequential rather than parallel

**The budget is set by the callers that actually wait for it.** D16 set 6 seconds against
the voice greeting target, but the realtime voice path (`startAIcallByRealtime`,
`start.go:1102`) never runs discovery. Discovery runs at three sites, all inside a start
RPC and all under the same budget: `startAIcallByMessaging` (`start.go:1184`, before the
aicall is created), `refreshMcpToolMap` on conversation reuse (`start.go:375`), and
`writeInsightSessionMetadata` (`insight_session.go:225`, which B10 turns off). Those RPCs
are called with the requesthandler default of 3 seconds (`requestTimeoutDefault`,
`bin-common-handler/pkg/requesthandler/main.go:150`) by `POST /aicalls`, by the
service-agent aicall path and by the flow `ai_task` action (`AIV1ServiceTypeTaskStart`);
only the flow AI-talk action allows 30 seconds (`actionhandle.go:1092`). ai-manager serves
the RPC on `context.Background()` (`listenhandler/main.go:279`), so when discovery pushes it
past 3 seconds the caller gets a timeout while ai-manager goes on:
- `POST /aicalls` then deletes the activeflow it made as orphaned
  (`servicehandler/aicall.go:127-131`), leaving an aicall without its activeflow.
- The flow `ai_task` action fails while ai-manager still creates the aicall, starts a
  pipecat task and schedules its termination.
A 6-second budget would allow both for a correctly configured AI.

**What the rest of the start costs, measured.** Every server records how long it takes to
serve each RPC (`ai_manager_receive_request_process_time`). Queried from production
Prometheus on 2026-09-29 over the preceding 30 days, when no AI had a whitelist, so
discovery cost nothing: `POST /v1/services/type/aicall` (the flow AI-talk path) served 13
starts, 12 under 50 ms and all 13 under 100 ms; `POST /v1/aicalls` served none, and
`POST /v1/services/type/task` (the flow `ai_task` path) has no series at all, so it was not
called in the retention window. The sample is small and comes from the 30-second path; the
`ai_task` path adds two more RPCs to its rest-of-start (the pipecat start and the delayed
terminate), unmeasured. The rest of the start on the measured path is well under a second,
not near three.

So D16 is amended **for PR B1 only**: the whole resolution is bounded by
`mcpDiscoveryBudget = 2 seconds` and each server by `mcpDiscoveryPerServer = 1.5 seconds`,
both as context deadlines, so the slot wait, the OAuth wait and the listing all spend from
them. That leaves about a second for the rest of the start and for RPC queueing, which
the caller's 3 seconds also covers. The revisit trigger is that same histogram, which
already exists: its buckets at 1000 and 3000 ms for `/v1/aicalls`,
`/v1/services/type/aicall` and `/v1/services/type/task` show when starts approach the
caller's timeout, without waiting for B18. It measures from when ai-manager starts
processing (`listenhandler/main.go:280`), which excludes the time the request waited in the
queue; the caller's 3 seconds include it. So it is read together with the callers' own
timeout errors for those RPCs, which record starts the histogram undercounts.

**What 1.5 seconds per server buys.** Listing a server is a new TCP and TLS connection
(every call builds its own client, `client.go:131-135`), DNS, then `initialize`, the
notification (a POST that waits for its reply) and `tools/list`: about six round trips.
From the primary region (us-central1), a server in Seoul (round trip about 135 ms,
estimated) lists in about 0.8 to 1.0 seconds, Sydney (about 170 ms) in about 1.1; these are
estimates, not measurements. A second server as far away usually misses the 2-second budget,
and a serverless host after idle (cold starts of one to several seconds) loses its tools on
the first session after the idle period. On main those servers work, slowly; after PR B1
they yield nothing, visible only in ai-manager's logs and as tools missing from the aicall's
`mcp_tool_map`. That is accepted for PR B1 because nothing is presented to a model yet, and
because the alternative is the orphaned aicall above. The session `DELETE` is not one of
those round trips; it fits inside the same 1.5 seconds (below).

**PR B2 activation gates added by this section:** a budget per entry point (the flow
AI-talk path allows 30 seconds), a decision on cold-start behaviour, a customer-visible
last-discovery status per server (or B15's probe), and B18's histogram including "skipped:
budget spent".

The configured `mcp_tool_call_timeout_seconds` still bounds each call inside the budget,
and alone bounds `CallTool`. **Discovery ignores a larger configured timeout**: a self-hosted
install that raised it for a slow server loses that server's tools at session start. The
flag's help text and the release notes say so. The loop stops as soon as the budget has
passed. Worst case for one session start falls from about 84 seconds (round 14's
measurement, eight servers) to about 2 seconds.

**The session close stays synchronous and fits inside the call's deadline.** Today
`doJSONRPCRequest` sends the session `DELETE` before returning, with a 250 ms floor past the
deadline (`session.go:220-230`, `client.go:160-162`), so a listing that finishes near its
deadline returns up to 250 ms after it, and a caller whose own deadline equals the listing's
loses tools that were in fact listed. Rounds 5 to 8 tried to take the close off the caller's
path, first with an asynchronous close and then with a result callback delivered before the
close; each needed further machinery (a bound on outstanding closes that one customer could
exhaust for all, split ownership of the call's client, a late-joiner window in B20, and one
start holding both slots through its closes). Both are withdrawn. The fix is to reserve the
close's time instead:

- `doJSONRPCRequest` already derives the call's deadline `T` from `h.timeout` and the
  caller's context (`client.go:124-125`, the earlier of the two). The protocol requests
  (`initialize`, the notification, the method, any replacement session) now run under
  `T - sessionCloseFloor`, and every `closeSession` in the call, including those on
  `openSession`'s failure paths (`session.go:176,182,186,194`), keeps measuring its budget
  against `T` as today (`session.go:220-228`): the time left, at most the 2-second close
  timeout, at least the 250 ms floor. Since the requests end by `T - 250 ms`, the floor
  never pushes a close past `T`. **The whole call, close included, ends by `T`** (within
  a few milliseconds of scheduling slack, which tests allow for), where today it can end
  250 ms after. `openSession` receives only the shortened context, so `T` is passed to it
  (stored on the session) for its close sites; the exported API does not change. A caller
  with less than 250 ms left fails `initialize` at once, and a refused `initialize` adopts
  no session id (`session.go:167`), so there is nothing to close.
- For `CallTool` this is 250 ms less of its 10 seconds. It is not customer-visible: pipecat
  calls `tool_execute` with the 3-second default (`requesthandler/ai_aicalls.go:247`), so it
  has given up long before. That 3 versus 10 second mismatch (a tool keeps running after
  its caller failed) is existing, and is added to PR B2's activation gates next to the
  per-entry-point budget.
- The comments that state the old guarantee (`session.go:48-51`, "at most
  sessionCloseFloor past its deadline", and `client.go:112-118`) are rewritten in the same
  change.
- The close stays on the call's goroutine with the call's client, and the deferred
  `CloseIdleConnections` is unchanged. No signature changes, so no mock changes for this.
- Discovery passes a per-server deadline, so a listing and its close together take at most
  1.5 seconds and release the slot by then. Listing stays strictly sequential within a
  start, as on main: a start holds at most one slot at a time.

The cost is 250 ms less for the protocol requests of each server, which the 1.5-second
figure above already had to absorb on main's behaviour.

**The budget replaces the 2-second slot allowance.** Today a resolution may spend at most
2 seconds in total blocked on a discovery slot (`mcpDiscoverySlotWait`, A.27). A 2-second
budget for the whole resolution bounds that and everything else, so the allowance and
its `waitLeft` bookkeeping are removed. The bound on **discovery slots** is unchanged: one
customer's slow servers holding both slots can delay another customer's session start by
at most 2 seconds, which then starts without MCP tools, as today.

**That is not isolation of the RPC workers.** ai-manager serves RPCs with 10 workers per pod
(`listenhandler/main.go:261`), and the same queue carries `tool_execute` for built-in tools
in live calls. A start that discovers holds a worker for up to 2 seconds, so a pod serves
about five such starts a second. One customer creating aicalls within the API rate limit
(16.7 per second, burst 33, `bin-api-manager/internal/config/main.go:146-147`) for an AI with
slow servers can occupy every worker on both pods, and every other customer's RPCs queue
behind them. That is not worse than main, where each start could hold a worker for eight
times ten seconds, and it is not reachable today (no AI has a whitelist). Per-customer
isolation of the workers (for example, at most two discovering starts in flight per
customer per pod, beyond which a start proceeds without MCP tools) is added to PR B2's
activation gates.

This departs from B13's title, "parallel fan-out". Discovery runs under a process-wide
limit of two concurrent listings, added in round 9 for memory. Parallel listing inside
one resolution would not add capacity; it would let one customer's session start take
both slots at once, and round 14 measured that one customer holding slots already
starves six of ten concurrent starts. The budget, not parallelism, is what bounds
latency. Revisit with the slot count (A.27) once B18's histogram exists.

**Whitelist order is priority order**, since a slow early server spends the budget of
later ones. The `mcp_server_ids` descriptions say so. Each resolution that skips a server
logs one summary line with the AI id, the elapsed time and each skipped server with its
reason (per-server timeout, budget spent, listing failed), in place of the scattered
per-server lines A.23 asked to consolidate.

### B20, single-flight per server

Concurrent listings of the same server share one execution, keyed by server id: a burst
of session starts for one AI lists its servers once rather than once per start. It is
hand-rolled (a map of in-flight executions under one mutex), since
`golang.org/x/sync/singleflight` is not vendored and adds nothing here.

- **The shared execution is detached from every caller.** It runs under
  `context.WithoutCancel` with its own `mcpDiscoveryPerServer` (1.5 seconds) deadline,
  which covers its wait for a slot, so its outcome never depends on whose call started
  it. Without this, a leader with 100 ms of its budget left would hand a deadline
  failure to every caller that joined it (design review round 1). Joining never extends
  its deadline, and the execution, its session close included, ends and releases its slot
  by that deadline (above).
- **Starting and joining.** A caller with less than `mcpDiscoveryStartFloor = 500 ms` of
  its budget left does not start a new execution (the server is skipped as "budget
  spent"), but may join one already in flight: joining costs nothing, and starting a
  second one would list the server twice and spend a slot.
- **An execution nobody waits for does not take a slot.** Waiters are counted under the
  map's mutex, and the caller that creates an execution is counted in the same critical
  section that inserts its map entry, before the execution's goroutine starts; the execution checks the count before it queues for a slot and again once
  it has one. When it is zero, the execution removes its own map entry **in that same
  critical section** and gives up, releasing the slot, so no caller can join an execution
  that has given up. No cancellation is involved: an execution already listing runs to its
  deadline.
- **Each caller waits only until its own deadline** (the smaller of its per-server bound
  and what is left of its 2-second budget), and a caller that gives up is recorded as
  "timed out waiting" in its own summary.
- **Finishing.** When the execution ends, it takes the mutex once and, in that critical
  section, stores its result, closes its done channel, and removes its map entry only if
  the entry still points to this execution (an execution that gave up has already removed
  its own, and a newer one may have taken the key). Every caller that joined received the
  entry before that critical section and reads the stored result when done closes; a
  caller arriving after it finds no entry and starts a new execution. This runs in a
  `defer`, so a panic still finishes the execution, with an error result.
- **D14 holds.** The usability check runs in each resolution before joining, never inside
  the shared execution. The execution's result depends only on the server row, which
  `ListTools` re-reads, so any two callers allowed to list a server may share one. The
  shared tool slice is read-only to every caller.
- Nothing is kept after the execution ends: B14's cache stays in PR B2. Since B28 now lands
  in this PR, B20 is an optimisation rather than a correctness bound; it is kept because
  it is small and the CEO chose to land all five items.

### Round 14's smaller findings, taken in the same change

The credential is replaced with `[REDACTED]` in any error body quoted from a non-2xx
response, in its raw form, without its `Bearer ` prefix, and JSON-escaped (a server
echoing the request's `Authorization` header otherwise wrote it to our logs;
pre-existing). The config comment and flag help say the timeout bounds a whole call.

### Tests, generated code and docs

- Listen handlers: POST and PUT over the cap, duplicated, identical-to-stored over the
  cap, Insight with a list, the type-only PUT, clear-and-switch in one request, and an
  already-Insight AI with a stored whitelist saving unchanged.
- `UpdateMcpServerIDs` refusing a new list on Insight; `ai.EffectiveType`.
- Discovery: Insight, duplicates, the eight-id limit, the aggregate and per-server
  deadlines with slow fake servers, the stop once the budget is spent, another
  customer's slow servers holding both slots costing at most the budget, the summary
  line, a listing plus a stalled `DELETE` ending by the per-server deadline with the tools
  returned, a handshake failure's close also ending by it, and `CallTool` ending by its
  timeout.
- B19: a 401 on `initialize`, on the notification and on `tools/list` (retried), a 401
  on `tools/call` (final), a 401 then a 404 and the reverse (one replacement session
  each), 403 with and without `invalid_token`, the re-read returning a newer token
  without a refresh, a re-read row that is deleted or no longer OAuth, recently written
  tokens refused without vendor contact (including a just-created row with NULL
  `tm_update`), a row whose URL host is not the vendor's never forcing a refresh, a dead
  grant backing off ten minutes in both vendors' shapes (`invalid_grant`, and GitHub's
  `bad_refresh_token` with HTTP 200), joining an
  exchange in flight, and the `DELETE` not triggering it.
- B20 under `-race`: callers sharing one request, a waiter leaving at its own deadline
  while the execution completes, an execution whose waiters all left not taking a slot
  and not being joined after it gave up, a caller under the start floor joining but not
  starting, a caller arriving after an execution finished starting a new one, an execution
  that gave up and was replaced not removing its successor's entry when it finishes, a
  panic finishing with an error, a leader with little budget left not failing a
  fresh waiter.
- `Test_Protocol_SessionCloseBounded` (`protocol_test.go:661-704`) changes: its limit of
  timeout plus 250 ms plus slack pins the old guarantee, and a 300 ms timeout would leave
  the three handshake requests 50 ms. It uses about a second and asserts the call ends by
  the timeout plus a small tolerance. `Test_Protocol_WholeCallDeadline` stays.
- Redaction of all three credential forms.
- Regenerate `pkg/aihandler/mock_main.go` and `pkg/mcpoauthhandler/mock_main.go`.
  Regenerate `bin-openapi-manager` and `bin-api-manager` from the OpenAPI change. Add the
  three reasons to `restful_api_errors.rst`, correct the Insight text, rebuild Sphinx
  and commit its output.

### Release notes, beyond the list below

`maxItems: 8` on the request schemas means a validating generated client rejects an
over-cap list before sending it, even the identical stored list the server would accept;
and the identical-list exemption is order-sensitive, so a client that sorts or otherwise
reorders the ids of an AI holding more than eight fails every save until the list is trimmed
to eight. Both are stated, with a query a self-hosted operator can run to
find AIs over the cap or holding a whitelist while Insight.

### Customer-visible changes, for the PR body

http URLs and redirects refused; at most 128 tools per server and 256 per session;
tools with invalid names dropped; at most eight servers per AI and no duplicates;
Insight AIs cannot be given a whitelist (one already held is kept, with no effect) and
do not discover; discovery bounded to two
seconds per session start and a second and a half per server, in whitelist order,
whatever the configured timeout; the timeout
setting bounds a whole call; one extra `DELETE` per call to a stateful server; an
OAuth server answering 401 (or 403 with `error="invalid_token"`) while listing tools, on
the vendor's own host, has its token refreshed and the call
retried once; a successful forced refresh at most once per 30 minutes per server (up to
twice across two replicas until A.27's lease lands), a failed one retried at most once a
minute per pod, or once per ten minutes when the vendor rejects the grant or VoIPBin's
client registration.
