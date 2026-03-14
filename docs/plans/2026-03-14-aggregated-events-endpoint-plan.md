# Aggregated Events Endpoint Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `GET /v1/aggregated-events` endpoint that returns all events related to a single activeflow execution in chronological order.

**Architecture:** New ClickHouse materialized column (`activeflow_id`) + new RPC handler in `bin-timeline-manager` + new HTTP endpoint in `bin-api-manager`. Zero changes to existing event publishing services.

**Tech Stack:** Go, ClickHouse (materialized columns), RabbitMQ RPC, OpenAPI 3.0 + oapi-codegen

**Design doc:** `docs/plans/2026-03-14-aggregated-events-endpoint-design.md`

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/`

---

### Task 1: ClickHouse Migration — Add `activeflow_id` materialized column

**Files:**
- Create: `bin-timeline-manager/migrations/000003_add_activeflow_id_column.up.sql`
- Create: `bin-timeline-manager/migrations/000003_add_activeflow_id_column.down.sql`

**Step 1: Create the up migration**

```sql
ALTER TABLE events
ADD COLUMN IF NOT EXISTS activeflow_id String
MATERIALIZED if(event_type LIKE 'activeflow_%', JSONExtractString(data, 'id'), JSONExtractString(data, 'activeflow_id'));
```

**Step 2: Create the down migration**

```sql
ALTER TABLE events
DROP COLUMN IF EXISTS activeflow_id;
```

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint
git add bin-timeline-manager/migrations/000003_add_activeflow_id_column.up.sql bin-timeline-manager/migrations/000003_add_activeflow_id_column.down.sql
git commit -m "NOJIRA-Add-aggregated-events-endpoint

- bin-timeline-manager: Add activeflow_id materialized column migration"
```

---

### Task 2: Timeline Manager — Add request/response models for aggregated events

**Files:**
- Modify: `bin-timeline-manager/models/event/request.go` — add `AggregatedEventListRequest`
- Modify: `bin-timeline-manager/models/event/response.go` — add `AggregatedEventListResponse`

**Step 1: Add AggregatedEventListRequest to `models/event/request.go`**

Add after the existing `EventListRequest`:

```go
// AggregatedEventListRequest represents the request for listing aggregated events
// across all resource types for a given activeflow.
type AggregatedEventListRequest struct {
	ActiveflowID uuid.UUID `json:"activeflow_id"`

	// Pagination
	PageToken string `json:"page_token,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}
```

**Step 2: Add AggregatedEventListResponse to `models/event/response.go`**

Add after the existing `EventListResponse`:

```go
// AggregatedEventListResponse represents the response for aggregated event list queries.
type AggregatedEventListResponse struct {
	Result        []*Event `json:"result"`
	NextPageToken string   `json:"next_page_token,omitempty"`
}
```

**Step 3: Commit**

```bash
git add bin-timeline-manager/models/event/request.go bin-timeline-manager/models/event/response.go
git commit -m "NOJIRA-Add-aggregated-events-endpoint

- bin-timeline-manager: Add request/response models for aggregated events RPC"
```

---

### Task 3: Timeline Manager — Add DB query for aggregated events

**Files:**
- Modify: `bin-timeline-manager/pkg/dbhandler/main.go` — add `AggregatedEventList` to interface
- Modify: `bin-timeline-manager/pkg/dbhandler/event.go` — add query implementation

**Step 1: Add to DBHandler interface in `pkg/dbhandler/main.go`**

Add to the interface:

```go
AggregatedEventList(ctx context.Context, activeflowID string, pageToken string, pageSize int) ([]*event.Event, error)
```

**Step 2: Add query implementation in `pkg/dbhandler/event.go`**

Add after the existing `EventList` function. Follow the same pattern:

```go
// buildAggregatedEventQuery constructs the SQL query for listing aggregated events by activeflow_id.
func buildAggregatedEventQuery(activeflowID string, pageToken string, pageSize int) (string, []interface{}) {
	query := `
		SELECT timestamp, event_type, publisher, data_type, data
		FROM events
		WHERE activeflow_id = ?
	`
	args := []interface{}{activeflowID}

	// Pagination by timestamp
	if pageToken != "" {
		query += " AND timestamp < ?"
		args = append(args, pageToken)
	}

	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, pageSize)

	return query, args
}

