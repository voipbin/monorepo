# VOIP-1258: WebSocket Event Subscription Broker-Level Scoped Routing — Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.
> Each task = one worktree commit, no exceptions. Run full service test suite after each task
> that touches `bin-common-handler` (its consumers must not break silently).

**Goal:** Convert `QueueNameWebhookEvent`'s RabbitMQ exchange from `fanout` to `topic`, with
scope-first routing keys (`customer_id.<uuid>.#` / `owner_id.<uuid>.#`), so `bin-api-manager`
pods only receive events for scopes that have a live local websocket subscriber — eliminating
unconditional per-pod processing cost and the `agent_id` → `owner_id` public API rename.

**Architecture:** 4 phases, each independently mergeable and independently safe to ship (no
phase depends on a later phase being deployed to be correct — each is additive or dual-write
until the final cutover step). Phase order: (1) shared plumbing in `bin-common-handler`
(additive, zero existing call sites touched) → (2) `bin-webhook-manager` computes routing keys
and dual-publishes → (3) exchange migration + non-websocket consumers move to the new exchange
→ (4) `bin-api-manager` dynamic bind/unbind + owner_id rename + old-exchange decommission.

**Tech Stack:** Go, RabbitMQ (`amqp091-go` via `bin-common-handler/pkg/rabbitmqhandler`), gomock
(`go generate`), existing `sock`/`hook`/`websockhandler` packages in `bin-api-manager`.

**Design doc (source of truth for all decisions/citations):**
`bin-api-manager/docs/plans/2026-07-14-voip-1258-topic-exchange-scoped-routing-design.md`

---

## Phase 1: `bin-common-handler` — additive routing-key/topic-kind plumbing

**Objective:** Expose routing-key-aware publish and topic-kind-aware exchange declaration on
`NotifyHandler`/`SockHandler`, without touching any existing method or call site.

### Task 1.1: Add `QueueBind`/`QueueUnbind` awareness check to `SockHandler` interface

**Files:**
- Modify: `bin-common-handler/pkg/sockhandler/main.go`
- Modify: `bin-common-handler/pkg/sockhandler/mock_sockhandler.go` (regenerated)

**Step 1:** `QueueBind` already exists as a public method on `rabbit` (concrete struct,
`rabbitmqhandler/queue.go:163`) but is NOT on the `SockHandler` interface. `QueueUnbind` does
not exist anywhere in `bin-common-handler` yet (verified: zero hits). Both are needed for Phase
4's dynamic bind/unbind. Add the following to the interface (this is the FINAL, authoritative
list — `EventPublish` at the `SockHandler` level already accepts a `key string` parameter,
confirmed at `sockhandler/main.go:20`, so no new `EventPublishWithKey` method is needed here;
the routing-key gap is one layer up, at `NotifyHandler`, addressed separately in Task 1.2):

```go
type SockHandler interface {
	Connect()
	Close()

	ConsumeMessage(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, numWorkers int, messageConsume sock.CbMsgConsume) error
	ConsumeRPC(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cbConsume sock.CbMsgRPC) error

	TopicCreate(name string) error
	TopicCreateWithKind(name string, kind string) error // NEW, Task 1.4

	EventPublish(topic string, key string, evt *sock.Event) error
	EventPublishWithDelay(topic string, key string, evt *sock.Event, delay int) error

	RequestPublish(ctx context.Context, queueName string, req *sock.Request) (*sock.Response, error)
	RequestPublishWithDelay(queueName string, req *sock.Request, delay int) error

	QueueCreate(name string, queueType string) error
	QueueSubscribe(name string, topic string) error
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error   // NEW, Task 1.3
	QueueUnbind(name, key, exchange string, args amqp.Table) error              // NEW, Task 1.3
}
```

Add `amqp "github.com/rabbitmq/amqp091-go"` to the import block (needed for `amqp.Table`).

**Step 2:** Regenerate mock:
```bash
cd bin-common-handler/pkg/sockhandler && go generate ./...
```

**Step 3:** Verify build:
```bash
cd bin-common-handler && go build ./...
```
Expected: FAILS at this point — `rabbit` struct's `QueueBind` signature already matches, but
`QueueUnbind` doesn't exist on `rabbit` yet, so `rabbit` no longer satisfies `SockHandler`. This
is expected and resolved by Task 1.3. Do not treat this failure as a blocker to committing this
task IF committing Task 1.1 and 1.3 together — see Task grouping note below.

**Commit grouping note:** Tasks 1.1 and 1.3 must land in the SAME commit (interface change +
implementation), since 1.1 alone leaves `bin-common-handler` non-compiling. Task 1.2 and 1.4 can
each be separate commits after 1.1+1.3 land.

### Task 1.2: Add `PublishEventWithRoutingKey` to `NotifyHandler`

**Files:**
- Modify: `bin-common-handler/pkg/notifyhandler/main.go` (interface)
- Modify: `bin-common-handler/pkg/notifyhandler/publish.go` (implementation)
- Modify: `bin-common-handler/pkg/notifyhandler/mock_main.go` (regenerated)
- Test: `bin-common-handler/pkg/notifyhandler/publish_test.go` (new or extend existing)

**Step 1: Write failing test**

```go
// bin-common-handler/pkg/notifyhandler/publish_test.go
func TestPublishEventWithRoutingKey(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &notifyHandler{
		sockHandler: mockSock,
		queueNotify: "test.queue",
		publisher:   "test-service",
	}

	data := map[string]string{"foo": "bar"}
	routingKey := "customer_id.abc123.call.call_updated.xyz789"

	mockSock.EXPECT().EventPublish("test.queue", routingKey, gomock.Any()).Return(nil)

	h.PublishEventWithRoutingKey(context.Background(), "call_updated", routingKey, data)

	// PublishEventWithRoutingKey is fire-and-forget like PublishEvent; assert via mock call above.
}
```

**Step 2: Run test to verify failure**

```bash
cd bin-common-handler/pkg/notifyhandler && go test ./... -run TestPublishEventWithRoutingKey -v
```
Expected: FAIL — `PublishEventWithRoutingKey` undefined.

**Step 3: Write minimal implementation**

```go
// bin-common-handler/pkg/notifyhandler/main.go — add to NotifyHandler interface:
type NotifyHandler interface {
	PublishEvent(ctx context.Context, eventType string, data interface{})
	PublishEventRaw(ctx context.Context, eventType string, dataType string, data []byte)
	PublishEventWithRoutingKey(ctx context.Context, eventType string, routingKey string, data interface{}) // NEW

	PublishWebhook(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookMessage)
	PublishWebhookEvent(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookMessage)
}
```

```go
// bin-common-handler/pkg/notifyhandler/publish.go — add:

// PublishEventWithRoutingKey publishes event to the event queue with an explicit AMQP routing
// key, for topic-kind exchanges. Unlike PublishEvent (which always publishes with an empty
// routing key, correct for fanout exchanges), this lets the caller target scope-based topic
// bindings. See VOIP-1258 design doc §6.
func (h *notifyHandler) PublishEventWithRoutingKey(ctx context.Context, eventType string, routingKey string, data interface{}) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "PublishEventWithRoutingKey",
		"evnet_type":  eventType,
		"routing_key": routingKey,
	})

	m, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Could not marshal the message. err: %v", err)
		return
	}

	if err := h.publishEventWithKey(routingKey, string(eventType), string(wmwebhook.DataTypeJSON), m, requestTimeoutDefault); err != nil {
		log.Errorf("Could not publish the event with routing key. err: %v", err)
		return
	}
}

// publishEventWithKey publishes an event to the event queue with the given routing key.
func (h *notifyHandler) publishEventWithKey(routingKey string, eventType string, dataType string, data json.RawMessage, timeout int) error {
	evt := &sock.Event{
		Type:      eventType,
		Publisher: string(h.publisher),
		DataType:  dataType,
		Data:      data,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	defer cancel()
	_ = ctx // reserved for future use, matches existing publishEvent's unused-ctx pattern

	start := time.Now()
	err := h.sockHandler.EventPublish(string(h.queueNotify), routingKey, evt)
	elapsed := time.Since(start)
	promNotifyProcessTime.WithLabelValues(evt.Type).Observe(float64(elapsed.Milliseconds()))
	if err != nil {
		return fmt.Errorf("could not publish the event. err: %v", err)
	}
	promNotifyTotal.WithLabelValues(evt.Type).Inc()

	return nil
}
```

**Step 4: Run test to verify pass**

```bash
cd bin-common-handler/pkg/notifyhandler && go generate ./... && go test ./... -run TestPublishEventWithRoutingKey -v
```
Expected: PASS

**Step 5: Verify no existing call site broke**

```bash
cd bin-common-handler && go build ./... && go test ./pkg/notifyhandler/...
```
Expected: all pass, zero changes needed to any existing `PublishEvent`/`PublishWebhookEvent`
call site (confirmed additive per design doc §6/§4).

**Step 6: Commit**

```bash
git add bin-common-handler/pkg/notifyhandler/
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Add PublishEventWithRoutingKey to NotifyHandler

- bin-common-handler: Add additive PublishEventWithRoutingKey method for topic-exchange scoped publishing (VOIP-1258), zero changes to existing PublishEvent/PublishWebhookEvent call sites"
```

### Task 1.3: Implement `QueueBind`/`QueueUnbind` on `rabbit`, satisfy `SockHandler`

**Files:**
- Modify: `bin-common-handler/pkg/rabbitmqhandler/queue.go`

**Step 1:** `QueueBind` already exists on `rabbit` (`queue.go:163`) with the exact signature
needed — no change required there, it's already compatible with the interface addition in Task
1.1. Only `QueueUnbind` needs to be added:

```go
// bin-common-handler/pkg/rabbitmqhandler/queue.go — add after QueueBind, and MODIFY QueueBind
// itself per the "Decision: implement option (a)" caveat above:

// QueueBind binds queue and exchange with a key. Appends to the tracked bind set for this
// queue name (does not overwrite), so redeclareAll() can restore ALL active bindings after a
// broker reconnect, not just the most recent one (VOIP-1258 round-1 implementation-plan review
// finding F2).
func (r *rabbit) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	queue := r.queueGet(name)
	if queue == nil {
		return fmt.Errorf("no queue found")
	}

	if err := queue.channel.QueueBind(name, key, exchange, noWait, args); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range r.queueBinds[name] {
		if b.key == key && b.exchange == exchange {
			return nil // already tracked, idempotent re-bind
		}
	}
	r.queueBinds[name] = append(r.queueBinds[name], &queueBind{
		name:     name,
		key:      key,
		exchange: exchange,
		noWait:   noWait,
		args:     args,
	})
	return nil
}

// QueueUnbind unbinds queue and exchange with a key, removing the matching entry from the
// tracked bind set (not the whole map key, unless the set becomes empty).
func (r *rabbit) QueueUnbind(name, key, exchange string, args amqp.Table) error {
	queue := r.queueGet(name)
	if queue == nil {
		return fmt.Errorf("no queue found")
	}

	if err := queue.channel.QueueUnbind(name, key, exchange, args); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	binds := r.queueBinds[name]
	for i, b := range binds {
		if b.key == key && b.exchange == exchange {
			r.queueBinds[name] = append(binds[:i], binds[i+1:]...)
			break
		}
	}
	if len(r.queueBinds[name]) == 0 {
		delete(r.queueBinds, name)
	}
	return nil
}
```

Also update the `r.queueBinds` field declaration (find its struct, likely in
`bin-common-handler/pkg/rabbitmqhandler/main.go` near the `queues`/`exchanges` map
declarations) from `map[string]*queueBind` to `map[string][]*queueBind`, and update
`redeclareAll()`'s snapshot loop (`main.go:275-278`) to flatten the nested slices:

```go
// bin-common-handler/pkg/rabbitmqhandler/main.go:275-278 — MODIFY:
bindsCopy := make([]*queueBind, 0, len(r.queueBinds))
for _, binds := range r.queueBinds { // now a [][]*queueBind value per key
	bindsCopy = append(bindsCopy, binds...)
}
```

**Caveat, RESOLVED (was previously deferred, now decided per round-1 implementation-plan review
finding F2)**: `r.queueBinds` is keyed by queue `name` only (`queue.go:174`, `QueueBind`'s
existing code: `r.queueBinds[name] = &queueBind{...}`) — it overwrites on each bind, so it
currently only tracks the MOST RECENT bind per queue name, not a set of all binds. **This is not
merely a bookkeeping gap: `rabbitmqhandler/main.go:260-307`'s `redeclareAll()` iterates
`r.queueBinds` and automatically re-issues `QueueBind` for each tracked entry on every broker
reconnect, to restore state after a connection drop.** Since only the LAST bind survives in the
map, once Phase 4's `scopeRefCount` (Task 4.1) has bound a single per-pod queue to N different
scope patterns, a broker reconnect will silently restore only ONE of those N bindings — every
other live subscriber's scope goes dark (receives zero events) until that specific connection
happens to re-subscribe (which nothing today triggers automatically on reconnect). **This is a
real production-availability bug, not a hypothetical**, and treating `QueueBind`/`QueueUnbind`
as fire-and-forget from `bin-api-manager`'s side (the originally-considered option (b)) does NOT
avoid it, because the bug lives inside `rabbitmqhandler`'s own internal reconnect logic, a layer
below where `bin-api-manager`'s refcount component operates.

