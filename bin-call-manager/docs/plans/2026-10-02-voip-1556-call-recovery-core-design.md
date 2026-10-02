# VOIP-1556: call-manager call recovery core fixes

Status: APPROVED, revision 10 (rounds 8 and 9 both APPROVED by both reviewers; revision 10 applies round 9 MINOR/NIT only). Ticket: VOIP-1556. Branch: `VOIP-1556-Fix-call-manager-recovery`.

This is the call-manager half of VOIP-1556. The Asterisk half (F14 tag re-register, Record-Route on the request, route set
freeze) shipped in monorepo-voip PR #144 (main 7d27526, deployed 2026-10-01 20:26Z, verified). Recovery stays disabled in
production (`RECOVERY_ENABLED` unset); enabling it is a separate CEO decision and is not part of this PR.

## 1. Issue validity (re-checked 2026-10-02 on monorepo main 02f3229de)

The VOIP-1556 issue analysis (agent note `VOIP-1556-analysis.md`, two consecutive APPROVED rounds) was made on main
4a14812da. Since then no commit touched the recovery path. The findings below were re-read in the current code:

| ID | Defect | Code (current main) |
|---|---|---|
| F3 | Selection includes ended channels: `ChannelGetsForRecovery` filters `asterisk_id`, `type`, `tm_create` only | `pkg/dbhandler/channel.go:456-475` |
| F4 | Channels with an empty SIP Call-ID are attempted; Homer lookup fails with `call ID cannot be empty` | `recovery.go:40-56`, `recoveryhandler_homer.go:66-68` |
| (answered) | Calls that were never answered are attempted; there is no established dialog to re-INVITE | same selection |
| F5b | Every Homer row is parsed with `sip.ParseMsg`; one unparseable or foreign row fails the whole recovery | `recoveryhandler_homer.go:129-137` |
| F13 | Call-ID, From/To URI and tag, CSeq are taken from the last message in Homer order, whoever sent it; Homer order is not causal | `recoveryhandler.go:168-183` |
| F13b | `determineRole` treats the first INVITE in list order without `VBOUT-SDP_Transport` as UAS; `findFirstSuccessfulResponse` matches the first INVITE's CSeq, which fails after a 401/407 challenge | `recoveryhandler.go:117-138, 188-204, 248-256` |
| F11 | ROUTES/RECORD-ROUTES are dropped unless they contain more than one entry (`len(split(",")) > 1`) | `recoveryhandler.go:155-165` |
| NPE | `From.Param.Get("tag").Value` dereferences nil when a tag is missing | `recoveryhandler.go:177, 181` |
| F7 | The recovery channel gets `TypeNone` (`ContextCallRecovery` is missing from the type map); its destroy is logged as unsupported and never reaches call handling | `channelhandler/arievent.go:56-75`, `arieventhandler/ari_channel.go:63-76` |
| F8 | On StasisStart the recovery channel is dialled, bridged, made the call's channel and the current action re-run, all before the remote answered | `start.go:339-376` |
| F9 | `CallSetChannelIDAndBridgeID` is an unconditional update; it can overwrite a call that was hung up meanwhile | `dbhandler/call.go:699-704` |
| F9b | `Hangup` reads the call by channel id, then (after a bridge destroy that waits for the ARI request to the dead Asterisk, duration not measured) records the hangup unconditionally (`CallSetHangup`), stops the activeflow and hangs up chained calls, even if the call moved to another channel in between | `hangup.go:26-100`, `db.go:433-445`, `dbhandler/call.go:285-294` |
| F9c | `hangingUpWithCause` reads the call, writes `terminating` and hangs up `c.ChannelID` as read; if the call moved to another channel in between, the old (dead) channel is hung up and the call is left `terminating` with a live channel that nothing hangs up any more | `hangup.go:123-177` |
| Dup | Nothing prevents two recovery legs for the same call (repeated container_died, manual `/v1/recovery`); the loser's BYE carries the same dialog identifiers and ends the recovered call | `recovery.go` (no guard) |

Not in scope (CEO decision 2026-10-02, "켜도 안전하게 동작하는 핵심"): restoring side state (recording, snoop, external
media, conference membership, chained/connect legs, groupcall), a concurrency limit N, re-recovery of an already recovered
call, the manual `/v1/recovery` public semantics and docs (F10), Grafana. Section 8 lists them with triggers. Calls whose
side state this PR cannot restore are skipped (section 4.1), not recovered half-way.

Recoverable class, stated explicitly: this PR recovers answered single-leg Flow calls only (IVR, play/TTS, recording,
digits, webhook-only flows, outbound campaign play). Not recovered: connect/forwarding (master and chained legs), queue
and conference calls, AI talk and streaming transcription (external media), groupcall calls. What share of production
calls falls in the recoverable class is an accepted unknown here; a read-only count of progressing calls with all skip
fields empty is an input to the go-live decision, not to this PR.

## 2. Goals

1. Only calls that can be recovered completely are attempted: answered, not ended, with a SIP Call-ID, still owned by the
   channel, without side state that recovery cannot restore, and not already being recovered.