// AggregatedEventList queries events from ClickHouse filtered by activeflow_id.
func (h *dbHandler) AggregatedEventList(
	ctx context.Context,
	activeflowID string,
	pageToken string,
	pageSize int,
) ([]*event.Event, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "AggregatedEventList",
		"activeflow_id":  activeflowID,
		"page_token":     pageToken,
		"page_size":      pageSize,
	})

	if h.conn == nil {
		return nil, errors.New("clickhouse connection not established")
	}

	query, args := buildAggregatedEventQuery(activeflowID, pageToken, pageSize)

	log.Debugf("Executing query: %s with args: %v", query, args)

	rows, err := h.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "could not query aggregated events")
	}
	defer func() { _ = rows.Close() }()

	result := []*event.Event{}
	for rows.Next() {
		var e event.Event
		var publisherStr, data string
		if err := rows.Scan(&e.Timestamp, &e.EventType, &publisherStr, &e.DataType, &data); err != nil {
			return nil, errors.Wrap(err, "could not scan event row")
		}
		e.Publisher = commonoutline.ServiceName(publisherStr)
		e.Data = []byte(data)
		result = append(result, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating rows")
	}

	return result, nil
}
```

**Step 3: Regenerate mocks**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/bin-timeline-manager
go generate ./pkg/dbhandler/...
```

**Step 4: Write unit test for `buildAggregatedEventQuery` in `pkg/dbhandler/event_test.go`**

Add test cases that verify:
- Basic query with just activeflowID
- Query with pageToken (adds `AND timestamp < ?`)
- Query with pageSize limit

**Step 5: Run tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/bin-timeline-manager
go test ./pkg/dbhandler/...
```

**Step 6: Commit**

```bash
git add bin-timeline-manager/pkg/dbhandler/
git commit -m "NOJIRA-Add-aggregated-events-endpoint

- bin-timeline-manager: Add AggregatedEventList DB query with activeflow_id filter"
```

---

### Task 4: Timeline Manager — Add request DTO and event handler business logic

**Files:**
- Create: `bin-timeline-manager/pkg/listenhandler/models/request/aggregated_event.go`
- Create: `bin-timeline-manager/pkg/listenhandler/models/response/aggregated_event.go`
- Modify: `bin-timeline-manager/pkg/eventhandler/main.go` — add `AggregatedList` to interface
- Modify: `bin-timeline-manager/pkg/eventhandler/event.go` — add business logic

**Step 1: Create request DTO `pkg/listenhandler/models/request/aggregated_event.go`**

```go
package request

import (
	"github.com/gofrs/uuid"
)

