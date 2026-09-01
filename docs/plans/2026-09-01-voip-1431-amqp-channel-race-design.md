# VOIP-1431: Fix scopeRefCount vs. consumer-registration AMQP channel race — design

Builds on the approved issue analysis:
[2026-09-01-voip-1431-scoperefcount-amqp-safety-analysis.md](2026-09-01-voip-1431-scoperefcount-amqp-safety-analysis.md)
(6 review rounds, 2 consecutive approvals). That document establishes: the hazard is real,
current, and has two independent trigger windows (broker reconnect; pod startup/restart); the
root cause is `rabbitmqhandler.QueueBind`/`QueueUnbind` and `startConsumers`'s `Qos`/`Consume`
sharing one long-lived `*amqp.Channel` (`queue.channel`) with no synchronization, on top of an
unfixed gap in `amqp091-go` v1.10.0 itself (no lock around `call()`, shared un-correlated
`ch.rpc` reply channel). This document picks a mechanism and specifies the change.

**Revision note:** round 1 of this document's own review loop found the mechanism choice sound
but the implementation sketch, test plan, and scope section unimplementable as originally
written. Round 2 found the round-1 fix itself was unimplementable for a different reason (Go
has no return-type covariance, so naively widening `amqpConnection.Channel()`'s return type
breaks `*amqp.Connection`'s implicit satisfaction of that interface — a genuine compile error,
not caught until round 2) and that the proposed race fix only locked the *read* side while the
*write* site remained unlocked, which provides zero actual synchronization. Round 3 confirmed
both of those fixes (the `realConnection` adapter, the locked write) are now correct, but caught
that the "pre-existing unsynchronized readers of `r.connection`" count — introduced to explain
scope boundaries — had been wrong twice in a row (five, then six; actually nine), and that a
stale-test-comment pointer named the wrong test. Both corrected below, along with a factual
cleanup of the `realConnection` nil-handling rationale (the code was already correct; its stated
justification wasn't). Round 4 confirmed the corrected count and pointer, but found the "nine"
count wasn't quite exhaustive either — a 10th `r.connection` access exists at `connect()`'s own
`NotifyClose` call, genuinely race-free (same-goroutine, reads what it just wrote) but
unmentioned; added with its exclusion reasoning. Round 5 (a whole-document pass, not a
narrow recheck) found that §4's test plan had never been revisited after §2/§3 grew across
rounds 2-4: widening `amqpChannel` breaks three hand-written test mocks that were never told to
implement the new method, and none of the new production code from those rounds
(`realConnection`, `connectionGet()`, the locked write in `connect()`) had any test coverage or
even a `-race` mention, despite the locked write's entire justification being "an unlocked
version is exactly what `-race` would catch." Also flagged, as a documented trade-off rather than
a defect: the `amqpChannel` widening this ripple stems from isn't required by the race fix
itself — only by §4's mock-based structural test — and that trade-off wasn't previously stated.
All addressed in §2 and §4 below. Round 6 (first APPROVE in this loop) independently re-verified
every fix against source with no remaining findings beyond one cosmetic §5 wording gap (file
list not naming `main.go` explicitly, unlike §2's convention) and an inaccurate "this project's
convention" attribution for the round-trip-testing note (traced to personal working-style
guidance, not an established in-repo precedent) — both corrected below. Round 6 also gave the
opinion, not as a blocking finding, that the narrower no-widening variant named in §2 might be
worth adopting outright given this package's admission-rule blast radius (changes here verify
across 38 services); left as-is per that round's own framing of it as a legitimate,
already-transparent implementation-time choice rather than something this design document needs
to pre-decide.

## 1. Chosen approach: dedicated ephemeral channel for QueueBind/QueueUnbind

Of the two options the analysis left open:

- **Option 1 (mutex serializing queue-channel RPCs)** — rejected. Requires a mutex distinct from
  `r.mu` (reusing `r.mu` deadlocks — see analysis §7), and even done correctly it serializes
  every `QueueBind`/`QueueUnbind`/`startConsumers` call across the whole rabbit instance,
  including unrelated queues, for the lifetime of each network round-trip. Ongoing maintenance
  cost (a lock-ordering rule future contributors must remember) for a problem that has a
  structural fix available.
