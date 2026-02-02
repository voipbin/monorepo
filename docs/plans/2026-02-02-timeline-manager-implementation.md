# timeline-manager Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a read-only OLAP query service for historical events stored in ClickHouse.

**Architecture:** RabbitMQ RPC service following the existing monorepo pattern (like flow-manager). Single `/v1/events` endpoint with flexible filtering. Uses ClickHouse instead of MySQL, golang-migrate for schema management.

**Tech Stack:** Go, ClickHouse, RabbitMQ, golang-migrate, Cobra/Viper, Prometheus

---

## Task 1: Create Project Structure

**Files:**
- Create: `bin-timeline-manager/go.mod`
- Create: `bin-timeline-manager/.gitignore`

**Step 1: Create the directory and go.mod**

```bash
mkdir -p bin-timeline-manager
```

Create `bin-timeline-manager/go.mod`:
```go
module monorepo/bin-timeline-manager

go 1.25.3

replace monorepo/bin-common-handler => ../bin-common-handler

require (
	github.com/ClickHouse/clickhouse-go/v2 v2.43.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang-migrate/migrate/v4 v4.18.1
	github.com/joonix/log v0.0.0-20251205082533-cd78070927ea
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.23.2
	github.com/sirupsen/logrus v1.9.4
	github.com/spf13/cobra v1.10.2
	github.com/spf13/viper v1.21.0
	go.uber.org/mock v0.6.0
	monorepo/bin-common-handler v0.0.0-00010101000000-000000000000
)
```

**Step 2: Create .gitignore**

Create `bin-timeline-manager/.gitignore`:
```
bin/
coverage.out
```

**Step 3: Commit**

```bash
git add bin-timeline-manager/go.mod bin-timeline-manager/.gitignore
git commit -m "NOJIRA-Add-timeline-manager-service

- bin-timeline-manager: Create initial project structure with go.mod"
```

---

## Task 2: Create Configuration Package

**Files:**
- Create: `bin-timeline-manager/internal/config/main.go`
- Create: `bin-timeline-manager/internal/config/main_test.go`

**Step 1: Create config/main.go**

Create `bin-timeline-manager/internal/config/main.go`:
```go
package config

import (
	"sync"

	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	globalConfig Config
	once         sync.Once
)

// Config holds process-wide configuration values.
type Config struct {
	RabbitMQAddress         string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	ClickHouseAddress       string
	ClickHouseDatabase      string
}

func Bootstrap(cmd *cobra.Command) error {
	initLog()
	if errBind := bindConfig(cmd); errBind != nil {
		return errors.Wrapf(errBind, "could not bind config")
	}
	return nil
}

func bindConfig(cmd *cobra.Command) error {
	viper.AutomaticEnv()
	f := cmd.PersistentFlags()

	f.String("rabbitmq_address", "", "RabbitMQ server address")
	f.String("prometheus_endpoint", "", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", "", "Prometheus listen address")
	f.String("clickhouse_address", "", "ClickHouse server address")
	f.String("clickhouse_database", "default", "ClickHouse database name")

	bindings := map[string]string{
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"clickhouse_address":        "CLICKHOUSE_ADDRESS",
		"clickhouse_database":       "CLICKHOUSE_DATABASE",
	}

	for flagKey, envKey := range bindings {
		if errBind := viper.BindPFlag(flagKey, f.Lookup(flagKey)); errBind != nil {
			return errors.Wrapf(errBind, "could not bind flag. key: %s", flagKey)
		}
		if errBind := viper.BindEnv(flagKey, envKey); errBind != nil {
			return errors.Wrapf(errBind, "could not bind the env. key: %s", envKey)
		}
	}

	return nil
}

func Get() *Config {
	return &globalConfig
}

func LoadGlobalConfig() {
	once.Do(func() {
		globalConfig = Config{
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			ClickHouseAddress:       viper.GetString("clickhouse_address"),
			ClickHouseDatabase:      viper.GetString("clickhouse_database"),
		}
		logrus.Debug("Configuration has been loaded and locked.")
	})
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}
```

**Step 2: Create config test**

