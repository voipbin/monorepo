package listenhandler

import (
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/activeflowhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler"
)

func Test_V1ActiveflowsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		expectRefereceType activeflow.ReferenceType
		expectRefereceID   uuid.UUID

		expectFlowID uuid.UUID
		af           *activeflow.Activeflow
		expectRes    *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/activeflows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"reference_type": "call", "reference_id": "b66c4922-a7a4-11ec-8e1b-6765ceec0323", "flow_id": "24092c98-05ee-11eb-a410-17d716ff3d61"}`),
			},

			activeflow.ReferenceTypeCall,
			uuid.FromStringOrNil("b66c4922-a7a4-11ec-8e1b-6765ceec0323"),

			uuid.FromStringOrNil("24092c98-05ee-11eb-a410-17d716ff3d61"),
			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("bd89ee76-a7a4-11ec-a1bd-8315ed90b9d1"),

				FlowID:     uuid.FromStringOrNil("24092c98-05ee-11eb-a410-17d716ff3d61"),
				CustomerID: uuid.FromStringOrNil("cd607242-7f4b-11ec-a34f-bb861637ee36"),

				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b66c4922-a7a4-11ec-8e1b-6765ceec0323"),

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
				Data:       []byte(`{"id":"bd89ee76-a7a4-11ec-a1bd-8315ed90b9d1","customer_id":"cd607242-7f4b-11ec-a34f-bb861637ee36","flow_id":"24092c98-05ee-11eb-a410-17d716ff3d61","reference_type":"call","reference_id":"b66c4922-a7a4-11ec-8e1b-6765ceec0323","current_action":{"id":"00000000-0000-0000-0000-000000000001","next_id":"00000000-0000-0000-0000-000000000000","type":""},"execute_count":0,"forward_action_id":"00000000-0000-0000-0000-000000000000","actions":[],"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockActive.EXPECT().Create(gomock.Any(), tt.expectRefereceType, tt.expectRefereceID, tt.expectFlowID).Return(tt.af, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestV1ActiveflowsIDNextGet(t *testing.T) {
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
				URI:      "/v1/activeflows/cec5b926-06a7-11eb-967e-fb463343f0a5/next",
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

			mockActive.EXPECT().GetNextAction(gomock.Any(), tt.callID, tt.currentActionID).Return(&tt.nextAction, nil)
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

func TestV1ActiveflowsIDForwardActionIDPut(t *testing.T) {

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
				URI:      "/v1/activeflows/6f14f3b8-5758-11ec-a413-772c32e3e51f/forward_action_id",
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
				URI:      "/v1/activeflows/6f14f3b8-5758-11ec-a413-772c32e3e51f/forward_action_id",
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

			mockActive.EXPECT().SetForwardActionID(gomock.Any(), tt.callID, tt.forwardActionID, tt.forwardNow).Return(nil)
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

func Test_v1ActiveflowsIDExecutePost(t *testing.T) {
	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		activeflowID uuid.UUID

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/activeflows/07c60d7c-a7ae-11ec-ad69-c3e765668a2b/execute",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("07c60d7c-a7ae-11ec-ad69-c3e765668a2b"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockActive.EXPECT().Execute(gomock.Any(), tt.activeflowID)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(100 * time.Millisecond)

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1ActiveflowsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		activeflowID   uuid.UUID
		responseDelete *activeflow.Activeflow

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/activeflows/4356d70a-adde-11ec-bff4-9fc5420b5bcb",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("4356d70a-adde-11ec-bff4-9fc5420b5bcb"),
			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("4356d70a-adde-11ec-bff4-9fc5420b5bcb"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4356d70a-adde-11ec-bff4-9fc5420b5bcb","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"execute_count":0,"forward_action_id":"00000000-0000-0000-0000-000000000000","actions":null,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockActive.EXPECT().Delete(gomock.Any(), tt.activeflowID).Return(tt.responseDelete, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