- **Option 2 (dedicated ephemeral channel for QueueBind/QueueUnbind) — chosen.** Matches this
  package's own dominant convention (`publishExchange`, `RequestPublish`,
  `publishRPCErrorResponse`, and `executeConsumeRPC`'s reply-publish path all already
  open-use-close a throwaway `r.connection.Channel()` per call). `queue.bind`/`queue.unbind` are
  broker-side operations scoped to the *queue*, not to the *issuing channel* — AMQP does not
  require the binding channel to be the same channel a consumer later reads from. Binding on a
  fresh channel is semantically identical to binding on `queue.channel`, and removes the race
  **by construction**: `QueueBind`/`QueueUnbind` no longer touch `queue.channel` at all, so
  nothing from that path can contend with `startConsumers`'s `Qos`/`Consume` on it. No lock, no
  lock-ordering rule, no contention with unrelated queues. (Two concurrent `startConsumers`
  calls on the *same* queue name — e.g. an overlapping initial `ConsumeMessage` registration and
  a reconnect-triggered `reconsumerAll` — would still contend with each other on `queue.channel`;
  that pairing is outside this ticket's two identified trigger windows and not addressed here —
  see the residual noted at the end of this section.)

### Why this closes both trigger windows with one change

Every caller of `QueueBind`/`QueueUnbind` — every service's startup bind, `redeclareAll`'s
bind-restoration loop, and `scopeRefCount` — goes through the same
`rabbitmqhandler.QueueBind`/`QueueUnbind` functions. Moving the network call in those two
functions off `queue.channel` onto a fresh channel means none of those callers touch
`queue.channel` any more, for either trigger window:

- **Window (a) — reconnect.** The actual racing pair (correcting an earlier draft of this
  document, which mis-described `redeclareAll`'s own bind-restoration loop and `reconsumerAll`
  as the two racing parties — they run sequentially on the same `reconnector()` goroutine and
  can never race each other) is `scopeRefCount`'s arbitrary-timing `QueueBind`/`QueueUnbind`
  call racing the `reconnector()` goroutine's `startConsumers`-driven `Qos`/`Consume` on the
  newly-replaced `queue.channel`. Since `scopeRefCount`'s call no longer touches `queue.channel`
  post-fix, this pairing is closed.
- **Window (b) — pod startup.** `scopeRefCount`'s call racing the async `ConsumeMessage`
  goroutine's `Qos`/`Consume` registration in `subscribehandler.Run()`. Same reasoning, same
  fix. No separate fix is needed for window (b)'s async dispatch itself — the analysis's §7
  "make `Run()` synchronous" item is superseded by this fix and is **not** implemented (would
  add a startup-latency cost for no remaining correctness benefit).

**Known residual, not addressed by this fix:** `QueueDeclare` (on reconnect) closes the
*previous* `queue.channel` while a concurrent `startConsumers` call may still be mid-`Qos`/
`Consume` on that same object — e.g. the `reconnector()` goroutine's re-declare racing a
still-in-flight startup `ConsumeMessage` goroutine from before the reconnect. This is a
`startConsumers`-vs-`startConsumers`/`QueueDeclare` pairing, not a `QueueBind`-vs-anything
pairing, so it falls outside both of this ticket's identified trigger windows and outside what
this fix's mechanism (moving `QueueBind`/`QueueUnbind` off the shared channel) can address. Not
fixed here; flagged so this document doesn't read as "closes the hazard class," only "closes the
two windows VOIP-1431 identified."

## 2. Scope

