# Centralize ClickHouse Writes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make bin-timeline-manager the sole ClickHouse writer by removing ClickHouse from notifyhandler and all 27 services, and adding a RabbitMQ event subscriber to timeline-manager.

**Architecture:** Remove ClickHouse connection logic from bin-common-handler/pkg/notifyhandler (constructor, connection loop, publish function). Add a subscribehandler to bin-timeline-manager that subscribes to all service event exchanges via RabbitMQ and writes received events to ClickHouse. Update ~55 NewNotifyHandler() call sites to remove the clickhouseAddress parameter.

**Tech Stack:** Go, RabbitMQ (AMQP), ClickHouse, gomock

**Design doc:** `docs/plans/2026-03-15-centralize-clickhouse-writes-design.md`

**Working directory:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Centralize-clickhouse-writes-in-timeline-manager`

---

### Task 1: Remove ClickHouse from bin-common-handler/pkg/notifyhandler

**Files:**
- Modify: `bin-common-handler/pkg/notifyhandler/main.go`
- Modify: `bin-common-handler/pkg/notifyhandler/publish.go`

**Step 1: Edit `main.go` — remove ClickHouse imports, fields, constructor param, and connection logic**

Remove from imports:
- `"sync/atomic"`
- `"time"` (only if no other usage — check; `time` is NOT used elsewhere, safe to remove)
- `"github.com/ClickHouse/clickhouse-go/v2"`

Remove from struct `notifyHandler`:
- `clickhouseAddress string`
- `chClient atomic.Value`

Remove constant:
- `clickhouseRetryInterval`

Update `NewNotifyHandler` signature — remove 5th param `clickhouseAddress string`:
```go
func NewNotifyHandler(sockHandler sockhandler.SockHandler, reqHandler requesthandler.RequestHandler, queueEvent commonoutline.QueueName, publisher commonoutline.ServiceName) NotifyHandler {
```

Remove from `NewNotifyHandler` body:
- `clickhouseAddress: clickhouseAddress,` assignment
- The entire `if clickhouseAddress != "" { go h.clickhouseConnectionLoop() }` block

Delete functions entirely:
- `newClickHouseClient()`
- `clickhouseConnectionLoop()`

The resulting `main.go` should have these imports only:
```go
import (
    "context"

    "github.com/gofrs/uuid"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/sirupsen/logrus"

    commonoutline "monorepo/bin-common-handler/models/outline"
    "monorepo/bin-common-handler/pkg/requesthandler"
    "monorepo/bin-common-handler/pkg/sockhandler"
)
```

**Step 2: Edit `publish.go` — remove ClickHouse import and publish function**

Remove from imports:
- `"github.com/ClickHouse/clickhouse-go/v2"`

Remove from `publishEvent()` function — delete these lines (122-127):
```go
	// Also publish to ClickHouse (fire-and-forget)
	if h.chClient.Load() != nil {
		go h.publishToClickHouse(eventType, dataType, data)
	}
```

Delete function entirely:
- `publishToClickHouse()`

**Step 3: Verify bin-common-handler compiles**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass. The `clickhouse-go/v2` dependency should be removed from `go.mod` by `go mod tidy` since nothing imports it anymore.

**Step 4: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-Centralize-clickhouse-writes-in-timeline-manager

- bin-common-handler: Remove ClickHouse connection logic from notifyhandler
- bin-common-handler: Remove clickhouseAddress parameter from NewNotifyHandler()
- bin-common-handler: Remove clickhouse-go/v2 dependency"
```

---

### Task 2: Add EventInsert to bin-timeline-manager dbhandler

**Files:**
- Modify: `bin-timeline-manager/pkg/dbhandler/main.go`
- Create: `bin-timeline-manager/pkg/dbhandler/event_insert.go`

**Step 1: Add `EventInsert` to the `DBHandler` interface in `main.go`**

Add to the `DBHandler` interface:
```go
type DBHandler interface {
    EventInsert(ctx context.Context, timestamp time.Time, eventType string, publisher string, dataType string, data string) error
    EventList(ctx context.Context, publisher string, resourceID uuid.UUID, events []string, pageToken string, pageSize int) ([]*event.Event, error)
    AggregatedEventList(ctx context.Context, activeflowID string, pageToken string, pageSize int) ([]*event.Event, error)
    WaitForConnection(ctx context.Context) error
}
```

Note: `EventInsert` takes primitive types (not `sock.Event`) because dbhandler should not depend on sock models. The subscribe handler does the mapping.

**Step 2: Create `event_insert.go` with the implementation**

Create file `bin-timeline-manager/pkg/dbhandler/event_insert.go`:
```go
package dbhandler

import (
    "context"
    "time"

    "github.com/pkg/errors"
    "github.com/sirupsen/logrus"
)

// EventInsert inserts a single event into ClickHouse.
// Uses PrepareBatch instead of Exec because Exec's positional parameter binding (?)
// formats time.Time with second precision (toDateTime), losing sub-second data for
// DateTime64(3) columns. PrepareBatch uses the binary columnar protocol which
// correctly preserves millisecond precision.
func (h *dbHandler) EventInsert(ctx context.Context, timestamp time.Time, eventType string, publisher string, dataType string, data string) error {
    log := logrus.WithFields(logrus.Fields{
        "func":       "EventInsert",
        "event_type": eventType,
        "publisher":  publisher,
    })

    if h.conn == nil {
        return errors.New("clickhouse connection not established")
    }

    batch, err := h.conn.PrepareBatch(ctx, "INSERT INTO events (timestamp, event_type, publisher, data_type, data)")
    if err != nil {
        return errors.Wrap(err, "could not prepare ClickHouse batch")
    }

    if err := batch.Append(timestamp, eventType, publisher, dataType, data); err != nil {
        return errors.Wrap(err, "could not append to ClickHouse batch")
    }

    if err := batch.Send(); err != nil {
        return errors.Wrap(err, "could not send ClickHouse batch")
    }

    log.Debugf("Inserted event into ClickHouse. event_type: %s, publisher: %s", eventType, publisher)
    return nil
}
```

**Step 3: Regenerate mocks and verify**

```bash
cd bin-timeline-manager && go generate ./... && go build ./cmd/...
```

Expected: Compiles. Mock updated with `EventInsert` method.

---

### Task 3: Add subscribehandler to bin-timeline-manager

**Files:**
- Create: `bin-timeline-manager/pkg/subscribehandler/main.go`

**Step 1: Create the subscribe handler**

Create file `bin-timeline-manager/pkg/subscribehandler/main.go`:
```go
package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
    "context"
    "fmt"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/sirupsen/logrus"

    commonoutline "monorepo/bin-common-handler/models/outline"
    "monorepo/bin-common-handler/models/sock"
    "monorepo/bin-common-handler/pkg/sockhandler"

    "monorepo/bin-timeline-manager/pkg/dbhandler"
)

// subscribeTargets lists all service event exchanges to subscribe to.
var subscribeTargets = []commonoutline.QueueName{
    commonoutline.QueueNameAIEvent,
    commonoutline.QueueNameAgentEvent,
    commonoutline.QueueNameAsteriskEventAll,
    commonoutline.QueueNameBillingEvent,
    commonoutline.QueueNameCallEvent,
    commonoutline.QueueNameCampaignEvent,
    commonoutline.QueueNameConferenceEvent,
    commonoutline.QueueNameContactEvent,
    commonoutline.QueueNameConversationEvent,
    commonoutline.QueueNameCustomerEvent,
    commonoutline.QueueNameEmailEvent,
    commonoutline.QueueNameFlowEvent,
    commonoutline.QueueNameMessageEvent,
    commonoutline.QueueNameNumberEvent,
    commonoutline.QueueNameOutdialEvent,
    commonoutline.QueueNamePipecatEvent,
    commonoutline.QueueNameQueueEvent,
    commonoutline.QueueNameRegistrarEvent,
    commonoutline.QueueNameRouteEvent,
    commonoutline.QueueNameSentinelEvent,
    commonoutline.QueueNameStorageEvent,
    commonoutline.QueueNameTagEvent,
    commonoutline.QueueNameTalkEvent,
    commonoutline.QueueNameTimelineEvent,
    commonoutline.QueueNameTranscribeEvent,
    commonoutline.QueueNameTransferEvent,
    commonoutline.QueueNameTTSEvent,
    commonoutline.QueueNameWebhookEvent,
}

var (
    metricsNamespace = "timeline_manager"

    promSubscribeEventProcessTime = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: metricsNamespace,
            Name:      "subscribe_event_process_time",
            Help:      "Process time of received subscribe event for ClickHouse insert",
            Buckets:   []float64{50, 100, 500, 1000, 3000},
        },
        []string{"publisher", "type"},
    )
)

func init() {
    prometheus.MustRegister(promSubscribeEventProcessTime)
}

// SubscribeHandler interface
type SubscribeHandler interface {
    Run() error
}

type subscribeHandler struct {
    sockHandler sockhandler.SockHandler
    dbHandler   dbhandler.DBHandler
}

// NewSubscribeHandler creates a new SubscribeHandler.
func NewSubscribeHandler(
    sockHandler sockhandler.SockHandler,
    dbHandler dbhandler.DBHandler,
) SubscribeHandler {
    return &subscribeHandler{
        sockHandler: sockHandler,
        dbHandler:   dbHandler,
    }
}

// Run creates the subscribe queue, binds to all event exchanges, and starts consuming.
func (h *subscribeHandler) Run() error {
    log := logrus.WithField("func", "Run")
    log.Info("Creating rabbitmq queue for event subscription.")

    subscribeQueue := string(commonoutline.QueueNameTimelineSubscribe)

    // Create durable queue
    if err := h.sockHandler.QueueCreate(subscribeQueue, "normal"); err != nil {
        return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
    }

    // Subscribe to all service event exchanges
    for _, target := range subscribeTargets {
        if errSubscribe := h.sockHandler.QueueSubscribe(subscribeQueue, string(target)); errSubscribe != nil {
            log.Errorf("Could not subscribe to target. target: %s, err: %v", target, errSubscribe)
            return errSubscribe
        }
        log.Debugf("Subscribed to event exchange. target: %s", target)
    }

    // Start consuming events
    go func() {
        if errConsume := h.sockHandler.ConsumeMessage(context.Background(), subscribeQueue, "timeline-manager", false, false, false, 10, h.processEventRun); errConsume != nil {
            log.Errorf("Could not consume subscribe events. err: %v", errConsume)
        }
    }()

    log.Infof("Subscribe handler started. subscribed to %d event exchanges.", len(subscribeTargets))
    return nil
}

// processEventRun dispatches event processing in a goroutine.
func (h *subscribeHandler) processEventRun(m *sock.Event) error {
    go h.processEvent(m)
    return nil
}

// processEvent inserts the received event into ClickHouse.
func (h *subscribeHandler) processEvent(m *sock.Event) {
    log := logrus.WithFields(logrus.Fields{
        "func":       "processEvent",
        "publisher":  m.Publisher,
        "event_type": m.Type,
    })

    start := time.Now()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := h.dbHandler.EventInsert(ctx, time.Now(), m.Type, m.Publisher, m.DataType, string(m.Data)); err != nil {
        log.Errorf("Could not insert event into ClickHouse. err: %v", err)
        return
    }

    elapsed := time.Since(start)
    promSubscribeEventProcessTime.WithLabelValues(m.Publisher, m.Type).Observe(float64(elapsed.Milliseconds()))
}
```

**Step 2: Generate mock and verify compilation**

```bash
cd bin-timeline-manager && go generate ./... && go build ./cmd/...
```

---

### Task 4: Add subscribehandler test

**Files:**
- Create: `bin-timeline-manager/pkg/subscribehandler/main_test.go`

**Step 1: Write tests for processEvent**

Create file `bin-timeline-manager/pkg/subscribehandler/main_test.go`:
```go
package subscribehandler

import (
    "encoding/json"
    "testing"

    "go.uber.org/mock/gomock"

    "monorepo/bin-common-handler/models/sock"
    "monorepo/bin-timeline-manager/pkg/dbhandler"
)

func Test_processEvent(t *testing.T) {
    tests := []struct {
        name      string
        event     *sock.Event
        expectErr bool
    }{
        {
            name: "normal",
            event: &sock.Event{
                Type:      "call_created",
                Publisher: "call-manager",
                DataType:  "application/json",
                Data:      json.RawMessage(`{"id":"test-id"}`),
            },
            expectErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()

            mockDB := dbhandler.NewMockDBHandler(mc)

            h := &subscribeHandler{
                dbHandler: mockDB,
            }

            mockDB.EXPECT().EventInsert(
                gomock.Any(),
                gomock.Any(), // timestamp
                tt.event.Type,
                tt.event.Publisher,
                tt.event.DataType,
                string(tt.event.Data),
            ).Return(nil)

            h.processEvent(tt.event)
        })
    }
}
```

**Step 2: Run tests**

```bash
cd bin-timeline-manager && go test ./pkg/subscribehandler/...
```

Expected: PASS

---

### Task 5: Wire subscribehandler into timeline-manager main.go

**Files:**
- Modify: `bin-timeline-manager/cmd/timeline-manager/main.go`

**Step 1: Add import and create subscribehandler in `runServices()`**

Add import:
```go
"monorepo/bin-timeline-manager/pkg/subscribehandler"
```

In `runServices()`, after the `runListen` call (line 167-169), add:
```go
    // Run subscribe handler to consume events from all services and write to ClickHouse
    if errSubscribe := runSubscribe(sockHandler, db); errSubscribe != nil {
        return errors.Wrapf(errSubscribe, "failed to run subscribe handler")
    }
```

**Step 2: Add `runSubscribe` function**

Add after the `runListen` function:
```go
func runSubscribe(sockHandler sockhandler.SockHandler, db dbhandler.DBHandler) error {
    log := logrus.WithField("func", "runSubscribe")

    subHandler := subscribehandler.NewSubscribeHandler(sockHandler, db)

    if errRun := subHandler.Run(); errRun != nil {
        log.Errorf("Error occurred in subscribe handler. err: %v", errRun)
        return errRun
    }

    return nil
}
```

**Step 3: Verify timeline-manager compiles and passes**

```bash
cd bin-timeline-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass.

**Step 4: Commit**

```bash
git add bin-timeline-manager/
git commit -m "NOJIRA-Centralize-clickhouse-writes-in-timeline-manager

- bin-timeline-manager: Add EventInsert method to dbhandler for ClickHouse writes
- bin-timeline-manager: Add subscribehandler to consume events from all service exchanges
- bin-timeline-manager: Wire subscribehandler into main.go alongside existing listenhandler"
```

---

### Task 6: Update all service NewNotifyHandler() call sites

**Files:**
- Modify: ~55 files across 27 services + voip-asterisk-proxy

**Overview:** Every call to `NewNotifyHandler()` currently passes 5 arguments. The 5th argument (`os.Getenv("CLICKHOUSE_ADDRESS")`) must be removed. The 4th argument varies by service but stays.

All call sites follow one of two patterns:

**Pattern A (53 call sites) — single-line:**
```go
notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameXxxEvent, serviceName, os.Getenv("CLICKHOUSE_ADDRESS"))
```
Change to:
```go
notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameXxxEvent, serviceName)
```

**Pattern B (1 call site) — multi-line in `bin-talk-manager/cmd/talk-manager/main.go:82-88`:**
```go
notifyHandler := commonnotify.NewNotifyHandler(
    sockHandler,
    reqHandler,
    commonoutline.QueueNameTalkEvent,
    commonoutline.ServiceNameTalkManager,
    os.Getenv("CLICKHOUSE_ADDRESS"),
)
```
Change to:
```go
notifyHandler := commonnotify.NewNotifyHandler(
    sockHandler,
    reqHandler,
    commonoutline.QueueNameTalkEvent,
    commonoutline.ServiceNameTalkManager,
)
```

**Step 1: Find all call sites**

```bash
grep -rn "NewNotifyHandler(" --include="*.go" | grep -v mock | grep -v _test.go | grep -v vendor
```

**Step 2: Update each call site**

Remove `, os.Getenv("CLICKHOUSE_ADDRESS")` from each call. For the single-line pattern, this means deleting the last argument before the closing `)`. For the multi-line pattern in talk-manager, delete the entire `os.Getenv("CLICKHOUSE_ADDRESS"),` line.

Also remove unused `"os"` import from files where `os.Getenv("CLICKHOUSE_ADDRESS")` was the only `os` usage. Check each file — some files use `os` for other things (e.g., `os.Exit`, `os.Getenv` for other vars).

**Complete list of files to update (grouped by service):**

1. `bin-agent-manager/cmd/agent-manager/main.go`
2. `bin-agent-manager/cmd/agent-control/main.go`
3. `bin-ai-manager/cmd/ai-manager/main.go`
4. `bin-ai-manager/cmd/ai-control/main.go` (2 call sites)
5. `bin-billing-manager/cmd/billing-manager/main.go`
6. `bin-billing-manager/cmd/billing-control/main.go`
7. `bin-call-manager/cmd/call-manager/main.go`
8. `bin-call-manager/cmd/call-control/main.go`
9. `bin-campaign-manager/cmd/campaign-manager/main.go`
10. `bin-campaign-manager/cmd/campaign-control/main.go`
11. `bin-conference-manager/cmd/conference-manager/main.go`
12. `bin-conference-manager/cmd/conference-control/main.go`
13. `bin-contact-manager/cmd/contact-manager/main.go`
14. `bin-contact-manager/cmd/contact-control/main.go`
15. `bin-conversation-manager/cmd/conversation-manager/main.go`
16. `bin-conversation-manager/cmd/conversation-control/main.go`
17. `bin-customer-manager/cmd/customer-manager/main.go`
18. `bin-customer-manager/cmd/customer-control/main.go` (2 call sites)
19. `bin-email-manager/cmd/email-manager/main.go`
20. `bin-email-manager/cmd/email-control/main.go`
21. `bin-flow-manager/cmd/flow-manager/main.go`
22. `bin-flow-manager/cmd/flow-control/main.go`
23. `bin-message-manager/cmd/message-manager/main.go`
24. `bin-message-manager/cmd/message-control/main.go`
25. `bin-number-manager/cmd/number-manager/main.go`
26. `bin-number-manager/cmd/number-control/main.go`
27. `bin-outdial-manager/cmd/outdial-manager/main.go`
28. `bin-outdial-manager/cmd/outdial-control/main.go`
29. `bin-pipecat-manager/cmd/pipecat-control/main.go`
30. `bin-queue-manager/cmd/queue-manager/main.go`
31. `bin-queue-manager/cmd/queue-control/main.go` (2 call sites)
32. `bin-registrar-manager/cmd/registrar-manager/main.go`
33. `bin-registrar-manager/cmd/registrar-control/main.go` (2 call sites)
34. `bin-route-manager/cmd/route-manager/main.go`
35. `bin-route-manager/cmd/route-control/main.go`
36. `bin-sentinel-manager/cmd/sentinel-manager/main.go`
37. `bin-storage-manager/cmd/storage-manager/main.go`
38. `bin-storage-manager/cmd/storage-control/main.go` (2 call sites)
39. `bin-tag-manager/cmd/tag-manager/main.go`
40. `bin-tag-manager/cmd/tag-control/main.go`
41. `bin-talk-manager/cmd/talk-manager/main.go` (**multi-line pattern**)
42. `bin-talk-manager/cmd/talk-control/main.go`
43. `bin-transcribe-manager/cmd/transcribe-manager/main.go`
44. `bin-transcribe-manager/cmd/transcribe-control/main.go`
45. `bin-transfer-manager/cmd/transfer-manager/main.go`
46. `bin-transfer-manager/cmd/transfer-control/main.go`
47. `bin-tts-manager/cmd/tts-manager/main.go`
48. `bin-tts-manager/cmd/tts-control/main.go`
49. `bin-webhook-manager/cmd/webhook-manager/main.go`
50. `bin-webhook-manager/cmd/webhook-control/main.go`
51. `voip-asterisk-proxy/cmd/asterisk-proxy/main.go`

**Step 3: After ALL call sites are updated, do NOT commit yet — proceed to Task 7.**

---

### Task 7: Verify all affected services

**Important:** Every service that imports `bin-common-handler` needs verification because the `NewNotifyHandler` signature changed. Run verification for each service.

**Step 1: Verify each service compiles**

For each service directory, run:
```bash
cd <service-dir> && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Run services in parallel where possible. The full list:
```
bin-agent-manager
bin-ai-manager
bin-billing-manager
bin-call-manager
bin-campaign-manager
bin-conference-manager
bin-contact-manager
bin-conversation-manager
bin-customer-manager
bin-email-manager
bin-flow-manager
bin-message-manager
bin-number-manager
bin-outdial-manager
bin-pipecat-manager
bin-queue-manager
bin-registrar-manager
bin-route-manager
bin-sentinel-manager
bin-storage-manager
bin-tag-manager
bin-talk-manager
bin-transcribe-manager
bin-transfer-manager
bin-tts-manager
bin-webhook-manager
voip-asterisk-proxy
```

