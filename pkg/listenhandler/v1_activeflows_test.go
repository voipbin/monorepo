package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/activeflowhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler"
)

func TestV1ActiveFlowsPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)
	mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		flowHandler:       mockFlowHandler,
		activeflowHandler: mockActive,
	}

	tests := []struct {
		name         string
		request      *rabbitmqhandler.Request
		expectCallID uuid.UUID
		expectFlowID uuid.UUID
		af           *activeflow.ActiveFlow
		expectRes    *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/active-flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"call_id": "1d8dacf4-05ee-11eb-9eae-037ddd66443e", "flow_id": "24092c98-05ee-11eb-a410-17d716ff3d61"}`),
			},
			uuid.FromStringOrNil("1d8dacf4-05ee-11eb-9eae-037ddd66443e"),
			uuid.FromStringOrNil("24092c98-05ee-11eb-a410-17d716ff3d61"),
			&activeflow.ActiveFlow{
				CallID:     uuid.FromStringOrNil("1d8dacf4-05ee-11eb-9eae-037ddd66443e"),
				FlowID:     uuid.FromStringOrNil("24092c98-05ee-11eb-a410-17d716ff3d61"),
				CustomerID: uuid.FromStringOrNil("cd607242-7f4b-11ec-a34f-bb861637ee36"),
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ExecuteCount:    0,
				ForwardActionID: action.IDEmpty,
				Actions:         []action.Action{},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"call_id":"1d8dacf4-05ee-11eb-9eae-037ddd66443e","flow_id":"24092c98-05ee-11eb-a410-17d716ff3d61","customer_id":"cd607242-7f4b-11ec-a34f-bb861637ee36","current_action":{"id":"00000000-0000-0000-0000-000000000001","type":""},"execute_count":0,"forward_action_id":"00000000-0000-0000-0000-000000000002","actions":[],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockActive.EXPECT().ActiveFlowCreate(gomock.Any(), tt.expectCallID, tt.expectFlowID).Return(tt.af, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			// if res.StatusCode != 200 {
			// 	t.Errorf("Wrong match. expect: 200, got: %d", res.StatusCode)
			// }

			// t.Errorf("%v", res)
		})
	}
}

func TestV1ActiveFlowsIDNextGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)
	mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		flowHandler:       mockFlowHandler,
		activeflowHandler: mockActive,
	}

	tests := []struct {
		name            string
		request         *rabbitmqhandler.Request
		callID          uuid.UUID
		currentActionID uuid.UUID
		nextAction      action.Action
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/active-flows/cec5b926-06a7-11eb-967e-fb463343f0a5/next",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"current_action_id": "6a1ce642-06a8-11eb-a632-978be835f982"}`),
			},
			uuid.FromStringOrNil("cec5b926-06a7-11eb-967e-fb463343f0a5"),
			uuid.FromStringOrNil("6a1ce642-06a8-11eb-a632-978be835f982"),
			action.Action{
				ID:   uuid.FromStringOrNil("63698276-06ab-11eb-9cbf-c771a09c1619"),
				Type: action.TypeEcho,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockActive.EXPECT().ActiveFlowNextActionGet(gomock.Any(), tt.callID, tt.currentActionID).Return(&tt.nextAction, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Wrong match. expect: 200, got: %d", res.StatusCode)
			}
		})
	}
}

func TestV1ActiveFlowsIDForwardActionIDPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)
	mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		flowHandler:       mockFlowHandler,
		activeflowHandler: mockActive,
	}

	tests := []struct {
		name            string
		request         *rabbitmqhandler.Request
		callID          uuid.UUID
		forwardActionID uuid.UUID
		forwardNow      bool
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/active-flows/6f14f3b8-5758-11ec-a413-772c32e3e51f/forward_action_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"forward_action_id": "6732dd5e-5758-11ec-92b1-bfe33ab190aa", "forward_now": true}`),
			},
			uuid.FromStringOrNil("6f14f3b8-5758-11ec-a413-772c32e3e51f"),
			uuid.FromStringOrNil("6732dd5e-5758-11ec-92b1-bfe33ab190aa"),
			true,
		},
		{
			"forward now false",
			&rabbitmqhandler.Request{
				URI:      "/v1/active-flows/6f14f3b8-5758-11ec-a413-772c32e3e51f/forward_action_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"forward_action_id": "6732dd5e-5758-11ec-92b1-bfe33ab190aa", "forward_now": false}`),
			},
			uuid.FromStringOrNil("6f14f3b8-5758-11ec-a413-772c32e3e51f"),
			uuid.FromStringOrNil("6732dd5e-5758-11ec-92b1-bfe33ab190aa"),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockActive.EXPECT().ActiveFlowSetForwardActionID(gomock.Any(), tt.callID, tt.forwardActionID, tt.forwardNow).Return(nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Wrong match. expect: 200, got: %d", res.StatusCode)
			}
		})
	}
}
