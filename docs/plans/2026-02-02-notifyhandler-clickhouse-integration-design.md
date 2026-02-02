# NotifyHandler ClickHouse Integration Design

## Overview

Add ClickHouse integration to `notifyhandler` to store all events for analytics, real-time monitoring, and audit trail purposes.

## Requirements

- If a ClickHouse address is provided to the notifyhandler, connect and send events to ClickHouse
- All events going through `PublishEvent`/`PublishEventRaw` should also be sent to ClickHouse
- Fire-and-forget publishing - don't block the main event flow
- Retry connection if ClickHouse is unavailable at startup or becomes unavailable

## Design

### Interface Changes

**Constructor signature change:**

```go
// New signature - clickhouseAddress is optional (empty string disables ClickHouse)
func NewNotifyHandler(
    sockHandler sockhandler.SockHandler,
    reqHandler requesthandler.RequestHandler,
    queueEvent commonoutline.QueueName,
    publisher commonoutline.ServiceName,
    clickhouseAddress string,  // NEW: e.g., "clickhouse:9000" or ""
) NotifyHandler
```

**Internal struct change:**

```go
type notifyHandler struct {
    sockHandler       sockhandler.SockHandler
    reqHandler        requesthandler.RequestHandler
    queueNotify       commonoutline.QueueName
    publisher         commonoutline.ServiceName

    clickhouseAddress string           // Store address for retry
    chClient          clickhouse.Conn  // nil until connected
}
```

The public interface (`NotifyHandler`) remains unchanged - no new methods needed.

### ClickHouse Table Schema

```sql
CREATE TABLE IF NOT EXISTS events (
    timestamp DateTime64(3),
    event_type LowCardinality(String),
    publisher LowCardinality(String),
    data_type LowCardinality(String),
    data String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (event_type, timestamp)
TTL timestamp + INTERVAL 1 YEAR
```

Design choices:
- `DateTime64(3)` - millisecond precision for event timestamps
- `LowCardinality(String)` - optimized for repeated values like event types
- `PARTITION BY toYYYYMM` - monthly partitions for efficient data management
- `ORDER BY (event_type, timestamp)` - optimized for queries filtering by event type
- `TTL 1 YEAR` - automatic data expiration

### Connection Handling

**ClickHouse client initialization:**

```go
func newClickHouseClient(address string) (clickhouse.Conn, error) {
    conn, err := clickhouse.Open(&clickhouse.Options{
        Addr: []string{address},
        Auth: clickhouse.Auth{
            Database: "default",
        },
        Settings: clickhouse.Settings{
            "max_execution_time": 60,
        },
        DialTimeout: 5 * time.Second,
    })
    if err != nil {
        return nil, err
    }
    // Ping to verify connection
    if err := conn.Ping(context.Background()); err != nil {
        return nil, err
    }
    return conn, nil
}
```

**Background connection retry loop:**

```go
const clickhouseRetryInterval = 30 * time.Second

func (h *notifyHandler) clickhouseConnectionLoop() {
    log := logrus.WithFields(logrus.Fields{
        "func":    "clickhouseConnectionLoop",
        "address": h.clickhouseAddress,
    })

    for {
        // Skip if already connected
        if h.chClient != nil {
            time.Sleep(clickhouseRetryInterval)
            continue
        }

        client, err := newClickHouseClient(h.clickhouseAddress)
        if err != nil {
            log.Errorf("Could not connect to ClickHouse, retrying in %v. err: %v",
                clickhouseRetryInterval, err)
            time.Sleep(clickhouseRetryInterval)
            continue
        }

        log.Info("Successfully connected to ClickHouse")
        h.chClient = client
        time.Sleep(clickhouseRetryInterval)
    }
}
```

Behavior:
- Runs in background goroutine
- Retries every 30 seconds if connection fails
- Logs error on each failed attempt
- Logs success when connected

### Event Publishing Logic

**Modified `publishEvent` function:**

```go
func (h *notifyHandler) publishEvent(eventType string, dataType string, data json.RawMessage, timeout int, delay int) error {
    // Existing RabbitMQ logic stays the same
    evt := &sock.Event{
        Type:      eventType,
        Publisher: string(h.publisher),
        DataType:  dataType,
        Data:      data,
    }

    // ... existing RabbitMQ publish code ...

    // NEW: Also publish to ClickHouse (fire-and-forget)
    if h.chClient != nil {
        go h.publishToClickHouse(eventType, dataType, data)
    }

    return nil
}
```

**New ClickHouse publish function:**

```go
func (h *notifyHandler) publishToClickHouse(eventType string, dataType string, data []byte) {
    log := logrus.WithFields(logrus.Fields{
        "func":       "publishToClickHouse",
        "event_type": eventType,
    })

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err := h.chClient.Exec(ctx,
        "INSERT INTO events (timestamp, event_type, publisher, data_type, data) VALUES (?, ?, ?, ?, ?)",
        time.Now(),
        eventType,
        string(h.publisher),
        dataType,
        string(data),
    )
    if err != nil {
        log.Errorf("Could not publish to ClickHouse. err: %v", err)
        return
    }
}
```

Key points:
- Runs in separate goroutine - doesn't block RabbitMQ publishing
- Uses its own timeout (5 seconds)
- Logs errors but doesn't fail the main operation
- Silently skips if `chClient` is nil (not connected yet)

### Updated Initialization

```go
func NewNotifyHandler(
    sockHandler sockhandler.SockHandler,
    reqHandler requesthandler.RequestHandler,
    queueEvent commonoutline.QueueName,
    publisher commonoutline.ServiceName,
    clickhouseAddress string,
) NotifyHandler {
    h := &notifyHandler{
        sockHandler:        sockHandler,
        reqHandler:         reqHandler,
        queueNotify:        queueEvent,
        publisher:          publisher,
        clickhouseAddress:  clickhouseAddress,
    }

    // Existing RabbitMQ topic setup
    if err := sockHandler.TopicCreate(string(queueEvent)); err != nil {
        logrus.Errorf("Could not declare the event exchange. err: %v", err)
        return nil
    }

    // NEW: Start ClickHouse connection loop (if address provided)
    if clickhouseAddress != "" {
        go h.clickhouseConnectionLoop()
    }

    namespace := commonoutline.GetMetricNameSpace(publisher)
    initPrometheus(namespace)

    return h
}
```

## Files to Modify

| File | Changes |
|------|---------|
| `bin-common-handler/pkg/notifyhandler/main.go` | Add `clickhouseAddress` field, `chClient` field, `newClickHouseClient()`, `clickhouseConnectionLoop()` |
| `bin-common-handler/pkg/notifyhandler/publish.go` | Add `publishToClickHouse()`, call it from `publishEvent()` |
| `bin-common-handler/go.mod` | Add `github.com/ClickHouse/clickhouse-go/v2` dependency |
| All 20+ services using notifyhandler | Update `NewNotifyHandler()` call to add 5th parameter |

## Migration for Existing Services

All services using `notifyhandler` will need a minor update:

```go
// Before
notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, queueName, serviceName)

// After - pass empty string to disable ClickHouse
notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, queueName, serviceName, "")

// Or with ClickHouse enabled
notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, queueName, serviceName, os.Getenv("CLICKHOUSE_ADDRESS"))
```

## New Dependency

```
github.com/ClickHouse/clickhouse-go/v2
```

## Prerequisites

- Create `events` table in ClickHouse before deploying (manual or migration script)

## Testing

- Unit tests with mock ClickHouse client
- Integration test verifying events are written to ClickHouse