2. The dialog values sent to Asterisk are Asterisk's own side of the dialog, derived from message content, independent of
   the order Homer returns rows in; ambiguous captures fail closed.
3. Homer rows that are not parseable SIP for this Call-ID do not abort recovery.
4. The recovery leg takes over the call only after the remote has answered the recovery INVITE, and only if the call is
   still the same live call. Otherwise the recovery leg is torn down and nothing else changes.
5. A progressing call's hangup is recorded only by the destroy of the channel that owns the call at that moment.
6. The recovery channel has a call type, so its state changes and its destroy go through call handling.

## 3. Smallest change considered (and why it is not enough)

Fixing only F13 and F5b makes the recovery INVITE correct, but leaves: ended calls being re-dialled (F3), the leg being
bridged and the Flow action re-run before answer (F8), a hung-up call being overwritten (F9) or a recovered call being
marked hung up by the old channel's late destroy (F9b), duplicate legs ending the recovered call (Dup), and the recovered
call never being hung up through call handling when its channel ends (F7). With recovery enabled each of these produces a
customer-visible effect. So the minimal safe set is goals 1-6; each fix below is a condition, a key or a small method.

## 4. Design

### 4.1 Selection and preconditions (F3, F4, answered, side state, Dup)

`dbhandler.ChannelGetsForRecovery` gets three more conditions (NULL convention: `ChannelCreate` writes nil for
`tm_end`/`tm_answer`, `dbhandler/channel.go:93-97`):

```go
Where(squirrel.Eq{string(channel.FieldTMEnd): nil}).
Where(squirrel.NotEq{string(channel.FieldTMAnswer): nil}).
Where(squirrel.NotEq{string(channel.FieldSIPCallID): ""}).   // also excludes NULL
```

- `tm_end` is set by `ChannelEndAndDelete` (`dbhandler/channel.go:283-292`, together with `tm_delete`) when a
  ChannelDestroyed is processed, including the health check's fake destroy; a channel on a dead Asterisk has none yet.
- `tm_answer` is set when the channel goes Up (`ChannelSetStateAnswer`, `channel.go:211-219`).
- Existing conditions (asterisk id, type call, created within 24 h), order and limit stay.

`recoveryRun`, after loading the call by channel id (`GetByChannelID`, so the call is owned by this channel at read time),
skips with an info log when:
- `c.Status != call.StatusProgressing`;
- the call has side state or peers this PR does not restore: `c.ConfbridgeID != uuid.Nil`, `len(c.ChainedCallIDs) > 0`,
  `c.MasterCallID != uuid.Nil`, `c.GroupcallID != uuid.Nil`, or `len(c.ExternalMediaIDs) > 0`. These calls end as today
  (health check hangup). Rationale: a call in a conference, or connected to another leg (master side: `ChainedCallIDs`;
  peer side: `MasterCallID`; groupcall legs: `GroupcallID`), loses or is ended with its peer when the call Asterisk dies,
  and re-running `confbridge_join` against a terminating conference leaves a silent progressing call; an AI/external media
  call loses its media session. An active
  recording is not restored (its channel died with the Asterisk), but the call itself is still recovered; documented
  limitation (section 8);
- the direction is neither incoming nor outgoing;
- the recovery claim is not acquired (below).

Recovery claim (Dup): new cache method `CallRecoveryClaim(ctx, callID uuid.UUID, ttl time.Duration) (bool, error)`, a
go-redis `SetNX` on key `call:recovery:<call id>`, exposed to callhandler through a dbhandler pass-through of the same
name (the existing pattern for cache-only data, `dbhandler/call_application.go:11-16`; callhandler has no cache handle).
The claim is taken after the preconditions above and before the Homer query. TTL 180 s: Homer HTTP timeout 30 s
(`recoveryhandler.go:73`) plus dial timeout 60 s (`pkg/callhandler/main.go:179`) plus margin. Not acquired: skip (info).
Redis error: skip (fail closed, warning). The key is not deleted; it expires. The claim covers the window before the old
channel's fake destroy (30-40 s); after it, the old channel has `tm_end` and is not selected any more. A switched call is
owned by the recovery channel, whose `sip_call_id` is empty, so it is not selected again either. Side effect, intended: a
claim taken for a call whose recovery then fails (for example no Homer data) also makes a manual `/v1/recovery` skip that
call for 180 s.

### 4.2 Homer rows (F5b)

In `getSIPMessages`, for each returned row:
- skip it (debug) if `row.CallID != callID`;
- parse it; on a parse error skip it (warning with the row id), do not fail;
- skip it if the parsed message's Call-ID differs from `callID`.
If no message remains, the existing `no SIP messages provided` error is returned.

### 4.3 Dialog reconstruction (F13, F13b, F11, NPE)

`GetRecoveryDetail(ctx, callID string, role asteriskRole)`: the role comes from call-manager's own record:
`call.DirectionOutgoing` means Asterisk sent the initial INVITE (UAC), `call.DirectionIncoming` means it received it
(UAS). `call.Direction` is set only by the outgoing call path (`outgoing_call.go:317`, also used by groupcall and connect
legs) and the incoming path (`start.go:648`); `ContextCallService` re-enters Stasis on an existing channel and creates no
call. Any other direction is an error.

