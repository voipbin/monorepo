# VOIP-1571: Skip the duplicate Done for an already completed campaigncall on a late call hangup

Status: Approved, revision 11 (design review: 2 consecutive full-approval rounds, rounds 9 and 10; MINOR/NIT items applied)
Date: 2026-10-07
Ticket: VOIP-1571
Revision history (newest first):

- Revision 11 (from design review round 10, MINOR/NIT only): all subscriber cases 8 to 10 call `processEventCMCallHungup` directly (a case that goes through `processEvent` needs the event `Publisher` set to `call-manager` to be dispatched); log content is not asserted.
- Revision 10 (from design review round 9, NIT only): section 8 notes that the no-merge-without-instruction rule is stated by both the root and the user-level `CLAUDE.md`.
- Revision 9 (from design review round 8): section 8 groups the rules by their real source (root `CLAUDE.md`, the user-level `/home/pchero/CLAUDE.md`, CI, environment) and no longer attributes everything to the root file; the new test function names are fixed.
- Revision 8 (from design review round 7, MINOR/NIT only): the subscriber tests that assert the return value call `processEventCMCallHungup` directly (`processEvent` returns nothing); the optional verification names a reproducible failure and the wait; the same-PR statement rests on the one-PR rule; repository rules and CI conditions are listed separately.
- Revision 7 (from design review round 6): sections 8 (PR plan with the repository rules and the files expected in the diff) and 9 (verification after deploy, stating that production has no running campaign) added; an optional case for a stored dialing campaigncall with a failed hangup added.
- Revision 6 (from design review round 5, MINOR/NIT only): the accepted limit states that this change removes the incidental repair of a partial `Done` by a duplicate hangup; the unchanged interface (no mock regeneration) is stated; the lint note moved out of the code block; the outdial file path is complete; the duplicate no-impact sentence is removed; revision 3 wording corrected.
- Revision 5 (from design review round 4): the guard is written in the De Morgan form that passes the CI lint (QF1001 rejected the negated conjunction); the log sketch has real arguments; two mutations added; the customer-visible result change is stated.
- Revision 4 (from design review round 3): the subscriber assignment is `_, err = ...` (`err` is already declared at `callmanager.go:27`, so `:=` does not compile); the evidence the design depends on is summarized in section 1 so the document can be read without the analysis note; the revision history is a list.
- Revision 3 (from design review round 2, MINOR/NIT only): the docs target is fixed (`domain.md`, Campaigncall Lifecycle); explicit cases for a stored dialing and progressing campaigncall; the subscriber test needs per-case expectations; the assignment form of the subscriber call is specified in revision 4; the first table row reads "any state other than done".
- Revision 2 (from design review round 1): the error path of the subscriber no longer returns early; it keeps the original intent (log the error, still run the campaign-level handler) using the `CampaignID` of the campaigncall the subscriber already loaded, so a stop signal is not lost; the current nil dereference is a process crash (the event runs in a goroutine without `recover`); the consumer wording is corrected; the late-success exception side effect is stated as an accepted limit; the test plan asserts return values and tightens case 6.

Issue analysis (approved, revision 6, review rounds 4 and 5): `~/agent-hermes/notes/tracks/VOIP-1571-analysis.md`. This document restates only what the design needs; the evidence (callers, event ordering, production numbers, counterexamples) lives there.

## 1. Problem

campaign-manager runs `campaigncallHandler.Done` a second time for a campaigncall that is already `done` when a late `call_hangup` arrives for the same call. `Done` (`pkg/campaigncallhandler/status.go:16-49`) has no already-done check, so every call writes the campaigncall again, publishes `campaigncall_updated`, counts `promCampaigncallDoneTotal`, and overwrites the outdial target status (`UpdateStatus` is an unconditional write).

Trigger: `executeCall` marks the campaigncall `Fail` when `CallV1CallCreateWithID` returns an error (`pkg/campaignhandler/execute.go:270`). A `call_hangup` for the same call id can still follow: since VOIP-1562 a call whose channel row never appeared is finalized as `failed` about 40 s after creation, and a create RPC that errors while call-manager still creates the call could already do it before. `processEventCMCallHungup` (`pkg/subscribehandler/callmanager.go`) then calls `EventHandleReferenceCallHungup`, which calls `Done` again.

Evidence the design relies on (all code-verified; details in the analysis note, which lives outside the repository):