Create `bin-timeline-manager/internal/config/main_test.go`:
```go
package config

import (
	"testing"
)

func TestGet(t *testing.T) {
	cfg := Get()
	if cfg == nil {
		t.Error("Get() returned nil")
	}
}
```

**Step 3: Run test to verify**

```bash
cd bin-timeline-manager && go mod tidy && go test ./internal/config/...
```

Expected: PASS

**Step 4: Commit**

```bash
git add bin-timeline-manager/internal/
git commit -m "NOJIRA-Add-timeline-manager-service

- bin-timeline-manager: Add configuration package with Cobra/Viper"
```

---

## Task 3: Create Event Models

**Files:**
- Create: `bin-timeline-manager/models/event/event.go`
- Create: `bin-timeline-manager/models/event/request.go`

**Step 1: Create event.go**

Create `bin-timeline-manager/models/event/event.go`:
```go
package event

import (
	"encoding/json"

	commonoutline "monorepo/bin-common-handler/models/outline"
)

// Event represents a single event from ClickHouse.
type Event struct {
	Timestamp string                    `json:"timestamp"`
	EventType string                    `json:"event_type"`
	Publisher commonoutline.ServiceName `json:"publisher"`
	DataType  string                    `json:"data_type"`
	Data      json.RawMessage           `json:"data"`
}

// EventListResponse represents the response for event list queries.
type EventListResponse struct {
	Result        []*Event `json:"result"`
	NextPageToken string   `json:"next_page_token,omitempty"`
}
```

**Step 2: Create request.go**

Create `bin-timeline-manager/models/event/request.go`:
```go
package event

import (
	"github.com/gofrs/uuid"

	commonoutline "monorepo/bin-common-handler/models/outline"
)

// EventListRequest represents the request for listing events.
type EventListRequest struct {
	Publisher commonoutline.ServiceName `json:"publisher"`
	ID        uuid.UUID                 `json:"id"`
	Events    []string                  `json:"events"`

	// Pagination
	PageToken string `json:"page_token,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}

// Default and max page sizes
const (
	DefaultPageSize = 100
	MaxPageSize     = 1000
)
```

**Step 3: Verify compilation**

```bash
cd bin-timeline-manager && go mod tidy && go build ./models/...
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add bin-timeline-manager/models/
git commit -m "NOJIRA-Add-timeline-manager-service

- bin-timeline-manager: Add event models for request/response"
```

---

## Task 4: Create Database Handler

**Files:**
- Create: `bin-timeline-manager/pkg/dbhandler/main.go`
- Create: `bin-timeline-manager/pkg/dbhandler/mock_main.go`
- Create: `bin-timeline-manager/pkg/dbhandler/event.go`
- Create: `bin-timeline-manager/pkg/dbhandler/event_test.go`

**Step 1: Create main.go with interface**

Create `bin-timeline-manager/pkg/dbhandler/main.go`:
```go
package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-timeline-manager/models/event"
)

const clickhouseRetryInterval = 30 * time.Second

// DBHandler interface for database operations.
type DBHandler interface {
	EventList(ctx context.Context, publisher commonoutline.ServiceName, resourceID uuid.UUID, events []string, pageToken string, pageSize int) ([]*event.Event, error)
}

type dbHandler struct {
	address  string
	database string
	conn     clickhouse.Conn
}

// NewHandler creates a new DBHandler.
func NewHandler(address, database string) DBHandler {
	h := &dbHandler{
		address:  address,
		database: database,
	}

	go h.connectionLoop()

	return h
}

func (h *dbHandler) connectionLoop() {
	log := logrus.WithFields(logrus.Fields{
		"func":    "connectionLoop",
		"address": h.address,
	})

	for {
		if h.conn != nil {
			time.Sleep(clickhouseRetryInterval)
			continue
		}

		conn, err := clickhouse.Open(&clickhouse.Options{
			Addr: []string{h.address},
			Auth: clickhouse.Auth{
				Database: h.database,
			},
			Settings: clickhouse.Settings{
				"max_execution_time": 60,
			},
			DialTimeout: 5 * time.Second,
		})
		if err != nil {
			log.Errorf("Could not connect to ClickHouse, retrying in %v. err: %v", clickhouseRetryInterval, err)
			time.Sleep(clickhouseRetryInterval)
			continue
		}

		if err := conn.Ping(context.Background()); err != nil {
			log.Errorf("Could not ping ClickHouse, retrying in %v. err: %v", clickhouseRetryInterval, err)
			time.Sleep(clickhouseRetryInterval)
			continue
		}

		log.Info("Successfully connected to ClickHouse")
		h.conn = conn
		time.Sleep(clickhouseRetryInterval)
	}
}
```

**Step 2: Create event.go with query logic**

Create `bin-timeline-manager/pkg/dbhandler/event.go`:
```go
package dbhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-timeline-manager/models/event"
)