From the filtered messages (any order; retransmissions and capture-hop copies allowed):

1. Dialog-creating 2xx: responses with status 200-299, `CSeqMethod == INVITE` and a To tag, whose From tag and CSeq match
   a To-tagless INVITE in the capture. Take the lowest CSeq among them (later INVITE 2xx answer re-INVITEs, which carry a
   To tag). If those lowest-CSeq 2xx carry more than one distinct To tag (forked 2xx), error.
2. Initial INVITE: the To-tagless INVITE with that CSeq and that From tag. This selects the INVITE that created the
   dialog, also after a 401/407 challenge (the challenged INVITE has a lower CSeq and no 2xx).
3. Consistency: if several copies of the initial INVITE or of the dialog-creating 2xx exist, the fields the role actually
   uses from that message (table below: UAC uses From/To of the INVITE and To tag, Contact, Record-Route of the 2xx; UAS
   uses From, To, Contact, Record-Route of the INVITE and the To tag of the 2xx) must be equal across copies; otherwise
   error. With `HOMER_WHITELIST` excluding the
   Kamailio outer hop (production), copies are byte-identical (analysis section 2); without it, outer-hop copies differ and
   recovery fails closed instead of depending on order.
4. Local and remote side (matches how the shipped Asterisk patch applies the variables: FROM_* to `dlg->local`, TO_* to
   `dlg->remote`, CSEQ to `local.cseq`, ROUTES as the frozen route set, RECORD-ROUTES as Record-Route headers):

| | UAC (outgoing) | UAS (incoming) |
|---|---|---|
| local tag (FROM_TAG) | initial INVITE From tag | 2xx To tag |
| local URI, display (FROM_URI, FROM_DISPLAY) | initial INVITE From | initial INVITE To |
| remote tag (TO_TAG) | 2xx To tag | initial INVITE From tag |
| remote URI, display (TO_URI, TO_DISPLAY) | initial INVITE To | initial INVITE From |
| request URI | 2xx Contact (fallback: INVITE To URI) | initial INVITE Contact (fallback: INVITE From URI) |
| ROUTES | 2xx Record-Route, reversed | initial INVITE Record-Route, in order |
| RECORD-ROUTES | 2xx Record-Route | initial INVITE Record-Route |

5. CSeq: the highest CSeq among all messages whose From tag equals the local tag (exactly the transactions Asterisk
   started: its requests carry its tag in From and the responses echo it), plus 100. RFC 3261 12.2.1.1 requires CSeq to
   increase, not to be contiguous; the margin covers Asterisk requests missing from the capture (HEP loss, ingest delay,
   session refresh just before the crash), which would otherwise draw a 500 (12.2.2); pjsip only checks `> 0`
   (patch) and increments. For UAS with no request from Asterisk the value is 0 and the CSEQ variable is left out of the
   channel variables (today `recovery.go:97` always sends `strconv.Itoa`, and "0" makes the patch log a warning): pjsip then uses a random initial local CSeq
   (`sip_dialog.c:277-278`), which the remote accepts because it has seen no request in that direction.
6. ROUTES and RECORD-ROUTES are kept whenever non-empty (F11: a single Record-Route is a valid route set).
7. Tag reads go through a helper that returns "" when the header or the tag parameter is missing. Validation fails
   closed: Call-ID, both tags, both URIs and the request URI must be non-empty.

Known limitation, accepted: a remote target refresh (later re-INVITE or UPDATE changing the remote Contact) is not
followed; Kamailio routes by the route set and the remote's address rarely changes mid-call.

`determineRole` and `findFirstSuccessfulResponse` are replaced by the steps above.

### 4.4 Recovery leg type (F7)

- `channelhandler.getChannelType` maps `channel.ContextCallRecovery` to `channel.TypeCall`. The recovery channel's
  StateChange already reaches `callhandler.ARIChannelStateChange` (`ari_channel.go:180-186`, not type-gated); its
  Destroyed now reaches `callhandler.ARIChannelDestroyed` → `Hangup` (`ari_channel.go:63-69`, TypeCall only).
- Not-found handling in `Hangup` is NOT changed. Today `CallGetByChannelID` not found returns an error and the consumer
  redelivers the event; ordinary incoming calls rely on that (a caller hanging up between StasisStart and `h.Create` in
  `startCallTypeFlow` gets its hangup recorded on the redelivery). For a recovery leg that never took over, and for the
  old channel's late destroy after a switch, the cost is the existing error log plus up to three harmless redeliveries.
- The recovery channel has no `direction` Stasis arg, so `UpdateSIPInfoByChannelVariable` does not run for it and its
  `sip_call_id` stays empty: a recovered call is not selected for recovery again (non-goal, section 8).

### 4.5 Answer-gated, conditional switch (F8, F9)

- `recoveryRun` adds `recovery_channel_id=<old channel id>` to the recovery channel's Stasis args, next to `call_id`
  (new constant `StasisDataTypeRecoveryChannelID` in `models/channel/main.go`, beside the other `StasisDataType` keys;
  `parseStasisData` already stores every key=value pair).
