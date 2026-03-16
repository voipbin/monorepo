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

	// Use valid Call JSON that can be unmarshaled into cmcall.Call
	callData := json.RawMessage(`{"id":"550e8400-e29b-41d4-a716-446655440000","customer_id":"` + customerID.String() + `","status":"progressing"}`)
	hangupData := json.RawMessage(`{"id":"550e8400-e29b-41d4-a716-446655440000","customer_id":"` + customerID.String() + `","status":"hangup"}`)

	responseEvents := &tmevent.AggregatedEventListResponse{
		Result: []*tmevent.Event{
			{
				Timestamp: testTimestamp,
				EventType: "call_created",
				Data:      callData,
			},
			{
				Timestamp: testTimestamp.Add(time.Minute),
				EventType: "call_hangup",
				Data:      hangupData,
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

	// Verify data is now *cmcall.WebhookMessage (not raw JSON)
	if _, ok := res[0].Data.(*cmcall.WebhookMessage); !ok {
		t.Errorf("Wrong data type. expect: *cmcall.WebhookMessage, got: %T", res[0].Data)
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

	// Use valid Call JSON that can be unmarshaled into cmcall.Call
	callData := json.RawMessage(`{"id":"` + callID.String() + `","customer_id":"` + customerID.String() + `","status":"progressing"}`)

	responseEvents := &tmevent.AggregatedEventListResponse{
		Result: []*tmevent.Event{
			{
				Timestamp: testTimestamp,
				EventType: "call_created",
				Data:      callData,
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
	callID := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000")
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

	// Include internal-only field (channel_id) to verify it gets stripped by ConvertWebhookMessage
	testData := json.RawMessage(`{"id":"` + callID.String() + `","customer_id":"` + customerID.String() + `","status":"hangup","channel_id":"ast-channel-123"}`)

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

	// Verify data is now *cmcall.WebhookMessage (converted from internal struct)
	webhookMsg, ok := res[0].Data.(*cmcall.WebhookMessage)
	if !ok {
		t.Errorf("Wrong data type. expect: *cmcall.WebhookMessage, got: %T", res[0].Data)
	} else {
		// Verify fields are properly converted
		if webhookMsg.Status != cmcall.StatusHangup {
			t.Errorf("Wrong status. expect: hangup, got: %s", webhookMsg.Status)
		}
		if webhookMsg.ID != callID {
			t.Errorf("Wrong ID. expect: %s, got: %s", callID, webhookMsg.ID)
		}
	}
}

func Test_AggregatedEventList_UnknownEventTypeSkipped(t *testing.T) {
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

	// Mix of known (call_) and unknown (confbridge_) event types
	callData := json.RawMessage(`{"id":"550e8400-e29b-41d4-a716-446655440000","customer_id":"` + customerID.String() + `","status":"progressing"}`)
	unknownData := json.RawMessage(`{"id":"aaa","bridge_id":"internal-bridge-123"}`)

	responseEvents := &tmevent.AggregatedEventListResponse{
		Result: []*tmevent.Event{
			{
				Timestamp: testTimestamp,
				EventType: "call_created",
				Data:      callData,
			},
			{
				Timestamp: testTimestamp.Add(time.Second),
				EventType: "confbridge_created",
				Data:      unknownData,
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

	// Only the known call event should be in results (confbridge_ is unsupported and skipped)
	if len(res) != 1 {
		t.Errorf("Wrong result length. expect: 1, got: %d", len(res))
	}

	if res[0].EventType != "call_created" {
		t.Errorf("Wrong event type. expect: call_created, got: %s", res[0].EventType)
	}
}

// Test_convertAggregatedEventData_LongestPrefixMatch verifies that overlapping prefixes
// resolve to the most specific (longest) converter.
func Test_convertAggregatedEventData_LongestPrefixMatch(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		data      string
		wantType  string // expected Go type name of the Data field
	}{
		{
			name:      "conversation_message_ beats conversation_",
			eventType: "conversation_message_created",
			data:      `{"id":"550e8400-e29b-41d4-a716-446655440000"}`,
			wantType:  "*message.WebhookMessage",
		},
		{
			name:      "conversation_ matches conversation event",
			eventType: "conversation_created",
			data:      `{"id":"550e8400-e29b-41d4-a716-446655440000"}`,
			wantType:  "*conversation.WebhookMessage",
		},
		{
			name:      "extension_direct_ beats extension_",
			eventType: "extension_direct_created",
			data:      `{"id":"550e8400-e29b-41d4-a716-446655440000"}`,
			wantType:  "*extensiondirect.WebhookMessage",
		},
		{
			name:      "extension_ matches extension event",
			eventType: "extension_created",
			data:      `{"id":"550e8400-e29b-41d4-a716-446655440000"}`,
			wantType:  "*extension.WebhookMessage",
		},
		{
			name:      "transcribe_speech_ beats transcribe_",
			eventType: "transcribe_speech_started",
			data:      `{"id":"550e8400-e29b-41d4-a716-446655440000"}`,
			wantType:  "*streaming.WebhookMessage",
		},
		{
			name:      "transcribe_ matches transcribe event",
			eventType: "transcribe_created",
			data:      `{"id":"550e8400-e29b-41d4-a716-446655440000"}`,
			wantType:  "*transcribe.WebhookMessage",
		},
		{
			name:      "account_ case-sensitive does not match Account_",
			eventType: "account_created",
			data:      `{"id":"550e8400-e29b-41d4-a716-446655440000"}`,
			wantType:  "*account.WebhookMessage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &tmevent.Event{
				Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				EventType: tt.eventType,
				Data:      json.RawMessage(tt.data),
			}

			result, err := convertAggregatedEventData(event)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			gotType := fmt.Sprintf("%T", result.Data)
			if gotType != tt.wantType {
				t.Errorf("Wrong data type for %s. want: %s, got: %s", tt.eventType, tt.wantType, gotType)
			}
		})
	}
}

// Test_convertAggregatedEventData_UnsupportedType verifies that unknown event types return an error.
func Test_convertAggregatedEventData_UnsupportedType(t *testing.T) {
	event := &tmevent.Event{
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		EventType: "unknown_event_type",
		Data:      json.RawMessage(`{"id":"test"}`),
	}

	_, err := convertAggregatedEventData(event)
	if err == nil {
		t.Error("Expected error for unsupported event type, got nil")
	}
}