// EventList queries events from ClickHouse.
func (h *dbHandler) EventList(
	ctx context.Context,
	publisher commonoutline.ServiceName,
	resourceID uuid.UUID,
	events []string,
	pageToken string,
	pageSize int,
) ([]*event.Event, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventList",
		"publisher":   publisher,
		"resource_id": resourceID,
		"events":      events,
		"page_token":  pageToken,
		"page_size":   pageSize,
	})

	if h.conn == nil {
		return nil, errors.New("clickhouse connection not established")
	}

	query := `
		SELECT timestamp, event_type, publisher, data_type, data
		FROM events
		WHERE publisher = ?
		  AND resource_id = ?
	`
	args := []interface{}{string(publisher), resourceID.String()}

	// Add event type filters
	if len(events) > 0 {
		conditions := buildEventConditions(events)
		if conditions != "" {
			query += " AND (" + conditions + ")"
		}
	}

	// Pagination by timestamp
	if pageToken != "" {
		query += " AND timestamp < ?"
		args = append(args, pageToken)
	}

	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, pageSize)

	log.Debugf("Executing query: %s with args: %v", query, args)

	rows, err := h.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "could not query events")
	}
	defer rows.Close()

	var result []*event.Event
	for rows.Next() {
		var e event.Event
		var data string
		if err := rows.Scan(&e.Timestamp, &e.EventType, &e.Publisher, &e.DataType, &data); err != nil {
			return nil, errors.Wrap(err, "could not scan event row")
		}
		e.Data = []byte(data)
		result = append(result, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating rows")
	}

	return result, nil
}

// buildEventConditions converts event patterns to SQL conditions.
// Examples:
//   - "activeflow_created" -> "event_type = 'activeflow_created'"
//   - "activeflow_*" -> "event_type LIKE 'activeflow_%'"
//   - "*" -> "" (no filter)
func buildEventConditions(events []string) string {
	var conditions []string

	for _, e := range events {
		if e == "*" {
			// Wildcard all - no filter needed
			return ""
		}

		if strings.HasSuffix(e, "_*") {
			// Prefix wildcard: "activeflow_*" -> LIKE 'activeflow_%'
			prefix := strings.TrimSuffix(e, "*")
			conditions = append(conditions, fmt.Sprintf("event_type LIKE '%s%%'", prefix))
		} else {
			// Exact match
			conditions = append(conditions, fmt.Sprintf("event_type = '%s'", e))
		}
	}

	return strings.Join(conditions, " OR ")
}
```

**Step 3: Create event_test.go**

Create `bin-timeline-manager/pkg/dbhandler/event_test.go`:
```go
package dbhandler

import (
	"testing"
)

func TestBuildEventConditions(t *testing.T) {
	tests := []struct {
		name     string
		events   []string
		expected string
	}{
		{
			name:     "exact match",
			events:   []string{"activeflow_created"},
			expected: "event_type = 'activeflow_created'",
		},
		{
			name:     "wildcard prefix",
			events:   []string{"activeflow_*"},
			expected: "event_type LIKE 'activeflow_%'",
		},
		{
			name:     "multiple patterns",
			events:   []string{"activeflow_created", "flow_*"},
			expected: "event_type = 'activeflow_created' OR event_type LIKE 'flow_%'",
		},
		{
			name:     "wildcard all",
			events:   []string{"*"},
			expected: "",
		},
		{
			name:     "empty",
			events:   []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildEventConditions(tt.events)
			if result != tt.expected {
				t.Errorf("buildEventConditions(%v) = %q, want %q", tt.events, result, tt.expected)
			}
		})
	}
}
```

**Step 4: Run tests**

```bash
cd bin-timeline-manager && go generate ./... && go test ./pkg/dbhandler/...
```

Expected: PASS

**Step 5: Commit**

```bash
git add bin-timeline-manager/pkg/dbhandler/
git commit -m "NOJIRA-Add-timeline-manager-service

