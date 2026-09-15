# VOIP-1525: recording_start auto-answer on ringing calls

Status: Draft → in review
Ticket: https://voipbin.atlassian.net/browse/VOIP-1525
Branch: VOIP-1525-Fix-recording-start-auto-answer-ringing

## Problem statement

Production call recording has produced zero new recordings since 2026-06-30, while
call traffic is healthy (2,938 calls in the last 7 days). Every `recording_start`
execution fails. Root cause confirmed from production Loki logs + MariaDB:

`recording_start` executes while the call is still in `ringing` status (before
answer/progressing), and `RecordingStart` rejects it:

```
bin-call-manager/pkg/callhandler/recording.go:36-40

	if c.RecordingID != uuid.Nil {
		return nil, fmt.Errorf("recording is already progressing. recording_id: %s", c.RecordingID)
	} else if c.Status != call.StatusProgressing {
		return nil, fmt.Errorf("the call is not progressing. status: %s", c.Status)
	}
```

Production evidence (call `7d8381a4-533e-416d-a0ae-b2030bc85efc`, 2026-09-14 17:03:25 UTC):

```
func: actionExecuteRecordingStart
message: "Could not start the recording. err: the call is not progressing. status: ringing"
severity: ERROR
```

The three failed calls in the last 7 days all run the flow `recording_start → talk →
hangup` with no `answer` action. The `talk` action auto-answers a non-progressing call
(media.go:32-38) and then succeeds; `recording_start` does not, so when it is the first
action it always fails on a ringing inbound call.

The `c.Status != StatusProgressing` guard has existed since the service's initial 2024
commit; this is not a recent regression. It is a UX/semantics gap: the platform silently
fails recording on a not-yet-answered call and moves to the next action, so the caller
believes recording happened but no file exists.

## Goals

1. When `recording_start` runs on a call that is not yet `progressing`, the platform
   answers the call immediately (no wait) and then starts recording, mirroring the
   established `Talk` auto-answer precedent.
2. Recording succeeds end-to-end for the canonical test flow `recording_start → talk →
   hangup` (no explicit `answer` action).
3. No behavior change for calls that are already `progressing` when `recording_start`
   runs (the common case where an `answer`/`talk` preceded it).

## Non-goals

- Confbridge recording (`confbridgeHandler.RecordingStart`, recording.go in
  confbridgehandler) is out of scope. A confbridge only records once it is progressing;
  its lifecycle differs and no production failure was observed there. Deferred.
- `RecordingStop` is out of scope — stopping requires an active recording, which by
  definition implies the call was progressing. No change.
- Changing flow-authoring guidance / docs to require an `answer` before
  `recording_start` — the whole point of this fix is that it should not be required.
- Backfilling / recovering the recordings lost between 2026-06-30 and the deploy. Those
  calls have ended; nothing to recover.

## Root cause detail: why "just call Answer" is not sufficient

`channelHandler.Answer` (channelhandler/etc.go) sends the ARI `AstChannelAnswer` command
to Asterisk and returns. It does NOT synchronously flip the call's status. The call's
`Status` transitions to `progressing` only later, asynchronously, when Asterisk emits the
`ChannelStateChange` (→ Up) ARI event, which `callHandler` handles via
`ARIChannelStateChange → updateStatusProgressing` (arievent.go:88-115, status.go:43-58).

Therefore the guard cannot be satisfied by "call Answer, re-`Get`, re-check status" — the
in-DB/cache `c.Status` will still read `ringing` immediately after the Answer command
returns. The `Talk` precedent does not re-check status either: it answers if not
progressing and proceeds directly (media.go:32-38). The fix must remove the hard status
rejection from the recording-start path the same way, replacing "reject if not
progressing" with "answer if not progressing, then proceed."

Asterisk-side safety: `Talk` calls `channelHandler.Answer` then immediately
`channelHandler.Play` on the same channel and this works in production (the failed calls'
`talk` action played TTS successfully right after auto-answering). Starting a recording is
the same class of ARI operation on a just-answered channel, so the immediate-no-wait
approach is consistent with existing, working behavior.

## Affected files

| File | Change |
|---|---|
| `bin-call-manager/pkg/callhandler/recording.go` | `RecordingStart`: replace the `else if c.Status != StatusProgressing { return err }` rejection with an auto-answer-if-not-progressing branch, then proceed. Add `github.com/pkg/errors` import (current guard uses only `fmt`; the new Answer-error path uses `errors.Wrap`). |
| `bin-call-manager/pkg/callhandler/recording_test.go` | Add/adjust tests: (a) ringing call → Answer called → recording starts; (b) already-progressing call → Answer NOT called → recording starts (unchanged path); (c) hung-up call → surfaced error, no recording. |

Note: no change to `action.go` `actionExecuteRecordingStart`, the RPC listenhandlers
(`v1_calls.go:745`), or the confbridge path. Placing the fix in the shared
`RecordingStart` method covers all three call entry points (flow action, RPC/API, AI tool)
uniformly — exactly how `Talk`'s auto-answer covers all of its callers.

## Exact change (before → after)

Current (recording.go:30-40):

```go
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return nil, err
	}

	if c.RecordingID != uuid.Nil {
		return nil, fmt.Errorf("recording is already progressing. recording_id: %s", c.RecordingID)
	} else if c.Status != call.StatusProgressing {
		return nil, fmt.Errorf("the call is not progressing. status: %s", c.Status)
	}
```

Proposed:

```go
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return nil, err
	}

	if c.RecordingID != uuid.Nil {
		return nil, fmt.Errorf("recording is already progressing. recording_id: %s", c.RecordingID)
	}

	// answer the call if it has not been answered yet, then start recording without
	// waiting for the status transition (mirrors the Talk auto-answer behavior).
	// A hung-up call is rejected by channelHandler.Answer (tm_delete set), so a
	// terminated call still cannot start a recording.
	if c.Status != call.StatusProgressing {
		if errAnswer := h.channelHandler.Answer(ctx, c.ChannelID); errAnswer != nil {
			return nil, errors.Wrap(errAnswer, "could not answer the call before recording")
		}
	}
```

Rationale for keeping the `RecordingID != uuid.Nil` check first: an already-recording call
must still be rejected regardless of status; only the status rejection is being relaxed.

Terminated-call safety: `channelHandler.Answer` first `Get`s the channel and returns
`"the channel has hungup already"` when `cn.TMDelete != nil` (channelhandler/etc.go). So a
`hangup`/`terminating` call fails cleanly at Answer instead of attempting to record a dead
channel — the relaxed guard does not open a "record a dead call" path.

## Behavior matrix

| Call status when recording_start runs | Before | After |
|---|---|---|
| `progressing` (answer/talk ran earlier) | records | records (unchanged, Answer not called) |
| `ringing` / `dialing` (not yet answered) | ERROR "not progressing", no recording | Answer sent, recording starts |
| already recording (`RecordingID != nil`) | ERROR "already progressing" | ERROR "already progressing" (unchanged) |
| `hangup` / `terminating` (channel gone) | ERROR "not progressing" | ERROR at Answer "channel has hungup already", no recording |

## Test strategy

`recording_test.go` `Test_RecordingStart` (table-driven, matching existing style):

1. **ringing → auto-answer → record**: call with `Status: StatusRinging`,
   `RecordingID: uuid.Nil`. Expect `channelHandler.Answer(ctx, ChannelID)` called,
   `recordingHandler.Start` called, `UpdateRecordingID` called, no error.
2. **progressing → no answer → record**: call with `Status: StatusProgressing`. Expect
   `channelHandler.Answer` NOT called (strict gomock, no EXPECT), `recordingHandler.Start`
   called. Locks the "unchanged for the common case" invariant.
3. **already recording → reject**: `RecordingID != uuid.Nil`. Expect error, no Answer, no
   Start.
4. **answer fails (hung-up channel) → reject**: `Status: StatusRinging`, Answer returns
   error. Expect error surfaced, `recordingHandler.Start` NOT called, `UpdateRecordingID`
   NOT called (explicit no-EXPECT assertions on both).

Also add: Open Question 1 confirmation — auto-answer as a side effect of a raw RPC/API
`recording_start` (v1_calls.go:745) on a ringing call is intended, not flow-only. This is
consistent with `Talk` exposing the same auto-answer on all its callers.

Verification workflow (in `bin-call-manager`): `go mod tidy && go mod vendor && go generate
./... && go test ./... && golangci-lint run -v --timeout 5m`.

## Rollout / risk

- Blast radius: one method in one service. RPC/flow/AI-tool recording-start paths all
  gain auto-answer; none lose functionality.
- Risk: a `recording_start` on a ringing OUTBOUND call would now answer it. Outbound
  recording is normally started after the callee answers (progressing), so this branch
  rarely triggers for outbound; if it does, answering our own outbound leg is a no-op at
  Asterisk when already up, and rejected if the channel is gone. Low risk; called out for
  reviewer scrutiny.
- No schema change, no migration, no API contract change, no webhook change.
- Post-deploy verification: re-run a live inbound test call with a
  `recording_start → talk → hangup` flow and confirm a `call_recordings` row is created
  with status progressing→ended and a file is produced (per prod-incident-diagnosis
  post-deploy live-traffic rung). Do not manufacture cost-incurring calls; use the
  existing internal test number path.

## Open questions

| # | Question | Recommendation | Owner |
|---|---|---|---|
| 1 | Place auto-answer in shared `RecordingStart` (covers RPC+flow+AI) vs only in `actionExecuteRecordingStart` (flow-action only)? | Shared method — mirrors `Talk`, keeps all entry points consistent. | reviewer/pchero |
| 2 | Should a warn/info log be emitted when auto-answer fires (observability of the implicit answer)? | Yes, one Debug/Info line "answered call before recording" — cheap, aids future diagnosis. | reviewer |
| 3 | Confbridge recording parity — apply same auto-answer to `confbridgeHandler.RecordingStart`? | Defer; no production failure observed, different lifecycle. Track separately if needed. | pchero |

## Iter-1 review response summary

Independent reviewer verdict: APPROVED. Root cause, change correctness (relaxes only the
status rejection; already-recording + terminated-call safety preserved), pkg/errors
semantics, and test-invariant locking all confirmed. Non-blocking advisories applied to
v2:
- Line-number alignment (30-40, guard at 36-40).
- `github.com/pkg/errors` import noted explicitly in the affected-files table.
- Test 4 now explicitly asserts `UpdateRecordingID` NOT called on Answer failure.
- Open Question 1 (auto-answer intended on raw RPC/API surface, not flow-only) confirmed
  in Test strategy section, consistent with Talk.
- Terminating-vs-channel-TMDelete race acknowledged as pre-existing (Talk shares it), not
  novel, not blocking.