**Decision: implement option (a).** Change `r.queueBinds` from `map[string]*queueBind` to
`map[string][]*queueBind` (one queue name maps to a SET of binds, not a single overwritten
entry). This requires:
- Updating the field declaration in `rabbitmqhandler`'s struct (find via `grep -n "queueBinds"
  bin-common-handler/pkg/rabbitmqhandler/*.go` — check ALL read/write sites, not just
  `queue.go:174` and `main.go:275-278`, before changing the type).
- `QueueBind` (Task 1.3, below): append to the slice instead of overwriting, but first check
  whether an identical `(name, key, exchange)` triple already exists in the slice (idempotent
  re-bind, since `Acquire()` in Task 4.1 may call `QueueBind` again for a pattern that's already
  bound if `scopeRefCount`'s in-memory state and the broker's actual state ever diverge after a
  reconnect — defensive, avoids duplicate slice entries).
- `QueueUnbind` (Task 1.3, below): remove the matching entry from the slice, not the whole map
  key, and only delete the map key entirely when the slice becomes empty.
- `redeclareAll()` (`main.go:298-303`): already iterates a flat list of all binds — this loop is
  UNCHANGED once the underlying data structure holds every bind instead of just the last one;
  the fix is entirely in how `queueBinds` is populated/depopulated, not in how it's replayed.
- **`bin-common-handler/pkg/rabbitmqhandler/main_test.go` MUST also be updated** — verified via
  round-2 implementation-plan review finding: this test file has ~20 direct references to
  `map[string]*queueBind{}` literals and single-value field access (e.g. `main_test.go:606-613,
  660-667`: `r.queueBinds["test-queue"].key`, `.exchange`), none of which compile against the
  new `map[string][]*queueBind` type. Before starting Task 1.3's implementation, run
  `grep -n "queueBinds" bin-common-handler/pkg/rabbitmqhandler/main_test.go` and update EVERY
  hit to the new slice-based access pattern (e.g. `r.queueBinds["test-queue"][0].key` for
  single-bind test cases, or iterate the slice for multi-bind cases). This is a REQUIRED part of
  Task 1.3's Step 2/Step 4 (build/test verification) — do not consider Task 1.3 complete until
  `go test ./bin-common-handler/pkg/rabbitmqhandler/...` passes with the new type.

This is now a REQUIRED part of Task 1.3 (not deferred to Phase 4 planning as originally
written), since Phase 4's `scopeRefCount` component depends on reconnects correctly restoring
ALL of a pod's active scope bindings, not just the most recent one.

**Step 2: Verify build**

```bash
cd bin-common-handler && go build ./...
```
Expected: PASS (this resolves the Task 1.1 compile failure).

**Step 3: Write test**

```go
// bin-common-handler/pkg/rabbitmqhandler/queue_test.go — add:
func TestQueueUnbind_Success(t *testing.T) {
	// follow existing TestQueueBind_Success pattern at queue_test.go (search for it),
	// mock channel.QueueUnbind, assert no error
}
```

**Step 4: Run tests**

```bash
cd bin-common-handler/pkg/rabbitmqhandler && go test ./... -run TestQueueUnbind -v
```
Expected: PASS

**Step 5: Commit (together with Task 1.1's interface change)**

```bash
git add bin-common-handler/pkg/sockhandler/ bin-common-handler/pkg/rabbitmqhandler/
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Add QueueBind/QueueUnbind to SockHandler interface

- bin-common-handler: Expose QueueBind (already existed on rabbit) and new QueueUnbind on the SockHandler interface, needed for VOIP-1258 Phase 4 dynamic bind/unbind"
```

### Task 1.4: Add `TopicCreateWithKind` to `SockHandler`/`rabbit`

**Files:**
- Modify: `bin-common-handler/pkg/sockhandler/main.go` (interface, already added in Task 1.1)
- Modify: `bin-common-handler/pkg/rabbitmqhandler/topic.go`

**Step 1: Write failing test**

```go
// bin-common-handler/pkg/rabbitmqhandler/topic_test.go — new file or extend
func TestTopicCreateWithKind_Topic(t *testing.T) {
	// mock ExchangeDeclare, assert called with kind="topic" when TopicCreateWithKind(name, "topic")
}
```

**Step 2: Run to verify failure**

```bash
cd bin-common-handler/pkg/rabbitmqhandler && go test ./... -run TestTopicCreateWithKind -v
```
Expected: FAIL — undefined.

**Step 3: Implement**

```go
// bin-common-handler/pkg/rabbitmqhandler/topic.go
package rabbitmqhandler

import "fmt"

func (h *rabbit) TopicCreate(name string) error {
	if errDeclare := h.ExchangeDeclare(name, "fanout", true, false, false, false, nil); errDeclare != nil {
		return fmt.Errorf("could not declare the queue for event. err: %v", errDeclare)
	}
	return nil
}

// TopicCreateWithKind declares an exchange with the given kind ("fanout", "topic", "direct",
// etc.), durable=true (matching TopicCreate's existing durability). Added for VOIP-1258 to
// support topic-kind exchanges without touching TopicCreate's existing fanout-only behavior.
func (h *rabbit) TopicCreateWithKind(name string, kind string) error {
	if errDeclare := h.ExchangeDeclare(name, kind, true, false, false, false, nil); errDeclare != nil {
		return fmt.Errorf("could not declare the exchange with kind %s. err: %v", kind, errDeclare)
	}
	return nil
}
```

**Step 4: Run test, verify pass**

```bash
cd bin-common-handler/pkg/rabbitmqhandler && go test ./... -run TestTopicCreateWithKind -v
```
Expected: PASS

**Step 5: Regenerate mocks, full build/test**

```bash
cd bin-common-handler && go generate ./... && go build ./... && go test ./...
```
Expected: all pass, zero regressions.

**Step 6: Commit**

```bash
git add bin-common-handler/pkg/rabbitmqhandler/topic.go bin-common-handler/pkg/rabbitmqhandler/topic_test.go bin-common-handler/pkg/sockhandler/mock_sockhandler.go
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Add TopicCreateWithKind for declaring non-fanout exchanges

- bin-common-handler: Add TopicCreateWithKind alongside existing TopicCreate (fanout-only), needed for VOIP-1258's new topic-kind exchange"
```

### Task 1.5: Phase 1 verification gate

**Step 1:** Full `bin-common-handler` test suite:
```bash
cd bin-common-handler && go build ./... && go test ./... -v
```
Expected: all pass, zero failures.

**Step 2:** Build a SAMPLE of dependent services to confirm zero breakage (per design doc Open
Question 9's verification requirement):
```bash
cd bin-webhook-manager && go build ./...
cd bin-agent-manager && go build ./...
cd bin-api-manager && go build ./...
```
Expected: all pass unchanged (Phase 1 is purely additive).

---

## Phase 2: `bin-webhook-manager` — compute routing keys, dual-publish

**Objective:** Move `createTopics()`'s routing-key-computation logic (customer_id, owner_id
extraction, chat-participant fan-out) into `bin-webhook-manager`, executed before publish, and
dual-publish to both the old fanout exchange (unchanged, for safety) and the new topic exchange.

### Task 2.1: Add `reqHandler` dependency to `webhookHandler`

**Files:**
- Modify: `bin-webhook-manager/pkg/webhookhandler/main.go`
- Modify: `bin-webhook-manager/cmd/webhook-manager/main.go`
- Modify: `bin-webhook-manager/cmd/webhook-control/main.go`
- Modify: `bin-webhook-manager/pkg/webhookhandler/mock_webhookhandler.go` (regenerated)

**Step 1: Modify struct + constructor**

```go
// bin-webhook-manager/pkg/webhookhandler/main.go
import (
	// ... existing imports
	"monorepo/bin-common-handler/pkg/requesthandler" // NEW
)

type webhookHandler struct {
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
	reqHandler    requesthandler.RequestHandler // NEW

	accoutHandler     accounthandler.AccountHandler
	activeflowHandler activeflowhandler.ActiveflowHandler

	httpClient *http.Client
}

func NewWebhookHandler(
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	reqHandler requesthandler.RequestHandler, // NEW, inserted after notifyHandler
	messageTargetHandler accounthandler.AccountHandler,
	activeflowHandler activeflowhandler.ActiveflowHandler,
) WebhookHandler {

	h := &webhookHandler{
		db:            db,
		notifyHandler: notifyHandler,
		reqHandler:    reqHandler, // NEW

		accoutHandler:     messageTargetHandler,
		activeflowHandler: activeflowHandler,

		httpClient: newSafeHTTPClient(),
	}

	return h
}
```

**Step 2: Update both wiring sites**

Find and update `NewWebhookHandler(...)` call sites:
```bash
grep -n "NewWebhookHandler(" bin-webhook-manager/cmd/webhook-manager/main.go bin-webhook-manager/cmd/webhook-control/main.go
```

Both already construct `reqHandler` locally (verified in design doc §6) — pass it as the new
positional argument matching the constructor signature above.

**Step 3: Regenerate mock, build**

```bash
cd bin-webhook-manager/pkg/webhookhandler && go generate ./...
cd bin-webhook-manager && go build ./...
```
Expected: PASS (this is a same-package internal change; no external callers of
`NewWebhookHandler` outside `cmd/webhook-manager` and `cmd/webhook-control`, confirmed in design
doc §6).

**Step 4: Run existing webhookhandler tests**

```bash
cd bin-webhook-manager/pkg/webhookhandler && go test ./... -v
```
Expected: PASS (constructor signature changed but no existing test should call `NewWebhookHandler`
without the new param after the mock regen — if any test breaks, update its call site to pass a
mock `reqHandler`).

**Step 5: Commit**

```bash
git add bin-webhook-manager/
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Add reqHandler dependency to webhookHandler

- bin-webhook-manager: Add requesthandler.RequestHandler DI to webhookHandler, needed for chat-participant RPC relocation (VOIP-1258 §6 harder part 2)"
```

### Task 2.2: Port `commonWebhookData` envelope-parsing struct into webhook-manager

**Files:**
- Create: `bin-webhook-manager/pkg/webhookhandler/routingkey.go`
- Test: `bin-webhook-manager/pkg/webhookhandler/routingkey_test.go`

**Step 1: Write failing test**

```go
// bin-webhook-manager/pkg/webhookhandler/routingkey_test.go
func TestParseWebhookOwnerData(t *testing.T) {
	data := json.RawMessage(`{"customer_id":"a1b2c3d4-0000-0000-0000-000000000001","owner_id":"98765432-0000-0000-0000-000000000002","owner_type":"agent","id":"xyz-0000-0000-0000-000000000003"}`)

	d, err := parseWebhookOwnerData(data)

	require.NoError(t, err)
	require.Equal(t, "a1b2c3d4-0000-0000-0000-000000000001", d.CustomerID.String())
	require.Equal(t, "98765432-0000-0000-0000-000000000002", d.OwnerID.String())
}
```

**Step 2: Run to verify failure**

```bash
cd bin-webhook-manager/pkg/webhookhandler && go test ./... -run TestParseWebhookOwnerData -v
```
Expected: FAIL — undefined.

**Step 3: Implement (port from `bin-api-manager/pkg/subscribehandler/webhookmanager.go`)**

```go
// bin-webhook-manager/pkg/webhookhandler/routingkey.go
package webhookhandler

import (
	"encoding/json"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// webhookOwnerData mirrors bin-api-manager's commonWebhookData struct (ported per VOIP-1258 §6
// harder part 1). Used to extract customer_id/owner_id from the event data payload BEFORE
// publish, so the routing key can be computed at publish time instead of at consumption time.
type webhookOwnerData struct {
	commonidentity.Identity
	commonidentity.Owner
	AIcallID uuid.UUID `json:"aicall_id,omitempty"`
	ChatID   uuid.UUID `json:"chat_id,omitempty"`
}

// parseWebhookOwnerData unmarshals the event data payload to extract customer_id/owner_id/
// aicall_id/chat_id. Best-effort: returns zero-value fields (not an error) if optional fields
// are absent, matching createTopics()'s existing tolerance for partial data.
func parseWebhookOwnerData(data json.RawMessage) (*webhookOwnerData, error) {
	d := &webhookOwnerData{}
	if err := json.Unmarshal(data, d); err != nil {
		return nil, err
	}
	return d, nil
}
```

**Step 4: Run test, verify pass**

```bash
cd bin-webhook-manager/pkg/webhookhandler && go test ./... -run TestParseWebhookOwnerData -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add bin-webhook-manager/pkg/webhookhandler/routingkey.go bin-webhook-manager/pkg/webhookhandler/routingkey_test.go
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Port webhook owner-data envelope parsing into webhook-manager

- bin-webhook-manager: Add parseWebhookOwnerData, ported from bin-api-manager's commonWebhookData (VOIP-1258 §6 harder part 1)"
```

### Task 2.3: Implement `createRoutingKeys()` in webhook-manager (owner_id, hard-cutover)

**Files:**
- Modify: `bin-webhook-manager/pkg/webhookhandler/routingkey.go`
- Test: `bin-webhook-manager/pkg/webhookhandler/routingkey_test.go`

**Step 1: Write failing test**

```go
func TestCreateRoutingKeys_CustomerOnly(t *testing.T) {
	d := &webhookOwnerData{
		Identity: commonidentity.Identity{CustomerID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"), ID: uuid.FromStringOrNil("xyz-0000-0000-0000-000000000003")},
	}
	keys := createRoutingKeys(d, "call", "call_updated")
	require.Equal(t, []string{"customer_id.a1b2c3d4-0000-0000-0000-000000000001.call.call_updated.xyz-0000-0000-0000-000000000003"}, keys)
}

func TestCreateRoutingKeys_CustomerAndOwner(t *testing.T) {
	d := &webhookOwnerData{
		Identity: commonidentity.Identity{CustomerID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"), ID: uuid.FromStringOrNil("xyz-0000-0000-0000-000000000003")},
		Owner:    commonidentity.Owner{OwnerID: uuid.FromStringOrNil("98765432-0000-0000-0000-000000000002")},
	}
	keys := createRoutingKeys(d, "queue", "queue_updated")
	require.ElementsMatch(t, []string{
		"customer_id.a1b2c3d4-0000-0000-0000-000000000001.queue.queue_updated.xyz-0000-0000-0000-000000000003",
		"owner_id.98765432-0000-0000-0000-000000000002.queue.queue_updated.xyz-0000-0000-0000-000000000003",
	}, keys)
}
```

**Step 2: Run to verify failure**

```bash
cd bin-webhook-manager/pkg/webhookhandler && go test ./... -run TestCreateRoutingKeys -v
```
Expected: FAIL — undefined.

**Step 3: Implement (non-chat resource types only — chat handled in Task 2.4)**

```go
// bin-webhook-manager/pkg/webhookhandler/routingkey.go — add:
import "fmt"

// createRoutingKeys generates AMQP routing keys for the given event, scope-first
// (VOIP-1258 §5): "<scope>.<scope_id>.<resource>.<message_type>.<resource_id>".
// owner_id replaces the old client-facing agent_id prefix (hard cutover, Open Question 10).
func createRoutingKeys(d *webhookOwnerData, resource string, messageType string) []string {
	res := []string{}

	if d.CustomerID != uuid.Nil {
		res = append(res, fmt.Sprintf("customer_id.%s.%s.%s.%s", d.CustomerID, resource, messageType, d.ID))
	}
	if d.OwnerID != uuid.Nil {
		res = append(res, fmt.Sprintf("owner_id.%s.%s.%s.%s", d.OwnerID, resource, messageType, d.ID))
	}

	return res
}
```

**Step 4: Run test, verify pass**

```bash
cd bin-webhook-manager/pkg/webhookhandler && go test ./... -run TestCreateRoutingKeys -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add bin-webhook-manager/pkg/webhookhandler/routingkey.go bin-webhook-manager/pkg/webhookhandler/routingkey_test.go
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Implement createRoutingKeys for non-chat resource types

- bin-webhook-manager: Add scope-first routing key generation (customer_id/owner_id), owner_id per Open Question 10 hard-cutover decision"
```

### Task 2.4: Chat-participant fan-out routing keys (RPC relocation)

**Files:**
- Modify: `bin-webhook-manager/pkg/webhookhandler/routingkey.go`
- Test: `bin-webhook-manager/pkg/webhookhandler/routingkey_test.go`

**Step 1: Write failing test**

```go
func TestCreateRoutingKeysForChat(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &webhookHandler{reqHandler: mockReq}

	chatID := uuid.FromStringOrNil("chat0000-0000-0000-0000-000000000001")
	d := &webhookOwnerData{
		Identity: commonidentity.Identity{CustomerID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001")},
		ChatID:   chatID,
	}

	// NOTE: TalkV1ParticipantList returns []*tkparticipant.Participant (POINTER slice),
	// verified against bin-common-handler/pkg/requesthandler/main.go:1394 and
	// talk_participants.go:16 -- NOT a value slice. Using tkparticipant (bin-talk-manager's
	// models/participant package), not tmparticipant.
	mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), chatID).Return([]*tkparticipant.Participant{
		{Owner: commonidentity.Owner{OwnerID: uuid.FromStringOrNil("p1000000-0000-0000-0000-000000000001")}},
		{Owner: commonidentity.Owner{OwnerID: uuid.FromStringOrNil("p2000000-0000-0000-0000-000000000002")}},
	}, nil)

	keys := h.createRoutingKeysForChat(context.Background(), d, "chatmessage_created")

	require.Contains(t, keys, "customer_id.a1b2c3d4-0000-0000-0000-000000000001.talk.chatmessage_created.chat0000-0000-0000-0000-000000000001")
	require.Contains(t, keys, "owner_id.p1000000-0000-0000-0000-000000000001.talk.chatmessage_created.chat0000-0000-0000-0000-000000000001")
	require.Contains(t, keys, "owner_id.p2000000-0000-0000-0000-000000000002.talk.chatmessage_created.chat0000-0000-0000-0000-000000000001")
}
```

**Step 2: Run to verify failure**

```bash
cd bin-webhook-manager/pkg/webhookhandler && go test ./... -run TestCreateRoutingKeysForChat -v
```
Expected: FAIL — undefined.

**Step 3: Implement (moves `TalkV1ParticipantList` RPC to publish time, per §6 harder part 2)**

Check the exact `RequestHandler.TalkV1ParticipantList` signature and its participant model's
field name first:
```bash
grep -n "TalkV1ParticipantList" bin-common-handler/pkg/requesthandler/*.go
```

```go
// bin-webhook-manager/pkg/webhookhandler/routingkey.go — add:
import "context"

// createRoutingKeysForChat generates routing keys for chat/chatmessage/chatparticipant events,
// including one owner_id key per chat participant (fan-out). This RPC call was moved here from
// bin-api-manager's createTopics() per VOIP-1258 §6 harder part 2 -- it now runs ONCE at publish
// time regardless of subscriber count, instead of once per pod after the fact.
func (h *webhookHandler) createRoutingKeysForChat(ctx context.Context, d *webhookOwnerData, messageType string) []string {
	log := logrus.WithFields(logrus.Fields{"func": "createRoutingKeysForChat"})

	res := []string{}

	chatID := d.ChatID
	if chatID == uuid.Nil {
		chatID = d.ID
	}
	if chatID == uuid.Nil {
		return res
	}

	if d.CustomerID != uuid.Nil {
		res = append(res, fmt.Sprintf("customer_id.%s.talk.%s.%s", d.CustomerID, messageType, chatID))
	}
	if d.OwnerID != uuid.Nil {
		res = append(res, fmt.Sprintf("owner_id.%s.talk.%s.%s", d.OwnerID, messageType, chatID))
	}

	participants, err := h.reqHandler.TalkV1ParticipantList(ctx, chatID)
	if err != nil {
		log.Errorf("Could not get chat participants, publishing customer/owner-scoped keys only. err: %v", err)
		return res
	}

	// NOTE: participants is []*tkparticipant.Participant (pointer slice) -- verified against
	// bin-common-handler/pkg/requesthandler/main.go:1394. p is a pointer here, not a value.
	for _, p := range participants {
		if p.OwnerID == d.OwnerID {
			continue // already added above
		}
		res = append(res, fmt.Sprintf("owner_id.%s.talk.%s.%s", p.OwnerID, messageType, chatID))
	}

	return res
}
```

**Step 4: Run test, verify pass**

```bash
cd bin-webhook-manager/pkg/webhookhandler && go test ./... -run TestCreateRoutingKeysForChat -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add bin-webhook-manager/pkg/webhookhandler/routingkey.go bin-webhook-manager/pkg/webhookhandler/routingkey_test.go
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Implement chat-participant fan-out routing keys at publish time

- bin-webhook-manager: Move TalkV1ParticipantList RPC from api-manager's createTopics() to publish-time, per VOIP-1258 Open Question 4/§6"
```

### Task 2.5: Wire routing-key computation + dual-publish into `SendWebhookToCustomer`/`SendWebhookToURI`

**Files:**
- Modify: `bin-webhook-manager/pkg/webhookhandler/webhook.go`
- Test: `bin-webhook-manager/pkg/webhookhandler/webhook_test.go` (extend existing)

**Step 1:** Determine the new topic exchange name constant first (Phase 3, Task 3.1 defines
it — this task has a forward dependency; if executing phases sequentially, do Task 3.1 before
this task, OR use a placeholder constant here and rename in Task 3.1's commit). Recommended:
do Task 3.1 (exchange name + declare + `topicNotifyHandler` field) BEFORE this task, then return
here. **This ordering is not optional — verified via round-3 implementation-plan review: the
code in this task calls `h.topicNotifyHandler.PublishEventWithRoutingKey(...)`, a field that
does not exist on `webhookHandler` until Task 3.1 adds it. There is no safe way to implement
this task using `h.notifyHandler` "temporarily" — that field is bound to the OLD fanout
exchange, which silently ignores routing keys, so the event would appear to publish
successfully while never actually reaching the new topic exchange at all (no compiler error, no
test failure, a completely silent feature failure). Do Task 3.1 first, full stop.**

**Step 2: Write failing test**

```go
func TestSendWebhookToCustomer_DualPublishesWithRoutingKey(t *testing.T) {
	// extend existing SendWebhookToCustomer test setup; assert that
	// h.topicNotifyHandler.PublishEventWithRoutingKey (NOT h.notifyHandler) is called once per
	// generated routing key, in addition to the existing h.notifyHandler.PublishEvent call
	// (dual-publish, Task 3.1's transition-window requirement). Use two DISTINCT mocks
	// (mockNotifyHandler, mockTopicNotifyHandler) in the test setup and assert each receives
	// calls on the correct one -- a test using a single shared mock for both fields would not
	// have caught the h.notifyHandler/h.topicNotifyHandler mixup found in round-3 review.
}
```

**Step 3: Run to verify failure**

```bash
cd bin-webhook-manager/pkg/webhookhandler && go test ./... -run TestSendWebhookToCustomer_DualPublishesWithRoutingKey -v
```
Expected: FAIL.

**Step 4: Implement**

```go
// bin-webhook-manager/pkg/webhookhandler/webhook.go — modify SendWebhookToCustomer:

func (h *webhookHandler) SendWebhookToCustomer(ctx context.Context, customerID uuid.UUID, dataType webhook.DataType, data json.RawMessage) error {
	// ... existing code through h.notifyHandler.PublishEvent(ctx, webhook.EventTypeWebhookPublished, wh) unchanged ...

	// NEW: dual-publish to the topic exchange with computed routing keys (VOIP-1258 §6/§8).
	// This is IN ADDITION TO the existing fanout PublishEvent call above, not a replacement --
	// dual-publish continues until Phase 4's cutover step (Task 4.6) removes the old exchange.
	h.publishRoutingKeyedEvent(ctx, webhook.EventTypeWebhookPublished, data)

	h.sendWebhookToActiveflow(ctx, dataType, data)

	return nil
}
```

```go
// bin-webhook-manager/pkg/webhookhandler/routingkey.go — add:

// publishRoutingKeyedEvent computes routing keys for the given event data and publishes to the
// new topic exchange with each key. Best-effort: logs and returns on parse/RPC failure without
// blocking the primary (fanout) delivery path above it.
//
// CRITICAL: uses h.topicNotifyHandler (bound to QueueNameWebhookEventTopic, a topic-kind
// exchange -- constructed in Task 3.1), NOT h.notifyHandler (bound to the OLD fanout exchange).
// Calling PublishEventWithRoutingKey on h.notifyHandler would compile and "succeed" silently --
// fanout exchanges ignore routing keys entirely, so the event would be delivered but the
// scoping this whole feature exists for would never take effect. This exact mistake was caught
// in round-3 implementation-plan review: if Task 2.5 is implemented before Task 3.1 adds the
// topicNotifyHandler field, this function CANNOT be written correctly yet -- do not stub it
// with h.notifyHandler "temporarily," implement Task 3.1 first as already instructed above, and
// write this function only once topicNotifyHandler exists on the struct.
func (h *webhookHandler) publishRoutingKeyedEvent(ctx context.Context, eventType string, data json.RawMessage) {
	log := logrus.WithFields(logrus.Fields{"func": "publishRoutingKeyedEvent", "event_type": eventType})

	d, err := parseWebhookOwnerData(data)
	if err != nil {
		log.Errorf("Could not parse owner data for routing key computation. err: %v", err)
		return
	}

	// messageType/resource parsing: eventType is the wire event type e.g. "call_updated".
	// resource = first underscore-delimited segment, matching createTopics()'s existing
	// convention (webhookmanager.go:111-120 in the pre-migration bin-api-manager code).
	tmps := strings.SplitN(eventType, "_", 2)
	if len(tmps) < 2 {
		log.Errorf("Wrong event type format for routing key. event_type: %s", eventType)
		return
	}
	resource := tmps[0]

	var keys []string
	switch resource {
	case "chat", "chatmessage", "chatparticipant":
		keys = h.createRoutingKeysForChat(ctx, d, eventType)
	default:
		keys = createRoutingKeys(d, resource, eventType)
	}

	for _, key := range keys {
		h.topicNotifyHandler.PublishEventWithRoutingKey(ctx, eventType, key, json.RawMessage(data))
	}
}
```

Apply the SAME `h.publishRoutingKeyedEvent(ctx, webhook.EventTypeWebhookPublished, data)` call
to `SendWebhookToURI`, immediately after its existing `h.notifyHandler.PublishEvent(...)` call
(per design doc §6 symmetry note — both entry points feed the same downstream path).

**Step 5: Run test, verify pass**

```bash
cd bin-webhook-manager/pkg/webhookhandler && go test ./... -v
```
Expected: all pass, including new dual-publish test.

**Step 6: Commit**

```bash
git add bin-webhook-manager/pkg/webhookhandler/webhook.go bin-webhook-manager/pkg/webhookhandler/routingkey.go bin-webhook-manager/pkg/webhookhandler/webhook_test.go
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Dual-publish routing-keyed events from SendWebhookToCustomer/SendWebhookToURI

- bin-webhook-manager: Wire routing-key computation into both webhook entry points, dual-publish to new topic exchange alongside existing fanout publish (VOIP-1258 §6/§8 transition window)"
```

---

## Phase 3: Exchange migration — new topic exchange, non-websocket consumer migration

**Objective:** Declare the new topic exchange, migrate `bin-agent-manager` and
`bin-timeline-manager`'s `QueueNameWebhookEvent` bindings to it with `#` wildcard, verify
dual-publish is flowing correctly before touching `bin-api-manager`.

### Task 3.1: Define new exchange name constant, declare as topic/durable

**Files:**
- Modify: `bin-common-handler/models/outline/queuename.go`
- Modify: `bin-webhook-manager/cmd/webhook-manager/main.go`, `cmd/webhook-control/main.go`

**Step 1:** Add the new constant next to the existing one:

```go
// bin-common-handler/models/outline/queuename.go — near line 172:
QueueNameWebhookEvent      QueueName = "bin-manager.webhook-manager.event"
QueueNameWebhookEventTopic QueueName = "bin-manager.webhook-manager.event.topic" // NEW, VOIP-1258
```

**Step 2:** In `bin-webhook-manager`'s startup wiring, declare the new exchange as `topic`/
`durable=true` via `sockHandler.TopicCreateWithKind`, alongside the existing fanout declare
(which happens implicitly inside `NewNotifyHandler` today — the new exchange needs its own
explicit declare since it's not created via a second `NewNotifyHandler` call):

```go
// bin-webhook-manager/cmd/webhook-manager/main.go — after sockHandler is constructed, before
// webhookHandler is constructed:
if err := sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameWebhookEventTopic), "topic"); err != nil {
	logrus.Errorf("Could not declare the topic exchange. err: %v", err)
	// decide: fatal or continue-without-dual-publish? Recommend fatal for webhook-manager main,
	// since dual-publish is a hard requirement of the transition window.
}
```

Apply the same in `cmd/webhook-control/main.go`.

**Step 3:** Update `webhookHandler`'s `publishRoutingKeyedEvent` (Task 2.5) to target this
constant instead of `h.queueNotify` — this requires `PublishEventWithRoutingKey` to accept an
explicit queue/exchange name parameter, OR a second `NotifyHandler` instance pointed at the new
exchange. **Recommended**: give `notifyhandler.NewNotifyHandler` a second call in webhook-manager
startup, creating a SEPARATE `NotifyHandler` instance bound to `QueueNameWebhookEventTopic`, and
inject it into `webhookHandler` as a distinct field (`topicNotifyHandler`) alongside the existing
`notifyHandler`. Revise Task 2.1's struct:

```go
type webhookHandler struct {
	db                 dbhandler.DBHandler
	notifyHandler      notifyhandler.NotifyHandler // existing fanout exchange
	topicNotifyHandler notifyhandler.NotifyHandler // NEW: topic exchange, VOIP-1258
	reqHandler         requesthandler.RequestHandler
	// ...
}
```

Update `NewWebhookHandler`'s signature to accept `topicNotifyHandler` as a new parameter, and
`publishRoutingKeyedEvent` to call `h.topicNotifyHandler.PublishEventWithRoutingKey(...)` instead
of `h.notifyHandler...`. Update both `cmd/` wiring sites to construct and pass the second
`NotifyHandler`.

**Step 4:** Build and test:
```bash
cd bin-webhook-manager && go build ./... && go test ./...
```

**Step 5: Commit**

```bash
git add bin-common-handler/models/outline/queuename.go bin-webhook-manager/
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Declare new topic exchange, wire second NotifyHandler instance

- bin-common-handler: Add QueueNameWebhookEventTopic constant
- bin-webhook-manager: Declare new topic/durable exchange at startup, add topicNotifyHandler to webhookHandler for routing-keyed publishes"
```

### Task 3.2: Migrate `bin-agent-manager`'s `QueueNameWebhookEvent` subscription to `#` wildcard on the new exchange

**Files:**
- Modify: `bin-agent-manager/cmd/agent-manager/main.go`

**Step 1:** Find the existing subscription (design doc §8: `main.go:159`):
```bash
grep -n "QueueNameWebhookEvent" bin-agent-manager/cmd/agent-manager/main.go
```

**Step 2:** Add a SECOND subscription to the new topic exchange with a `#` wildcard binding
(keep the old fanout subscription in place during the transition window — remove only in Task
3.4 once both are confirmed flowing). This requires the queue-creation/subscribe helper to
support an explicit routing-key argument for `#`, since `QueueSubscribe(name, topic)` today
always binds with an empty key (`queue.go:158-159`, correct for fanout but not for topic). Add
a new call using `QueueBind` directly (added in Task 1.3):

```go
// after the existing per-pod queue is created and existing subscriptions are set up:
if err := sockHandler.QueueBind(queueNamePod, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil); err != nil {
	logrus.Errorf("Could not bind to the new topic exchange. err: %v", err)
}
```

**Step 3:** Build and manually verify (no automated test for cross-service AMQP binding without
an integration test harness — flag for Task 3.5's integration verification instead).

```bash
cd bin-agent-manager && go build ./...
```

**Step 4: Commit**

```bash
git add bin-agent-manager/cmd/agent-manager/main.go
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Bind agent-manager to new topic exchange with # wildcard

- bin-agent-manager: Add # wildcard binding to QueueNameWebhookEventTopic alongside existing fanout subscription, transition window per VOIP-1258 §8"
```

### Task 3.3: Migrate `bin-timeline-manager`'s `QueueNameWebhookEvent` subscription (same pattern)

**Files:**
- Modify: `bin-timeline-manager/pkg/subscribehandler/main.go`

**Step 1-4:** Same pattern as Task 3.2, applied to `bin-timeline-manager` (design doc §8:
`subscribehandler/main.go:54`).

**Commit:**
```bash
git add bin-timeline-manager/pkg/subscribehandler/main.go
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Bind timeline-manager to new topic exchange with # wildcard

- bin-timeline-manager: Add # wildcard binding to QueueNameWebhookEventTopic alongside existing fanout subscription, transition window per VOIP-1258 §8"
```

### Task 3.4: Deploy and verify dual-publish flowing correctly (manual verification gate)

**Objective:** Before proceeding to Phase 4, confirm in a real (staging/sandbox) environment
that events are landing on BOTH exchanges correctly and `bin-agent-manager`/
`bin-timeline-manager` are receiving identical event streams via their new `#` binding.

**Step 1:** Deploy Phase 1-3 changes to sandbox/staging.

**Step 2:** Trigger a test event (e.g. a call state change) and verify via RabbitMQ management
UI or `rabbitmqctl list_bindings` that:
- The new `QueueNameWebhookEventTopic` exchange exists, kind=topic, durable=true.
- `bin-agent-manager`'s and `bin-timeline-manager`'s per-pod queues show BOTH the old
  fanout binding AND the new `#` topic binding.
- A published event produces messages on both queues (no duplicate processing observed in
  service logs — `bin-agent-manager`/`bin-timeline-manager`'s business logic doesn't
  distinguish which exchange a message arrived from, so receiving it twice via two different
  bindings on the SAME queue would actually be fine architecturally IF the queue itself
  dedupes... but it does NOT dedupe by default. **Verify this explicitly**: if both bindings
  target the SAME per-pod queue, a single publish that matches both bindings' criteria
  (which won't happen here since old=fanout-implicit-match-all, new=`#`-explicit-match-all --
  BOTH match every message) will cause the SAME message to be delivered to the queue TWICE.
  **This is a real risk requiring resolution before Task 3.2/3.3 can be considered complete.**

**This is a blocking finding, not just a verification step — resolve before continuing:**
Since `bin-agent-manager`/`bin-timeline-manager` dual-publish means the SAME event arrives via
the old fanout exchange (unconditional) AND the new topic exchange (`#` matches everything) to
the SAME queue, each event would be processed TWICE by these services during the transition
window, corrupting agent-status-update / audit-log semantics (double-processing).

**Resolution options** (decide before implementing Task 3.2/3.3, revise those tasks accordingly):
- (a) Use a SEPARATE queue per exchange in `bin-agent-manager`/`bin-timeline-manager` during
  the transition window (two queues, two consumers, business logic must dedupe by event ID
  or the consumer must be made idempotent) -- adds complexity.
- (b) Skip dual-consumption for these two services entirely: since `bin-webhook-manager`
  ALREADY dual-publishes to both exchanges, `bin-agent-manager`/`bin-timeline-manager` only
  need to bind to ONE exchange at a time. Have them cut over directly from old fanout binding
  to new topic `#` binding in a SINGLE atomic change (no transition window needed for these
  two specifically, since their consumption logic doesn't care about routing key granularity
  at all -- `#` and fanout behave identically for a full-firehose consumer). **Recommended**:
  this eliminates the double-processing risk entirely and simplifies Tasks 3.2/3.3 to a single
  rebind rather than a dual-bind.

**Revise Tasks 3.2 and 3.3**: replace "add a SECOND subscription" with "REPLACE the existing
fanout subscription with the new topic `#` binding, in the same deploy as
`bin-webhook-manager`'s dual-publish going live" -- since `bin-webhook-manager` publishes to
BOTH exchanges simultaneously, these two consumers can point at either one at any time without
losing events, so a direct cutover (not a gradual dual-bind) is both simpler and safer for them
specifically. Only `bin-api-manager` (Phase 4) needs an actual multi-step, per-scope dynamic
transition, because its binding pattern is genuinely different (whole-queue-content changes),
not just an exchange-source change.

**Step 3 (after resolving above):** Re-verify no double-processing, single binding per consumer
at all times.

**Step 4:** No code commit for this task (verification + design correction only) — but DO patch
Tasks 3.2/3.3 in this plan document to reflect the single-bind, direct-cutover approach before
marking Phase 3 complete.

### Task 3.5: Patch Tasks 3.2/3.3 implementation to single-bind cutover

**Files:**
- Modify: `bin-agent-manager/cmd/agent-manager/main.go` (change from Task 3.2)
- Modify: `bin-timeline-manager/pkg/subscribehandler/main.go` (change from Task 3.3)

**Step 1:** Replace the additive `QueueBind` call from Task 3.2/3.3 with a direct swap: change
the exchange name in the EXISTING `QueueSubscribe(queueNamePod, string(commonoutline.QueueNameWebhookEvent))`
call to `commonoutline.QueueNameWebhookEventTopic`, and follow it with an explicit `#`
`QueueBind` (since `QueueSubscribe` always binds with an empty key, wrong for a topic exchange).
**Critically, this must also explicitly `QueueUnbind` the OLD fanout binding — verified via
round-1 implementation-plan review finding F3 that `bin-agent-manager`/`bin-timeline-manager`
use a DURABLE, shared, non-per-pod queue (`QueueCreate(subscribeQueue, "normal")`, confirmed at
`subscribehandler/main.go:120` for timeline-manager, not `"volatile"`/UUID-suffixed like
api-manager's per-pod queue) that PERSISTS the old binding across deploys. AMQP `QueueBind` is
additive — declaring a new binding does NOT remove a pre-existing one on the same queue. Without
an explicit `QueueUnbind`, this queue ends up bound to BOTH the old fanout exchange (implicit
match-all) AND the new topic exchange with `#` (also match-all) simultaneously, reproducing
Task 3.4's double-processing bug exactly, since `bin-webhook-manager` dual-publishes to both
exchanges during this window (Task 2.5/3.1).**

```go
// bin-agent-manager/cmd/agent-manager/main.go — REPLACE:
//   sockHandler.QueueSubscribe(queueNamePod, string(commonoutline.QueueNameWebhookEvent))
// WITH (bind new + unbind old, in that order, to avoid an event-loss window where the queue is
// briefly bound to neither exchange):
if err := sockHandler.QueueCreate(queueNamePod, "normal"); err != nil { /* ... */ } // queueType matches existing declaration; do not change to "volatile"
if err := sockHandler.QueueBind(queueNamePod, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil); err != nil {
	logrus.Errorf("Could not bind to the topic exchange. err: %v", err)
	// do NOT proceed to unbind the old exchange if this bind failed -- stay on the old
	// exchange rather than risk ending up bound to neither.
} else if err := sockHandler.QueueUnbind(queueNamePod, "", string(commonoutline.QueueNameWebhookEvent), nil); err != nil {
	logrus.Errorf("Could not unbind from the old fanout exchange. err: %v", err)
	// non-fatal: the queue is now bound to BOTH exchanges (double-processing resumes) --
	// alert/log loudly, this needs manual intervention (confirm the unbind succeeded via
	// RabbitMQ management API) rather than silently leaving the queue in a degraded state.
}
```

Same pattern for `bin-timeline-manager`, using its actual queue-type argument as currently
declared (`"normal"`, confirmed at `subscribehandler/main.go:120` — do not assume `"volatile"`;
verify the exact call before writing this code).

**Step 2:** Build, test, deploy, re-verify single delivery per event (repeat Task 3.4's
verification, this time checking no double-processing).

**Step 3: Commit**

```bash
git add bin-agent-manager/cmd/agent-manager/main.go bin-timeline-manager/pkg/subscribehandler/main.go
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Cut over agent-manager/timeline-manager to topic exchange directly