**In scope:**
- `bin-common-handler/pkg/rabbitmqhandler/main.go`: widen `amqpConnection.Channel()` to return
  `amqpChannel` (the existing test-seam interface) instead of the concrete `*amqp.Channel`, and
  add `PublishWithContext` to `amqpChannel`. **Why an adapter is required, not just a signature
  edit:** the package's doc comment states "`*amqp.Connection` implicitly satisfies this
  interface" (`main.go:53-55`) — true today, but Go has no return-type covariance, so once
  `Channel()`'s return type changes to the interface, `*amqp.Connection`'s concrete method
  `func (c *Connection) Channel() (*Channel, error)` no longer has a matching signature and
  `*amqp.Connection` stops satisfying `amqpConnection` — `r.connection = conn` at `connect()`
  (`main.go:221`) would fail to compile. This is resolved with a thin adapter, not by editing the
  interface alone (see §3 for the exact type). `*amqp.Channel` already implements every method
  `amqpChannel` needs, including `PublishWithContext` (used by `publish.go` and `consume.go`'s
  RPC-reply path, both of which already call `r.connection.Channel()` and would need the added
  method) — the *interface* widening is a pure widen; the *production code* needs the adapter to
  keep compiling, and the adapter's error path must return an explicit nil interface (not a
  typed-nil `*amqp.Channel` boxed into `amqpChannel`, which would compare `!= nil` and break
  `checkConnection`'s existing `if ch != nil { ch.Close() }` check at `main.go:239-241`) — see §3.
  **Ripple this adds to `main_test.go`, not just `main.go`:** widening `amqpChannel` breaks
  compilation of every existing implementor of that interface until each gets the new method —
  three hand-written mock types in `main_test.go` (`mockChannel`, line 16;
  `mockChannelWithConsumeCounter`, line 142; `closedCaptureMockChannel`, line 1165), each needing
  a `PublishWithContext(...)` stub. See §4 for the exact list and stub content.
  **Is the widening actually required by the race fix itself? No — stated explicitly, since it's
  the single largest source of ripple in this design and was the root cause of two of this
  document's five review rounds' findings.** `QueueBind`/`QueueUnbind` could instead open their
  ephemeral channel through the *existing*, un-widened `amqpConnection.Channel()` and use the
  concrete `*amqp.Channel` it already returns — exactly how `ExchangeDeclare`, `QueueDeclare`,
  `publishExchange`, and `RequestPublish` already do today, none of which needed any interface
  change. That smaller variant drops the adapter, the `PublishWithContext` addition, and the
  three-mock ripple entirely. What it costs: without the widening, `mockConnection.Channel()` can
  never return a working mock channel (only `(nil, nil)` or an error), so §4's structural
  regression test — "`QueueBind`/`QueueUnbind` open a channel *different* from `queue.channel`,"
  the one automated check that the actual fix (not touching the shared channel) is doing what it
  claims — cannot be written as a unit test; only reachable via the real-broker integration test
  path (§4's regression-test section, option 1). This design accepts the wider interface and its
  mock-update cost specifically to make that structural property unit-testable, not because the
  race fix needs it. If unit-testing that property is judged not worth the ripple at
  implementation time, the narrower variant (skip the widening, rely solely on the integration
  test for structural coverage) is a legitimate fallback — noted here so that choice is visible
  and deliberate, not discovered mid-implementation.
- Add `connectionGet()` — an `r.mu.RLock()`-guarded accessor for `r.connection` — used in
  `QueueBind`/`QueueUnbind`, **and** change `connect()`'s write of `r.connection`
  (`main.go:221`) to take `r.mu.Lock()` around the assignment. **Why the write must be locked
  too, not just the read:** a read-side lock alone provides no synchronization against an
  unlocked writer — `go test -race` still flags it, and the goal (closing off the exact
  goroutine pairing this ticket is about, `scopeRefCount` vs. the `reconnector()` goroutine) is
  not met by locking only one side. Verified `connect()` is safe to lock: neither `Connect()`
  (`main.go:147-151`) nor `reconnector()` (`main.go:188-203`) holds `r.mu` when calling
  `connect()`, so there is no re-entrancy hazard. This makes `r.connection` a genuinely
  race-free field for the two functions this ticket touches. It does **not** retroactively fix
  the nine *other* pre-existing unsynchronized readers of `r.connection` — grepped every
  `r.connection.<method>` call site in the package's non-test files to get an exhaustive count
  (an earlier draft of this document twice undercounted this, first as five, then as six; both
  counts were wrong): `publishExchange` (`publish.go:16`), `RequestPublish` (`publish.go:55`),
  `executeConsumeRPC`'s reply-publish path (`consume.go:307`), `publishRPCErrorResponse`
  (`consume.go:353`), `ExchangeDeclare` (`exchange.go:11`), `QueueDeclare` (`queue.go:103`),
  `checkConnection` (`main.go:235`), `Close()` (`main.go:177`), and `healthChecker()`
  (`main.go:259`). All nine read the field directly without going through `connectionGet()`, and
  remain a separate, pre-existing, out-of-scope issue (see below). (A tenth read exists at
  `main.go:225`, `r.connection.NotifyClose(r.errorChannel)` — deliberately excluded from this
  list, not missed: it's inside `connect()` itself, reading the value `connect()` just wrote at
  `main.go:221` in the same, single invocation. `connect()` is called only from `Connect()` and
  `reconnector()`, never concurrently with itself, so this read always happens-after its own
  write in program order on one goroutine and cannot race — unlike the other nine, which are
  genuinely different goroutines reading a value some other goroutine last wrote.)
- `bin-common-handler/pkg/rabbitmqhandler/queue.go`: `QueueBind`, `QueueUnbind` — issue the
  network call on a fresh channel obtained via `connectionGet().Channel()`, closed after the
  call, instead of `queue.channel`. The queue-existence check (`queueGet(name) == nil` → `"no
  queue found"`) stays as-is.
- `bin-common-handler/pkg/rabbitmqhandler/main_test.go`: update `mockConnection.Channel()` (and
  its `channelFunc` field type) to return a mock `amqpChannel` (not `nil, nil`), and update every
  existing `QueueBind`/`QueueUnbind` test (success- and error-path alike) that currently
  constructs `&rabbit{...}` with `connection: nil` to instead provide a `mockConnection` — see
  §4 for the specific test inventory affected, including the three that need more than adding a
  `mockConnection` (their assertions currently target `queue.channel`'s mock, which the new code
  no longer touches).
- `bin-api-manager/pkg/websockhandler/scoperefcount.go`: remove the `OPEN QUESTION (VOIP-1431)`
  doc comment on `Acquire` (lines 35-41) now that the question is answered and fixed; replace
  with a brief note that the underlying channel-sharing hazard is resolved at the
  `rabbitmqhandler` level (link to this design doc and the analysis doc), so a future reader
  isn't left staring at a stale "unconfirmed" marker on the exact file this ticket names.

**Explicitly out of scope (checked against actual usage, not assumed):**
- `QueueDelete` (`queue.go:22-34`) and `QueueQoS` (`queue.go:144-155`) — confirmed dead code by
  a stronger argument than a repo-wide grep (which can miss a caller and would decay over time):
  neither method appears in the `Rabbit` interface (`main.go:17-37`) nor the `SockHandler`
  interface (`bin-common-handler/pkg/sockhandler/main.go:13-33`), and `rabbit` is an unexported
  struct that only `NewRabbit` constructs, always returned behind one of those two interfaces —
  so these two methods are unreachable from outside the package *by construction*, not merely
  uncalled today. (The same-named `QueueDelete` methods in `bin-api-manager/server/queues.go`
  and `bin-queue-manager/pkg/queuehandler/db.go` are confirmed-unrelated domain methods: the
  former proxies an RPC to queue-manager, the latter runs a SQL update — neither touches AMQP.)
  Documentation-only touch: a one-line comment on each noting they share this hazard class and
  should get the same treatment if ever wired up — this is an in-scope doc edit, not a "leave
  entirely alone" item.
- `subscribehandler.Run()`'s async `ConsumeMessage` dispatch (window (b)'s literal trigger) —
  superseded per §1 above, not touched.