- `Done` has no already-done check and its effects are the campaigncall write with the `campaigncall_updated` webhook, the done metric, and an unconditional outdial target status write (`status.go:16-49`, `bin-outdial-manager/pkg/outdialtargethandler/outdialtarget.go:209-233`).
- The only unguarded duplicate source is the call-hangup path; `pkg/subscribehandler/flowmanager.go:35` already guards the activeflow path.
- Retry counting reads the outdial target `TryCount`, not campaigncall rows, and nothing else in the repository reads `campaigncall.Result`, so correcting a late Fail to Success changes the outdial target state and the result that customers see through the API and the `campaigncall_updated` webhook (the second webhook is accepted, section 3.1).
- `campaignStopNow` returns early (no webhook) when the campaign is already stopped, so the campaign-level step is idempotent.

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

Place: `campaigncallHandler.EventHandleReferenceCallHungup` (`pkg/campaigncallhandler/eventhandle.go`), right after the hangup reason is mapped to a result and before `h.Done`. This is the only place that knows both the already stored campaigncall (`cc`, loaded by the subscriber with `GetByReferenceID`, a fresh database read) and the new result. It reuses the existing pattern of `pkg/subscribehandler/flowmanager.go:35` (skip when already done) and keeps `Done` untouched.

Rule (i) of the analysis: a campaigncall that is already `done` is left alone, with one exception, a late success after a recorded failure:

```go
// the campaigncall is already done. a second Done would repeat the webhook, the metric and the outdial target update.
// the only legitimate second Done is a late success after a recorded failure (the create request errored although the
// call was created and answered): it corrects the result and the outdial target.
if cc.Status == campaigncall.StatusDone && (cc.Result != campaigncall.ResultFail || result != campaigncall.ResultSuccess) {
    log.Infof("The campaigncall is already done. Skipping. campaigncall_id: %s, stored_result: %s, new_result: %s", cc.ID, cc.Result, result)
    return cc, nil
}
```

Behavior table (stored state, new result):

| Stored | New | Action | Why |
|---|---|---|---|
| any state other than done (normally dialing or progressing) | any | `Done` | normal hangup |
| done, fail | fail | skip | the duplicate this ticket removes (Fail then Fail) |
| done, fail | success | `Done(Success)` | late success: the call was answered, the target must become `done`, not `idle` (redial risk) |
| done, success | success | skip | duplicate |
| done, success | fail | skip | a `done` target must not flip back to `idle`; practically excluded order |
| done, none | any | skip | `Done` always writes Success or Fail, so this is not expected; skipping is the safe default |

The guard is written in the De Morgan form (`Status == done && (Result != fail || result != success)`) because the repository lint (staticcheck QF1001) rejects the negated conjunction `!(a && b)`; the two forms are logically identical. `CampaigncallHandler` keeps its method set and signatures, so no mock is regenerated.

Return value on skip: `(cc, nil)`, the stored campaigncall, a natural "current state" result (the subscriber does not depend on it, section 3.2).

The late `Done(Success)` after `Done(Fail)` publishes a second `campaigncall_updated` (status done, result success) and counts the metric once more under `success`. That is the state correction, accepted.

The result mapping error path (`calcCampaigncallResultByCallHangupReason` fails for an unknown reason) is unchanged: it returns `nil, err`.

### 3.2 The subscriber and the existing nil dereference

`pkg/subscribehandler/callmanager.go`, `processEventCMCallHungup`:

- Existing defect: when `EventHandleReferenceCallHungup` returns an error, `newCC` is nil and the next statement dereferences `newCC.CampaignID`. The event runs as `go h.processEvent(m)` with no `recover` anywhere in the service (`subscribehandler/main.go`, grep of `recover()` finds none), so this is a process crash, not a recovered panic. The original intent is visible in the code: the error branch only logs and has no `return`, so the campaign-level handler is meant to run even when the campaigncall handling failed (it matters because `Done` writes the campaigncall as done before its outdial request, so after a failed outdial request the campaigncall is done and the campaign still needs its stop check).
- The fix keeps that intent: the subscriber uses the `CampaignID` of the campaigncall it already loaded (`cc.CampaignID`, from `GetByReferenceID`) for the campaign-level call, so it no longer depends on the returned value. On an error it logs and continues. No retry, no early return.
- After the change `newCC` is not used by the subscriber, so the call becomes `_, err = h.campaigncallHandler.EventHandleReferenceCallHungup(...)` (plain assignment: `err` is already declared by `cc, err := ...GetByReferenceID` at `callmanager.go:27`, so `:=` would fail with "no new variables"; a leftover `newCC` would fail as unused).
- On a skip (section 3.1) the campaign-level `EventHandleReferenceCallHungup(ctx, cc.CampaignID)` also runs. It is idempotent: it acts only when `isStoppable` is true (execute stopped and no `dialing` or `progressing` campaigncall), and `campaignStopNow` returns early without a webhook when the campaign is already `stop` (`pkg/campaignhandler/status_stop.go`). So a duplicate event cannot publish a second campaign webhook, and a stop that depends on this event is not lost.
- Event consumer facts: `processEventRun` always returns nil after starting the goroutine, and the result of `processEvent` is only logged (`subscribehandler/main.go`), so nothing is redelivered whatever the handler does.
- The wrong comment above the function ("confbridge_leaved") is corrected to the real event name (`call_hangup`).