- bin-timeline-manager: Add ClickHouse database handler with event queries"
```

---

## Task 5: Create Event Handler

**Files:**
- Create: `bin-timeline-manager/pkg/eventhandler/main.go`
- Create: `bin-timeline-manager/pkg/eventhandler/mock_main.go`
- Create: `bin-timeline-manager/pkg/eventhandler/event.go`
- Create: `bin-timeline-manager/pkg/eventhandler/event_test.go`

**Step 1: Create main.go**

Create `bin-timeline-manager/pkg/eventhandler/main.go`:
```go
package eventhandler

//go:generate mockgen -package eventhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

// EventHandler interface for event operations.
type EventHandler interface {
	List(ctx context.Context, req *event.EventListRequest) (*event.EventListResponse, error)
}

type eventHandler struct {
	db dbhandler.DBHandler
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(db dbhandler.DBHandler) EventHandler {
	return &eventHandler{
		db: db,
	}
}
```

**Step 2: Create event.go**

Create `bin-timeline-manager/pkg/eventhandler/event.go`:
```go
package eventhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-timeline-manager/models/event"
)

// List returns events matching the request criteria.
func (h *eventHandler) List(ctx context.Context, req *event.EventListRequest) (*event.EventListResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "List",
		"publisher": req.Publisher,
		"id":        req.ID,
		"events":    req.Events,
	})

	// Validate request
	if req.Publisher == "" {
		return nil, errors.New("publisher is required")
	}
	if req.ID == uuid.Nil {
		return nil, errors.New("id is required")
	}
	if len(req.Events) == 0 {
		return nil, errors.New("events filter is required")
	}

	// Apply defaults
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = event.DefaultPageSize
	}
	if pageSize > event.MaxPageSize {
		pageSize = event.MaxPageSize
	}

	// Query database (request pageSize + 1 to determine if more results exist)
	events, err := h.db.EventList(ctx, req.Publisher, req.ID, req.Events, req.PageToken, pageSize+1)
	if err != nil {
		log.Errorf("Could not list events. err: %v", err)
		return nil, errors.Wrap(err, "could not list events")
	}

	// Build response with pagination
	response := &event.EventListResponse{
		Result: events,
	}

	// If we got more than pageSize, there are more results
	if len(events) > pageSize {
		response.Result = events[:pageSize]
		// Use timestamp of last returned event as next page token
		response.NextPageToken = events[pageSize-1].Timestamp
	}

	return response, nil
}
```

**Step 3: Create event_test.go**

Create `bin-timeline-manager/pkg/eventhandler/event_test.go`:
```go
package eventhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