- bin-agent-manager, bin-timeline-manager: Replace fanout QueueNameWebhookEvent subscription with single # wildcard binding on QueueNameWebhookEventTopic, avoiding double-processing during webhook-manager's dual-publish window (VOIP-1258 Task 3.4 finding)"
```

---

## Phase 4: `bin-api-manager` — dynamic bind/unbind, owner_id rename, decommission old exchange

**Objective:** Implement scope-aware dynamic binding on the per-pod queue, rename
`agent_id`→`owner_id` in the client-facing wire protocol (hard cutover), remove the dead
`QueueNameAgentEvent`/`QueueNameTalkEvent` subscriptions, then decommission the old fanout
exchange once everything is confirmed on the new one.

### Task 4.1: Design the per-pod scope-refcount component

**Files:**
- Create: `bin-api-manager/pkg/websockhandler/scoperefcount.go`
- Test: `bin-api-manager/pkg/websockhandler/scoperefcount_test.go`

**Step 1:** Resolve Task 1.3's caveat: track scope→refcount entirely in this NEW component
(`bin-api-manager`-local), not in `rabbitmqhandler`'s internal `queueBinds` map (design doc §9
recommendation, option (b)).

**Step 2: Write failing test**

```go
func TestScopeRefCount_BindOnFirstSubscribe(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)

	rc := newScopeRefCount(mockSock, "pod-queue-1", "bin-manager.webhook-manager.event.topic")

	mockSock.EXPECT().QueueBind("pod-queue-1", "customer_id.abc.#", "bin-manager.webhook-manager.event.topic", false, nil).Return(nil)

	err := rc.Acquire("customer_id.abc.#")
	require.NoError(t, err)
}