- `startConsumers`'s `Qos`/`Consume` calls — stay on `queue.channel` as-is; that's correct and
  necessary (a consumer must read from the channel it registered on).
- Any other service's `QueueBind` usage — every other caller in the monorepo calls it
  synchronously at startup before `ConsumeMessage` starts. Spot-checked beyond the issue
  analysis's own examples: `bin-campaign-manager/pkg/subscribehandler/main.go:117-128`,
  `bin-conversation-manager/pkg/subscribehandler/main.go:119-130`, and
  `bin-number-manager/pkg/subscribehandler/main.go:107-118` (which carries an explicit comment,
  lines 97-102, that this block "MUST run synchronously here, BEFORE the ConsumeMessage
  goroutine below") — pattern holds broadly, not just in the cases already checked. They were
  never actually racing anything; this fix changes `rabbitmqhandler`'s shared implementation, so
  they get the same channel-isolation for free with zero call-site changes.
- The nine other pre-existing unsynchronized readers of `r.connection` (`publishExchange`,
  `RequestPublish`, `executeConsumeRPC`, `publishRPCErrorResponse`, `ExchangeDeclare`,
  `QueueDeclare`, `checkConnection`, `Close()`, `healthChecker()` — see the exhaustive list with
  line numbers above) — a real latent data race (flagged by `go test -race` if a reconnect
  happens to overlap any of them), but it predates this ticket, is not part of the hazard this
  ticket analyzed, and fixing all nine is a separate, broader cleanup better scoped as its own
  ticket rather than folded into this one. (Locking `connect()`'s write, per the bullet above,
  does not fix these nine readers — it only makes the *write* safe; each of these nine would
  still need to be changed to read through `connectionGet()` to actually benefit.)

## 3. Implementation sketch

```go
// realConnection adapts *amqp.Connection to amqpConnection's widened Channel() signature
// (returns amqpChannel, not the concrete *amqp.Channel). Go's interface satisfaction has no
// return-type covariance, so *amqp.Connection stops satisfying amqpConnection the moment
// Channel()'s return type widens -- this thin wrapper is the reason a wrapper is needed at all,
// not an optional convenience. Close() and NotifyClose() are inherited via embedding, since
// *amqp.Connection's signatures for those already match amqpConnection unchanged.
type realConnection struct {
	*amqp.Connection
}

// Channel opens a new channel and returns it as the amqpChannel interface. On error, returns an
// explicit nil interface -- NOT a typed-nil *amqp.Channel boxed into amqpChannel. This is
// defensive rather than fixing a currently-reachable bug: amqp091-go's real
// Connection.allocateChannel never returns a non-nil channel paired with a non-nil error, and
// checkConnection's own `if ch != nil` check (main.go:239-241) is unreachable on the error path
// anyway (it returns on `err != nil` first, main.go:236-238). Kept as good defensive practice
// for this new adapter regardless -- a future caller of Channel() that doesn't happen to check
// err first should still get a genuinely nil, not typed-nil, channel back.
func (rc *realConnection) Channel() (amqpChannel, error) {
	ch, err := rc.Connection.Channel()
	if err != nil {
		return nil, err
	}
	return ch, nil
}

// connectionGet returns the current AMQP connection under a read lock. Pairs with the locked
// write in connect() below -- a read-side-only lock against an unlocked writer provides no
// synchronization at all (an earlier draft of this document made exactly that mistake).
// QueueBind/QueueUnbind read it from arbitrary caller goroutines (VOIP-1431's scopeRefCount, in
// particular), so this accessor+locked-write pair exists specifically to keep that read/write
// from becoming a second, narrower version of the very race this ticket fixes. (The package's
// nine other callers of r.connection -- publishExchange, RequestPublish, executeConsumeRPC,
// publishRPCErrorResponse, ExchangeDeclare, QueueDeclare, checkConnection, Close(),
// healthChecker() -- still read the field directly, not through this accessor; that's a
// separate, pre-existing race, out of scope here; see §2.)
func (r *rabbit) connectionGet() amqpConnection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.connection
}

// connect() (existing function, main.go:206-230) changes only at the assignment:
//   conn, err := amqp.Dial(r.uri)
//   ...
//   r.mu.Lock()
//   r.connection = &realConnection{conn}
//   r.mu.Unlock()
// (was: `r.connection = conn`, unlocked). Safe to lock here: neither Connect() (main.go:147-151)
// nor reconnector() (main.go:188-203) holds r.mu when calling connect(), so there is no
// re-entrancy hazard.

// QueueBind binds queue and exchange with a key. Issues the bind on a dedicated,
// short-lived channel (VOIP-1431) rather than the queue's own long-lived channel
// (queue.channel) -- queue.channel is shared with startConsumers's Qos/Consume, and
// amqp091-go's Channel.QueueBind/Consume/Qos all route through an internal call()
// helper with no lock and a single un-correlated reply channel (ch.rpc); calling any
// two of them concurrently on the same *amqp.Channel can cross-deliver replies. See
// docs/plans/2026-09-01-voip-1431-scoperefcount-amqp-safety-analysis.md for the full
// analysis. queue.bind is scoped to the queue, not the issuing channel, so this is
// semantically identical to binding on queue.channel -- except immediately after a
// reconnect and before redeclareAll has re-declared this queue, where the old code
// failed fast (ErrClosed on the dead queue.channel) and the new code instead issues
// queue.bind against a queue that may not be re-declared yet on the live connection:
// benign either way. A durable queue survives the reconnect on the broker side, so the
// bind succeeds and is then harmlessly re-issued (idempotently, see the existing
// tracking-map dedup below) when redeclareAll's own replay runs. A volatile queue
// (x-expires) may have expired, in which case the broker closes THIS ephemeral channel
// with a 404 -- harmless precisely because the channel is ephemeral and nothing else is
// using it; the caller sees the bind fail, same outcome as today's ErrClosed case, just
// with a different error type.
func (r *rabbit) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	if r.queueGet(name) == nil {
		return fmt.Errorf("no queue found")
	}

	conn := r.connectionGet()
	if conn == nil {
		return amqp.ErrClosed // mirrors the prior behavior's implicit "no connection" failure mode; amqp091-go's own sentinel, already imported in this package
	}
	channel, err := conn.Channel()
	if err != nil {
		return err
	}
	defer func() { _ = channel.Close() }()

	if err := channel.QueueBind(name, key, exchange, noWait, args); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range r.queueBinds[name] {
		if b.key == key && b.exchange == exchange {
			return nil
		}
	}
	r.queueBinds[name] = append(r.queueBinds[name], &queueBind{
		name: name, key: key, exchange: exchange, noWait: noWait, args: args,
	})
	return nil
}
```

`QueueUnbind` gets the mirror-image change (dedicated channel for `channel.QueueUnbind`, same
`connectionGet()` use, same tracking-map update as today).

Notes:
- The existence check now uses `r.queueGet(name) == nil` directly rather than keeping the
  `*queue` struct around — the old code used `queue.channel` for the network call, so it needed
  the full struct; the new code only needs to know a queue entry exists, since the bind itself
  goes on a brand-new channel unrelated to `queue.channel`.
- The `conn == nil` branch returns an error, not a panic — `queueGet` and `connectionGet` are
  both plain accessors, and this whole path is a normal early-return, never a nil-pointer
  dereference. (An earlier draft of this document's own test plan incorrectly described the
  pre-fix version of the affected tests as panicking under this code; corrected in §4.)
- In current production usage `r.connection` is never nil by the time any `queueGet(name) !=
  nil` check passes (a queue entry only exists after `QueueDeclare` succeeded, which requires a
  connection) — the `conn == nil` branch above is defensive, not a case expected to trigger in
  practice, and exists to give an explicit, named failure mode (`amqp.ErrClosed`, already
  imported in this package via `queue.go`'s `amqp` import — no new sentinel needed) rather than
  a nil-pointer panic if that invariant is ever violated.
- `amqp091-go`'s channel-ID allocator (`Connection.allocateChannel`/`releaseChannel`) is itself
  mutex-protected and IDs are released on `Close()`, so opening/closing many short-lived
  channels concurrently (bounded here by `scopeRefCount`'s own serializing mutex, and by
  `redeclareAll`'s sequential bind-restoration loop) has no exhaustion or leak risk at this
  volume.

## 4. Test plan

- **Prerequisite, compile-breaking if skipped: stub `PublishWithContext` on every existing
  `amqpChannel` implementor.** Widening the interface (§2) breaks compilation of every mock that
  implements it until each gets the new method. Three in `main_test.go`: `mockChannel` (line 16),
  `mockChannelWithConsumeCounter` (line 142), `closedCaptureMockChannel` (line 1165). Each needs
  a `PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg
  amqp.Publishing) error` method — a plain no-op returning `nil` is sufficient unless a specific
  test needs to assert on it. Note `mock_rabbitmqhandler.go`'s generated mocks
  (`MockamqpChannel`/`MockamqpConnection`, from `//go:generate mockgen ... -source ./main.go` at
  `main.go:3`) are regenerated by `go generate ./...`, not hand-edited — the "full verification
  workflow" bullet below already runs this, but call it out here since it's the step that
  resolves this specific ripple for the generated mocks (the three hand-written ones above still
  need manual stubs).
- **Enable mocked channel coverage (prerequisite, see §2):** update `mockConnection.Channel()`
  in `main_test.go` to return a working mock `amqpChannel` (not `nil, nil`) — this requires
  changing `mockConnection.channelFunc`'s field type from `func() (*amqp.Channel, error)` to
  `func() (amqpChannel, error)`, not just the return statement, since the field type is exported
  through the whole mock. **Ripple effect to account for, not just the field itself:** the
  default `channelFunc` currently returns `(nil, nil)`. `TestCheckConnection_ReturnsNilOnSuccess`
  (`main_test.go:1517-1532`) only asserts on `mockConn.channelCallCnt`, so it's unaffected. The
  test that actually depends on and documents the `(nil, nil)` default is
  `TestHealthChecker_DoesNotCloseHealthyConnection` (`main_test.go:1600-1602`, comment: "//
  channelErr is nil — Channel() returns nil, nil (success)") — once the default returns a
  working mock channel instead, this test will actually invoke `Close()` on it; confirm the
  mock's `Close()` is a safe no-op (it already is, per the existing mock's pattern) and update
  this test's stale comment. Sweep for any other test relying on the same `(nil, nil)` default
  before assuming only these two are affected.
- **Existing-test inventory that breaks under this change, confirmed by name (verified against
  the current file, not assumed):** every test below constructs `&rabbit{...}` with `connection`
  left as the zero value and exercises `QueueBind`/`QueueUnbind`. Once those functions call
  `connectionGet().Channel()`, `connectionGet()` returns a nil `amqpConnection` and `QueueBind`/
  `QueueUnbind` correctly return `amqp.ErrClosed` per §3 (an early, incorrect draft of this plan
  said these tests would panic — they fail with an unexpected error instead, which is what
  needs fixing test-by-test):
  - `TestQueueBind_Success`, `TestQueueUnbind_Success`, `TestQueueUnbind_KeepsOtherBinds`,
    `TestQueueBind_IdempotentRebind`, `TestQueueSubscribe_CallsQueueBind` — add a
    `mockConnection` whose `channelFunc` returns a working mock channel; no other change needed,
    since these tests only assert on the tracking-map outcome, not on which channel object was
    used.
  - `TestQueueBind_ReturnsChannelError` and `TestQueueUnbind_ReturnsChannelError` — need **more
    than adding a `mockConnection`**: today they inject `queueBindErr`/`queueUnbindErr` on the
    mock channel stored as `queue.channel`, which the new code never calls. The error must be
    injected on the mock channel returned by the new `mockConnection`'s `channelFunc` instead, or
    the test passes vacuously without exercising the intended error path.
  - `TestRedeclareAll_RestoresAllBindsForSameQueue` (the VOIP-1258 finding-F2 regression test) —
    same relocation problem: it currently asserts `mockCh.queueBindCallCount == 2` against
    `queue.channel`'s mock; post-fix that counter stays 0 on the old target. Move the assertion
    to the new connection-provided channel's call count, or this regression test silently stops
    testing anything.
  - `TestQueueBindUnbind_ConcurrentAccess` — add a `mockConnection`. The new concurrency hazard
    this introduces is in the **connection** mock, not the channel mock: `mockConnection.Channel()`
    (`main_test.go:119-129`) does an unguarded `m.channelCallCnt++`, and post-fix every one of
    this test's concurrent goroutines calls it — that counter needs the same kind of guard
    `mockChannel.queueBindCallCount` already has (`main_test.go:36-37`; note the rest of
    `mockChannel`, including `mockChannel.Close()`'s `m.closeCalled++`, is *not* guarded either,
    and matters if a single shared channel instance is returned across calls rather than a fresh
    one per call — pick whichever matches this file's existing pattern, per the note below, and
    guard whatever ends up shared).
- **`mockChannel` gap to fill:** `mockChannel.QueueUnbind` currently has no call counter (unlike
  `QueueBind`, which does) — add one, since the new "different channel instance" assertion below
  needs it for `QueueUnbind` parity with `QueueBind`.
- **New coverage confirming the structural fix:** with the mock now capable of it, add a case
  asserting `QueueBind`/`QueueUnbind` invoke `Channel()` on the mock **connection** to obtain a
  **different** mock channel instance than the one already registered as `queue.channel` for
  that queue — this pins "does not touch `queue.channel`" directly, which is the actual
  structural property this fix relies on.
- **Coverage for the new production code itself (§2/§3), not just `QueueBind`/`QueueUnbind`'s use
  of it** — an earlier draft of this test plan covered the refactored functions but never
  circled back to test the pieces §2/§3 added along the way:
  - `connectionGet()` and the new `conn == nil → amqp.ErrClosed` branch in `QueueBind`/
    `QueueUnbind` — trivially unit-testable: construct `&rabbit{}` with `connection` left nil,
    call `connectionGet()` directly and assert it returns nil, then call `QueueBind`/`QueueUnbind`
    and assert `amqp.ErrClosed` specifically (not just "some error") — this is new production
    behavior with no prior test coverage at all.
  - `realConnection.Channel()`'s explicit-nil-on-error behavior — genuinely hard to unit test
    (`&realConnection{nil}` nil-panics on the embedded call, confirmed by tracing
    `*amqp.Connection.Channel()` → `openChannel()` → `allocateChannel()`, which unconditionally
    takes a mutex on the receiver; a working test needs a real `*amqp.Connection`, which needs a
    real or fake broker). State explicitly, in the test file, why round-trip testing wasn't
    performed here, rather than silently having zero coverage on this exact function — code
    review is the fallback verification for this one method.
  - **The locked write in `connect()` — this is the change with the least test coverage relative
    to how central it is to this ticket's correctness claim, and needs explicit attention, not a
    silent gap.** Its entire justification (§2/§3) is "a read-only lock alone provides no
    synchronization; `go test -race` still flags it" — so `-race` coverage of this exact path
    must appear somewhere in this plan, and as of this revision it didn't. Add `-race` to the
    verification workflow below. Whether a *targeted* test can be written that fails under
    `-race` on the pre-fix (unlocked-write) code and passes on the post-fix code — mirroring this
    project's mutation-testing convention (deliberately revert the lock, confirm the race
    detector catches it, then restore) — depends on whether a test can reliably force `connect()`
    and a `QueueBind`/`QueueUnbind` call to overlap in a bounded amount of test time; `connect()`
    loops on `amqp.Dial` with a 1s retry sleep and blocks until success, which makes deterministic
    overlap hard to engineer at unit-test speed. If a reliable trigger can't be found within
    reasonable effort, this is a case for the same real-broker integration test path as the
    regression test below, not a silent skip — decide and document at implementation time, don't
    leave `-race` as the only signal.
- **Regression test for the real AMQP-level race** (the point of this ticket, not just the
  refactor): a mocked-channel unit test cannot exercise real AMQP RPC-reply cross-delivery —
  that requires two genuinely concurrent `call()`-style RPCs on one real `*amqp.Channel`, which
  needs a real (or realistically-behaving fake) broker connection, not a mock. This is
  necessarily a "prove the bug existed, prove this class of fix prevents it" test, not a
  mocked-unit-test target. Options, in preference order, to finalize at implementation time:
  1. A focused integration test against a real RabbitMQ instance (check whether existing test
     infra in this repo already provides one for `bin-common-handler`, or for another service,
     before assuming none exists), spawning a goroutine that hammers `QueueBind`/`QueueUnbind`
     while another goroutine runs `ConsumeMessage`, asserting no error and no missed messages
     across N iterations — and, if run against the pre-fix code first, confirming it actually
     fails there (proves the test has real detection power, per this project's mutation-testing
     convention).
  2. If no real-broker test infra exists anywhere in the repo, document that gap explicitly
     rather than silently shipping without it — do not substitute a mocked test and imply it
     covers the race; a mocked test can only cover the structural refactor (bullet above), never
     the race itself.
- **Full verification workflow** (mandatory, root CLAUDE.md): `go mod tidy && go mod vendor &&
  go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` in
  `bin-common-handler`, plus a build check (`go build ./...`) in every consumer service to catch
  any interface-signature drift (the exported `Rabbit`/`SockHandler` interfaces are unchanged;
  only the unexported `amqpConnection`/`amqpChannel` test seams widen, so no consumer-service
  build should be affected — verify this holds rather than assuming it). **Additionally run
  `go test -race ./pkg/rabbitmqhandler/...`** — not covered by the plain `go test ./...` above,
  and specifically relevant here since the locked write in `connect()` (§2/§3) exists precisely
  because a race detector would flag its absence; this package's own tests (starting with
  `TestQueueBindUnbind_ConcurrentAccess`) should run clean under `-race` after this change.

## 5. Risk / rollback

- **Behavior change risk:** low, with one explicitly-analyzed exception. Steady-state
  bind/unbind semantics on the broker are unchanged (same queue, same key, same exchange); only
  the AMQP channel used to issue it changes, which is not wire-visible to any consumer. The one
  analyzed exception is the reconnect-window timing change noted in §3's `QueueBind` doc comment
  above (old: fails fast with `ErrClosed` on the dead channel; new: attempts the bind on the
  live connection, which may race `redeclareAll`'s own re-declare of the same queue) — assessed
  benign in both outcomes, but a real behavior change inside the exact window this ticket
  targets, not asserted away as "identical."
- **New-race risk:** addressed directly via `connectionGet()` (§2/§3) rather than left as a gap
  — without it, this change would trade the ticket's original race for a narrower one on
  `r.connection` in the same goroutine pairing.
- **Performance:** two extra synchronous round-trips per `QueueBind`/`QueueUnbind` call (channel
  open, and channel close — `Channel.Close()` is itself a synchronous RPC waiting for
  `channel.close-ok`, not fire-and-forget). For `scopeRefCount`, this is per websocket
  subscribe/unsubscribe transition — already a much heavier operation (HTTP upgrade, auth) than
  two extra AMQP round-trips, so not expected to be observable. For the reconnect path
  (`redeclareAll`'s bind-restoration loop), this adds a channel open/close pair per
  previously-tracked bind being restored — reconnects are rare and this runs once per reconnect,
  not on any hot path.
- **Rollback:** the `rabbitmqhandler` package changes — `main.go` (the `realConnection` adapter,
  `connectionGet()`, the locked write, the widened `amqpConnection`/`amqpChannel` interfaces),
  `queue.go` (`QueueBind`/`QueueUnbind`), and `main_test.go` (the three mock updates plus new
  test cases) — are a single, self-contained commit with no schema/API changes to the *exported*
  `Rabbit`/`SockHandler` interfaces; revert if an issue surfaces. The `scoperefcount.go` comment
  update is documentation-only and has no functional rollback concern.

## 6. Open items resolved from the analysis's §7

- Mechanism: dedicated channel (Option 2), decided above.
- `QueueDelete`/`QueueQoS`: confirmed dead code (by interface-absence, not just grep), left out
  of scope with a comment (§2 above).
- `subscribehandler.Run()` synchronization: not needed, superseded by this fix (§1 above).
- Full 2-day production log sweep: not needed to proceed — the fix is a structural elimination
  of the race, not conditional on how often it has historically fired.
- (New, surfaced during this document's own review) The `r.connection` unsynchronized-read risk
  this design would otherwise have introduced: resolved via `connectionGet()` paired with a
  locked write in `connect()` (a read-only lock alone does not synchronize anything), scoped to
  the two functions this ticket touches; the nine pre-existing readers (§2 has the exhaustive,
  line-numbered list — this count was wrong twice before landing on nine) are explicitly left as
  a separate, out-of-scope, pre-existing issue rather than silently expanded into.
- (New) The `amqpConnection.Channel()` interface widening requires a `realConnection` adapter
  around `*amqp.Connection` — Go has no return-type covariance, so the interface change alone
  does not compile without it. Specified in §3.