func TestList_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	tests := []struct {
		name    string
		req     *event.EventListRequest
		wantErr bool
	}{
		{
			name: "missing publisher",
			req: &event.EventListRequest{
				ID:     uuid.Must(uuid.NewV4()),
				Events: []string{"activeflow_*"},
			},
			wantErr: true,
		},
		{
			name: "missing id",
			req: &event.EventListRequest{
				Publisher: commonoutline.ServiceNameFlowManager,
				Events:    []string{"activeflow_*"},
			},
			wantErr: true,
		},
		{
			name: "missing events",
			req: &event.EventListRequest{
				Publisher: commonoutline.ServiceNameFlowManager,
				ID:        uuid.Must(uuid.NewV4()),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.List(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestList_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	handler := NewEventHandler(mockDB)

	testID := uuid.Must(uuid.NewV4())
	req := &event.EventListRequest{
		Publisher: commonoutline.ServiceNameFlowManager,
		ID:        testID,
		Events:    []string{"activeflow_*"},
		PageSize:  10,
	}

	expectedEvents := []*event.Event{
		{Timestamp: "2024-01-15T10:30:00.123Z", EventType: "activeflow_created"},
		{Timestamp: "2024-01-15T10:29:00.123Z", EventType: "activeflow_started"},
	}

	mockDB.EXPECT().
		EventList(gomock.Any(), req.Publisher, testID, req.Events, "", 11).
		Return(expectedEvents, nil)

	result, err := handler.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result.Result) != 2 {
		t.Errorf("List() returned %d events, want 2", len(result.Result))
	}

	if result.NextPageToken != "" {
		t.Errorf("List() NextPageToken = %q, want empty", result.NextPageToken)
	}
}
```

**Step 4: Generate mocks and run tests**

```bash
cd bin-timeline-manager && go generate ./... && go test ./pkg/eventhandler/...
```

Expected: PASS

**Step 5: Commit**

```bash
git add bin-timeline-manager/pkg/eventhandler/
git commit -m "NOJIRA-Add-timeline-manager-service

- bin-timeline-manager: Add event handler with business logic"
```

---

## Task 6: Create Listen Handler

**Files:**
- Create: `bin-timeline-manager/pkg/listenhandler/main.go`
- Create: `bin-timeline-manager/pkg/listenhandler/mock_main.go`
- Create: `bin-timeline-manager/pkg/listenhandler/v1_events.go`

**Step 1: Create main.go**

Create `bin-timeline-manager/pkg/listenhandler/main.go`:
```go
package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
)

var (
	regV1Events = regexp.MustCompile("/v1/events$")
)

var (
	metricsNamespace = "timeline_manager"

	promReceivedRequestProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "receive_request_process_time",
			Help:      "Process time of received request",
			Buckets:   []float64{50, 100, 500, 1000, 3000},
		},
		[]string{"type", "method"},
	)
)

func init() {
	prometheus.MustRegister(promReceivedRequestProcessTime)
}

// ListenHandler interface
type ListenHandler interface {
	Run(queue string) error
}

type listenHandler struct {
	sockHandler  sockhandler.SockHandler
	eventHandler eventhandler.EventHandler
}

// NewListenHandler creates a new ListenHandler.
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	eventHandler eventhandler.EventHandler,
) ListenHandler {
	return &listenHandler{
		sockHandler:  sockHandler,
		eventHandler: eventHandler,
	}
}

func simpleResponse(code int) *sock.Response {
	return &sock.Response{StatusCode: code}
}

func (h *listenHandler) Run(queue string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "Run",
		"queue": queue,
	})
	log.Info("Creating rabbitmq queue for listen.")

	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, "timeline-manager", false, false, false, 10, h.processRequest); errConsume != nil {
			log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processRequest",
		"request": m,
	})

	ctx := context.Background()

	var requestType string
	var err error
	var response *sock.Response

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))
	}()

	switch {
	case regV1Events.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/events"
		response, err = h.v1EventsPost(ctx, m)

	default:
		log.Errorf("Could not find corresponded request handler. data: %s", m.Data)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}

	if err != nil {
		log.Errorf("Could not process the request correctly. data: %s", m.Data)
		response = simpleResponse(400)
		err = nil
	}

	return response, err
}
```

**Step 2: Create v1_events.go**

Create `bin-timeline-manager/pkg/listenhandler/v1_events.go`:
```go
package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-timeline-manager/models/event"
)

func (h *listenHandler) v1EventsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1EventsPost",
	})

	// Parse request
	var req event.EventListRequest
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	// Call handler
	result, err := h.eventHandler.List(ctx, &req)
	if err != nil {
		log.Errorf("Could not list events. err: %v", err)
		return simpleResponse(500), nil
	}

	// Marshal response
	data, err := json.Marshal(result)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal response")
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
```

**Step 3: Generate mocks and verify**

```bash
cd bin-timeline-manager && go generate ./... && go build ./pkg/listenhandler/...
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add bin-timeline-manager/pkg/listenhandler/
git commit -m "NOJIRA-Add-timeline-manager-service

- bin-timeline-manager: Add RabbitMQ RPC listen handler"
```

---

## Task 7: Create Main Service Entry Point

**Files:**
- Create: `bin-timeline-manager/cmd/timeline-manager/main.go`

**Step 1: Create main.go**

Create `bin-timeline-manager/cmd/timeline-manager/main.go`:
```go
package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-timeline-manager/internal/config"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
	"monorepo/bin-timeline-manager/pkg/listenhandler"
)

