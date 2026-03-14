package servicehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	tmevent "monorepo/bin-timeline-manager/models/event"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_AggregatedEventList_NeitherProvided(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	agent := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	_, _, err := h.AggregatedEventList(ctx, agent, uuid.Nil, uuid.Nil, 10, "")
	if err == nil {
		t.Error("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "either activeflow_id or call_id is required" {
		t.Errorf("Wrong error message. expect: either activeflow_id or call_id is required, got: %s", err.Error())
	}
}

func Test_AggregatedEventList_BothProvided(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	activeflowID := uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a")
	callID := uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a")

	agent := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	_, _, err := h.AggregatedEventList(ctx, agent, activeflowID, callID, 10, "")
	if err == nil {
		t.Error("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "only one of activeflow_id or call_id is allowed" {
		t.Errorf("Wrong error message. expect: only one of activeflow_id or call_id is allowed, got: %s", err.Error())
	}
}

func Test_AggregatedEventList_ActiveflowNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	activeflowID := uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a")

	agent := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	mockReq.EXPECT().FlowV1ActiveflowGet(ctx, activeflowID).Return(nil, fmt.Errorf("not found"))

	_, _, err := h.AggregatedEventList(ctx, agent, activeflowID, uuid.Nil, 10, "")
	if err == nil {
		t.Error("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "not found" {
		t.Errorf("Wrong error message. expect: not found, got: %s", err.Error())
	}
}

func Test_AggregatedEventList_ActiveflowNoPermission(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	activeflowID := uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a")
	activeflowCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentCustomerID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	agent := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: agentCustomerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	testActiveflow := &fmactiveflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         activeflowID,
			CustomerID: activeflowCustomerID,
		},
	}

	mockReq.EXPECT().FlowV1ActiveflowGet(ctx, activeflowID).Return(testActiveflow, nil)

	_, _, err := h.AggregatedEventList(ctx, agent, activeflowID, uuid.Nil, 10, "")
	if err == nil {
		t.Error("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "user has no permission" {
		t.Errorf("Wrong error message. expect: user has no permission, got: %s", err.Error())
	}
}

func Test_AggregatedEventList_ActiveflowSuccess(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	activeflowID := uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a")
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	testTimestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	agent := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	testActiveflow := &fmactiveflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         activeflowID,
			CustomerID: customerID,
		},
	}

	testData := json.RawMessage(`{"key":"value"}`)

	responseEvents := &tmevent.AggregatedEventListResponse{
		Result: []*tmevent.Event{
			{
				Timestamp: testTimestamp,
				EventType: "call_created",
				Data:      testData,
			},
			{
				Timestamp: testTimestamp.Add(time.Minute),
				EventType: "call_hangup",
				Data:      testData,
			},
		},
		NextPageToken: "next-token",
	}

	mockReq.EXPECT().FlowV1ActiveflowGet(ctx, activeflowID).Return(testActiveflow, nil)
	mockReq.EXPECT().TimelineV1AggregatedEventList(ctx, &tmevent.AggregatedEventListRequest{
		ActiveflowID: activeflowID,
		PageSize:     10,
		PageToken:    "",
	}).Return(responseEvents, nil)

	res, nextToken, err := h.AggregatedEventList(ctx, agent, activeflowID, uuid.Nil, 10, "")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) != 2 {
		t.Errorf("Wrong result length. expect: 2, got: %d", len(res))
	}

	if nextToken != "next-token" {
		t.Errorf("Wrong next token. expect: next-token, got: %s", nextToken)
	}
}

func Test_AggregatedEventList_CallNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	callID := uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a")

	agent := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	mockReq.EXPECT().CallV1CallGet(ctx, callID).Return(nil, fmt.Errorf("not found"))

	_, _, err := h.AggregatedEventList(ctx, agent, uuid.Nil, callID, 10, "")
	if err == nil {
		t.Error("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "not found" {
		t.Errorf("Wrong error message. expect: not found, got: %s", err.Error())
	}
}

func Test_AggregatedEventList_CallNoPermission(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	callID := uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a")
	callCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentCustomerID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	agent := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: agentCustomerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	testCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID:         callID,
			CustomerID: callCustomerID,
		},
	}

	mockReq.EXPECT().CallV1CallGet(ctx, callID).Return(testCall, nil)

	_, _, err := h.AggregatedEventList(ctx, agent, uuid.Nil, callID, 10, "")
	if err == nil {
		t.Error("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "user has no permission" {
		t.Errorf("Wrong error message. expect: user has no permission, got: %s", err.Error())
	}
}

