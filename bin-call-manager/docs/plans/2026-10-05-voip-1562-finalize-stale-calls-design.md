# VOIP-1562: Finalize stale non-final calls (close three gaps in the existing health checks)

Status: Approved, revision 5 (design review: 2 consecutive full-approval rounds, rounds 4 and 5; NIT items applied)
Date: 2026-10-05
Ticket: VOIP-1562
Issue analysis (approved, revision 4, review rounds 3 and 4): `~/agent-hermes/notes/tracks/VOIP-1562-analysis.md`. This document restates only what the design needs; the evidence (database buckets, Loki traces, row counts) lives there.

Revision 5 (from design review round 4, MINOR/NIT only): the guard's effect on the bridge destroy retry and the no-retry event path stated; the live-call wording narrowed; imports named; the unassertable log mutation dropped and three mutations added; the activeflow-stop difference explained.

Revision 4 (from design review round 3): the `Hangup` guard keys on `tm_hangup` (a `call-control update-status hangup` writes only the status, and the late destroy event must still run the cleanup); a missing channel row on any status other than `dialing`/`canceling` only logs; one explicit failure rule (no new retry loops); recovery timing and R3 final state stated; docs wording and test plan corrected.

Revision 3 (from design review round 2): the watchdog now tells a truly missing channel row (`dbhandler.ErrNotFound`) from any other read error and only acts on the first; the missing-channel finalize applies only to `dialing` and `canceling` calls; a live channel is never hung up by the fallback; the R2 latency is corrected (about 40 s, no 3 s waits); the groupcall statement is corrected per caller path; the R3 evidence and blast radius are restated; test plan and mutations extended.

Revision 2 (from design review round 1): R2 no longer finalizes the call at the failure site. Finalizing immediately is unsafe when the channel create request fails ambiguously (the request errors, Asterisk still creates the channel): the late `StasisStart` would dial a call that is already `hangup`. R2 now uses the call health check as the single watchdog (section 3.1), which also removes the `outgoing_call.go` change. The R3 risk statement is corrected with evidence (section 3.3). Citations, tests, docs and rollout are corrected.

## 1. Problem

A call can stay in a non-final status (`dialing`, `canceling`, `progressing`) forever. On 2026-10-04, 97 such rows (oldest 2026-02-18) were closed by a one-off, CEO-approved database cleanup. Three verified causes can recreate them:

- R1 (56 rows, about 1.5 new rows per day since 2026-08-25): a ring-all groupcall with `hangup_others` hangs up a leg within about 50 ms of creation. The `ChannelDestroyed` event is processed before `StasisStart`, so the channel `type` is still empty and `EventHandlerChannelDestroyed` drops it (its `default:` branch only logs "Unsupported channel type", `arieventhandler/ari_channel.go`). The call is `canceling`. The call-level health check (`callhandler/health.go`) later reaches its retry limit and calls `HangingUp`, which returns early for a `canceling` call (`hangup.go:151-154`). Nothing finalizes the call.
- R2 (6 rows confirmed, outage 2026-10-02 00:19 UTC): `CreateCallOutgoing` inserts the call row and then `setVariablesCall` or `createChannelOutgoing` fails (Asterisk `Internal Server Error`). The row stays `dialing` (or `canceling` after the master's hangup) with no channel row. The call health check stops at its first tick when the channel lookup fails (`health.go:45-49`).
- R3 (the deadline for dead channels): the channel health check publishes a fake `ChannelDestroyed` after its retry limit, but the event has an empty `type`, so the router drops it (`subscribehandler/asterisk_proxy.go`, no handler). The channel row never ends and the call health check keeps resetting its retry count. Seen on 2026-09-30 07:44 UTC for two progressing calls. This is the "hard deadline" that the VOIP-1556 call recovery design assumes exists.

## 2. Non-goals

- No periodic reaper or new scheduled job (no measured signal; revisit trigger in section 7).
- No change to `ChannelDestroyed` routing for untyped channels, and no change to the order in which events are processed.
- No on/off flag. All fixes are always active.
- No change to groupcall counters: `HangupGroupcall` (`groupcallhandler/start.go:156` and `:233`) decrements only `groupcall_count`, so the `call_count` of a groupcall whose member call failed to be created stays above zero on those two paths today; this is an existing defect and is not touched here.
- No handling of the 29 older bucket C rows of unknown cause (analysis section 3, F2; the R2 watchdog may cover some future ones, but this is not promised), the legacy rows, incoming calls whose `StasisStart` arrives after the destroy, or `call_groupcalls` rows whose call count was never decremented by the old stuck calls (the operations query of 3.4 looks at `call_calls` only).
- No new metrics or alerts.

## 3. Changes (all in bin-call-manager)

### 3.1 R1 and R2: complete the call-level health check

`pkg/callhandler/health.go`, `HealthCheck`. Two edits and two new private methods.

Edit 1, channel lookup failure (currently lines 45-49, which log and `return`, ending the chain). The lookup becomes `h.db.ChannelGet(ctx, c.ChannelID)` (the call handler already holds the database handler) so that a truly missing row can be told from a read failure; `channelHandler.Get` cannot (it waits `defaultExistTimeout`, 3 s, and wraps every failure as not-found, `channelhandler/db.go:108-129`, `getWithTimeout` at `:413-445`). `ChannelGet` reads the cache first, then the database, and returns `dbhandler.ErrNotFound` only when the row is absent (`dbhandler/channel.go:127-143`, `:343`); any other failure is a different error.

```go
	cn, err := h.db.ChannelGet(ctx, c.ChannelID)
	switch {
	case errors.Is(err, dbhandler.ErrNotFound):
		// the channel row does not exist (yet): count it like an ended channel and keep watching
		retryCount++
	case err != nil:
		// a read failure says nothing about the channel: log and end the chain, exactly as today
		log.Errorf(...)
		return
	case cn.TMEnd != nil || cn.TMDelete != nil:
		retryCount++
	default:
		retryCount = 0
	}
	// send health check (unchanged)
```

A row that appears late (for example `ChannelCreated` arrives after a slow create request) is found at the next tick and resets the count. A read failure (any error other than not-found) still ends the chain, as today.

Edit 2, retry-exceeded branch (lines 23-27). Keep the log line (its text is a verification signature); replace the `HangingUp` call with `healthHangup(ctx, id, retryCount)`.

```go
// healthHangup finishes a call that stayed bound to an ended or missing channel for more than the max retry count.
func (h *callHandler) healthHangup(ctx context.Context, id uuid.UUID, retryCount int) {
	// read the latest state from the database, not the cache, immediately before acting
	c, err := h.db.CallGetFromDB(ctx, id)
	if err != nil {
		log.Errorf(...)
		return
	}

	// a call that finished meanwhile is left alone
	if c.Status == call.StatusHangup || c.TMHangup != nil || c.TMDelete != nil { return }

	cn, err := h.db.ChannelGet(ctx, c.ChannelID)
	switch {
	case errors.Is(err, dbhandler.ErrNotFound):
		// no channel row after 3 failed checks. Only a dialing call (or one moved to canceling by a hangup request)
		// can legitimately have no channel row: the dial never started and there is nothing to hang up.
		if c.Status == call.StatusDialing || c.Status == call.StatusCanceling {
			h.hangupFailedCall(ctx, c)
			return
		}
		// any other status never has a missing row in normal operation: log and return, as the chain does today
		log.Errorf(...)
		return

	case err != nil:
		// read failure: decide nothing; log and return, as today
		log.Errorf(...)
		return

	case cn.TMEnd == nil && cn.TMDelete == nil:
		// the channel is alive again (the counted failures were transient): start over
		reschedule with retry count 0 (`reqHandler.CallV1CallHealth(ctx, id, defaultHealthDelay, 0)`, as at `health.go:62`) and return

	case c.Status == call.StatusCanceling || c.Status == call.StatusTerminating:
		// HangingUp returns early for these statuses: finish through Hangup, the function the
		// ChannelDestroyed event would have run.
		if _, err := h.Hangup(ctx, cn); err != nil { log.Errorf(...) }
		return
	}

	// every other case keeps the existing behavior
	_, _ = h.HangingUp(ctx, id, call.HangupReasonNormal)
}

// hangupFailedCall finalizes a call whose channel never appeared.
func (h *callHandler) hangupFailedCall(ctx context.Context, c *call.Call) {
	if _, _, err := h.UpdateHangupInfo(ctx, c.ID, "", call.HangupReasonFailed, call.HangupByLocal); err != nil {
		log.Errorf(...)
		return
	}
	// a call created with a dummy activeflow (flow service was down at creation) has uuid.Nil: nothing to stop.
	// (`Hangup` calls the stop unconditionally; the extra guard here follows the earlier failure path in
	// `CreateCallOutgoing` and avoids a request that cannot succeed.)
	if c.ActiveflowID != uuid.Nil {
		if _, err := h.reqHandler.FlowV1ActiveflowStop(ctx, c.ActiveflowID); err != nil { log.Errorf(...) }
	}
}
```

(The snippets are sketches: `log` is the function's `logrus` entry as in the rest of the file. `health.go` needs two new imports: `errors` from `github.com/pkg/errors`, which the other call handler files use, and `monorepo/bin-call-manager/pkg/dbhandler`; `callhandler` already imports `dbhandler` elsewhere, so there is no cycle.)

Failure rule: this design adds no retry loop. Every read or write failure in `HealthCheck`, `healthHangup` and `hangupFailedCall` is logged and ends the chain, exactly as the existing code does for a call read failure (`health.go:30-36`) and a failed `HangingUp`. The only reschedule is the live-channel case (not a failure: the counted failures were transient and the watch must continue). A call left stuck by a failure at the moment of the fourth tick is detected by the operations query of 3.4 (section 7).

Behavior decisions:
- R1 outcome: `Hangup` gets the call's own ended channel (`ID = call.ChannelID`, cause 16, type call after the late `StasisStart`); `CallGetByChannelID` finds the call; `CalculateHangupReason(outgoing, canceling, 16)` gives `cancel` and `CalculateHangupBy(canceling)` gives `local`; `isRetryable` is false (cause 16 and status not dialing/ringing, `hangup.go:240-258`), so no failover channel is created; `call_hangup` is published, the activeflow is stopped, the groupcall is notified (this also lets a groupcall that was waiting on the stuck leg finish).
- R2 outcome: the call becomes `hangup`, reason `failed`, `hangup_by` local, `call_hangup` published (it follows the `call_created` already published), hangup metric counted once, activeflow stopped. For a call that a master hangup had moved to `canceling` the reason is also `failed` (the dial never started). Latency: about 40 s after creation: ticks every 10 s (no channel waits, `ChannelGet` does not block), the retry count reaches 3 on the third tick, and the fourth tick finalizes.
- Why not finalize at the failure site: when the create request fails ambiguously (request error or timeout, channel created by Asterisk anyway), `ChannelCreated` creates the row within milliseconds and the late `StasisStart` runs `startContextOutgoingCall`, which dials without checking the call status (`start.go:272-315`); a call already in `hangup` cannot be updated back (`IsUpdatableStatus`, `models/call/call.go`), so a live dial would go untracked. Waiting for three failed checks practically closes that window (the create request itself times out after 3 s, `requestTimeoutDefault`); a request replayed from a queue after 40 s would still be an unhandled corner, and the code path that dials without checking the status stays as it is.
- The groupcall is deliberately not notified in `hangupFailedCall` (`Hangup` would). In the observed R2 path the groupcall caller already decrements `call_count` (`groupcallhandler/start.go:358` and `dial.go:129` call `HangupCall`); a second decrement would double-count. The other two callers (`start.go:156` and `:233`, `HangupGroupcall`) decrement a different counter (section 2).
- Mid-transition safety for the `Hangup` branch: only `canceling` and `terminating` calls take it, which a call recovery switch never touches (it requires `progressing` and the old channel, `dbhandler/call.go:709-710`). Route failover (dialing/ringing only, `hangup.go:258`) overwrites `channel_id` unconditionally (`dbhandler/call.go:522-528`); if it lands between the read and `Hangup`, `CallGetByChannelID(old channel)` finds no call and `Hangup` returns an error that is only logged. The final write is not made conditional on the channel id (that would change the shared `Hangup`).
- The master call's `ChainedCallIDs` may contain the orphan; the master's hangup calls `HangingUp(orphan)`, which moves it to `canceling` and fails to hang up the missing channel (logged); the watchdog finalizes it as above; a later master `HangingUp(orphan)` returns early on a final call.
- The watchdog does not act on a call whose channel reads as alive: `hangupFailedCall` runs only for `dialing`/`canceling` with a row that is absent after three consecutive absent reads; `Hangup` runs only for `canceling`/`terminating` with a channel read as ended; `HangingUp` (today's behavior) runs only for a channel read as ended; an alive channel reschedules; every read error ends the chain. The limits (a transient control-plane failure that makes the channel health check declare a live channel dead, and an ambiguous create failure whose channel row appears very late) are stated in 3.3 and below.

### 3.2 `Hangup` guard against a second run

`pkg/callhandler/hangup.go`, `Hangup`, right after the debug log that follows the `CallGetByChannelID` lookup (the lookup is at the top of the function):

```go
	if c.TMHangup != nil {
		log.Infof("The call has hungup already. Skipping. call_id: %s, channel_id: %s", c.ID, cn.ID)
		return c, nil
	}
```

The condition is `tm_hangup`, not `status`. Every real finalization writes `tm_hangup` (`CallSetHangup`, `CallSetHangupIfChannel`), while the documented manual fallback `call-control call update-status --status hangup` writes only the status (`callhandler/db.go` `dbUpdateStatus` comment: the hangup event "must be done with Hangup()"; `CallSetStatus` in `dbhandler/call.go`). A call forced to `hangup` that way must still get the full cleanup (`tm_hangup`, webhook, activeflow stop, groupcall, chained calls) from the real `ChannelDestroyed` event, so the guard must not skip it. It also matches the final-call test in `HealthCheck` (`Status || TMHangup || TMDelete`).

Why: `Hangup` has no early return for a final call. A second run rewrites `hangup_reason`, `hangup_by` and `tm_hangup`, republishes `call_hangup`, counts metrics twice and repeats the groupcall, activeflow and chained-call steps (`UpdateHangupInfo` is unconditional for every status except `progressing`). R3 makes a second run possible: the fake event can finalize a call first and a real `ChannelDestroyed` can arrive later if the failure was transient. Limits: the guard is a read-then-act check, so two replicas entering `Hangup` at the same moment can both pass it, as they can today; it removes the realistic late-event duplicates, not a same-instant race. Callers: `ARIChannelDestroyed` (`arievent.go:31`) ignores the returned call; `hangingUpWithCause` (`hangup.go:197`) uses it as the result and receives the finalized call, which is correct. A `progressing` call is unaffected; the VOIP-1556 owner-conditioned write stays as is. Two more effects. First, `Hangup` destroys the call bridge before this point (`hangup.go:35-39`) and the guard sits after the lookup but before that step, so a late real destroy no longer retries a bridge destroy that failed in the first run (the bridge is also removed by its own events and by Asterisk when the channels are gone). Second, if a first run wrote `hangup` and then failed before publishing the webhook, a second run no longer republishes it; the asterisk-proxy event path does not retry a failed handler today, so this changes nothing in practice. No existing test reaches `Hangup` with a `hangup` call.

### 3.3 R3: give the fake destroy event a type

`pkg/channelhandler/health.go`, `publishFakeEventChannelDestoryed`:
- `Event: ari.Event{Type: ari.EventTypeChannelDestroyed}`. Verified with a throwaway round trip: with `Type` set and an empty `Timestamp`, `ari.Parse` returns the typed event and no error; the handler needs only `Channel.ID` and `Cause` (`EventHandlerChannelDestroyed`). `Application`, `Timestamp` and `AsteriskID` stay empty (the `asterisk_id` metric label shows an empty value for these events).
- Log the publish failure (`log.Errorf`), which is swallowed today.

Effect: after the retry limit, the channel row ends, and `Delete` and `ARIChannelDestroyed` run `Hangup` for a call channel. The call health check also sees the ended channel and finalizes through 3.1. A second run is stopped by 3.2.

Risk, stated with the evidence found in review (not as an assumption): the channel health check cannot tell an unreachable Asterisk from a failing ARI request: `AstChannelGet` errors are not classified (404, 5xx and timeouts are all failures, `channelhandler/health.go`), and a dead Asterisk shows up as a timeout, so timeouts must count. Therefore an ARI request outage that lasts longer than about 40 s while the WebSocket event path is still alive would end the affected calls in the database (a stalled RPC path or proxy hits every channel of that Asterisk at once, so the affected set can be all of its live calls) (activeflow stopped, `call_hangup` published) while media continues. Blast radius: it is not limited to calls. When the fake event ends a channel, its row gets `tm_delete`, and later operations on that channel (playback, hold, mute and similar, for example `channelhandler/etc.go`) are rejected for a channel that may still be alive. Observed frequency: in the 14 days of available logs (from 2026-09-21), the deadline fired once, on 2026-09-30 07:44 UTC (the Asterisk recreate, six channels). It did not fire in the 2026-10-02 00:19 UTC episode (failed channel create requests seen in three bursts at 00:19:00, :28 and :58, a span of at least 58 s, with no later failure in the logs); this is weak evidence only, because the failing requests were channel creates, the later traffic was not channel creation, and it is not known whether any live channel was being health-checked during that time. No mitigation is added: any classification of errors would be a guess, and a flag is excluded. The fake event is handled once: a handler error is only logged and the event is not redelivered (`subscribehandler/asterisk_proxy.go`: handler errors are logged and `nil` is returned), so if the `Hangup` it triggers fails, the call-level health check (3.1) is the backup. Final state for a progressing call finalized by the deadline: `hangup`, reason `normal`, `hangup_by` remote (`CalculateHangupBy(progressing)`), `call_hangup` published, activeflow stopped. A call that recovery moved to a new channel is unaffected (`CallGetByChannelID(old channel)` finds no call; the progressing hangup write is owner-conditioned, VOIP-1556). A recovery still in progress when the deadline arrives (the deadline is about 30 to 40 s after the container death; the recovery budget is longer) loses the race: the call is finalized and the recovery switch is refused and its recovery leg hung up. The VOIP-1556 design already accepts this ordering (`2026-10-02-voip-1556-call-recovery-core-design.md`: the switch is expected well before the old channel's fake destroy, lines 100, 283 and 290); it is a decision for the CEO that this design makes that deadline real. The CEO decision of 2026-10-03 (no drain, rely on recovery) already depends on this deadline existing. Rollback is reverting the PR. This risk needs the CEO's explicit acceptance (section 8).

### 3.4 Docs

`docs/operations.md`:
- add a section "Stale non-final calls" before "Debugging Guide": the read-only detection query, the log lines that show the mechanism working ("Exceeded max call health check retry count", "Exceeded max channel health check retry count", "The call has hungup already. Skipping."), and the revisit trigger of section 7;
- rewrite the Failure Modes row about calls orphaned after an Asterisk container died: call recovery is unconditionally on (the row still says it is disabled by default); the channel health deadline now finalizes the leftover call channel about 30 to 40 s after the container is declared dead (fake `ChannelDestroyed`, runs `Hangup`), and the call health check is the backup about 40 s after a channel is found ended; `call-control call update-status --status hangup` remains the manual fallback and now keeps the real destroy event's cleanup (guard keyed on `tm_hangup`);
- extend the recovery noise paragraph (about the not-found log for an old channel) with the fake destroy event of the old channel, and correct its statement that the event consumer retries (it does not: handler errors are only logged, `asterisk_proxy.go`).

## 4. Tests (written together with the implementation)

- `callhandler/health_test.go`:
  - extend `Test_HealthCheck`: its three existing cases mock `channelHandler.Get` and must be changed to `mockDB.EXPECT().ChannelGet(...)` (the lookup moves to the database handler), then add: channel row missing (`ErrNotFound`) increments the retry count and reschedules; other channel read error ends the chain (no reschedule); `retryCount` 3 calls `healthHangup` instead of `HangingUp` directly (wiring of edit 2).
  - new `Test_healthHangup` (table, strict mocks; `GroupcallID` left empty to avoid the goroutine; the handler struct includes the bridge handler): canceling call with an ended channel finalizes through `Hangup` once (mock order: `CallGetFromDB`, `ChannelGet`, `CallGetByChannelID`, bridge `Destroy`, `CallSetHangup`, `CallGet`, webhook, activeflow stop; `Hangup` stops the activeflow even for `uuid.Nil`, unlike `hangupFailedCall`); terminating call with an ended channel; channel missing with `dialing` and with `canceling` finalizes as `failed`/local with webhook and activeflow stop, and a dummy (nil) activeflow is not stopped; channel missing with `ringing`, `progressing` and `terminating` finalizes nothing and calls nothing (log and return); live channel reschedules with retry count 0 and calls nothing else; call or channel read error ends the chain (no reschedule); final call does nothing (one case per condition: only `Status` is `hangup`, only `TMHangup` is set, only `TMDelete` is set); dialing/ringing/progressing with an ended channel go through `HangingUp` (its own mock sequence: `CallGet` and the status update); `Hangup` error is only logged.
- `callhandler/hangup_test.go`: new cases in `Test_Hangup` (its current table sets the `CallGetByChannelID` and bridge `Destroy` expectations for every case, so those become per-case): a call with `tm_hangup` set (returns the call; no bridge destroy, no write, no webhook); a call with status `hangup` and `tm_hangup` nil (continues and runs the full cleanup, the force-updated case).
- `channelhandler/health_test.go`: replace the expectation in `Test_publishFakeEventChannelDestoryed` (which pins `"type":""`) with a round trip: the payload parses with `ari.Parse` to `EventTypeChannelDestroyed` and carries the channel ID and cause 16; a publish error returns without effect (the error log itself is not asserted; the repo's tests use no log hook).
- `subscribehandler` test: the existing `Test_processEvent_AsteriskProxy_ChannelDestroyed` already covers a typed payload; add one case that feeds the exact bytes the fake event publishes (so the router-map coverage promised in the analysis is tied to the real payload).
- Mutation checks in a `git archive` copy, each must fail at least one test: remove the not-found retry increment; make a read error count as a retry or reschedule; revert edit 2 (call `HangingUp` directly); remove each of the `Status`, `TMHangup` and `TMDelete` conditions of the final-call check in `healthHangup`; remove the `ActiveflowID != uuid.Nil` check in `hangupFailedCall`; remove the activeflow stop in `hangupFailedCall`; make the live-channel reschedule keep the old retry count instead of 0; change the `canceling`/`terminating` condition; remove the live-channel reschedule; remove the `dialing`/`canceling` restriction of the missing-channel branch (a missing row on `progressing` must not finalize); remove the missing-channel branch; remove the `Hangup` guard; change the guard back to the status; restore the empty event type.

## 5. Risks and alternatives

- Risks are listed with their mitigations in 3.1, 3.2 and 3.3. The largest behavior change is R3 (a deadline that has never run starts running), explicitly accepted or not in section 8.
- Alternatives rejected in the analysis (section 4): a periodic reaper (B), routing untyped destroy events to the call handler (D), changing the early return in `hangingUpWithCause` (E), finalizing at the failed add-to-bridge in the late `StasisStart` path (F). Rejected in this design: finalizing the call at the channel create failure site (unsafe for an ambiguous create failure, see 3.1); a best-effort channel hangup at the failure site (the channel row does not exist yet, so it cannot work reliably).

## 6. Rollout and verification

- Merge, then deploy by the CEO's normal path. No schema or config change, no flag.
- Before or after deploy, if a sandbox is available: stop the call Asterisk container with a virtual call up and confirm the call is finalized about 40 s after its channel is declared dead (optional rehearsal of a mechanism that has not run in production).
- After deploy: the Loki search for "Unsupported channel type. type: " still shows the dropped events (the cause is not removed, its effect is), each such call that stayed `canceling` is now followed within about 40 s by "Exceeded max call health check retry count" and a `Hangup` log (a dropped event for a call that finished normally has no such follow-up); the operations query of 3.4 returns 0 rows after the first deploy window; the next Asterisk recreate shows "Exceeded max channel health check retry count" followed by a `Hangup` for a leftover call channel.

## 7. Revisit trigger (reaper)

Add a periodic job only if, after this is deployed, the operations query returns non-final calls older than 2 hours again (the longest naturally ended answered call in 90 days was 3599 s). The query is run by the on-duty operator after each bin-call-manager or Asterisk deploy; there is no alert.

## 8. Decisions needed

- Confirm the `Hangup` guard of 3.2 (it changes a shared path; adopted with its limits stated there).
- Accept the R3 risk of 3.3 (enable the channel health deadline). The recommendation is to accept: it is the mechanism VOIP-1556 and the no-drain decision assume, it fired once in 14 days, and it did not fire in the one observed ARI error episode.

## Review history

- Round 1 (2026-10-05): A CHANGES_REQUESTED (MAJOR: ambiguous create failure then late StasisStart; MINOR: groupcall citations, chained-call wording, health chain breaks on read error, citations, test mock order; NIT: `TMDelete` consistency, undefined `log`); B CHANGES_REQUESTED (MAJOR: R3 blast radius unverified, R2 ambiguous failure; MINOR: guard atomicity wording, `TMDelete`, mutation list, groupcall citations, docs scope, rollout, `call_groupcalls` non-goal). Addressed in revision 2.
- Round 2 (2026-10-05): A CHANGES_REQUESTED (MAJOR: counted read failures could lead the fallback to hang up a live channel; MINOR: R2 latency, test claims, missing wiring test; NIT: pseudo-code, citation split); B CHANGES_REQUESTED (MAJOR: groupcall double-count claim wrong for two caller paths, missing-channel finalize not restricted by status; MINOR: latency, 'window removed' overclaim, R3 evidence strength and blast radius; NIT: guard partial failure). Addressed in revision 3.
- Round 3 (2026-10-05): A CHANGES_REQUESTED (MINOR: the 'keeps today's HangingUp' claim for a missing row on other statuses is false, existing `Test_HealthCheck` mocks must move to `ChannelGet`; NIT: line numbers, sketch return); B CHANGES_REQUESTED (MAJOR: guard keyed on status regresses the documented `update-status hangup` fallback, key on `tm_hangup`; MINOR: missing-row branch for other statuses, inconsistent failure rule, recovery timing interplay, docs wording; NIT: R3 evidence wording, final state). Addressed in revision 4.
- Round 4 (2026-10-05): A APPROVED (MINOR: unassertable log mutation, `Hangup` activeflow stop in the mock sequence; NIT: import names, missing mutation); B APPROVED (MINOR: guard skips a bridge destroy retry, 'live call cannot be hung up' overclaim, 'consumer retries' premise in operations.md; NIT: missing mutations, activeflow stop difference). Applied in revision 5.
