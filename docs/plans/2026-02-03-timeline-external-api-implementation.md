# Timeline External API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a public API endpoint for customers to retrieve timeline events for their resources in WebhookMessage format.

**Architecture:** Resource-centric URL pattern where customers query `/v1/timelines/{resource_type}/{resource_id}/events`. The API validates resource ownership via existing helper functions, then queries timeline-manager and converts event data to WebhookMessage format using existing converters.

**Tech Stack:** Go, OpenAPI 3.0, oapi-codegen, Gin HTTP framework, RabbitMQ RPC

---

## Task 1: Add OpenAPI Specification

**Files:**
- Create: `bin-openapi-manager/openapi/paths/timelines/resource_events.yaml`
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Create the path file for timeline events**

Create `bin-openapi-manager/openapi/paths/timelines/resource_events.yaml`:

```yaml
get:
  summary: Get timeline events for a resource
  description: Returns timeline events for the specified resource in WebhookMessage format.
  tags:
    - Timeline
  parameters:
    - name: resource_type
      in: path
      required: true
      description: The type of resource (calls, conferences, flows, activeflows)
      schema:
        type: string
        enum:
          - calls
          - conferences
          - flows
          - activeflows
    - name: resource_id
      in: path
      required: true
      description: The UUID of the resource
      schema:
        type: string
        format: uuid
    - $ref: '#/components/parameters/PageSize'
    - $ref: '#/components/parameters/PageToken'
  responses:
    '200':
      description: A list of timeline events.
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
      description: Invalid request parameters
    '404':
      description: Resource not found
```

**Step 2: Add Timeline tag to openapi.yaml**

In `bin-openapi-manager/openapi/openapi.yaml`, add the Timeline tag after existing tags (around line 186):

```yaml
  - name: Timeline
    description: Operations related to timeline events
```

**Step 3: Add TimelineManagerEvent schema**

In `bin-openapi-manager/openapi/openapi.yaml`, add the schema in the components/schemas section:

```yaml
    TimelineManagerEvent:
      type: object
      description: A timeline event with WebhookMessage data
      properties:
        timestamp:
          type: string
          format: date-time
          description: When the event occurred
        event_type:
          type: string
          description: Type of event (e.g., call_created, conference_started)
        data:
          type: object
          description: Event data in WebhookMessage format
```

**Step 4: Add path reference to openapi.yaml**

In `bin-openapi-manager/openapi/openapi.yaml`, add the path in the paths section:

```yaml
  /timelines/{resource_type}/{resource_id}/events:
    $ref: './paths/timelines/resource_events.yaml'
```

**Step 5: Run verification**

Run: `cd bin-openapi-manager && go generate ./...`
Expected: No errors, `gens/models/gen.go` regenerated

**Step 6: Commit**

```bash
git add bin-openapi-manager/openapi/paths/timelines/resource_events.yaml
git add bin-openapi-manager/openapi/openapi.yaml
git commit -m "$(cat <<'EOF'
Add timeline events API specification

- bin-openapi-manager: Add GET /timelines/{resource_type}/{resource_id}/events endpoint
- bin-openapi-manager: Add Timeline tag and TimelineManagerEvent schema
EOF
)"
```

---

## Task 2: Regenerate bin-api-manager OpenAPI Server

**Files:**
- Modify: `bin-api-manager/gens/openapi_server/gen.go` (auto-generated)

**Step 1: Update vendor dependencies**

Run: `cd bin-api-manager && go mod tidy && go mod vendor`
Expected: Vendor updated with new openapi-manager types

**Step 2: Regenerate server code**

Run: `cd bin-api-manager && go generate ./...`
Expected: `gens/openapi_server/gen.go` updated with new `GetTimelinesResourceTypeResourceIdEvents` interface method

**Step 3: Verify the generated interface**

Run: `grep -A 5 "GetTimelinesResourceTypeResourceIdEvents" bin-api-manager/gens/openapi_server/gen.go`
Expected: Interface method exists with parameters for resource_type, resource_id, and pagination params

**Step 4: Commit**

```bash
git add bin-api-manager/go.mod bin-api-manager/go.sum
git add bin-api-manager/vendor/
git add bin-api-manager/gens/
git commit -m "$(cat <<'EOF'
Regenerate API server with timeline endpoint

- bin-api-manager: Update vendor dependencies
- bin-api-manager: Regenerate OpenAPI server code
EOF
)"
```

---

## Task 3: Add ServiceHandler Timeline Method

**Files:**
- Create: `bin-api-manager/pkg/servicehandler/timeline.go`
- Modify: `bin-api-manager/pkg/servicehandler/main.go`

