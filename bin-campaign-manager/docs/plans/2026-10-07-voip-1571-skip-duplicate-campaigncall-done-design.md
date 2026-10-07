# VOIP-1571: Skip the duplicate Done for an already completed campaigncall on a late call hangup

Status: Draft, revision 1
Date: 2026-10-07
Ticket: VOIP-1571
Issue analysis (approved, revision 6, review rounds 4 and 5): `~/agent-hermes/notes/tracks/VOIP-1571-analysis.md`. This document restates only what the design needs; the evidence (callers, event ordering, production numbers, counterexamples) lives there.

## 1. Problem

campaign-manager runs `campaigncallHandler.Done` a second time for a campaigncall that is already `done` when a late `call_hangup` arrives for the same call. `Done` (`pkg/campaigncallhandler/status.go:16-49`) has no already-done check, so every call writes the campaigncall again, publishes `campaigncall_updated`, counts `promCampaigncallDoneTotal`, and overwrites the outdial target status (`UpdateStatus` is an unconditional write).

Trigger: `executeCall` marks the campaigncall `Fail` when `CallV1CallCreateWithID` returns an error (`pkg/campaignhandler/execute.go:270`). A `call_hangup` for the same call id can still follow: since VOIP-1562 a call whose channel row never appeared is finalized as `failed` about 40 s after creation, and a create RPC that errors while call-manager still creates the call could already do it before. `processEventCMCallHungup` (`pkg/subscribehandler/callmanager.go`) then calls `EventHandleReferenceCallHungup`, which calls `Done` again.

Production impact today: none (all 9325 campaigns stopped, 14 campaigncalls ever, the last in 2022). It affects self-hosted installations that run campaigns.

## 2. Non-goals

- No guard inside `Done` and no change to its signature or to the other `Done` callers (analysis section 4, alternative B).
- No atomic conditional database update (the two `Done` calls are not concurrent).
- No change to the `execute.go:270` path itself, to the `execute.go:332-336` nil dereference, or to the `isStoppable` fail-open (analysis section 6).
- No reaper, flag, or metric. No change to call-manager.
- No handling of the order where the hangup arrives before the `execute.go:270` Done (analysis section 5, question 2: practically excluded; only a guard inside `Done` could cover it).

## 3. Design

All code is in bin-campaign-manager.

### 3.1 The guard (option A, rule (i))

Place: `campaigncallHandler.EventHandleReferenceCallHungup` (`pkg/campaigncallhandler/eventhandle.go`), right after the hangup reason is mapped to a result and before `h.Done`. This is the only place that knows both the already stored campaigncall (`cc`, loaded by the subscriber with `GetByReferenceID`, a fresh database read) and the new result. It reuses the existing pattern of `flowmanager.go:35` (skip when already done) and keeps `Done` untouched.

Rule (i) of the analysis: a campaigncall that is already `done` is left alone, with one exception, a late success after a recorded failure:

```go
// the campaigncall is already done. a second Done would repeat the webhook, the metric and the outdial target update.
// the only legitimate second Done is a late success after a recorded failure (the create request errored although the
// call was created and answered): it corrects the result and the outdial target.
if cc.Status == campaigncall.StatusDone && !(cc.Result == campaigncall.ResultFail && result == campaigncall.ResultSuccess) {
    log.WithFields(...).Infof("The campaigncall is already done. Skipping. campaigncall_id: %s, stored_result: %s, new_result: %s", ...)
    return cc, nil
}
```

Behavior table (stored state, new result):

| Stored | New | Action | Why |
|---|---|---|---|
| dialing or progressing | any | `Done` | normal hangup |
| done, fail | fail | skip | the duplicate this ticket removes (Fail then Fail) |
| done, fail | success | `Done(Success)` | late success: the call was answered, the target must become `done`, not `idle` (redial risk) |
| done, success | success | skip | duplicate |
| done, success | fail | skip | a `done` target must not flip back to `idle`; practically excluded order |
| done, none | any | skip | `Done` always writes Success or Fail, so this is not expected; skipping is the safe default |

Return value on skip: `(cc, nil)`, the stored campaigncall, not `nil`. The caller then continues with the campaign-level handler for `cc.CampaignID` (section 3.2).

The late `Done(Success)` after `Done(Fail)` publishes a second `campaigncall_updated` (status done, result success) and counts the metric once more under `success`. That is the state correction, accepted.

The result mapping error path (`calcCampaigncallResultByCallHangupReason` fails for an unknown reason) is unchanged: it returns `nil, err`.

### 3.2 The subscriber and the existing nil dereference

`pkg/subscribehandler/callmanager.go`, `processEventCMCallHungup`:

- When `EventHandleReferenceCallHungup` returns an error, `newCC` is nil and the next statement dereferences `newCC.CampaignID` (panic). The function logs the error and returns `nil` instead of continuing. The event handler already treats a returned nil as handled (errors are not redelivered), so behavior for the error path becomes "log and stop" instead of a panic. No retry is added.
- On a skip (section 3.1) `newCC` is the stored campaigncall, so the campaign-level `EventHandleReferenceCallHungup(ctx, newCC.CampaignID)` still runs. It is idempotent: it acts only when `isStoppable` is true (execute stopped and no `dialing` or `progressing` campaigncall), and `campaignStopNow` returns early without a webhook when the campaign is already `stop` (`pkg/campaignhandler/status_stop.go`). So a duplicate event cannot publish a second campaign webhook, and a stop that depends on this event is not lost.
- The wrong comment above the function ("confbridge_leaved") is corrected to the real event name (`call_hangup`).

### 3.3 Failure rule

No new retry loops. A skip is a normal return. Read errors are logged and end the event handling as today. Concurrency: the guard is read-then-act. The duplicate events are far apart in time (about 40 s for the VOIP-1562 trigger, the call duration for a late success), so a same-instant race is not a realistic case; two replicas handling the same event in the same instant could both pass, which is the existing behavior and is left as is.

### 3.4 Docs

If `bin-campaign-manager/docs` has a section that describes the campaigncall lifecycle or the hangup event, add two sentences there (an already done campaigncall is not completed again, except a late success after a failure). Otherwise no doc change. The implementation step checks the files (`domain.md`, `operations.md`) and reports which one was edited.

## 4. Tests (written with the implementation, not before)

`pkg/campaigncallhandler/eventhandle_test.go`, `Test_EventHandleReferenceCallHungup` (existing eight reason cases stay unchanged and use a dialing campaigncall). New cases, with strict gomock expectations so any unexpected `Done` call fails the test:

1. done + fail stored, failed hangup: no `Done` call (no database write, no webhook, no outdial request), returns the stored campaigncall.
2. done + fail stored, hangup `busy` (fail): same skip.
3. done + success stored, normal hangup: skip.
4. done + success stored, failed hangup: skip.
5. done + fail stored, normal hangup: `Done(Success)` runs (database write, webhook, outdial `done`).
6. done with an empty result stored: skip.
7. unknown reason: unchanged error and `nil` result.

`pkg/subscribehandler/callmanager_test.go`, `Test_processEventCMCallHangup` (today one case):

8. `GetByReferenceID` fails: returns nil, no handler call.
9. `EventHandleReferenceCallHungup` returns an error: no panic, the campaign handler is not called, returns nil.
10. skip result (returns the stored campaigncall): the campaign handler is called with its `CampaignID`.

Mutation checks (each must fail at least one test), run in an isolated `git archive` copy:

- remove the whole guard; drop the `Status == done` term; drop the exception term (late success blocked); invert the exception (`Fail` to `Success`); change `ResultFail` in the exception to `ResultSuccess`; return `nil, nil` instead of `cc, nil` on skip; remove the nil return in the subscriber; call the campaign handler with `cc.CampaignID` replaced by an empty id.

Existing tests are not weakened (cases and expectations stay as they are).

## 5. Rollout and rollback

- No schema, configuration, flag, or data change. Rollback is a revert of the PR.
- In-flight events at deploy time are handled by the new code on arrival. A campaigncall done before the deploy with a late hangup after the deploy is skipped by the new rule, which is the intended behavior.
- Customer-visible change: fewer duplicate `campaigncall_updated` webhooks. One extra `campaigncall_updated` remains by design for a late success after a failure.

## 6. Risks and accepted limits

- A skipped duplicate cannot repair a partial `Done` (the outdial request failed inside the first `Done`, so the target stays `progressing`). This is a theoretical edge case: no reaper exists today, so such a target would stay `progressing` regardless of this change except through this late hangup. Accepted (analysis section 5, question 2). Revisit trigger: an outdial target stuck in `progressing` with a done campaigncall is seen in production.
- The late `idle` overwrite of a target redialed by a newer campaigncall is removed for the hangup path only. The `execute.go:270` order (hangup first) is not covered (non-goal).
- The rule adds one condition to a handler that already contains the result mapping. No new state, no new field.

## 7. Open points for review

1. Is the single exception (late success after fail) worth its one extra condition, versus plain skip-if-done (option (ii))? The design says yes because the failure mode is a redial of an answered number.
2. Is returning the stored campaigncall on skip, with the campaign-level handler still running, the right contract (versus returning early from the subscriber)?
