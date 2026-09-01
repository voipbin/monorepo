# VOIP-1431: scopeRefCount runtime QueueBind/Unbind vs. AMQP channel safety — issue analysis

**Status:** Issue confirmed valid and current. Root cause identified and verified against
library source, an authoritative upstream issue, and production logs. Two independent trigger
windows identified (§4) — round-1 review of this document caught that an earlier draft
described only one and incorrectly claimed no race exists outside it; corrected. Round-3 review
caught that an earlier draft's §6/§7 named a specific fix mechanism that actually deadlocks;
corrected to leave mechanism selection to the design stage, with the deadlock noted as a
constraint on option 1. Round-4 review caught that §7's precedent for the dedicated-channel
option mischaracterized `ExchangeDeclare`/`QueueDeclare` as "ephemeral" when they actually open
a channel and keep it open indefinitely (that kept-open channel *is* `queue.channel`); corrected
to cite only the genuinely ephemeral precedents and describe Declare separately. No fix
implemented yet — this document is the analysis stage only, per the mandatory review workflow.

## 1. Issue validity re-check

The ticket's own background (a pre-existing warning comment, deleted by VOIP-1425, describing a
2026-07-14 production incident on bin-agent-manager) already established that concurrent
QueueBind against an actively-consuming channel is a real hazard *somewhere* in this codebase.
The open question was narrower: does it still apply to `scopeRefCount`'s specific usage
(runtime QueueBind/Unbind driven by websocket subscribe/unsubscribe), and if so, under what
conditions.

**Re-confirmed as still valid** — see §3 for the verified mechanism and §4 for the precise
trigger window in this codebase's current state (re-read directly from source, not assumed
from the ticket description).

## 2. Code re-inspected

- `bin-api-manager/pkg/websockhandler/scoperefcount.go` — `Acquire`/`Release`, both under
  `scopeRefCount.mu`.
- `bin-api-manager/pkg/websockhandler/main.go:33-53` — confirms `scopeRefCount` is **one
  instance shared across all websocket connections on a pod** (explicit comment on the struct
  field), not per-connection.