### 3.3 Failure rule

No new retry loops. A skip is a normal return. A failed `GetByReferenceID` returns as today (the campaigncall does not exist). A failed `EventHandleReferenceCallHungup` is logged and the campaign-level step still runs. Concurrency: the guard is read-then-act. The duplicate events are far apart in time (about 40 s for the VOIP-1562 trigger, the call duration for a late success), so a same-instant race is not a realistic case; two replicas handling the same event in the same instant could both pass, which is the existing behavior and is left as is.

### 3.4 Docs

`bin-campaign-manager/docs/domain.md`, section "Campaigncall Lifecycle" (line 102 at the base commit): add two sentences (a campaigncall that is already done is not completed again by a later call hangup, except a late success after a recorded failure, which corrects the result and the outdial target). No other doc change.

## 4. Tests (written with the implementation, not before)

`pkg/campaigncallhandler/eventhandle_test.go`, `Test_EventHandleReferenceCallHungup`. The existing eight reason cases stay unchanged (their campaigncall has no status, so the guard passes). The existing test discards the returned campaigncall and every case sets the same `Done` expectations, so the new cases go into a separate test function named `Test_EventHandleReferenceCallHungup_alreadyDone` with a per-case choice of the `Done` expectation set and an assertion on the returned value (`(cc, nil)` on skip). New cases, with strict gomock expectations so any unexpected `Done` call fails the test:

0. stored dialing and stored progressing, normal hangup (one case each), plus stored dialing with a failed hangup: `Done` runs with the mapped result. This pins the first table row, because the existing eight cases carry no status (a value production never has).
1. done + fail stored, failed hangup: no `Done` call (no database write, no webhook, no outdial request), returns the stored campaigncall.
2. done + fail stored, hangup `busy` (fail): same skip.
3. done + success stored, normal hangup: skip.
4. done + success stored, failed hangup: skip.
5. done + fail stored, normal hangup: `Done(Success)` runs (database write, webhook, outdial `done`).
6. done with an empty result stored: skip, once with a normal hangup and once with a failed hangup (so a loosened exception such as "stored is not success" is caught).
7. unknown reason: unchanged error and `nil` result.

`pkg/subscribehandler/callmanager_test.go`, `Test_processEventCMCallHangup` (today one case with fixed expectations; the new cases need per-case expectations, so they go into a separate test function named `Test_processEventCMCallHangup_errorAndSkip`):

8. `GetByReferenceID` fails: returns nil, no handler call.
9. `EventHandleReferenceCallHungup` returns an error: no panic, the campaign handler is still called with `cc.CampaignID`, returns nil.
10. skip result (returns the stored campaigncall): the campaign handler is called with `cc.CampaignID`.

The subscriber cases (8, 9 and 10) call `processEventCMCallHungup(ctx, m)` directly, because `processEvent` returns nothing (`subscribehandler/main.go:141`); the existing `Test_processEventCMCallHangup` goes through `processEvent`. Both run synchronously, so a panic from the nil dereference fails the test.

Mutation checks (each must fail at least one test), run in an isolated `git archive` copy:

- remove the whole guard; drop the `Status == done` term; drop the exception term (late success blocked); invert the exception (`Fail` to `Success`); change `ResultFail` in the exception to `ResultSuccess`; loosen the exception to `stored != success`; swap `||` for `&&` in the guard; change `Status == done` to `Status != done`; return `nil, nil` instead of `cc, nil` on skip (killed by the return assertion); in the subscriber: use `newCC.CampaignID` again (case 9), return early on an error (case 9), drop the campaign-level call (case 10), pass an empty id (cases 9 and 10).

Existing tests are not weakened (cases and expectations stay as they are).

## 5. Rollout and rollback