func Test_AggregatedEventList_CallNoActiveflow(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	callID := uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a")
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")

	agent := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	testCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID:         callID,
			CustomerID: customerID,
		},
		ActiveflowID: uuid.Nil,
	}

	mockReq.EXPECT().CallV1CallGet(ctx, callID).Return(testCall, nil)

	_, _, err := h.AggregatedEventList(ctx, agent, uuid.Nil, callID, 10, "")
	if err == nil {
		t.Error("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "not found" {
		t.Errorf("Wrong error message. expect: not found, got: %s", err.Error())
	}
}

func Test_AggregatedEventList_CallSuccess(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	callID := uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a")
	activeflowID := uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a")
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	testTimestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	agent := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	testCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID:         callID,
			CustomerID: customerID,
		},
		ActiveflowID: activeflowID,
	}

	testData := json.RawMessage(`{"key":"value"}`)

	responseEvents := &tmevent.AggregatedEventListResponse{
		Result: []*tmevent.Event{
			{
				Timestamp: testTimestamp,
				EventType: "call_created",
				Data:      testData,
			},
		},
		NextPageToken: "",
	}

	mockReq.EXPECT().CallV1CallGet(ctx, callID).Return(testCall, nil)
	mockReq.EXPECT().TimelineV1AggregatedEventList(ctx, &tmevent.AggregatedEventListRequest{
		ActiveflowID: activeflowID,
		PageSize:     10,
		PageToken:    "",
	}).Return(responseEvents, nil)

	res, nextToken, err := h.AggregatedEventList(ctx, agent, uuid.Nil, callID, 10, "")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) != 1 {
		t.Errorf("Wrong result length. expect: 1, got: %d", len(res))
	}

	if nextToken != "" {
		t.Errorf("Wrong next token. expect: empty, got: %s", nextToken)
	}
}

func Test_AggregatedEventList_TimelineRPCError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	activeflowID := uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a")
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")

	agent := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	testActiveflow := &fmactiveflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         activeflowID,
			CustomerID: customerID,
		},
	}

	mockReq.EXPECT().FlowV1ActiveflowGet(ctx, activeflowID).Return(testActiveflow, nil)
	mockReq.EXPECT().TimelineV1AggregatedEventList(ctx, &tmevent.AggregatedEventListRequest{
		ActiveflowID: activeflowID,
		PageSize:     10,
		PageToken:    "",
	}).Return(nil, fmt.Errorf("timeline service unavailable"))

	_, _, err := h.AggregatedEventList(ctx, agent, activeflowID, uuid.Nil, 10, "")
	if err == nil {
		t.Error("Wrong match. expect: error, got: nil")
	}
	if err.Error() != "internal error" {
		t.Errorf("Wrong error message. expect: internal error, got: %s", err.Error())
	}
}

func Test_AggregatedEventList_SuccessWithConversion(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	activeflowID := uuid.FromStringOrNil("c3d4e5f6-8f36-11ed-a01a-efb53befe93a")
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	testTimestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	agent := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	}

	testActiveflow := &fmactiveflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         activeflowID,
			CustomerID: customerID,
		},
	}

	testData := json.RawMessage(`{"call_id":"abc-123","status":"hangup"}`)

	responseEvents := &tmevent.AggregatedEventListResponse{
		Result: []*tmevent.Event{
			{
				Timestamp: testTimestamp,
				EventType: "call_hangup",
				Data:      testData,
			},
		},
		NextPageToken: "",
	}

	mockReq.EXPECT().FlowV1ActiveflowGet(ctx, activeflowID).Return(testActiveflow, nil)
	mockReq.EXPECT().TimelineV1AggregatedEventList(ctx, &tmevent.AggregatedEventListRequest{
		ActiveflowID: activeflowID,
		PageSize:     10,
		PageToken:    "",
	}).Return(responseEvents, nil)

	res, _, err := h.AggregatedEventList(ctx, agent, activeflowID, uuid.Nil, 10, "")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) != 1 {
		t.Errorf("Wrong result length. expect: 1, got: %d", len(res))
	}

	// Verify timestamp format
	expectedTimestamp := "2024-01-15T10:30:00.000Z"
	if res[0].Timestamp != expectedTimestamp {
		t.Errorf("Wrong timestamp. expect: %s, got: %s", expectedTimestamp, res[0].Timestamp)
	}

	// Verify event type
	if res[0].EventType != "call_hangup" {
		t.Errorf("Wrong event type. expect: call_hangup, got: %s", res[0].EventType)
	}

	// Verify data is preserved as json.RawMessage
	dataBytes, ok := res[0].Data.(json.RawMessage)
	if !ok {
		t.Errorf("Wrong data type. expect: json.RawMessage, got: %T", res[0].Data)
	} else {
		expectedData := `{"call_id":"abc-123","status":"hangup"}`
		if string(dataBytes) != expectedData {
			t.Errorf("Wrong data. expect: %s, got: %s", expectedData, string(dataBytes))
		}
	}
}