// V1DataAggregatedEventsPost represents the request for listing aggregated events.
type V1DataAggregatedEventsPost struct {
	ActiveflowID uuid.UUID `json:"activeflow_id"`

	// Pagination
	PageToken string `json:"page_token,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}
```

**Step 2: Create response DTO `pkg/listenhandler/models/response/aggregated_event.go`**

```go
package response

import (
	"monorepo/bin-timeline-manager/models/event"
)

// V1DataAggregatedEventsPost represents the response for aggregated event list queries.
type V1DataAggregatedEventsPost struct {
	Result        []*event.Event `json:"result"`
	NextPageToken string         `json:"next_page_token,omitempty"`
}
```

**Step 3: Add `AggregatedList` to EventHandler interface in `pkg/eventhandler/main.go`**

```go
type EventHandler interface {
	List(ctx context.Context, req *request.V1DataEventsPost) (*response.V1DataEventsPost, error)
	AggregatedList(ctx context.Context, req *request.V1DataAggregatedEventsPost) (*response.V1DataAggregatedEventsPost, error)
}
```

**Step 4: Implement `AggregatedList` in `pkg/eventhandler/event.go`**

Add after the existing `List` method. Follow the same pattern (validate, apply defaults, query DB with pageSize+1, build response):

```go
// AggregatedList returns all events matching the given activeflow ID.
func (h *eventHandler) AggregatedList(ctx context.Context, req *request.V1DataAggregatedEventsPost) (*response.V1DataAggregatedEventsPost, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "AggregatedList",
		"activeflow_id": req.ActiveflowID,
	})

	// Validate request
	if req.ActiveflowID == uuid.Nil {
		return nil, errors.New("activeflow_id is required")
	}

	// Apply defaults
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = request.DefaultPageSize
	}
	if pageSize > request.MaxPageSize {
		pageSize = request.MaxPageSize
	}

	// Query database (request pageSize + 1 to determine if more results exist)
	events, err := h.db.AggregatedEventList(ctx, req.ActiveflowID.String(), req.PageToken, pageSize+1)
	if err != nil {
		log.Errorf("Could not list aggregated events. err: %v", err)
		return nil, errors.Wrap(err, "could not list aggregated events")
	}

	// Build response with pagination
	res := &response.V1DataAggregatedEventsPost{
		Result: events,
	}

	// If we got more than pageSize, there are more results
	if len(events) > pageSize {
		res.Result = events[:pageSize]
		res.NextPageToken = events[pageSize-1].Timestamp.Format(commonutil.ISO8601Layout)
	}

	return res, nil
}
```

**Step 5: Regenerate mocks**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/bin-timeline-manager
go generate ./pkg/eventhandler/...
```

**Step 6: Write unit test in `pkg/eventhandler/event_test.go`**

Test cases for `AggregatedList`:
- Valid request returns events
- Missing activeflow_id returns error
- Pagination: when DB returns pageSize+1 results, NextPageToken is set
- Pagination: when DB returns <= pageSize results, NextPageToken is empty
- PageSize defaults and max capping

**Step 7: Run tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/bin-timeline-manager
go test ./pkg/eventhandler/...
```

**Step 8: Commit**

```bash
git add bin-timeline-manager/pkg/listenhandler/models/ bin-timeline-manager/pkg/eventhandler/
git commit -m "NOJIRA-Add-aggregated-events-endpoint

- bin-timeline-manager: Add AggregatedList event handler with request/response DTOs"
```

---

### Task 5: Timeline Manager — Add RPC listen handler

**Files:**
- Modify: `bin-timeline-manager/pkg/listenhandler/main.go` — add regex route + switch case
- Create: `bin-timeline-manager/pkg/listenhandler/v1_aggregated_events.go` — RPC handler

**Step 1: Add regex in `pkg/listenhandler/main.go`**

Add to the `var` block at top:

```go
regV1AggregatedEvents = regexp.MustCompile("/v1/aggregated-events$")
```

Add to `processRequest` switch block (before `default:`):

```go
case regV1AggregatedEvents.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
	requestType = "/aggregated-events"
	response, err = h.v1AggregatedEventsPost(ctx, m)
```

**Step 2: Create `pkg/listenhandler/v1_aggregated_events.go`**

Follow the exact pattern of `v1_events.go`:

```go
package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/request"
)

func (h *listenHandler) v1AggregatedEventsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1AggregatedEventsPost",
	})

	// Parse request
	var req request.V1DataAggregatedEventsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	// Call handler
	result, err := h.eventHandler.AggregatedList(ctx, &req)
	if err != nil {
		log.Errorf("Could not list aggregated events. err: %v", err)
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

**Step 3: Regenerate mocks**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/bin-timeline-manager
go generate ./pkg/listenhandler/...
```

**Step 4: Run full verification for timeline-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/bin-timeline-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
git add bin-timeline-manager/
git commit -m "NOJIRA-Add-aggregated-events-endpoint

- bin-timeline-manager: Add v1AggregatedEventsPost RPC listen handler"
```

---

### Task 6: Common Handler — Add RPC client for aggregated events

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go` — add method to RequestHandler interface
- Create: `bin-common-handler/pkg/requesthandler/timeline_aggregated_events.go` — RPC client

**Step 1: Add to RequestHandler interface in `main.go`**

Add near the existing `TimelineV1EventList` line (~1304):

```go
TimelineV1AggregatedEventList(ctx context.Context, req *tmevent.AggregatedEventListRequest) (*tmevent.AggregatedEventListResponse, error)
```

**Step 2: Create `timeline_aggregated_events.go`**

Follow the exact pattern of `timeline_events.go`:

```go
package requesthandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	tmevent "monorepo/bin-timeline-manager/models/event"
)

// TimelineV1AggregatedEventList sends a request to timeline-manager
// to list aggregated events for a given activeflow.
func (r *requestHandler) TimelineV1AggregatedEventList(ctx context.Context, req *tmevent.AggregatedEventListRequest) (*tmevent.AggregatedEventListResponse, error) {
	uri := "/v1/aggregated-events"

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodPost, "timeline/aggregated-events", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tmevent.AggregatedEventListResponse
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**Step 3: Regenerate mocks and run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/bin-common-handler
go generate ./pkg/requesthandler/...
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-Add-aggregated-events-endpoint

- bin-common-handler: Add TimelineV1AggregatedEventList RPC client method"
```

---

### Task 7: OpenAPI — Define aggregated-events endpoint

**Files:**
- Create: `bin-openapi-manager/openapi/paths/timelines/aggregated_events.yaml`
- Modify: `bin-openapi-manager/openapi/openapi.yaml` — add path reference

**Step 1: Create `paths/timelines/aggregated_events.yaml`**

```yaml
get:
  summary: Get aggregated timeline events
  description: |
    Returns all timeline events associated with a single activeflow execution.
    Query by activeflow_id directly, or by call_id (which resolves to the call's activeflow).
    Exactly one of activeflow_id or call_id must be provided.
  tags:
    - Timeline
  parameters:
    - name: activeflow_id
      in: query
      required: false
      description: "The UUID of the activeflow. Obtained from the `id` field of `GET /activeflows` or from the `activeflow_id` field of a call."
      schema:
        type: string
        format: uuid
        example: "550e8400-e29b-41d4-a716-446655440000"
    - name: call_id
      in: query
      required: false
      description: "The UUID of the call. Obtained from the `id` field of `GET /calls`. The call's activeflow_id will be used to query events."
      schema:
        type: string
        format: uuid
        example: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
    - $ref: '#/components/parameters/PageSize'
    - $ref: '#/components/parameters/PageToken'
  responses:
    '200':
      description: A list of aggregated timeline events sorted by timestamp descending.
      content:
        application/json:
          schema:
            allOf:
              - $ref: '#/components/schemas/CommonPagination'
              - type: object
                properties:
                  result:
                    type: array
                    items:
                      $ref: '#/components/schemas/TimelineManagerEvent'
    '400':
      description: "Invalid request. Either both or neither query params provided."
    '404':
      description: "Resource not found or call has no activeflow."
```

**Step 2: Add path reference in `openapi.yaml`**

Add near the existing timeline paths (~line 6911):

```yaml
  /aggregated-events:
    $ref: './paths/timelines/aggregated_events.yaml'
```

**Step 3: Regenerate OpenAPI types and api-manager server code**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./...

cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/bin-api-manager
go mod tidy && go mod vendor && go generate ./...
```

**Step 4: Verify the generated code includes the new endpoint**

Check that `bin-api-manager/gens/openapi_server/gen.go` contains a new function signature like `GetAggregatedEvents`.

**Step 5: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/gens/
git commit -m "NOJIRA-Add-aggregated-events-endpoint

- bin-openapi-manager: Add GET /aggregated-events OpenAPI spec
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

### Task 8: API Manager — Add servicehandler for aggregated events

**Files:**
- Create: `bin-api-manager/pkg/servicehandler/aggregated_events.go`

**Step 1: Implement the service handler**

```go
package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	tmevent "monorepo/bin-timeline-manager/models/event"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// AggregatedEventList retrieves aggregated timeline events for an activeflow.
func (h *serviceHandler) AggregatedEventList(
	ctx context.Context,
	a *amagent.Agent,
	activeflowID uuid.UUID,
	callID uuid.UUID,
	pageSize int,
	pageToken string,
) ([]*TimelineEvent, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "AggregatedEventList",
		"customer_id":   a.CustomerID,
		"activeflow_id": activeflowID,
		"call_id":       callID,
	})

	// Validate: exactly one of activeflow_id or call_id must be provided
	if activeflowID == uuid.Nil && callID == uuid.Nil {
		log.Info("Neither activeflow_id nor call_id provided")
		return nil, "", fmt.Errorf("either activeflow_id or call_id is required")
	}
	if activeflowID != uuid.Nil && callID != uuid.Nil {
		log.Info("Both activeflow_id and call_id provided")
		return nil, "", fmt.Errorf("only one of activeflow_id or call_id is allowed")
	}

	// Resolve to activeflow_id
	var resolvedActiveflowID uuid.UUID
	if activeflowID != uuid.Nil {
		// Query by activeflow_id: validate ownership
		af, err := h.activeflowGet(ctx, activeflowID)
		if err != nil {
			log.Infof("Could not get activeflow: %v", err)
			return nil, "", fmt.Errorf("not found")
		}
		log.WithField("activeflow", af).Debugf("Retrieved activeflow info. activeflow_id: %s", af.ID)

		if !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			log.Info("Agent has no permission")
			return nil, "", fmt.Errorf("user has no permission")
		}
		resolvedActiveflowID = af.ID
	} else {
		// Query by call_id: get call, extract activeflow_id
		c, err := h.callGet(ctx, callID)
		if err != nil {
			log.Infof("Could not get call: %v", err)
			return nil, "", fmt.Errorf("not found")
		}
		log.WithField("call", c).Debugf("Retrieved call info. call_id: %s", c.ID)

		if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			log.Info("Agent has no permission")
			return nil, "", fmt.Errorf("user has no permission")
		}

		if c.ActiveflowID == uuid.Nil {
			log.Info("Call has no activeflow")
			return nil, "", fmt.Errorf("not found")
		}
		resolvedActiveflowID = c.ActiveflowID
	}

	// Query timeline-manager
	req := &tmevent.AggregatedEventListRequest{
		ActiveflowID: resolvedActiveflowID,
		PageSize:     pageSize,
		PageToken:    pageToken,
	}

	resp, err := h.reqHandler.TimelineV1AggregatedEventList(ctx, req)
	if err != nil {
		log.Errorf("Failed to query aggregated events: %v", err)
		return nil, "", fmt.Errorf("internal error")
	}

	// Return raw events as-is (data is already WebhookMessage JSON)
	result := make([]*TimelineEvent, 0, len(resp.Result))
	for _, ev := range resp.Result {
		result = append(result, &TimelineEvent{
			Timestamp: ev.Timestamp.Format("2006-01-02T15:04:05.000Z"),
			EventType: ev.EventType,
			Data:      ev.Data,
		})
	}

	return result, resp.NextPageToken, nil
}
```

**Step 2: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/aggregated_events.go
git commit -m "NOJIRA-Add-aggregated-events-endpoint

- bin-api-manager: Add AggregatedEventList servicehandler"
```