- `bin-common-handler/pkg/rabbitmqhandler/queue.go` — `QueueBind`/`QueueUnbind`: call
  `queue.channel.QueueBind`/`QueueUnbind` directly; `r.mu` (the rabbit struct's own mutex) is
  only taken *after* the network call succeeds, to update the Go-level tracking map
  (`r.queueBinds`) — it does **not** protect the network call itself.
- `bin-common-handler/pkg/rabbitmqhandler/consume.go` — `startConsumers`: calls
  `queue.channel.Qos(...)` then `queue.channel.Consume(...)` on the **same** `*amqp.Channel`
  object, once at initial registration and again on every reconnect via `redeclareAll()` →
  `reconsumerAll()` → `startConsumers()`.
- `bin-common-handler/pkg/rabbitmqhandler/main.go` — `redeclareAll()` (runs on the
  `reconnector()` goroutine after a broker reconnect): re-declares each queue (opens a **new**
  `*amqp.Channel`, replacing `r.queues[name].channel`), then loops over previously-tracked
  binds calling `r.QueueBind(...)` for each (restoring prior bindings), then calls
  `reconsumerAll()` which re-registers `Qos`/`Consume` on that same new channel.

## 3. Root cause, verified against library source (not assumed)

Pinned version: `github.com/rabbitmq/amqp091-go v1.10.0` (`bin-common-handler/go.mod`). Read
directly from `$(go env GOPATH)/pkg/mod/github.com/rabbitmq/amqp091-go@v1.10.0/channel.go`:

- `Channel.QueueBind`, `QueueUnbind`, `Consume`, `Qos`, `ExchangeDeclare` all route through the
  internal `ch.call(req, res)` helper, which does `ch.send(req)` then blocks reading **one**
  message off `ch.rpc` — a single, un-buffered, per-Channel reply channel shared by every
  in-flight synchronous RPC on that Channel.
- `ch.dispatch()`'s `default:` case pushes **any** non-content, non-delivery reply frame
  (`queueBindOk`, `queueUnbindOk`, `basicConsumeOk`, `basicQosOk`, etc.) onto that same
  `ch.rpc`, with no per-request correlation ID — `call()` matches purely by Go type via
  `reflect.TypeOf`.
- Only `Ack`, `Nack`, `Reject`, and `PublishWithDeferredConfirm` take `ch.m.Lock()` before
  calling `send()`. `QueueBind`/`QueueUnbind`/`Consume`/`Qos` take **no lock at all** — grepped
  every `ch.m.Lock()` call site in `channel.go` to confirm this is exhaustive, not an
  oversight in reading.
- `Connection.send()` **does** hold a write mutex (`c.sendM`) around the raw frame write, so
  concurrent calls cannot corrupt the wire byte stream itself — but that only serializes the
  *writes*; it does nothing to prevent two concurrent `call()` invocations from each reading
  the *other's* reply off the shared `ch.rpc`.

**Net effect:** two goroutines calling any two of {QueueBind, QueueUnbind, Consume, Qos,
ExchangeDeclare, QueueDeclare} concurrently on the *same* `*amqp.Channel` can cross-deliver
replies — one goroutine gets a reply of the wrong type (`ErrCommandInvalid`) while the other
hangs waiting for a reply that already went to the first, until connection-level error/timeout.

### External corroboration

- [rabbitmq/amqp091-go#242](https://github.com/rabbitmq/amqp091-go/issues/242) — "Add mutex
  guard to Channel methods", filed 2024-01-30, **still open, unresolved**. Reports the exact
  error `Exception (503) Reason: 'unexpected command received'` from concurrent
  `Channel.QueueDeclare` calls on one channel, and explicitly notes the same asymmetry found
  independently above: Ack/Nack/Reject/PublishWithDeferredConfirm are guarded,
  declare/bind/consume methods are not. This is a confirmed, currently-unfixed gap in the
  library itself — not speculation, and not unique to VoIPBin's usage.
- Predecessor library `streadway/amqp` has the same design and the same class of reports
  (issues #119, #327: "Channel is not thread safe" / "NOT thread safe for concurrent
  consume/publish").

## 4. Precise trigger windows in this codebase (not "any concurrent use is broken")

Because `scopeRefCount` is a single per-pod instance and `Acquire`/`Release` both hold
`scopeRefCount.mu` for their entire body including the QueueBind/Unbind call, **scopeRefCount's
own binds/unbinds cannot race each other** — multiple websocket clients subscribing/unsubscribing
concurrently are already serialized before reaching the AMQP channel.

There are **two independent** trigger windows where something *other than scopeRefCount itself*
touches the same channel concurrently with it. (Round-1 review of this document caught that an
earlier draft claimed only the first of these existed — corrected here.)

**(a) Broker reconnect.** `rabbitmqhandler`'s own reconnection sequence (`redeclareAll` →
bind-restoration loop → `reconsumerAll` → `startConsumers`'s `Qos`/`Consume`) runs on the
separate `reconnector()` goroutine, on the newly-replaced channel object, with no coordination
with scopeRefCount's mutex. A websocket client subscribe/unsubscribe landing inside the window
between the new channel being installed and `reconsumerAll` finishing races it.

**(b) Pod startup/restart — no reconnect required.** Re-read `bin-api-manager/cmd/api-manager/main.go:190-191`:
`run()` launches `go runSubscribe(...)` and `go runListenHTTP(...)` as two independent,
unsynchronized goroutines. Inside `runSubscribe` → `subscribehandler.Run()`
(`bin-api-manager/pkg/subscribehandler/main.go:78-107`): `QueueCreate` (which opens the
channel) runs synchronously, but the actual `Qos`+`Consume` registration
(`ConsumeMessage` → `startConsumers`) is dispatched into **yet another goroutine**
(`main.go:100-104`) and `Run()` returns immediately without waiting for it. Meanwhile
`runListenHTTP` stands up the HTTP server that serves the websocket upgrade endpoint with zero
dependency on that consumer-registration goroutine having completed. So on **every pod
startup or rolling restart** — not just reconnects — there is a window between the channel
being created and `Qos`/`Consume` being registered on it, during which a websocket client that
connects and subscribes fast enough triggers `scopeRefCount.Acquire` → `QueueBind` on that same
not-yet-fully-registered channel concurrently with the async `startConsumers` call. This is the
same hazard class as (a), with no reconnect involved at all.

Outside these two windows (steady-state operation with a stable connection, no pod restart in
progress), there is no race.

## 5. Production evidence check

Queried `bin-api-manager`'s live container (`voipbin-api-manager` on bm-nyc-01, up 2 days) via
Komodo's `GetContainerLog` (5000-line / ~1.3MB tail, covering roughly 2026-09-01 02:45–04:20
UTC):

- **No** `unexpected command received`, `503`, `ErrCommandInvalid`, or AMQP channel-closed
  errors in that window.
- **No** RabbitMQ reconnect events (`Connecting to rabbitmq` / `Reconnecting after connection
  closed`) in that window either — meaning the precondition for the race (§4) simply didn't
  occur in the sampled window.
- The websocket-subscription code path is actively exercised in production
  (`subscriptionRunPinger: Sent ping message to client` present repeatedly), so this isn't a
  dead/unused feature — it's a live feature that hasn't yet had a reconnect coincide with a
  subscribe/unsubscribe in the sampled log window.

This is consistent with the ticket's own note ("기능 저하 확인된 것은 아님") — **absence of
observed incidents does not disprove the race; it means the trigger conditions haven't recently
coincided in the ~2-hour sample.** This check only covered trigger window (a) (reconnect
events); it did **not** check window (b) (pod startup/restart — the container has been up 2
days, so its own boot-time window is well outside the sampled tail). A full incident-log sweep
across the 2-day uptime, and specifically around the most recent deploy/restart timestamp, would
be needed to check window (b) and to fully rule out past occurrences of window (a); neither was
performed here (see §7).

## 6. Proceed-or-not analysis

**Proceed.** This is a confirmed, currently-real defect (not a stale/already-fixed concern, not
a misunderstanding of the code) in a live, actively-used feature (websocket event
subscriptions), with:
- A verified mechanism (library source read directly, not inferred).
- Independent, authoritative external corroboration (open upstream GitHub issue against the
  exact pinned library version, matching the exact operation family).
- Two precise, identified trigger conditions (§4) — this is not "rewrite everything," it's "the
  channel-setup path (startup and reconnect alike) and the runtime-rebind path need to not
  touch the same channel unsynchronized." Both trigger windows resolve to the same underlying
  gap — `startConsumers`'s `Qos`/`Consume` registration (called both at initial startup and by
  `reconsumerAll` on reconnect) racing `QueueBind`/`QueueUnbind` — so a single fix addresses
  both, not two separate fixes.
- No evidence of active production impact right now, so this is not an emergency, but it is a
  legitimate latent correctness bug worth fixing before it manifests as a harder-to-diagnose
  incident (a repeat of the exact 2026-07-14 bin-agent-manager pattern this ticket's own
  background cites).

Priority stays Medium (matches Jira) — not escalating, since no active incident. Recommend
moving to design: the fix belongs in `rabbitmqhandler` (protect the shared per-queue channel's
`QueueBind`/`QueueUnbind` against concurrent execution with `startConsumers` — both its
initial-registration call site and its `reconsumerAll`-on-reconnect call site — on the same
rabbit instance), not in `scopeRefCount` — a rabbitmqhandler-level fix protects every current
and future caller of `QueueBind`/`QueueUnbind` against reconnect races, not just this one call
site. The specific mechanism (which mutex, or avoiding the shared channel entirely) is a design
decision, not resolved here — see §7's options; round-3 review of this document caught that an
earlier draft named a specific mechanism ("hold `r.mu` for the network-call duration") that is
not actually viable (see §7 option 1's note), which is exactly the kind of detail this
analysis-stage document should not have pre-committed to.

## 7. Open items for the design stage (not resolved here)

- **Option 1 — reuse `r.mu` as the serialization point: NOT VIABLE as literally stated.**
  An earlier draft of this document proposed "hold `r.mu` for the network-call duration of
  these ops." Round-3 review caught that this deadlocks immediately: `QueueBind`
  (`queue.go:166-167`) and `startConsumers` (`consume.go:33`) both begin by calling `queueGet`,
  which takes `r.mu.RLock()` (`queue.go:13-18`) — locking `r.mu.Lock()` at the top of either
  function and then calling `queueGet` is a write-lock-then-read-lock on the same
  `sync.RWMutex` from the same goroutine, which Go's `RWMutex` does not support re-entrantly.
  `r.mu` is currently a *data guard* for the `r.queues`/`r.queueBinds` maps, held only briefly;
  repurposing it as an *in-flight-network-call guard* held for a whole RPC round-trip is a
  different invariant with a different hold duration, and mixing the two is what causes the
  deadlock. If a dedicated mutex is used instead of `r.mu` itself, this option becomes viable,
  but it still serializes ALL queue/exchange RPCs across the whole rabbit instance, including on
  unrelated queues — a real contention cost, though narrower than it first appears:
  `publishExchange`, `RequestPublish`, and `publishRPCErrorResponse` (`publish.go`,
  `consume.go:307,353`) each open, use, and close their own throwaway channel per call and would
  not be blocked by this lock. `ExchangeDeclare` (`exchange.go:11-37`) and `QueueDeclare`
  (`queue.go:102-142`) are a separate case (see Option 2's note below) — they also would not be
  blocked, but not because they're throwaway; they simply never touch the *pre-existing*
  `queue.channel` at all. Only `QueueBind`, `QueueUnbind`, `QueueDelete`, `QueueQoS`, and
  `startConsumers`'s `Qos`/`Consume` share the long-lived `queue.channel` and would contend.
- **Option 2 — dedicated short-lived channel for `QueueBind`/`QueueUnbind`, matching part of
  this package's own convention.** `publishExchange`, `RequestPublish`, and
  `publishRPCErrorResponse` genuinely open-use-close an ephemeral `r.connection.Channel()` per
  call — a real, directly-applicable precedent for issuing `QueueBind`/`QueueUnbind` the same
  way. (`ExchangeDeclare`/`QueueDeclare` are a *weaker*, different data point, not the same
  pattern: verified they open a channel and never close it — they store it into
  `r.exchanges[name].channel`/`r.queues[name].channel` and keep it open indefinitely, only
  closing the *previous* one when a future re-declare replaces it. That stored, kept-open
  channel is `queue.channel` itself — the very channel this whole document is about — so these
  two functions are the origin of the long-lived shared channel, not an example of avoiding
  one.) `queue.bind`/`queue.unbind` are broker-side operations scoped to the queue, not to the
  issuing channel, so binding/unbinding on a fresh ephemeral channel (following the
  publish/RPC-reply pattern) is semantically identical to doing it on `queue.channel` — and
  removes the race by construction: no lock, no cross-queue contention, no lock-ordering rules
  for future maintainers to get wrong. Trade-offs to weigh at design time: one channel
  open/close per websocket subscribe/unsubscribe transition, and how this interacts with
  `QueueDeclare`'s channel-replacement on reconnect.
- Whichever option is chosen, it must cover all five current users of the shared
  `queue.channel` object — `QueueBind` (`queue.go:172`), `QueueUnbind` (`queue.go:201`),
  `QueueDelete` (`queue.go:28`), `QueueQoS` (`queue.go:150`), and `startConsumers`'s
  `Qos`+`Consume` (`consume.go:44,48`, covering both the initial-registration call site in
  `subscribehandler.Run()` and the `reconsumerAll`-on-reconnect call site) — not just
  `QueueBind`/`QueueUnbind` vs. `startConsumers` as this document's earlier drafts framed it;
  `QueueDelete` and `QueueQoS` route through the same unguarded `call()` path and belong to the
  same hazard class.
- Whether `subscribehandler.Run()`'s async `go func() { ConsumeMessage(...) }()` (window (b))
  should instead be synchronous/awaited before `Run()` returns and before `runListenHTTP`
  starts accepting connections, as a defense-in-depth measure independent of the
  `rabbitmqhandler`-level mutex fix — would eliminate window (b) entirely rather than just
  making it safe to race.
- A full 2-day log sweep (not just the ~2-hour tail sampled here) covering both trigger windows
  — specifically including the timestamp of the pod's most recent startup/restart (window (b))
  in addition to any reconnect events (window (a)) — to increase confidence no past incident
  was already caused by this and mis-attributed to something else.