const serviceName = commonoutline.ServiceNameTimelineManager

var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "timeline-manager",
		Short: "Voipbin Timeline Manager Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.LoadGlobalConfig()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemon()
		},
	}

	if errBind := config.Bootstrap(rootCmd); errBind != nil {
		logrus.Fatalf("Failed to bootstrap config: %v", errBind)
	}

	if errExecute := rootCmd.Execute(); errExecute != nil {
		logrus.Errorf("Command execution failed: %v", errExecute)
		os.Exit(1)
	}
}

func runDaemon() error {
	initSignal()
	initProm(config.Get().PrometheusEndpoint, config.Get().PrometheusListenAddress)

	log := logrus.WithField("func", "runDaemon")
	log.WithField("config", config.Get()).Info("Starting timeline-manager...")

	if errStart := runServices(); errStart != nil {
		return errors.Wrapf(errStart, "could not start services")
	}

	<-chDone
	log.Info("Timeline-manager stopped safely.")
	return nil
}

func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-chSigs
		logrus.Infof("Received signal: %v", sig)
		chDone <- true
	}()
}

func initProm(endpoint, listen string) {
	if endpoint == "" || listen == "" {
		logrus.Debug("Prometheus metrics server disabled")
		return
	}

	http.Handle(endpoint, promhttp.Handler())
	go func() {
		logrus.Infof("Prometheus metrics server starting on %s%s", listen, endpoint)
		if err := http.ListenAndServe(listen, nil); err != nil {
			logrus.Errorf("Prometheus server error: %v", err)
		}
	}()
}

func runServices() error {
	db := dbhandler.NewHandler(config.Get().ClickHouseAddress, config.Get().ClickHouseDatabase)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	evtHandler := eventhandler.NewEventHandler(db)

	if errListen := runListen(sockHandler, evtHandler); errListen != nil {
		return errors.Wrapf(errListen, "failed to run service listen")
	}

	return nil
}

func runListen(sockListen sockhandler.SockHandler, evtHandler eventhandler.EventHandler) error {
	log := logrus.WithField("func", "runListen")

	listenHdlr := listenhandler.NewListenHandler(sockListen, evtHandler)

	if errRun := listenHdlr.Run(string(commonoutline.QueueNameTimelineRequest)); errRun != nil {
		log.Errorf("Error occurred in listen handler. err: %v", errRun)
	}

	return nil
}
```

**Step 2: Verify build**

```bash
cd bin-timeline-manager && go mod tidy && go build ./cmd/timeline-manager/...
```

Expected: Build succeeds (may fail due to missing ServiceNameTimelineManager - will fix in next task)

**Step 3: Commit**

```bash
git add bin-timeline-manager/cmd/timeline-manager/
git commit -m "NOJIRA-Add-timeline-manager-service

- bin-timeline-manager: Add main service entry point"
```

---

## Task 8: Add Service Constants to bin-common-handler

**Files:**
- Modify: `bin-common-handler/models/outline/service.go`
- Modify: `bin-common-handler/models/outline/queue.go`

**Step 1: Add ServiceNameTimelineManager**

Edit `bin-common-handler/models/outline/service.go` to add:
```go
ServiceNameTimelineManager ServiceName = "timeline-manager"
```

**Step 2: Add QueueNameTimelineRequest**

Edit `bin-common-handler/models/outline/queue.go` to add:
```go
QueueNameTimelineRequest QueueName = "bin-manager.timeline-manager.request"
```

**Step 3: Run verification**

```bash
cd bin-common-handler && go mod tidy && go test ./...
```

Expected: PASS

**Step 4: Commit**

```bash
git add bin-common-handler/models/outline/
git commit -m "NOJIRA-Add-timeline-manager-service

- bin-common-handler: Add timeline-manager service name and queue constants"
```

---

## Task 9: Create Migrations

**Files:**
- Create: `bin-timeline-manager/migrations/000001_create_events_table.up.sql`
- Create: `bin-timeline-manager/migrations/000001_create_events_table.down.sql`
- Create: `bin-timeline-manager/migrations/000002_add_resource_id_column.up.sql`
- Create: `bin-timeline-manager/migrations/000002_add_resource_id_column.down.sql`

**Step 1: Create migration files**

Create `bin-timeline-manager/migrations/000001_create_events_table.up.sql`:
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
TTL toDateTime(timestamp) + INTERVAL 1 YEAR;
```