---

### Task 9: API Manager — Add HTTP server handler

**Files:**
- Create: `bin-api-manager/server/aggregated_events.go`

**Step 1: Implement the server handler**

Match the generated function name from oapi-codegen (check `gens/openapi_server/gen.go` for exact name). It will likely be `GetAggregatedEvents`. Follow the pattern from `server/timelines.go`:

```go
package server

import (
	"net/http"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetAggregatedEvents(c *gin.Context, params openapi_server.GetAggregatedEventsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAggregatedEvents",
		"request_address": c.ClientIP(),
	})

	// Get agent from context
	tmp, exists := c.Get("agent")
	if !exists {
		log.Error("Could not find agent info")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "authentication required"})
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithField("customer_id", a.CustomerID)

	// Parse query params
	var activeflowID, callID uuid.UUID
	if params.ActiveflowId != nil {
		parsed, err := uuid.FromString(params.ActiveflowId.String())
		if err != nil {
			log.Infof("Invalid activeflow_id: %v", err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "invalid activeflow_id"})
			return
		}
		activeflowID = parsed
	}
	if params.CallId != nil {
		parsed, err := uuid.FromString(params.CallId.String())
		if err != nil {
			log.Infof("Invalid call_id: %v", err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "invalid call_id"})
			return
		}
		callID = parsed
	}

	// Parse pagination params
	pageSize := 100
	if params.PageSize != nil {
		pageSize = *params.PageSize
		if pageSize <= 0 || pageSize > 1000 {
			pageSize = 100
		}
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	// Call servicehandler
	events, nextPageToken, err := h.serviceHandler.AggregatedEventList(c.Request.Context(), &a, activeflowID, callID, pageSize, pageToken)
	if err != nil {
		log.Infof("Failed to get aggregated events: %v", err)
		if err.Error() == "user has no permission" || err.Error() == "not found" {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "resource not found"})
			return
		}
		if err.Error() == "either activeflow_id or call_id is required" || err.Error() == "only one of activeflow_id or call_id is allowed" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "internal error"})
		return
	}

	// Build response
	res := struct {
		Result        interface{} `json:"result"`
		NextPageToken string      `json:"next_page_token,omitempty"`
	}{
		Result:        events,
		NextPageToken: nextPageToken,
	}

	c.JSON(http.StatusOK, res)
}
```

**Note:** The exact function signature and param type names will be determined by oapi-codegen output. Check `bin-api-manager/gens/openapi_server/gen.go` after Task 7 and adjust accordingly.

**Step 2: Commit**

```bash
git add bin-api-manager/server/aggregated_events.go
git commit -m "NOJIRA-Add-aggregated-events-endpoint

- bin-api-manager: Add GetAggregatedEvents HTTP server handler"
```

---

### Task 10: Full verification and final commit

**Step 1: Run full verification for all changed services**

```bash
# Timeline manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/bin-timeline-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# Common handler
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# OpenAPI manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# API manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Fix any issues**

Address any lint errors, test failures, or compilation issues. Common things to check:
- Generated mock files are up to date
- Import paths are correct
- The oapi-codegen function name matches what's in `gen.go`
- No unused imports

**Step 3: Push and create PR**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aggregated-events-endpoint
git push -u origin NOJIRA-Add-aggregated-events-endpoint
```

Then create PR following the monorepo PR format.