- `startContextCallRecovery` (StasisStart) only dials (`Dial`, as today). No bridge, no call update, no action. If the
  dial fails, it hangs up the recovery channel by its own id and returns nil (an undialled channel in Stasis is owned by
  no call and would otherwise stay; returning an error would only make the consumer redeliver and re-dial).
- Concurrent duplicate StasisStart for the recovery channel (duplicate delivery): the second `Dial` fails and, by the
  rule above, hangs up the recovery leg; that recovery fails and the call is cleaned up by the health check as today.
  This includes a StasisStart redelivered after the switch (call-manager restarted before the ack): the second `Dial`
  fails and hangs up the leg that now owns the call, ending the call. Accepted residual (same outcome as today's
  StasisStart error path, `ari_stasis.go:52-55`).
- Error policy for recovery events: call-manager acks events after processing and redelivers on error (5 s, 30 s, 120 s,
  `bin-common-handler/pkg/rabbitmqhandler/consume.go:23-28, 133-190`). Every recovery handler below cleans up itself and
  returns nil for handled outcomes, so redelivery never repeats a partial switch.
- `callhandler.ARIChannelStateChange`: when the channel's Stasis context is `ContextCallRecovery`, it handles only Up
  (calls `recoverySwitch`) and returns for every state, before the existing status update and `answerCallBridgePeers`
  (the call is already progressing; Ringing on the recovery leg must not touch the call). Other contexts are unchanged.