Create `bin-timeline-manager/migrations/000001_create_events_table.down.sql`:
```sql
DROP TABLE IF EXISTS events;
```

Create `bin-timeline-manager/migrations/000002_add_resource_id_column.up.sql`:
```sql
ALTER TABLE events
ADD COLUMN resource_id String MATERIALIZED JSONExtractString(data, 'id');
```

Create `bin-timeline-manager/migrations/000002_add_resource_id_column.down.sql`:
```sql
ALTER TABLE events
DROP COLUMN resource_id;
```

**Step 2: Commit**

```bash
git add bin-timeline-manager/migrations/
git commit -m "NOJIRA-Add-timeline-manager-service

- bin-timeline-manager: Add ClickHouse migration files"
```

---

## Task 10: Create timeline-control CLI

**Files:**
- Create: `bin-timeline-manager/cmd/timeline-control/main.go`

**Step 1: Create CLI tool**

Create `bin-timeline-manager/cmd/timeline-control/main.go`:
```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-timeline-manager/internal/config"
	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
)

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "timeline-control",
		Short: "Voipbin Timeline Management CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if errBind := viper.BindPFlags(cmd.Flags()); errBind != nil {
				return errors.Wrap(errBind, "failed to bind flags")
			}
			config.LoadGlobalConfig()
			return nil
		},
	}

	if err := config.Bootstrap(cmdRoot); err != nil {
		cobra.CheckErr(errors.Wrap(err, "failed to bind infrastructure config"))
	}

	// Event commands
	cmdEvent := &cobra.Command{Use: "event", Short: "Event operations"}
	cmdEvent.AddCommand(cmdEventList())
	cmdRoot.AddCommand(cmdEvent)

	// Migrate commands
	cmdMigrate := &cobra.Command{Use: "migrate", Short: "Migration operations"}
	cmdMigrate.AddCommand(cmdMigrateUp())
	cmdMigrate.AddCommand(cmdMigrateDown())
	cmdMigrate.AddCommand(cmdMigrateStatus())
	cmdRoot.AddCommand(cmdMigrate)

	return cmdRoot
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSON")
	}
	fmt.Println(string(data))
	return nil
}

// Event commands

func cmdEventList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List events",
		RunE:  runEventList,
	}

	flags := cmd.Flags()
	flags.String("publisher", "", "Publisher service name (required)")
	flags.String("id", "", "Resource ID (required)")
	flags.String("events", "", "Event patterns comma-separated (required, e.g., 'activeflow_*,flow_created')")
	flags.Int("page-size", 100, "Page size")
	flags.String("page-token", "", "Page token for pagination")

	return cmd
}

func runEventList(cmd *cobra.Command, args []string) error {
	publisher := viper.GetString("publisher")
	if publisher == "" {
		return fmt.Errorf("publisher is required")
	}

	idStr := viper.GetString("id")
	if idStr == "" {
		return fmt.Errorf("id is required")
	}
	id := uuid.FromStringOrNil(idStr)
	if id == uuid.Nil {
		return fmt.Errorf("invalid id format")
	}

	eventsStr := viper.GetString("events")
	if eventsStr == "" {
		return fmt.Errorf("events is required")
	}
	events := strings.Split(eventsStr, ",")

	db := dbhandler.NewHandler(config.Get().ClickHouseAddress, config.Get().ClickHouseDatabase)
	handler := eventhandler.NewEventHandler(db)

	req := &event.EventListRequest{
		Publisher: commonoutline.ServiceName(publisher),
		ID:        id,
		Events:    events,
		PageSize:  viper.GetInt("page-size"),
		PageToken: viper.GetString("page-token"),
	}

	result, err := handler.List(context.Background(), req)
	if err != nil {
		return errors.Wrap(err, "failed to list events")
	}

	return printJSON(result)
}

// Migration commands

func getMigrate() (*migrate.Migrate, error) {
	addr := config.Get().ClickHouseAddress
	db := config.Get().ClickHouseDatabase
	if addr == "" {
		return nil, fmt.Errorf("CLICKHOUSE_ADDRESS is required")
	}

	dsn := fmt.Sprintf("clickhouse://%s/%s?x-multi-statement=true", addr, db)
	return migrate.New("file://migrations", dsn)
}

func cmdMigrateUp() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := getMigrate()
			if err != nil {
				return errors.Wrap(err, "failed to initialize migrate")
			}
			defer m.Close()

			if err := m.Up(); err != nil && err != migrate.ErrNoChange {
				return errors.Wrap(err, "migration failed")
			}

			fmt.Println("Migrations applied successfully")
			return nil
		},
	}
}

func cmdMigrateDown() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Rollback last migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := getMigrate()
			if err != nil {
				return errors.Wrap(err, "failed to initialize migrate")
			}
			defer m.Close()

			if err := m.Steps(-1); err != nil {
				return errors.Wrap(err, "rollback failed")
			}

			fmt.Println("Rollback completed successfully")
			return nil
		},
	}
}

func cmdMigrateStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := getMigrate()
			if err != nil {
				return errors.Wrap(err, "failed to initialize migrate")
			}
			defer m.Close()

			version, dirty, err := m.Version()
			if err != nil && err != migrate.ErrNilVersion {
				return errors.Wrap(err, "failed to get version")
			}

			fmt.Printf("Current version: %d\n", version)
			fmt.Printf("Dirty: %v\n", dirty)
			return nil
		},
	}
}
```