**Step 1: Create timeline.go with helper functions and main method**

Create `bin-api-manager/pkg/servicehandler/timeline.go`:

```go
package servicehandler

import (
	"context"
	"encoding/json"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcall "monorepo/bin-call-manager/models/call"
	commonoutline "monorepo/bin-common-handler/models/outline"
	cfconference "monorepo/bin-conference-manager/models/conference"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmflow "monorepo/bin-flow-manager/models/flow"
	tmevent "monorepo/bin-timeline-manager/models/event"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// TimelineEvent represents a timeline event with converted WebhookMessage data.
type TimelineEvent struct {
	Timestamp string      `json:"timestamp"`
	EventType string      `json:"event_type"`
	Data      interface{} `json:"data"`
}

// resourceTypeConfig maps resource types to their configuration.
type resourceTypeConfig struct {
	ServiceName  commonoutline.ServiceName
	EventPattern string
}

// resourceTypeConfigs defines the mapping from resource_type to ServiceName and event pattern.
var resourceTypeConfigs = map[string]resourceTypeConfig{
	"calls":       {ServiceName: commonoutline.ServiceNameCallManager, EventPattern: "call_*"},
	"conferences": {ServiceName: commonoutline.ServiceNameConferenceManager, EventPattern: "conference_*"},
	"flows":       {ServiceName: commonoutline.ServiceNameFlowManager, EventPattern: "flow_*"},
	"activeflows": {ServiceName: commonoutline.ServiceNameFlowManager, EventPattern: "activeflow_*"},
}

// TimelineEventList retrieves timeline events for a resource.
func (h *serviceHandler) TimelineEventList(
	ctx context.Context,
	a *amagent.Agent,
	resourceType string,
	resourceID uuid.UUID,
	pageSize int,
	pageToken string,
) ([]*TimelineEvent, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "TimelineEventList",
		"customer_id":   a.CustomerID,
		"username":      a.Username,
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})

	// Validate resource type
	config, ok := resourceTypeConfigs[resourceType]
	if !ok {
		log.Info("Invalid resource type")
		return nil, "", fmt.Errorf("invalid resource type")
	}

	// Validate resource ownership
	customerID, err := h.validateResourceOwnership(ctx, resourceType, resourceID)
	if err != nil {
		log.Infof("Resource validation failed: %v", err)
		return nil, "", err
	}

	// Check permission
	if !h.hasPermission(ctx, a, customerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("Agent has no permission")
		return nil, "", fmt.Errorf("user has no permission")
	}

	// Query timeline events
	req := &tmevent.EventListRequest{
		Publisher:  config.ServiceName,
		ResourceID: resourceID,
		Events:     []string{config.EventPattern},
		PageSize:   pageSize,
		PageToken:  pageToken,
	}

	resp, err := h.reqHandler.TimelineV1EventList(ctx, req)
	if err != nil {
		log.Errorf("Failed to query timeline: %v", err)
		return nil, "", fmt.Errorf("internal error")
	}

	// Convert events to WebhookMessage format
	result := make([]*TimelineEvent, 0, len(resp.Result))
	for _, event := range resp.Result {
		converted, err := h.convertEventToWebhookMessage(resourceType, event)
		if err != nil {
			log.Warnf("Failed to convert event: %v", err)
			continue // Skip failed conversions
		}
		result = append(result, converted)
	}

	return result, resp.NextPageToken, nil
}

// validateResourceOwnership validates the resource exists and returns its customer ID.
func (h *serviceHandler) validateResourceOwnership(ctx context.Context, resourceType string, resourceID uuid.UUID) (uuid.UUID, error) {
	switch resourceType {
	case "calls":
		c, err := h.callGet(ctx, resourceID)
		if err != nil {
			return uuid.Nil, err
		}
		return c.CustomerID, nil

	case "conferences":
		c, err := h.conferenceGet(ctx, resourceID)
		if err != nil {
			return uuid.Nil, err
		}
		return c.CustomerID, nil

	case "flows":
		f, err := h.flowGet(ctx, resourceID)
		if err != nil {
			return uuid.Nil, err
		}
		return f.CustomerID, nil

	case "activeflows":
		af, err := h.activeflowGet(ctx, resourceID)
		if err != nil {
			return uuid.Nil, err
		}
		return af.CustomerID, nil

	default:
		return uuid.Nil, fmt.Errorf("unsupported resource type")
	}
}

// convertEventToWebhookMessage converts a timeline event to WebhookMessage format.
func (h *serviceHandler) convertEventToWebhookMessage(resourceType string, event *tmevent.Event) (*TimelineEvent, error) {
	var data interface{}

	switch resourceType {
	case "calls":
		var call cmcall.Call
		if err := json.Unmarshal(event.Data, &call); err != nil {
			return nil, err
		}
		data = call.ConvertWebhookMessage()

	case "conferences":
		var conf cfconference.Conference
		if err := json.Unmarshal(event.Data, &conf); err != nil {
			return nil, err
		}
		data = conf.ConvertWebhookMessage()

	case "flows":
		var flow fmflow.Flow
		if err := json.Unmarshal(event.Data, &flow); err != nil {
			return nil, err
		}
		data = flow.ConvertWebhookMessage()

	case "activeflows":
		var af fmactiveflow.Activeflow
		if err := json.Unmarshal(event.Data, &af); err != nil {
			return nil, err
		}
		data = af.ConvertWebhookMessage()

	default:
		return nil, fmt.Errorf("unsupported resource type for conversion")
	}

	return &TimelineEvent{
		Timestamp: event.Timestamp.Format("2006-01-02T15:04:05.000Z"),
		EventType: event.EventType,
		Data:      data,
	}, nil
}
```