- No schema, configuration, flag, or data change. Rollback is a revert of the PR.
- In-flight events at deploy time are handled by the new code on arrival. A campaigncall done before the deploy with a late hangup after the deploy is skipped by the new rule, which is the intended behavior.
- Customer-visible change: fewer duplicate `campaigncall_updated` webhooks. One extra `campaigncall_updated` remains by design for a late success after a failure.

## 6. Risks and accepted limits

- A skipped duplicate cannot repair a partial `Done` (the outdial request failed inside the first `Done`, so the target stays `progressing`). This is a theoretical edge case: no reaper exists today. Before this change a duplicate hangup incidentally re-sent the outdial request and could repair such a target; this change removes that incidental repair (the same-result duplicate is now skipped). Accepted (analysis section 5, question 2). Revisit trigger: an outdial target stuck in `progressing` with a done campaigncall is seen in production.
- The late `idle` overwrite of a target redialed by a newer campaigncall is removed for the hangup path only. The `execute.go:270` order (hangup first) is not covered (non-goal).
- The late-success exception (`Done(Success)` after a recorded failure) writes the target `done` unconditionally (`UpdateStatus` has no guard). If the target was redialed by a newer campaigncall in the meantime, it is overwritten with `done` and the newer campaigncall's own `Done` decides the final state again. This is accepted: the number was answered, and the effect is smaller than the redial of an answered number that plain skip would cause. Code-derived, not reproduced.
- The rule adds one condition to a handler that already contains the result mapping. No new state, no new field.

## 7. Open points for review

1. Is the single exception (late success after fail) worth its one extra condition, versus plain skip-if-done (option (ii))? The design says yes because the failure mode is a redial of an answered number.
2. Resolved in revision 2: the subscriber uses `cc.CampaignID`, so the return value of the skip no longer matters to it.

## 8. PR plan

One PR for the whole change (branch and worktree `VOIP-1571-Skip-duplicate-campaigncall-done`; the design document ships in the same PR, following the one-PR-per-task rule).

Files expected in the diff (all under `bin-campaign-manager`):

- `pkg/campaigncallhandler/eventhandle.go` (the guard)
- `pkg/subscribehandler/callmanager.go` (use `cc.CampaignID`, log and continue, comment fix)
- `pkg/campaigncallhandler/eventhandle_test.go` and `pkg/subscribehandler/callmanager_test.go` (tests, section 4)
- `docs/domain.md` (two sentences in "Campaigncall Lifecycle")
- `docs/plans/2026-10-07-voip-1571-skip-duplicate-campaigncall-done-design.md` (this document)

No generated file changes: `CampaigncallHandler` keeps its method set, so no mock is regenerated.

Rules that apply, by source:

- Root `CLAUDE.md` (monorepo): the verification workflow before a commit, run in `bin-campaign-manager`: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` (the generate step must leave no diff); work only in a worktree; the PR title equals the branch name; no AI attribution; fetch `origin/main` and check for conflicts before the PR and before any merge; squash merge only; no merge without the CEO's instruction (also stated at user level).
- User-level `/home/pchero/CLAUDE.md` (the CEO's instructions): one PR per task; the PR body is a narrative paragraph followed by `bin-campaign-manager:` bullets, with no headers and no test plan section; no merge without the CEO's explicit instruction.
- `bin-campaign-manager/CLAUDE.md` adds nothing that conflicts (its rules are about the self-scheduling execute loop and the `stopping` state, which this change does not touch).
- CI: `scripts/check-test-conventions.sh` runs as a CircleCI job on the added lines (test names start with `Test_`, no testify, the gomock controller variable is `mc`). It is not part of the root `CLAUDE.md` workflow, so it is run in addition to it, from the repository root, and the planned tests follow it.
- Environment: commits use the configured repository author.

Branch, commit and PR title: `VOIP-1571-Skip-duplicate-campaigncall-done`. The PR text states the zero production impact (section 1) and the accepted limits of section 6.

## 9. Verification after deploy

Production has no running campaign (section 1), so there is nothing to observe in production after the deploy: the verification is the test evidence (the planned cases and the mutation results, run before the PR) and the CI result. After the deploy it is enough to check that campaign-manager starts and stays up. For a self-hosted or sandbox environment that does run campaigns, an optional check is to make the campaign call fail AFTER the call row exists (a channel creation failure such as `setVariablesCall` or `createChannelOutgoing`; a failure before the call row is created, for example a customer or balance check, produces no `call_hangup` and cannot show the guard) and, about 40 s later when the failed hangup arrives, to confirm exactly one `campaigncall_updated` webhook and one `Done` metric increment for it, and the log line "The campaigncall is already done. Skipping." for the late hangup. This optional check is not a gate.