**Step 2: Verify build**

```bash
cd bin-timeline-manager && go mod tidy && go build ./cmd/timeline-control/...
```

Expected: Build succeeds

**Step 3: Commit**

```bash
git add bin-timeline-manager/cmd/timeline-control/
git commit -m "NOJIRA-Add-timeline-manager-service

- bin-timeline-manager: Add timeline-control CLI with event queries and migrations"
```

---

## Task 11: Create Dockerfile

**Files:**
- Create: `bin-timeline-manager/Dockerfile`

**Step 1: Create Dockerfile**

Create `bin-timeline-manager/Dockerfile`:
```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o timeline-manager ./cmd/timeline-manager
RUN go build -o timeline-control ./cmd/timeline-control

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/timeline-manager .
COPY --from=builder /app/timeline-control .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080
CMD ["./timeline-manager"]
```

**Step 2: Commit**

```bash
git add bin-timeline-manager/Dockerfile
git commit -m "NOJIRA-Add-timeline-manager-service

- bin-timeline-manager: Add Dockerfile"
```

---

## Task 12: Run Full Verification and Final Commit

**Step 1: Run verification for bin-timeline-manager**

```bash
cd bin-timeline-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All pass

**Step 2: Run verification for bin-common-handler**

```bash
cd bin-common-handler && \
go mod tidy && \
go test ./...
```

Expected: All pass

**Step 3: Verify all services still build (since bin-common-handler changed)**

```bash
for dir in bin-*/; do
  if [ -f "$dir/go.mod" ]; then
    echo "=== $dir ===" && \
    (cd "$dir" && go mod tidy && go build ./cmd/...) || echo "FAILED: $dir"
  fi
done
```

Expected: All services build

**Step 4: Final summary commit if needed**

If all tasks were committed individually, no final commit needed. Otherwise:

```bash
git status
# If there are uncommitted changes:
git add -A
git commit -m "NOJIRA-Add-timeline-manager-service

- bin-timeline-manager: Complete service implementation
- bin-common-handler: Add service constants"
```

---

## Summary

| Task | Description |
|------|-------------|
| 1 | Create project structure with go.mod |
| 2 | Create configuration package |
| 3 | Create event models |
| 4 | Create database handler with ClickHouse queries |
| 5 | Create event handler with business logic |
| 6 | Create listen handler for RabbitMQ RPC |
| 7 | Create main service entry point |
| 8 | Add service constants to bin-common-handler |
| 9 | Create migration files |
| 10 | Create timeline-control CLI |
| 11 | Create Dockerfile |
| 12 | Run full verification |

**Total files created:** ~20 files
**Services modified:** bin-common-handler (2 files)