Also verify services that may not have direct NewNotifyHandler calls but depend on bin-common-handler:
```
bin-api-manager
bin-openapi-manager
bin-dbscheme-manager
```

**Step 2: Fix any compilation errors**

If any service fails, check:
1. Missing removal of `os.Getenv("CLICKHOUSE_ADDRESS")` argument
2. Unused `"os"` import (if CLICKHOUSE_ADDRESS was the only os.Getenv call)
3. Multi-line call patterns that were missed

**Step 3: Commit all service updates**

```bash
git add .
git commit -m "NOJIRA-Centralize-clickhouse-writes-in-timeline-manager

Remove ClickHouse direct publishing from all services. Events are now written
to ClickHouse exclusively by bin-timeline-manager via RabbitMQ event subscription.

- bin-agent-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-ai-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-billing-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-call-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-campaign-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-conference-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-contact-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-conversation-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-customer-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-email-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-flow-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-message-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-number-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-outdial-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-pipecat-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-queue-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-registrar-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-route-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-sentinel-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-storage-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-tag-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-talk-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-transcribe-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-transfer-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-tts-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- bin-webhook-manager: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls
- voip-asterisk-proxy: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler calls"
```

---

### Task 8: Push and create PR

**Step 1: Push branch**

```bash
git push -u origin NOJIRA-Centralize-clickhouse-writes-in-timeline-manager
```

**Step 2: Create PR**

Title: `NOJIRA-Centralize-clickhouse-writes-in-timeline-manager`

Body:
```
Centralize ClickHouse event writes in bin-timeline-manager. Previously all 27 services
maintained their own ClickHouse connections via notifyhandler and published events directly.
Now timeline-manager subscribes to all service event exchanges via RabbitMQ and is the sole
ClickHouse writer. This reduces operational burden (one ClickHouse URL to configure), eliminates
27 unnecessary connections, and improves resilience (RabbitMQ buffers events vs fire-and-forget).

- bin-common-handler: Remove ClickHouse connection logic and clickhouse-go dependency from notifyhandler
- bin-common-handler: Remove clickhouseAddress parameter from NewNotifyHandler()
- bin-timeline-manager: Add EventInsert method to dbhandler for ClickHouse writes
- bin-timeline-manager: Add subscribehandler to consume events from all 28 service event exchanges
- bin-timeline-manager: Wire subscribehandler into main.go alongside existing listenhandler
- All 27 services + voip-asterisk-proxy: Remove CLICKHOUSE_ADDRESS from NewNotifyHandler() calls
```