**Step 2: Add interface method to main.go**

In `bin-api-manager/pkg/servicehandler/main.go`, add to the `ServiceHandler` interface:

```go
	// timeline
	TimelineEventList(ctx context.Context, a *amagent.Agent, resourceType string, resourceID uuid.UUID, pageSize int, pageToken string) ([]*TimelineEvent, string, error)
```

**Step 3: Add required imports to main.go**

Ensure `tmevent "monorepo/bin-timeline-manager/models/event"` is imported in main.go if used.

**Step 4: Regenerate mocks**

Run: `cd bin-api-manager && go generate ./pkg/servicehandler/...`
Expected: `mock_main.go` updated with new method

**Step 5: Run tests**

Run: `cd bin-api-manager && go test ./pkg/servicehandler/...`
Expected: All tests pass

**Step 6: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/timeline.go
git add bin-api-manager/pkg/servicehandler/main.go
git add bin-api-manager/pkg/servicehandler/mock_main.go
git commit -m "$(cat <<'EOF'
Add timeline servicehandler methods

- bin-api-manager: Add TimelineEventList method with resource validation
- bin-api-manager: Add WebhookMessage conversion for call, conference, flow, activeflow
EOF
)"
```

---

## Task 4: Add Server Endpoint Handler

**Files:**
- Create: `bin-api-manager/server/timelines.go`

**Step 1: Create timelines.go**

Create `bin-api-manager/server/timelines.go`:

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

func (h *server) GetTimelinesResourceTypeResourceIdEvents(c *gin.Context, resourceType string, resourceId string, params openapi_server.GetTimelinesResourceTypeResourceIdEventsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTimelinesResourceTypeResourceIdEvents",
		"request_address": c.ClientIP(),
		"resource_type":   resourceType,
		"resource_id":     resourceId,
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

	// Validate resource_type
	validTypes := map[string]bool{"calls": true, "conferences": true, "flows": true, "activeflows": true}
	if !validTypes[resourceType] {
		log.Info("Invalid resource type")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "invalid resource type"})
		return
	}

	// Parse resource_id
	resourceUUID := uuid.FromStringOrNil(resourceId)
	if resourceUUID == uuid.Nil {
		log.Info("Invalid resource id")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "invalid resource id"})
		return
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
	events, nextPageToken, err := h.serviceHandler.TimelineEventList(c.Request.Context(), &a, resourceType, resourceUUID, pageSize, pageToken)
	if err != nil {
		log.Infof("Failed to get timeline events: %v", err)
		if err.Error() == "user has no permission" || err.Error() == "not found" {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "resource not found"})
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

**Step 2: Run build**

Run: `cd bin-api-manager && go build ./cmd/...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add bin-api-manager/server/timelines.go
git commit -m "$(cat <<'EOF'
Add timeline endpoint handler