func TestScopeRefCount_NoRebindOnSecondSubscribe(t *testing.T) {
	// second Acquire() for the same scope should NOT call QueueBind again (refcount 2, no AMQP call)
}

func TestScopeRefCount_UnbindOnLastRelease(t *testing.T) {
	// Acquire twice, Release twice -> QueueUnbind called exactly once, on the second Release
}

func TestScopeRefCount_NoUnbindWhileRefsRemain(t *testing.T) {
	// Acquire twice, Release once -> QueueUnbind NOT called
}
```

**Step 3: Run to verify failure**

```bash
cd bin-api-manager/pkg/websockhandler && go test ./... -run TestScopeRefCount -v
```
Expected: FAIL — undefined.

**Step 4: Implement**

```go
// bin-api-manager/pkg/websockhandler/scoperefcount.go
package websockhandler

import (
	"sync"

	"monorepo/bin-common-handler/pkg/sockhandler"
)

// scopeRefCount tracks, per api-manager pod, how many local websocket connections currently
// have at least one active subscription for each AMQP binding pattern (e.g.
// "customer_id.<uuid>.#"), and binds/unbinds the pod's per-pod queue accordingly. This is the
// component that makes the fanout->topic conversion actually reduce per-pod processing: a scope
// with zero local subscribers is never bound, so the broker never delivers it to this pod.
// See VOIP-1258 design doc §9.
type scopeRefCount struct {
	mu       sync.Mutex
	counts   map[string]int // binding pattern -> active subscriber count
	sock     sockhandler.SockHandler
	queue    string
	exchange string
}