- `recoverySwitch(ctx, cn)`. The call id and the old channel id come from the Stasis data; nothing is read before the
  commit.
  1. Commit first, before touching Asterisk: generate the bridge id (`UUIDCreate`), then new dbhandler method
     `CallSetChannelIDAndBridgeIDIfOwned(ctx, id uuid.UUID, oldChannelID, newChannelID, bridgeID string) (bool, error)`:
     `UPDATE call_calls SET channel_id=?, bridge_id=?, tm_update=? WHERE id=? AND channel_id=? AND status=?` with
     `call.StatusProgressing` bound; returns whether a row changed (channel_id always changes, so MySQL's affected rows
     is 1 on success; the driver default does not set CLIENT_FOUND_ROWS); on success the cache is refreshed as in
     `CallUpdate`. Only the delivery whose update changes the row creates a bridge, so a concurrent or redelivered
     duplicate of the Up event never creates or joins one.
  2. Not changed, or the update returned an error (it may or may not have applied): read the call from the DB
     (`CallGetFromDB(call_id)`).
     - Not found (no such call, e.g. a missing or invalid `call_id` in the Stasis data): treated as refused below.
     - Other read error: return the error (no side effect; the event is redelivered and step 1 decides again). If the DB
       stays unavailable through all redeliveries the event is dropped and the Up recovery leg stays until the remote
       hangs up; accepted residual (a DB outage during a recovery).
     - `channel_id == cn.ID`: this channel already owns the call (a duplicate or redelivered Up whose twin committed, or
       an update that applied before its error). Return nil, no action.
     - Otherwise (the call was hung up, is hanging up, or is owned by another channel): hang up the recovery channel
       itself with `channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNormalClearing)` (never
       `callHandler.HangingUp`, which acts on the call's current channel). Info log, return nil. The BYE this sends is for
       the original dialog; this is correct because the claim (4.1) guarantees no other recovery leg exists for the
       call, so the call is ended or ending and the remote's dialog should end too.
  3. Changed: create the call bridge with the pre-generated id and join the recovery channel (`addCallBridge` takes the
     bridge id as a parameter instead of generating it; its other callers, `start.go:296, 588`, pass a fresh
     `UUIDCreate`, behaviour unchanged). A bridge or join failure here is a post-commit failure: hang up the recovery
     channel by its own id (`channelHandler.HangingUp(ctx, cn.ID, ...)`), return nil; the channel's destroy then records
     the call's hangup once through `Hangup` (the call is owned by `cn.ID`). If the destroy event is lost entirely, the
     call health check is the backstop.
  4. Arm the call duration limit on the new channel (`channelHandler.HangingUpWithDelay(ctx, cn.ID,
     ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration)`, as `start.go:238, 288` do for the original
     channel; the limit restarts, documented in section 8). A failure is logged (warning) and the switch continues.
  5. Re-read the call from the DB (`CallGetFromDB`, not the cache-first `h.Get`, because ownership is decided here). If
     it is no longer `progressing` or no longer owned by `cn.ID` (hung up meanwhile),
     return nil. Otherwise `actionExecute(ctx, c)`. A failure of the re-read or of `actionExecute` hangs up the call by
     the call id from the Stasis data (`callHandler.HangingUp(ctx, callID, call.HangupReasonNormal)`; `c` is nil after a
     failed re-read) and returns nil, so no silent progressing call
     remains.
- Accepted residual: if call-manager dies after the commit and before steps 3-5 finish, the redelivered Up finds the
  call owned by `cn.ID` (step 2) and does nothing; the call stays without bridge and action until the remote hangs up
  (a second failure during a recovery; rare).

### 4.6 Hangup follows the owning channel (F9b, F9c)

Writers of `call_calls.channel_id`: `CallCreate`, `CallSetForRouteFailover` (`dbhandler/call.go:522-528`, route failover;
taken when the status read at `hangup.go:26` is `dialing`/`ringing`, `isRetryable` `hangup.go:228`; the write itself has
no status condition) and the recovery switch (4.5, conditional on `progressing`). On automatic code paths call status
never returns to `progressing` once it left it (`IsUpdatableStatus`, `models/call/call.go:426-462`); the operator CLI
`call-control call update-status` bypasses this and is out of scope. A pre-existing race (an Up and a destroy of the same
dialling channel processed concurrently) could let failover rewrite a `progressing` call's channel; with this design
that case takes the skip path below, which is harmless.

Destroy side (`Hangup`), the only change: if the call read at `hangup.go:26` is `progressing`, the hangup write gets the
condition `AND channel_id = cn.ID` (new dbhandler method `CallSetHangupIfChannel(ctx, id uuid.UUID, channelID string,
reason, hangupBy) (bool, error)`; on success the cache is refreshed as in `CallUpdate`, so `UpdateHangupInfo`'s
following `CallGet` publishes the hangup state; `UpdateHangupInfo` gets the channel id, empty meaning unconditional, and
reports whether the row changed). If no row changed, the call was switched to a recovery channel after the read: info
log ("call moved to another channel before hangup"), `Hangup` returns the current call (non-nil, read with
`CallGetFromDB`; if that read fails, the error is returned and the redelivered destroy ends as not-found, harmless)
without follow-ups (webhook, metrics, RTP debug stop, activeflow stop, groupcall, chained). For
every other status the write is unconditional, exactly as today: a `dialing`/`ringing` call can only have been moved by
route failover (whose failure paths `Hangup` already handles as today), and a `terminating`/`canceling`/`hangup` call can
no longer be switched. A duplicate destroy of the same channel writes one row as today (`tm_hangup`/`tm_update` change)
and runs the follow-ups as today.

`createFailoverChannel` and the failover branch are not changed. `CallSetHangup` stays for the unconditional case or is
folded into the new method with an empty channel id; `UpdateHangupInfo`'s callers are `hangup.go:57` and
`db_test.go:798`.

Request side (`hangingUpWithCause`, F9c): after writing `terminating`/`canceling` (`hangup.go:150`, which applies to the
call whatever its channel), it reads the call's `channel_id` from the DB (new dbhandler method `CallGetFromDB`, a thin
export of the existing `callGetFromDB`; the cache refresh after a write is a separate read-then-set and can interleave
with another writer's) and hangs up that channel instead of the `c.ChannelID` read before the write. If that DB read
fails, it falls back to `c.ChannelID` (today's behaviour) with a warning. If a switch committed in between, the live
recovery channel is hung up and its destroy records the hangup; the call does not get stuck in `terminating`.

Effect on all calls: the destroy-side result differs from today only for a `progressing` call whose channel changed
between the read and the write (the recovery switch, or the pre-existing failover race above). The request-side result
differs only if the channel changed between the read at `hangup.go:130` and the DB read after the status write: by the
recovery switch, or by a route failover that committed a new channel id in that window. In the failover case the new
behaviour hangs up the new failover channel (today it is left dialling under a `canceling` call). Return contract kept:
when the channel read from the DB differs from `c.ChannelID` and hanging it up fails (its row does not appear within
the 3 s lookup wait of `channelHandler.Get`, because failover channel creation failed or ChannelCreated is late),
`hangingUpWithCause` logs a warning and returns `res, nil` as today. If creation failed, the old channel's destroy
records the hangup (unconditional write for a non-progressing call); if ChannelCreated is late, the failover channel is
left dialling under a `canceling` call, the same as today (pre-existing). The request side costs one extra SELECT by
primary key per hangup request. Pre-existing races not addressed here (for example a request-side
`terminating` write landing after a destroy recorded `hangup`) are unchanged.

Timeline with this design (crash at T0): sentinel event about 1 s, Homer query, dial and the remote's 200 normally within
a few seconds, so the switch happens well before the old channel's fake destroy at about T0 + 30-40 s (three failed channel health checks 10 s apart, `retryCount > 2`,
`channelhandler/health.go:22-27`,
`channelhandler/main.go:132-133`; longer if the ARI request to the dead Asterisk times out); that destroy then finds
no call (existing error log and redeliveries, 4.4). If recovery is slow and the destroy runs first: either it records the hangup before the switch (the
switch condition then fails, 4.5 step 2) or the switch lands between its read and its write (the hangup write then fails,
4.6, and the recovered call continues). If the remote never answers or rejects the re-INVITE, only the recovery leg ends
(its destroy finds no call: existing error log, redeliveries, then dropped) and the call is hung up by the old channel's
fake destroy from the channel health check, as today. A recovery leg destroyed by the remote around the commit: if the destroy is
processed after the commit, its `Hangup` finds the call and records the hangup. If before, `Hangup` finds no call and
returns an error (redelivered in 5 s); the commit then happens, step 3's join fails on the dead channel, and the
post-commit path hangs up only the recovery channel (4.5 step 3; it is already deleted, so this is a no-op); the
redelivered destroy finds the call owned by its channel and records the hangup once.
A request-side hangup (flow, API, call health check) racing the switch is covered by 4.6
request side.

### 4.7 Logging

One info line per decision: skipped (with reason), claimed, dialled, switched, switch refused, hangup skipped because the
call moved. Error lines only for unexpected failures. No metrics in this PR.

## 5. Affected files (bin-call-manager)

| File | Change |
|---|---|
| `pkg/dbhandler/channel.go`, `channel_test.go` | 4.1 selection conditions |
| `pkg/dbhandler/call.go`, `main.go`, `mock_main.go`, `call_test.go` | 4.5 `CallSetChannelIDAndBridgeIDIfOwned` (replaces `CallSetChannelIDAndBridgeID`, which has no other caller and is removed), 4.6 `CallSetHangupIfChannel`, `CallGetFromDB` |
| `pkg/cachehandler/handler.go`, `main.go`, `mock_main.go`, test; `pkg/dbhandler` pass-through | 4.1 `CallRecoveryClaim` |
| `pkg/callhandler/recovery.go`, `recovery_test.go` | 4.1 preconditions and claim, role from direction, 4.5 Stasis args |
| `pkg/callhandler/recoveryhandler.go`, `recoveryhandler_homer.go`, `mock_recoveryhandler.go`, new `recoveryhandler_test.go` | 4.2, 4.3 |
| `pkg/callhandler/start.go`, tests | 4.5 StasisStart dial only, `recoverySwitch`, `addCallBridge` bridge id parameter |
| `models/channel/main.go` | 4.5 `StasisDataTypeRecoveryChannelID` |
| `pkg/callhandler/arievent.go`, tests | 4.5 recovery context hook |
| `pkg/callhandler/hangup.go`, `db.go`, tests | 4.6 conditional hangup for progressing calls, request-side channel |
| `pkg/channelhandler/arievent.go`, tests | 4.4 type map |
| `docs/domain.md`, `docs/operations.md` | new behaviour; recovery still disabled; enabling is a gated crash test approved by the CEO, go-live after drain (VOIP-1560); `HOMER_WHITELIST` described as the Kamailio outer-hop exclusion that dialog reconstruction depends on (`operations.md:109` today says "IP whitelist for Homer recovery endpoint", which is wrong); update the "must not be enabled before VOIP-1556" / "Keep disabled until VOIP-1556" wording at `domain.md:97`, `operations.md:13` and `operations.md:111-113` |

## 6. Verification

- Unit tests:
  - dialog reconstruction (synthetic captures shaped like the sip-validator test calls, no customer data): outgoing and
    incoming; every permutation of row order for a small capture; duplicated hop copies; the remote's ACK, re-INVITE and
    BYE present; Asterisk re-INVITE present (CSeq); single Record-Route; 401/407 challenge then 2xx (initial INVITE is the
    second); forked 2xx with two To tags (error); differing hop copies (error); missing tag (error); no 2xx (error);
    foreign Call-ID and garbage rows skipped. Field-by-field assertions including CSeq.
  - selection query on the test database: ended, unanswered and empty-Call-ID channels excluded.
  - `recoveryRun`: skips for not progressing, confbridge, chained, master, groupcall, external media, bad direction,
    claim not acquired, claim error; Stasis args include `recovery_channel_id`.
  - `CallSetChannelIDAndBridgeIDIfOwned`: changes a progressing call owned by the old channel; not a hung-up call, not a
    call owned by another channel.
  - `CallSetHangupIfChannel` and `Hangup`: progressing call owned by `cn.ID` (written, follow-ups as today);
    progressing call moved to another channel (0 rows: no webhook, no metrics, no activeflow stop, no chained hangup);
    `dialing`/`ringing`/`terminating` calls (unconditional write as today, including route failover failure); duplicate
    destroy of the same channel (as today); not found (still an error, unchanged).
  - `hangingUpWithCause`: hangs up the channel read from the DB after the status write (switched call: the new channel);
    regression for terminating, canceling (outgoing dialling), the `TMEnd`-already-set immediate `Hangup` branch, and DB
    read failure falling back to the old value; route failover committed between the read and the status write (the new
    failover channel is hung up; if hanging it up fails, warning and `res, nil`).
  - `recoverySwitch`: switched path (update, then bridge with the pre-generated id, duration limit, action); refused
    path (0 rows, call not owned by `cn.ID`: recovery channel hung up by its own id, no bridge, no action); duplicate or
    redelivered Up (0 rows, call owned by `cn.ID`: nothing, nil); update error with the row applied (re-read shows
    `cn.ID`: nothing) and not applied (refused path); re-read error (error returned, no side effect); bridge/join error
    after commit (recovery channel hung up by its own id, nil); call hung up before step 5 (DB read shows not progressing: no action); action or re-read error after commit (call
    hung up, nil).
  - `startContextCallRecovery`: dial error hangs up the recovery channel and returns nil.
  - `recoveryRun`: CSEQ variable omitted when CSeq is 0.
  - `ARIChannelStateChange`: recovery context Ringing does nothing to the call; recovery Up calls the switch; a
    non-recovery Up behaves as before (regression).
  - type map entry.
- Mutation check of the key tests (roles, initial INVITE selection, CSeq, both conditional updates, refused switch, claim)
  in a `git archive` copy.
- Repo gates: `go build ./...`, `go test ./...`, `golangci-lint run`, monorepo `scripts/check-test-conventions.sh`.
- No production change and no production test in this PR. End-to-end proof is the gated enablement crash test (analysis
  section 4 step 4), a separate CEO decision after drain (VOIP-1560). A local call-manager integration run is not planned
  (it needs RabbitMQ, MySQL, Redis, Homer and two Asterisks wired together); the Asterisk side was proven end to end in the
  spike and by the image smoke test.

## 7. Rollout and risk

- Paths every call takes (each gives today's result except where noted):
  - `Hangup`: for a `progressing` call the hangup write requires `channel_id = cn.ID`; differs only when the recovery
    switch moved the call (no follow-ups). Other statuses: unconditional write as today.
  - `hangingUpWithCause`: one extra SELECT by primary key; hangs up the channel read from the DB after the status write,
    falling back to the value read before it on a read error; differs only when the channel changed in between
    (recovery switch, or route failover committing in that window: the new failover channel is hung up). Return
    contract unchanged (a failed hangup of a changed channel is logged and `res, nil` is returned).
  - `addCallBridge`: bridge id passed in by the caller (same behaviour).
  - `ARIChannelStateChange`: a context check before the existing code (only the recovery context branches).
- Recovery-only: the type map entry (only `call-recovery` channels), selection, claim, switch. Recovery is disabled
  (`RECOVERY_ENABLED` unset, guard first in `RecoveryStart`).
- Deploy: the normal call-manager pipeline. Rollback: revert the PR.
- Recoverable class: answered single-leg Flow calls only (section 1).
- Residual risks when recovery is later enabled: the current action is re-run (section 8); calls with conference, chained
  legs or external media are not recovered; an active recording is not restored; no concurrency limit; remote target
  refresh not followed.

## 8. Non-goals and follow-up triggers

| Item | Why not now | Trigger |
|---|---|---|
| Recovering calls with conference, chained/connect legs or external media (skipped in 4.1); restoring recording, snoop | CEO scope 2026-10-02 | enablement crash test or a real incident shows these calls matter |
| Concurrency limit N | no load signal; live calls per call Asterisk are few | after a crash, calls not switched before the fake destroy > 0 |
| Re-run vs resume semantics of the current action | changing Flow semantics is a product decision | enablement test shows duplicated customer-visible effects |
| Re-recovery of a recovered call | needs the recovery dialog's Call-ID and capture; rare (two crashes within one call) | a second crash during a recovered call is observed |
| `/v1/recovery` semantics, public docs (`architecture_rtc.rst`), Grafana (F10) | docs follow behaviour once enabled | go-live decision |
| Remote target refresh | rare | a recovery fails because the remote moved |
| Call duration limit restarts on the recovery channel; the old call bridge record is not cleaned up | computing the remaining time is extra code for a rare event; the bridge record is inert | a recovered call reaching the limit is observed |

## 9. Review history

Step and section numbers in each round refer to the revision reviewed in that round.

- Round 1 (A, B): both CHANGES_REQUESTED. MAJOR: initial INVITE/2xx selection fails after a 401/407 challenge and with
  forked 2xx or differing hop copies (A, B) → 4.3 steps 1-3; no duplicate-recovery guard, refused BYE can end a recovered
  call (A, B) → 4.1 claim, 4.5 step 4; "recovered call can be recovered again" false because `sip_call_id` stays empty (A)
  → 4.4, section 8; unconditional hangup write races the switch (B) → 4.6; failed action or conference/chained call leaves
  a silent live leg (B) → 4.1 skips, 4.5 step 5. MINOR: quiet not-found placement (A, B) → 4.4; which `HangingUp` (A, B)
  → 4.5 step 4; CSeq margin (A, B) → +100; duration limit (B) → 4.5 step 5; Ringing on the recovery leg (B) → 4.5 hook;
  docs and whitelist wording (B) → section 5; tests (B) → section 6; triggers (B) → section 8. NIT: owner check is implied
  by the lookup (A) → 4.1 wording; status constant bound (A); re-read call before action (A); `ChannelEndAndDelete` line
  (A); Q1 answered (direction set only by two paths) → 4.3.
- Round 2 (A, B): both CHANGES_REQUESTED. MAJOR: errors in switch steps 1-3 leave an Up recovery leg owned by no call (A,
  B) → 4.5 error handling; `hangingUpWithCause` racing the switch leaves the call `terminating` with a live channel (A) →
  4.6 request side (F9c); route failover rewrites `channel_id` before channel creation fails, so the conditional hangup
  would drop a real hangup (B) → 4.6 expected owner after failover. MINOR: cache access path (A) → dbhandler
  pass-through; TTL 90 s has no margin (A, B) → 180 s, claim position and rationale; duration limit restarts (A, B) →
  section 8; Up redelivery after commit (A) → 4.5 step 4 no-op; master/groupcall legs (A, B) → 4.1 skips; consistency
  fields per role (A) → 4.3 step 3; not-found log at `hangup.go:171` (A) → 4.4; safety net for a leg destroyed before
  commit (B) → 4.6 timeline. NIT: CSEQ 0 omitted (A); path of `main.go:179` (A); old bridge record (A) → section 8;
  `db_test.go:798` caller (A).
- Round 3 (A, B): both CHANGES_REQUESTED. MAJOR: redelivered Up after commit runs `addCallBridge` before the
  already-switched check and moves the live channel out of the call bridge (B) → 4.5 step 0 before any bridge; post-commit
  errors returned to the consumer are redelivered and then skipped, leaving a call without its action (A) → error policy,
  post-commit errors hang up the call and return nil; recoverable class not stated (B) → section 1 and 7, share as
  go-live input. MINOR: steps 1-3 return nil after cleanup (A, B); StasisStart dial error cleanup (B); cache-first reads in
  step 4, failover owner and request side (A, B) → step 0 uses `CallGetByChannelID` (DB), failover returns its own written
  id, request side `CallGetFromDB`; tests (B) → section 6. NIT: `sip_dialog.c` lines, goal 5 failover wording, claim
  blocking manual recovery (A) → 4.1; skip test bullets merged (B).
- Round 4 (A, B): both CHANGES_REQUESTED. MAJOR: a concurrent duplicate Up passes step 0 twice and the loser hangs up the
  just-switched recovery channel (A; B as MINOR) → commit before any bridge with a pre-generated bridge id, loser re-reads
  the call from the DB and does nothing when it already owns `cn.ID`; quiet not-found in `Hangup` removes the redelivery
  that records the hangup of an incoming call destroyed before `h.Create` (B) → quiet not-found dropped entirely, `Hangup`
  not-found unchanged. MINOR: step 0 DB error returns the error (A); `createFailoverChannel` failure return contract (A);
  duplicate StasisStart residual (A); fake destroy timing 20-40 s (B); tests (B). NIT: CLIENT_FOUND_ROWS (A); the 3 s
  bridge wait stated as an estimate (B).
- Round 5: A CHANGES_REQUESTED, B APPROVED. MAJOR (A; B as MINOR): a recovery leg destroyed around the commit gets its
  hangup recorded twice (post-commit `HangingUp` `TMEnd` branch, then the redelivered destroy) → conditional hangup also
  requires `status<>hangup`, recorded once; timeline rewritten. MINOR: step 4 re-read error (A, B) → return the error;
  step 1-2 errors could hang up a live switched channel (A) and steps 0/1 are redundant after commit-first (B NIT) →
  steps 0 and 1 removed, update errors go through the DB re-read; `CallGetFromDB` failure on the request side (B) →
  fall back to the old value; section 7 list (A, B) → rewritten; regression tests (B) → section 6. NIT: stale switch
  test text, duplicated failover contract (A); step 5 guard, post-commit crash residual, Stasis key constant (B).
- Round 6 (A, B): both CHANGES_REQUESTED. MAJOR (A, B): `status<>hangup` turned a destroy redelivered after a partial
  failure into a no-op, losing `call_hangup`, activeflow stop and chained/groupcall hangup for ordinary calls → on 0 rows
  re-read; same owner with status `hangup` runs the follow-ups without rewriting the record (at-least-once as today),
  other owner skips. MINOR: not-found in the switch re-read treated as refused (A); fake destroy 30-40 s (A); wording of
  "exactly once" and timeline (B). NIT: Stasis key name in tests, historical step numbers (A, B).
- Round 7: A APPROVED, B CHANGES_REQUESTED. MAJOR (B): `updateForRouteFailover` can commit a new channel id and then
  fail its `CallGet`, so the failover owner rule leaves an ordinary outbound call stuck; and the `status<>hangup` plus
  re-read branches are more than needed → destroy-side change reduced to "progressing call: `AND channel_id = cn.ID`, 0
  rows skip"; failover contract change, status condition and re-read removed (duplicate destroys behave as today).
  MINOR: post-commit bridge/join failure hangs up only the recovery channel so its destroy records the hangup once (A);
  metrics/RTP debug on the skip path (A, B: no follow-ups at all); pre-existing request-side race stated (A); docs
  wording at `operations.md:13, 111-113` (B). NIT: duplicate test bullets (A, B).
- Round 8: A APPROVED, B APPROVED (consecutive approvals: 1). MINOR/NIT applied without a reset: cache refresh after
  `CallSetHangupIfChannel` (A); step 5 ownership read from the DB (B); request-side effect also covers a route failover
  committing in the window, with the missing-row error stated (B); automatic-path wording for status monotonicity and the
  read-status condition of failover (A); return value on the skip path (A); StasisStart redelivered after the switch
  (B); fake destroy wording (B); removal of `CallSetChannelIDAndBridgeID` (B).
- Round 9: A APPROVED, B APPROVED (consecutive approvals: 2, design review closed). MINOR/NIT applied in revision 10
  without a reset: request-side return contract kept (warning and `res, nil` when hanging up a changed channel fails),
  wording of the 3 s wait and late ChannelCreated, extra SELECT stated in section 7, failure test (B); step 5 hangup by
  Stasis call id, skip-path read error, late ChannelCreated as pre-existing (A).