- bin-api-manager: Add GetTimelinesResourceTypeResourceIdEvents handler
- bin-api-manager: Handle pagination and error responses
EOF
)"
```

---

## Task 5: Add Unit Tests

**Files:**
- Create: `bin-api-manager/pkg/servicehandler/timeline_test.go`
- Create: `bin-api-manager/server/timelines_test.go`

**Step 1: Create servicehandler unit test**

Create `bin-api-manager/pkg/servicehandler/timeline_test.go`:

```go
package servicehandler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	tmevent "monorepo/bin-timeline-manager/models/event"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_TimelineEventList(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceID   uuid.UUID
		pageSize     int
		pageToken    string
		setupMocks   func(mc *gomock.Controller, h *serviceHandler)
		expectErr    bool
		expectCount  int
	}{
		{
			name:         "valid call timeline",
			resourceType: "calls",
			resourceID:   uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			pageSize:     100,
			pageToken:    "",
			setupMocks: func(mc *gomock.Controller, h *serviceHandler) {
				mockReqHandler := requesthandler.NewMockRequestHandler(mc)
				mockUtilHandler := utilhandler.NewMockUtilHandler(mc)
				h.reqHandler = mockReqHandler
				h.utilHandler = mockUtilHandler

				customerID := uuid.FromStringOrNil("6a93f71e-1234-5678-9abc-def012345678")
				resourceID := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000")

				// Mock call get
				mockReqHandler.EXPECT().CallV1CallGet(gomock.Any(), resourceID).Return(&cmcall.Call{
					Identity: commonidentity.Identity{
						ID:         resourceID,
						CustomerID: customerID,
					},
					TMDelete: "9999-01-01 00:00:00.000000",
				}, nil)

				// Mock timeline query
				callData, _ := json.Marshal(&cmcall.Call{
					Identity: commonidentity.Identity{
						ID:         resourceID,
						CustomerID: customerID,
					},
				})
				mockReqHandler.EXPECT().TimelineV1EventList(gomock.Any(), gomock.Any()).Return(&tmevent.EventListResponse{
					Result: []*tmevent.Event{
						{
							Timestamp: time.Now(),
							EventType: "call_created",
							Publisher: commonoutline.ServiceNameCallManager,
							Data:      callData,
						},
					},
					NextPageToken: "",
				}, nil)
			},
			expectErr:   false,
			expectCount: 1,
		},
		{
			name:         "invalid resource type",
			resourceType: "invalid",
			resourceID:   uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			pageSize:     100,
			pageToken:    "",
			setupMocks:   func(mc *gomock.Controller, h *serviceHandler) {},
			expectErr:    true,
			expectCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &serviceHandler{}
			tt.setupMocks(mc, h)

			agent := &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("agent-id"),
					CustomerID: uuid.FromStringOrNil("6a93f71e-1234-5678-9abc-def012345678"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}

			events, _, err := h.TimelineEventList(context.Background(), agent, tt.resourceType, tt.resourceID, tt.pageSize, tt.pageToken)

			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(events) != tt.expectCount {
				t.Errorf("expected %d events, got %d", tt.expectCount, len(events))
			}
		})
	}
}
```

**Step 2: Run tests**

Run: `cd bin-api-manager && go test ./pkg/servicehandler/... -run Test_TimelineEventList -v`
Expected: Tests pass

**Step 3: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/timeline_test.go
git commit -m "$(cat <<'EOF'
Add timeline servicehandler tests

- bin-api-manager: Add unit tests for TimelineEventList
EOF
)"
```

---

## Task 6: Run Full Verification

**Files:**
- Multiple services

**Step 1: Run verification for bin-openapi-manager**

Run: `cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All checks pass

**Step 2: Run verification for bin-api-manager**

Run: `cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All checks pass

**Step 3: Run verification for bin-common-handler**

Run: `cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All checks pass

**Step 4: Commit any remaining changes**

```bash
git add -A
git commit -m "$(cat <<'EOF'
Final verification and cleanup

- bin-api-manager: Fix any linting issues
- bin-openapi-manager: Fix any linting issues
EOF
)"
```

---

## Summary

This implementation adds a public timeline API with the following:

1. **OpenAPI Spec** (`bin-openapi-manager`): Defines `GET /timelines/{resource_type}/{resource_id}/events` with proper path parameters and response schema.

2. **ServiceHandler** (`bin-api-manager/pkg/servicehandler/timeline.go`):
   - Validates resource type (calls, conferences, flows, activeflows)
   - Validates resource ownership via existing helpers (callGet, conferenceGet, etc.)
   - Checks agent permission (CustomerAdmin or CustomerManager)
   - Queries timeline-manager via existing RPC method
   - Converts event data to WebhookMessage format using existing converters

3. **Server Handler** (`bin-api-manager/server/timelines.go`):
   - Handles HTTP request parsing
   - Validates parameters
   - Returns proper error responses (400, 404, 500)
   - Returns paginated results

4. **Tests**: Unit tests for servicehandler timeline methods.

The design reuses existing patterns:
- Resource validation via private helper methods (callGet, conferenceGet, etc.)
- Permission checking via hasPermission
- WebhookMessage conversion via ConvertWebhookMessage methods
- Pagination via page_size and page_token
