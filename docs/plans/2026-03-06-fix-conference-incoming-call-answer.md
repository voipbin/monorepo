# Fix Conference Incoming Call Answer

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an `answer` action before `conference_join` in the conference domain incoming call handler so the call transitions to `progressing` status and the conference health check doesn't terminate the call.

**Architecture:** When an incoming call arrives on the `conference.<domain>` SIP domain, `startIncomingDomainTypeConference()` creates a temporary flow with only a `conference_join` action. The call channel is never answered, so the call status stays `ringing`. The conference-manager health check expects `progressing` and terminates the conferencecall after 3 retries, hanging up the call. The fix prepends an `answer` action to the temp flow.

**Tech Stack:** Go, gomock

---

## Background

### Root Cause (call-id: 5bdf1c3c-e4e4-423d-af66-9f78be081fd0)

1. Incoming SIP/WSS call to `conference.voipbin.net` handled by `startIncomingDomainTypeConference()`
2. Temp flow created with only `conference_join` action (no `answer`)
3. Call joined the confbridge, but the call's own SIP channel was never answered (state: Ring, tm_answer: null)
4. Call status remained `ringing` instead of transitioning to `progressing`
5. Conference-manager health check (`health.go:72`) checks `c.Status != cmcall.StatusProgressing` — failed every 5s
6. After 3 retries, health check terminated the conferencecall, kicking the call and triggering hangup

### Why only this handler

The fix is scoped to `startIncomingDomainTypeConference()` only. The registrar variant (`startIncomingDomainTypeRegistrarDestinationTypeConference()`) is a different call path and not confirmed to have this issue. The `ServiceStart()` in conference-manager must NOT be changed because it would affect 1:1 connect calls where we must not answer before the remote endpoint answers.

---

### Task 1: Add answer action to startIncomingDomainTypeConference

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/start.go:477-485`
- Modify: `bin-call-manager/pkg/callhandler/start_test.go:169-176`

**Step 1: Update the temp flow actions in `startIncomingDomainTypeConference()`**

In `start.go`, change the actions slice at line 477-485 from:

```go
// create temp flow for conference join
actions := []fmaction.Action{
    {
        Type: fmaction.TypeConferenceJoin,
        Option: fmaction.ConvertOption(fmaction.OptionConferenceJoin{
            ConferenceID: cf.ID,
        }),
    },
}
```

To:

```go
// create temp flow for conference join.
// the answer action is required to transition the call status from ringing to progressing
// before joining the conference. without it, the conference-manager health check
// will terminate the conferencecall because it expects the call to be in progressing status.
actions := []fmaction.Action{
    {
        Type: fmaction.TypeAnswer,
    },
    {
        Type: fmaction.TypeConferenceJoin,
        Option: fmaction.ConvertOption(fmaction.OptionConferenceJoin{
            ConferenceID: cf.ID,
        }),
    },
}
```

**Step 2: Update the test expectation in `start_test.go`**

Change `expectActions` at line 169-176 from:

```go
expectActions: []fmaction.Action{
    {
        Type: fmaction.TypeConferenceJoin,
        Option: fmaction.ConvertOption(fmaction.OptionConferenceJoin{
            ConferenceID: uuid.FromStringOrNil("bad943d8-9b59-11ea-b409-4ba263721f17"),
        }),
    },
},
```

To:

```go
expectActions: []fmaction.Action{
    {
        Type: fmaction.TypeAnswer,
    },
    {
        Type: fmaction.TypeConferenceJoin,
        Option: fmaction.ConvertOption(fmaction.OptionConferenceJoin{
            ConferenceID: uuid.FromStringOrNil("bad943d8-9b59-11ea-b409-4ba263721f17"),
        }),
    },
},
```

**Step 3: Run verification**

```bash
cd bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-call-manager/pkg/callhandler/start.go bin-call-manager/pkg/callhandler/start_test.go
git commit -m "NOJIRA-Fix-conference-incoming-call-answer

- bin-call-manager: Add answer action before conference_join in startIncomingDomainTypeConference
- bin-call-manager: Update test expectation for conference incoming call flow"
```