func newScopeRefCount(sock sockhandler.SockHandler, queueName string, exchangeName string) *scopeRefCount {
	return &scopeRefCount{
		counts:   make(map[string]int),
		sock:     sock,
		queue:    queueName,
		exchange: exchangeName,
	}
}

// Acquire increments the refcount for the given binding pattern, binding the queue on the
// first (0->1) transition.
func (r *scopeRefCount) Acquire(pattern string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.counts[pattern] == 0 {
		if err := r.sock.QueueBind(r.queue, pattern, r.exchange, false, nil); err != nil {
			return err
		}
	}
	r.counts[pattern]++
	return nil
}

// Release decrements the refcount for the given binding pattern, unbinding the queue on the
// last (1->0) transition. No-op (not an error) if the pattern isn't currently tracked, to
// tolerate double-release from racing cleanup paths (explicit unsubscribe + abrupt disconnect).
func (r *scopeRefCount) Release(pattern string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.counts[pattern] <= 0 {
		return nil
	}
	r.counts[pattern]--
	if r.counts[pattern] == 0 {
		delete(r.counts, pattern)
		if err := r.sock.QueueUnbind(r.queue, pattern, r.exchange, nil); err != nil {
			return err
		}
	}
	return nil
}

// ReleaseAll releases every currently-held pattern for this connection's tracked set. Used on
// abrupt disconnect (VOIP-1258 §9) where no per-topic unsubscribe message was received. Callers
// must pass the SET of patterns this specific connection held (tracked separately per
// connection, not by scopeRefCount itself -- see Task 4.3).
func (r *scopeRefCount) ReleaseAll(patterns []string) {
	for _, p := range patterns {
		_ = r.Release(p) // best-effort; log at call site if needed
	}
}
```

**Step 5: Run test, verify pass**

```bash
cd bin-api-manager/pkg/websockhandler && go generate ./... && go test ./... -run TestScopeRefCount -v
```
Expected: PASS

**Step 6: Commit**

```bash
git add bin-api-manager/pkg/websockhandler/scoperefcount.go bin-api-manager/pkg/websockhandler/scoperefcount_test.go
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Add per-pod scope refcount component for dynamic AMQP bind/unbind

- bin-api-manager: Add scopeRefCount tracking active local subscribers per AMQP binding pattern, binds on 0->1 and unbinds on 1->0 transitions (VOIP-1258 §9)"
```

### Task 4.2: Convert client-facing topic string to AMQP binding pattern

**Files:**
- Create: `bin-api-manager/pkg/websockhandler/bindpattern.go`
- Test: `bin-api-manager/pkg/websockhandler/bindpattern_test.go`

**Step 1: Write failing test**

```go
func TestTopicToBindPattern(t *testing.T) {
	cases := []struct {
		topic    string
		expected string
	}{
		{"customer_id:abc123:call:*", "customer_id.abc123.#"},
		{"owner_id:def456:queue:*", "owner_id.def456.#"},
		{"customer_id:abc123", "customer_id.abc123.#"},
	}
	for _, c := range cases {
		got, err := topicToBindPattern(c.topic)
		require.NoError(t, err)
		require.Equal(t, c.expected, got)
	}
}
```

**Step 2: Run to verify failure**

```bash
cd bin-api-manager/pkg/websockhandler && go test ./... -run TestTopicToBindPattern -v
```
Expected: FAIL.

**Step 3: Implement**

```go
// bin-api-manager/pkg/websockhandler/bindpattern.go
package websockhandler

