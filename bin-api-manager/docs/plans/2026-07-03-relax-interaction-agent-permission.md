# Design: Allow CustomerAgent to use Interaction/Resolution APIs

## Problem

`square-talk` (the front-line agent chat/call app) needs to show which
Contact a caller/chatter belongs to, and let the agent resolve an
unrecognized peer into a Contact (create new or link to existing) — the
same flow `square-admin`'s "Unresolved Interactions" screen already
provides for Admin/Manager users.

Today `bin-api-manager/pkg/servicehandler/interaction.go` gates every
Interaction/Resolution operation behind:

```go
amagent.PermissionCustomerAdmin | amagent.PermissionCustomerManager
```

A regular agent JWT (bit `PermissionCustomerAgent`, 0x0010) fails this
check and gets `ErrPermissionDenied`. Since square-talk's users are
ordinary agents (not admins/managers), the feature is unusable there as-is.

## Decision

Widen the permission check on all five gated methods (read AND write) to
also accept `PermissionCustomerAgent`, using an explicit bitwise OR:

```go
amagent.PermissionCustomerAgent | amagent.PermissionCustomerAdmin | amagent.PermissionCustomerManager
```

**Not** `amagent.PermissionCustomerAll` (0x00F0). Round-1/round-2 design
review confirmed `PermissionCustomerAll` silently includes the reserved,
currently-undefined bit `0x0080` (`bin-agent-manager/models/agent/agent.go`).
If that bit is ever assigned to a new role in the future, these five call
sites would silently and immediately grant that new role access with no
code review touching this file. The explicit three-bit OR avoids that
forward-compatibility trap and is the only meaningful correction the
review loop produced — reviewers also confirmed `PermissionCustomerAll`
is otherwise used in exactly one place in the codebase (`storage_file.go`),
not the "established pattern" this doc originally implied.

Correction from `serviceagent_contact.go` comparison: that file's
`amagent.PermissionAll` (0xFFFF) is a special sentinel handled explicitly
in `Agent.HasPermission()` (`if perm == PermissionAll { return true }`) —
an unconditional pass, not a bitmask OR. It is a different mechanism than
what this change makes, even though the practical effect (any
authenticated identity of the matching customer passes) is similar. This
doc no longer claims the two are the same pattern, only that they reach a
similar practical outcome for a comparably-trusted class of operation.

No endpoint-scoping change, no new `/service_agents/interactions/*` path.
These endpoints already live at the top-level `/interactions/*` path (not
under `/service_agents/`) and are reachable by any authenticated identity
(agent JWT, accesskey, etc.) whose `customer_id` matches — `hasPermission`
already enforces the tenant boundary via `a.CustomerID != customerID`
before checking permission bits (confirmed in `etc.go`). Widening the
bitmask does not widen the tenant boundary; an agent still only ever sees
interactions within their own `customer_id`.

### Rejected alternative 1: scope by "interactions the agent personally handled"

No reliable, already-computed attribution of "which agent handled this
interaction" exists at the Interaction projection layer (Interaction is a
peer/local endpoint record derived from call/conversation source+dest, not
an agent-assignment record). Building this would require a new join/lookup
with real latency and correctness risk for marginal benefit.

### Rejected alternative 2: asymmetric read/write split (read=Agent+, write=Admin/Manager only)

The codebase has a real precedent for this shape
(`timeline_analysis.go`: `permTimelineAnalysisRead` includes CustomerAgent,
`permTimelineAnalysisWrite` does not), surfaced in design review. Under
this split, agents could see who a peer resolves to but could not create
or delete a Resolution themselves — new/unrecognized contacts would still
require an Admin/Manager to attach.

Explicitly rejected for this change: the entire point of bringing this
flow to square-talk is to let a front-line agent self-serve "create a new
Contact / link to an existing one" while handling a call or chat, without
routing through a supervisor. Read-only access for agents does not satisfy
that goal. `ResolutionCreate`/`ResolutionDelete` are opened to
`PermissionCustomerAgent` along with the three read methods.

**Known limitation, accepted for now:** `ResolutionDelete` has no
attribution check — any agent in the customer can delete any Resolution
regardless of who created it, and `ResolutionCreate`'s `resolved_by_id` is
client-supplied with no server-side check against the caller's own agent
ID (pre-existing gap, not introduced by this change). Design review
flagged this as a MUST-FIX in the review's own recommended shape, but a
deliberate scope decision here is to **not** add attribution
enforcement in this PR: there is no real incident driving it yet, and
adding an unverified control now adds complexity without confirmed value
(consistent with this team's general policy of not building robustness
against theoretical-only risk). Revisit if this is ever actually abused in
production — see "Not in scope" below.

## Changes

`bin-api-manager/pkg/servicehandler/interaction.go` — five call sites,
replace:

```go
if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
```
(or the `ia.CustomerID`/`res.CustomerID` variants)

with:

```go
if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAgent|amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
```

Affected methods: `InteractionList`, `InteractionListUnresolved`,
`InteractionGet`, `ResolutionCreate`, `ResolutionDelete`.

No OpenAPI spec change (the paths/schemas are unchanged, only the internal
authorization bitmask). No DB/migration change.

## Tests

Design review confirmed the existing five "permission denied" cases in
`interaction_test.go` all use `amagent.PermissionNone`, not
`PermissionCustomerAgent` — none of them flip. This is additive test
coverage, not a rewrite of existing cases:

- Add one new success-path test per method (5 total) in
  `interaction_test.go` asserting a `PermissionCustomerAgent`-only
  identity (no Admin/Manager bits) now succeeds.
- Keep existing `PermissionNone` / cross-`customer_id` rejection cases
  unchanged — they must still fail after this change.
- `server/interactions_test.go` has zero permission-path test cases
  today, and this PR does not add any there. That file's tests mock
  `ServiceHandler` entirely (`servicehandler.NewMockServiceHandler`), so a
  permission-bitmask regression in `interaction.go` is invisible at that
  layer regardless of what mock expectations are set — the actual
  authorization logic lives one layer down, in `pkg/servicehandler`, which
  is where the 5 new tests above already assert it directly. Adding a
  same-shaped test at the HTTP-handler layer would only prove the mock
  wiring works, not that permission enforcement works, so it is skipped
  as redundant rather than deferred as future work.

## Not in scope

- square-talk frontend consumption of these endpoints (separate PR/design).
- Any change to `ServiceAgentContact*` (already agent-accessible).
- Any change to the `/interactions/unresolved` since-window or pagination
  behavior.
- `ResolutionDelete` attribution/ownership enforcement and
  `ResolutionCreate`'s `resolved_by_id` spoofing gap (pre-existing, noted
  above as a known and currently-accepted limitation).
