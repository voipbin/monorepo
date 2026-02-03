package servicehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	tmevent "monorepo/bin-timeline-manager/models/event"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_TimelineEventList(t *testing.T) {

	callID := uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a")
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")

	testCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID:         callID,
			CustomerID: customerID,
		},
		TMDelete: defaultTimestamp,
	}

	callJSON, _ := json.Marshal(testCall)
	testTimestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string

		agent        *amagent.Agent
		resourceType string
		resourceID   uuid.UUID
		pageSize     int
		pageToken    string

		responseCall       *cmcall.Call
		responseEvents     *tmevent.EventListResponse
		expectEventRequest *tmevent.EventListRequest
		expectResLen       int
	}{
		{
			name: "valid call timeline request",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			resourceType: "calls",
			resourceID:   callID,
			pageSize:     10,
			pageToken:    "",

			responseCall: testCall,
			responseEvents: &tmevent.EventListResponse{
				Result: []*tmevent.Event{
					{
						Timestamp: testTimestamp,
						EventType: "call_created",
						Publisher: commonoutline.ServiceNameCallManager,
						Data:      callJSON,
					},
					{
						Timestamp: testTimestamp.Add(time.Minute),
						EventType: "call_hangup",
						Publisher: commonoutline.ServiceNameCallManager,
						Data:      callJSON,
					},
				},
				NextPageToken: "next-token",
			},
			expectEventRequest: &tmevent.EventListRequest{
				Publisher:  commonoutline.ServiceNameCallManager,
				ResourceID: callID,
				Events:     []string{"call_*"},
				PageSize:   10,
				PageToken:  "",
			},
			expectResLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.resourceID).Return(tt.responseCall, nil)
			mockReq.EXPECT().TimelineV1EventList(ctx, tt.expectEventRequest).Return(tt.responseEvents, nil)

			res, nextToken, err := h.TimelineEventList(ctx, tt.agent, tt.resourceType, tt.resourceID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.expectResLen {
				t.Errorf("Wrong result length. expect: %d, got: %d", tt.expectResLen, len(res))
			}

			if nextToken != tt.responseEvents.NextPageToken {
				t.Errorf("Wrong next token. expect: %s, got: %s", tt.responseEvents.NextPageToken, nextToken)
			}
		})
	}
}

func Test_TimelineEventList_error_invalid_resource_type(t *testing.T) {

	tests := []struct {
		name string

		agent        *amagent.Agent
		resourceType string
		resourceID   uuid.UUID
		pageSize     int
		pageToken    string
	}{
		{
			name: "invalid resource type",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			resourceType: "invalid_type",
			resourceID:   uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a"),
			pageSize:     10,
			pageToken:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			_, _, err := h.TimelineEventList(ctx, tt.agent, tt.resourceType, tt.resourceID, tt.pageSize, tt.pageToken)
			if err == nil {
				t.Error("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_TimelineEventList_error_resource_not_found(t *testing.T) {

	tests := []struct {
		name string

		agent        *amagent.Agent
		resourceType string
		resourceID   uuid.UUID
		pageSize     int
		pageToken    string

		responseCallError error
	}{
		{
			name: "resource not found",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			resourceType: "calls",
			resourceID:   uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a"),
			pageSize:     10,
			pageToken:    "",

			responseCallError: fmt.Errorf("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.resourceID).Return(nil, tt.responseCallError)

			_, _, err := h.TimelineEventList(ctx, tt.agent, tt.resourceType, tt.resourceID, tt.pageSize, tt.pageToken)
			if err == nil {
				t.Error("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_TimelineEventList_error_permission_denied(t *testing.T) {

	callID := uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a")
	callCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentCustomerID := uuid.FromStringOrNil("different-customer-id")

	tests := []struct {
		name string

		agent        *amagent.Agent
		resourceType string
		resourceID   uuid.UUID
		pageSize     int
		pageToken    string

		responseCall *cmcall.Call
	}{
		{
			name: "permission denied - different customer",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: agentCustomerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			resourceType: "calls",
			resourceID:   callID,
			pageSize:     10,
			pageToken:    "",

			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         callID,
					CustomerID: callCustomerID,
				},
				TMDelete: defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.resourceID).Return(tt.responseCall, nil)

			_, _, err := h.TimelineEventList(ctx, tt.agent, tt.resourceType, tt.resourceID, tt.pageSize, tt.pageToken)
			if err == nil {
				t.Error("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_convertEventToWebhookMessage(t *testing.T) {

	callID := uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a")
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")

	testCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID:         callID,
			CustomerID: customerID,
		},
		TMDelete: defaultTimestamp,
	}
	callJSON, _ := json.Marshal(testCall)
	testTimestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name         string
		resourceType string
		event        *tmevent.Event
		expectRes    *TimelineEvent
		expectErr    bool
	}{
		{
			name:         "convert call event",
			resourceType: "calls",
			event: &tmevent.Event{
				Timestamp: testTimestamp,
				EventType: "call_created",
				Publisher: commonoutline.ServiceNameCallManager,
				Data:      callJSON,
			},
			expectRes: &TimelineEvent{
				Timestamp: "2024-01-15T10:30:00.000Z",
				EventType: "call_created",
				Data:      testCall.ConvertWebhookMessage(),
			},
			expectErr: false,
		},
		{
			name:         "invalid resource type",
			resourceType: "invalid",
			event: &tmevent.Event{
				Timestamp: testTimestamp,
				EventType: "invalid_created",
				Data:      callJSON,
			},
			expectRes: nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &serviceHandler{}

			res, err := h.convertEventToWebhookMessage(tt.resourceType, tt.event)
			if tt.expectErr {
				if err == nil {
					t.Error("Wrong match. expect: error, got: nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.Timestamp != tt.expectRes.Timestamp {
				t.Errorf("Wrong timestamp. expect: %s, got: %s", tt.expectRes.Timestamp, res.Timestamp)
			}

			if res.EventType != tt.expectRes.EventType {
				t.Errorf("Wrong event type. expect: %s, got: %s", tt.expectRes.EventType, res.EventType)
			}

			if !reflect.DeepEqual(res.Data, tt.expectRes.Data) {
				t.Errorf("Wrong data. expect: %v, got: %v", tt.expectRes.Data, res.Data)
			}
		})
	}
}