import (
	"fmt"
	"strings"
)

// topicToBindPattern converts a client-facing subscribe topic string (colon-delimited,
// "<scope>:<scope_id>:<resource>:<resource_id_or_*>") to the AMQP binding pattern
// (dot-delimited, "<scope>.<scope_id>.#") used for scope-first topic exchange binding.
// See VOIP-1258 design doc §5.
func topicToBindPattern(topic string) (string, error) {
	tmps := strings.Split(topic, ":")
	if len(tmps) < 2 {
		return "", fmt.Errorf("invalid topic format: %s", topic)
	}
	return fmt.Sprintf("%s.%s.#", tmps[0], tmps[1]), nil
}
```

**Step 4: Run test, verify pass**

```bash
cd bin-api-manager/pkg/websockhandler && go test ./... -run TestTopicToBindPattern -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add bin-api-manager/pkg/websockhandler/bindpattern.go bin-api-manager/pkg/websockhandler/bindpattern_test.go
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Add topic-string-to-AMQP-binding-pattern conversion

- bin-api-manager: Add topicToBindPattern, converts client colon-delimited topic strings to dot-delimited AMQP binding patterns (VOIP-1258 §5)"
```

### Task 4.3: Wire scopeRefCount into subscribe/unsubscribe and abrupt-disconnect paths

**Files:**
- Modify: `bin-api-manager/pkg/websockhandler/subscription.go`
- Modify: `bin-api-manager/pkg/websockhandler/main.go` (per-pod `scopeRefCount` instance, per-connection pattern tracking)
- Test: `bin-api-manager/pkg/websockhandler/subscription_test.go` (extend existing)

**Step 1:** Add a per-connection pattern set to track what THIS connection currently holds (for
`ReleaseAll` on abrupt disconnect):

```go
// bin-api-manager/pkg/websockhandler/subscription.go — subscriptionRun's local state:
func (h *websockHandler) subscriptionRun(ctx context.Context, w http.ResponseWriter, r *http.Request, a *auth.AuthIdentity) error {
	// ... existing setup ...

	heldPatterns := make(map[string]bool) // NEW: patterns THIS connection currently holds
	var heldMu sync.Mutex                  // NEW

	newCtx, newCancel := context.WithCancel(ctx)
	go h.subscriptionRunWebsock(newCtx, newCancel, a, ws, zmqSub, heldPatterns, &heldMu) // pass new params
	go h.subscriptionRunZMQSub(newCtx, newCancel, ws, zmqSub, &writeMu)
	go h.subscriptionRunPinger(newCtx, newCancel, ws, &writeMu)

	<-newCtx.Done()
	log.Debugf("Websocket connection has been closed. agent_id: %s", a.AgentID())

	// NEW: abrupt-disconnect cleanup -- release everything this connection held, regardless of
	// whether an explicit unsubscribe message was ever received.
	heldMu.Lock()
	patterns := make([]string, 0, len(heldPatterns))
	for p := range heldPatterns {
		patterns = append(patterns, p)
	}
	heldMu.Unlock()
	h.scopeRefCount.ReleaseAll(patterns)

	return nil
}
```

**Step 2:** Wire subscribe/unsubscribe in `subscriptionHandleMessage` to call `Acquire`/`Release`
and update `heldPatterns`:

```go
// bin-api-manager/pkg/websockhandler/subscription.go — modify subscriptionHandleMessage
// signature to accept heldPatterns/heldMu (threaded from subscriptionRunWebsock):
func (h *websockHandler) subscriptionHandleMessage(ctx context.Context, a *auth.AuthIdentity, zmqSub zmqsubhandler.ZMQSubHandler, m *hook.Hook, heldPatterns map[string]bool, heldMu *sync.Mutex) error {
	// ... existing validateTopics call unchanged ...

	switch m.Type {
	case hook.TypeSubscribe:
		for _, topic := range m.Topics {
			if errSub := zmqSub.Subscribe(topic); errSub != nil {
				return errSub
			}

			// NEW: acquire the AMQP binding
			pattern, errConv := topicToBindPattern(topic)
			if errConv != nil {
				log.Errorf("Could not convert topic to bind pattern. topic: %s, err: %v", topic, errConv)
				continue // zmqSub subscribe already succeeded; local filter still works even if
				         // broker-side scoping doesn't -- degrades to "receives more than needed"
				         // not "receives nothing", a safe failure mode
			}
			if errAcq := h.scopeRefCount.Acquire(pattern); errAcq != nil {
				log.Errorf("Could not acquire the AMQP binding. pattern: %s, err: %v", pattern, errAcq)
				continue
			}
			heldMu.Lock()
			heldPatterns[pattern] = true
			heldMu.Unlock()
		}

	case hook.TypeUnsubscribe:
		for _, topic := range m.Topics {
			if errSub := zmqSub.Unsubscribe(topic); errSub != nil {
				return errSub
			}

			// NEW: release the AMQP binding
			pattern, errConv := topicToBindPattern(topic)
			if errConv != nil {
				continue
			}
			if errRel := h.scopeRefCount.Release(pattern); errRel != nil {
				log.Errorf("Could not release the AMQP binding. pattern: %s, err: %v", pattern, errRel)
			}
			heldMu.Lock()
			delete(heldPatterns, pattern)
			heldMu.Unlock()
		}
	}

	return nil
}
```

**Step 3:** Add `scopeRefCount` field to `websockHandler` struct, threaded from `cmd/api-manager/
main.go` where the per-pod queue and `sockHandler` actually live. **Verified via round-1
implementation-plan review finding F4: `websockHandler` (`pkg/websockhandler/main.go:30-33`)
currently has NO `sockhandler.SockHandler` field, and its constructor `NewWebsockHandler(reqHandler,
streamHandler)` (called from `cmd/api-manager/main.go:136`) takes neither a sock handler nor a
queue name — the per-pod queue (`queueNamePod`) and its `sockHandler` are constructed entirely
in `cmd/api-manager/main.go:157`, outside the `websockhandler` package. This requires a new
cross-package wiring, not just a struct field addition:**

```go
// bin-api-manager/pkg/websockhandler/main.go — MODIFY:
type websockHandler struct {
	reqHandler    requesthandler.RequestHandler
	streamHandler streamhandler.StreamHandler
	scopeRefCount *scopeRefCount // NEW: shared across all connections on this pod
}

// NewWebsockHandler creates a new HookHandler
func NewWebsockHandler(
	reqHandler requesthandler.RequestHandler,
	streamHandler streamhandler.StreamHandler,
	sockHandler sockhandler.SockHandler, // NEW param
	queueNamePod string, // NEW param -- the SAME per-pod queue name main.go already constructs
) WebsockHandler {

	res := &websockHandler{
		reqHandler:    reqHandler,
		streamHandler: streamHandler,
		scopeRefCount: newScopeRefCount(sockHandler, queueNamePod, string(commonoutline.QueueNameWebhookEventTopic)), // NEW
	}

	endpointInit()

	return res
}
```

```go
// bin-api-manager/cmd/api-manager/main.go — MODIFY the existing NewWebsockHandler call site
// at line 136 to pass the sockHandler and queueNamePod locals that are ALREADY constructed at
// line 157 today. This requires either (a) reordering so queue/sockHandler construction happens
// BEFORE the NewWebsockHandler(...) call (currently it's the reverse order, verify with
// `grep -n "queueNamePod\|NewWebsockHandler" bin-api-manager/cmd/api-manager/main.go` before
// writing this change), or (b) passing them as forward references if Go's initialization order
// allows it. Read the full current main.go control flow before implementing this reordering --
// do not assume it's a trivial one-line change.
websockHandler := websockhandler.NewWebsockHandler(reqHandler, streamHandler, sockHandler, queueNamePod)
```

**Step 4: Write test for the wiring**

```go
func TestSubscriptionHandleMessage_AcquiresBindingOnSubscribe(t *testing.T) {
	// mock zmqSub.Subscribe, mock scopeRefCount's underlying sockHandler.QueueBind,
	// assert both called for a subscribe message
}

func TestSubscriptionRun_ReleasesAllOnAbruptDisconnect(t *testing.T) {
	// simulate ctx cancellation without an explicit unsubscribe message,
	// assert scopeRefCount.Release called for every previously-acquired pattern
}
```

**Step 5: Run tests**

```bash
cd bin-api-manager/pkg/websockhandler && go test ./... -v
```
Expected: all pass.

**Step 6: Commit**

```bash
git add bin-api-manager/pkg/websockhandler/
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Wire dynamic AMQP bind/unbind into subscribe/unsubscribe and disconnect paths

- bin-api-manager: Call scopeRefCount.Acquire/Release on explicit subscribe/unsubscribe, ReleaseAll on abrupt disconnect via subscriptionRun's existing teardown path (VOIP-1258 §9)"
```

### Task 4.4: Rename `agent_id` -> `owner_id` in `validateTopics`/`validateTopic` (hard cutover)

**Files:**
- Modify: `bin-api-manager/pkg/websockhandler/etc.go`
- Modify: `bin-api-manager/pkg/websockhandler/etc_test.go`

**Step 1: Write failing test**

```go
// bin-api-manager/pkg/websockhandler/etc_test.go — update existing agent_id test cases:
func TestValidateTopics_OwnerScope(t *testing.T) {
	a := &auth.AuthIdentity{ /* agent with AgentID() == "98765432-..." */ }
	topics := []string{"owner_id:98765432-0000-0000-0000-000000000002:queue:*"}
	require.True(t, h.validateTopics(context.Background(), a, topics))
}

func TestValidateTopics_RejectsOldAgentIdPrefix(t *testing.T) {
	a := &auth.AuthIdentity{}
	topics := []string{"agent_id:98765432-0000-0000-0000-000000000002:queue:*"}
	require.False(t, h.validateTopics(context.Background(), a, topics)) // hard cutover: old prefix now invalid
}
```

**Step 2: Run to verify failure**

```bash
cd bin-api-manager/pkg/websockhandler && go test ./... -run "TestValidateTopics_OwnerScope|TestValidateTopics_RejectsOldAgentIdPrefix" -v
```
Expected: `TestValidateTopics_OwnerScope` FAILS (still says `agent_id`),
`TestValidateTopics_RejectsOldAgentIdPrefix` PASSES already (old behavior happens to reject
unknown prefixes... verify this against current `default: return false` branch — should already
pass since `owner_id` isn't yet a recognized case, meaning the NEW test needs `owner_id` to be
accepted and `agent_id` to be REJECTED after the rename, i.e. write both tests to assert
POST-rename behavior, run BEFORE the code change to confirm `TestValidateTopics_OwnerScope`
fails and `TestValidateTopics_RejectsOldAgentIdPrefix`... actually also fails pre-change, since
`agent_id` is currently ACCEPTED, not rejected. Both tests should fail before the code change).

**Step 3: Implement (rename, both functions)**

```go
// bin-api-manager/pkg/websockhandler/etc.go — in validateTopics, change:
//   case "agent_id":
//       if tmpID != a.AgentID() {
// TO:
		case "owner_id":
			if tmpID != a.AgentID() {
				return false
			}
```

Apply the identical change in `validateTopic` (the second near-duplicate function, line 139).
Update the two `default:` comment lines from `// the first part should be "customer_id" or
"agent_id"` to `// the first part should be "customer_id" or "owner_id"`.

**Step 4: Run tests, verify pass**

```bash
cd bin-api-manager/pkg/websockhandler && go test ./... -v
```
Expected: all pass, including the two new tests.

**Step 5: Update RST docs (per design doc §5, CLAUDE.md RST-sync rule)**

```bash
grep -rn "agent_id:" bin-api-manager/docsdev/source/websocket_overview.rst bin-api-manager/docsdev/source/websocket_struct.rst bin-api-manager/docsdev/source/websocket_tutorial.rst
```
Replace each `agent_id:` example with `owner_id:`. Rebuild:
```bash
cd bin-api-manager/docsdev && sphinx-build -M html source build
```
Force-add the build output per CLAUDE.md's RST-sync convention (build/ is gitignored elsewhere
but the sync rule requires committing rendered output — confirm current convention with
`git status` after build, follow existing pattern in the repo).

**Step 6: Commit**

```bash
git add bin-api-manager/pkg/websockhandler/etc.go bin-api-manager/pkg/websockhandler/etc_test.go bin-api-manager/docsdev/
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Rename agent_id to owner_id in websocket wire protocol (breaking change)

- bin-api-manager: Rename agent_id scope prefix to owner_id in validateTopics/validateTopic, hard cutover per Open Question 10 (no dual-accept period)
- bin-api-manager: Update websocket RST docs (overview/struct/tutorial) to reflect owner_id"
```

### Task 4.5: Remove dead `QueueNameAgentEvent`/`QueueNameTalkEvent` subscriptions

**Files:**
- Modify: `bin-api-manager/cmd/api-manager/main.go`

**Step 1:** Locate the `subscribeTargets` list (design doc §7: `main.go:159-162`):
```bash
grep -n "subscribeTargets" bin-api-manager/cmd/api-manager/main.go
```

**Step 2:** Before removing, run one more verification pass per Open Question 6 (confirm
nothing else depends on these subscriptions):
```bash
grep -rn "QueueNameAgentEvent\|QueueNameTalkEvent" bin-api-manager/
```
Expected: only the `subscribeTargets` list itself references them (confirmed in design doc §2's
single-case switch analysis — should show no other dependents).

**Step 3:** Remove both from the list, and switch the baseline exchange target to the new topic
exchange WITH an explicit `#` wildcard binding — do NOT use `QueueSubscribe`'s empty-key bind for
this target. **Verified via round-2 implementation-plan review finding: on a `topic`-kind
exchange (unlike `fanout`), an empty routing key binding only matches messages published with an
empty routing key. Since every VOIP-1258 publish path (Task 2.3/2.4/2.5) publishes with
non-empty scope-first keys (`customer_id.xxx...`/`owner_id.xxx...`), using
`QueueSubscribe(queue, target)`'s existing empty-key bind
(`rabbitmqhandler/queue.go:158-159`) against `QueueNameWebhookEventTopic` would deliver ZERO
events to bin-api-manager's baseline subscription — a silent total event-loss regression the
moment this task deploys, contradicting the plan's own phased-safety principle that no phase
should depend on a later phase to be correct.** This task and Task 4.6's `#`-fallback binding
must land TOGETHER, not sequentially:

```go
// BEFORE:
subscribeTargets := []string{
	string(commonoutline.QueueNameWebhookEvent),
	string(commonoutline.QueueNameAgentEvent),
	string(commonoutline.QueueNameTalkEvent),
}
// ... elsewhere, existing subscribehandler.Run() loop calls
// sockHandler.QueueSubscribe(queue, target) for each target, which binds with an empty key --
// correct for fanout, WRONG for the new topic exchange.

// AFTER: remove subscribeTargets' use of QueueSubscribe for the new exchange entirely; bind it
// explicitly with "#" instead, immediately after the per-pod queue is created:
if err := sockHandler.QueueBind(queueNamePod, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil); err != nil {
	logrus.Errorf("Could not bind to the topic exchange. err: %v", err)
}
// subscribeTargets no longer includes QueueNameWebhookEvent/AgentEvent/TalkEvent at all --
// the topic exchange binding above replaces it, done here (not deferred to Task 4.6).
```

Task 4.6's "keep the `#` fallback for the first deploy, remove later" framing still applies, but
the `#` binding itself must exist from THIS task's deploy onward, not added later — Task 4.6 is
now solely about REMOVING the fallback once dynamic per-scope bind/unbind (Task 4.3) is confirmed
stable, not about adding it for the first time.

**Step 4:** Build:
```bash
cd bin-api-manager && go build ./...
```

**Step 5: Commit**

```bash
git add bin-api-manager/cmd/api-manager/main.go
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Remove dead QueueNameAgentEvent/QueueNameTalkEvent subscriptions

- bin-api-manager: Remove subscriptions never acted on by processEvent's single-case switch (VOIP-1258 §7), reduces per-pod RabbitMQ consumption cost"
```

### Task 4.6: Switch api-manager's baseline subscription from fanout to topic exchange, remove old exchange dual-publish

**Files:**
- Modify: `bin-api-manager/cmd/api-manager/main.go`
- Modify: `bin-webhook-manager/pkg/webhookhandler/webhook.go` (remove dual-publish, Task 2.5's temporary code)
- Modify: `bin-webhook-manager/pkg/webhookhandler/main.go` (remove `notifyHandler` field, keep only `topicNotifyHandler`)

**This is the final cutover step — deploy and verify Tasks 4.1-4.5 in staging/production for a
full transition window (Open Question 1: window length TBD) BEFORE executing this task.**

**Step 1:** Change `bin-api-manager`'s baseline `#` fallback binding (added in Task 4.5, kept as
a safety net during initial dynamic-bind/unbind rollout) — REMOVE it now that Task 4.3's
per-scope dynamic bind/unbind is confirmed stable in production, so pods only receive events for
scopes with a live local subscriber (the actual point of this whole design). This is now a
REMOVAL step, not an initial addition — Task 4.5 already added and deployed the `#` binding as
part of its own safe cutover.

**Step 2:** Remove the dual-publish from `bin-webhook-manager`: delete the old
`h.notifyHandler.PublishEvent(ctx, webhook.EventTypeWebhookPublished, wh)` calls in
`SendWebhookToCustomer`/`SendWebhookToURI`, keeping only `publishRoutingKeyedEvent`'s topic-
exchange publish. Remove the now-unused `notifyHandler` field and its `NewNotifyHandler` startup
declare for the OLD fanout exchange (keep `topicNotifyHandler`, rename back to `notifyHandler`
for cleanliness if desired — optional cosmetic follow-up).

**Step 3:** Decommission the old exchange (`QueueNameWebhookEvent`) — infrastructure-level
action (delete via RabbitMQ management API/UI or a migration script), NOT a code change. Do
this ONLY after confirming zero consumers remain bound to it.

**Step 4:** Build, full test suite, deploy.

```bash
cd bin-webhook-manager && go build ./... && go test ./...
cd bin-api-manager && go build ./... && go test ./...
```

**Step 5: Commit**

```bash
git add bin-webhook-manager/ bin-api-manager/cmd/api-manager/main.go
git -c user.name="Sungtae Kim" -c user.email="pchero21@gmail.com" commit -m "Cut over to topic exchange exclusively, remove dual-publish

- bin-webhook-manager: Remove old fanout exchange dual-publish, topic exchange is now the sole delivery path
- bin-api-manager: Switch baseline subscription to the new topic exchange"
```

---

## Cross-cutting verification (run after EVERY phase, not just at the end)

```bash
# From monorepo root
cd bin-common-handler && go build ./... && go test ./...
cd ../bin-webhook-manager && go build ./... && go test ./...
cd ../bin-api-manager && go build ./... && go test ./...
cd ../bin-agent-manager && go build ./... && go test ./...
cd ../bin-timeline-manager && go build ./... && go test ./...
cd ../bin-queue-manager && go build ./... && go test ./...  # unaffected, sanity check only
cd ../bin-talk-manager && go build ./... && go test ./...    # unaffected, sanity check only
```

## Deferred to follow-up tickets (explicitly out of this plan's scope)

- Load/chaos testing the `scopeRefCount` component under concurrent connect/disconnect storms
  (Open Question 2 — testing strategy still needs a dedicated design pass before this ships to
  production traffic at scale).
