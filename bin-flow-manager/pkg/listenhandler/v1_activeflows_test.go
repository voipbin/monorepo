package listenhandler

import (
	"reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/pkg/activeflowhandler"
	"monorepo/bin-flow-manager/pkg/flowhandler"
)

func Test_v1ActiveflowsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		expectID           uuid.UUID
		expectRefereceType activeflow.ReferenceType
		expectRefereceID   uuid.UUID
		expectFlowID       uuid.UUID

		responseActiveflow *activeflow.Activeflow
		expectRes          *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/activeflows",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"a508739b-d98d-40fb-8a47-61e9a70958cd","reference_type": "call", "reference_id": "b66c4922-a7a4-11ec-8e1b-6765ceec0323", "flow_id": "24092c98-05ee-11eb-a410-17d716ff3d61"}`),
			},

			uuid.FromStringOrNil("a508739b-d98d-40fb-8a47-61e9a70958cd"),
			activeflow.ReferenceTypeCall,
			uuid.FromStringOrNil("b66c4922-a7a4-11ec-8e1b-6765ceec0323"),
			uuid.FromStringOrNil("24092c98-05ee-11eb-a410-17d716ff3d61"),

			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("a508739b-d98d-40fb-8a47-61e9a70958cd"),

				FlowID:     uuid.FromStringOrNil("24092c98-05ee-11eb-a410-17d716ff3d61"),
				CustomerID: uuid.FromStringOrNil("cd607242-7f4b-11ec-a34f-bb861637ee36"),

				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b66c4922-a7a4-11ec-8e1b-6765ceec0323"),

				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ExecuteCount:    0,
				ForwardActionID: action.IDEmpty,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a508739b-d98d-40fb-8a47-61e9a70958cd","customer_id":"cd607242-7f4b-11ec-a34f-bb861637ee36","flow_id":"24092c98-05ee-11eb-a410-17d716ff3d61","status":"","reference_type":"call","reference_id":"b66c4922-a7a4-11ec-8e1b-6765ceec0323","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000001","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"empty id",
			&sock.Request{
				URI:      "/v1/activeflows",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"reference_type": "call", "reference_id": "b66c4922-a7a4-11ec-8e1b-6765ceec0323", "flow_id": "24092c98-05ee-11eb-a410-17d716ff3d61"}`),
			},

			uuid.Nil,
			activeflow.ReferenceTypeCall,
			uuid.FromStringOrNil("b66c4922-a7a4-11ec-8e1b-6765ceec0323"),
			uuid.FromStringOrNil("24092c98-05ee-11eb-a410-17d716ff3d61"),

			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("7a18e7e0-7d67-44f2-9591-58cc7d8f5610"),

				FlowID:     uuid.FromStringOrNil("24092c98-05ee-11eb-a410-17d716ff3d61"),
				CustomerID: uuid.FromStringOrNil("cd607242-7f4b-11ec-a34f-bb861637ee36"),

				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b66c4922-a7a4-11ec-8e1b-6765ceec0323"),

				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ExecuteCount:    0,
				ForwardActionID: action.IDEmpty,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7a18e7e0-7d67-44f2-9591-58cc7d8f5610","customer_id":"cd607242-7f4b-11ec-a34f-bb861637ee36","flow_id":"24092c98-05ee-11eb-a410-17d716ff3d61","status":"","reference_type":"call","reference_id":"b66c4922-a7a4-11ec-8e1b-6765ceec0323","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000001","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)
			mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				flowHandler:       mockFlowHandler,
				activeflowHandler: mockActive,
			}

			mockActive.EXPECT().Create(gomock.Any(), tt.expectID, tt.expectRefereceType, tt.expectRefereceID, tt.expectFlowID).Return(tt.responseActiveflow, nil)
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

func Test_v1ActiveflowsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		pageToken string
		pageSize  uint64

		responseFilters     map[string]string
		responseActiveflows []*activeflow.Activeflow

		expectRes *sock.Response
	}{
		{
			"1 item",
			&sock.Request{
				URI:    "/v1/activeflows?page_token=2020-10-10%2003:30:17.000000&page_size=10&filter_customer_id=16d3fcf0-7f4c-11ec-a4c3-7bf43125108d&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			"2020-10-10 03:30:17.000000",
			10,

			map[string]string{
				"customer_id": "16d3fcf0-7f4c-11ec-a4c3-7bf43125108d",
				"deleted":     "false",
			},
			[]*activeflow.Activeflow{
				{
					ID:         uuid.FromStringOrNil("ae07a96a-cbda-11ed-bbc9-5324a1a8b94b"),
					CustomerID: uuid.FromStringOrNil("16d3fcf0-7f4c-11ec-a4c3-7bf43125108d"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ae07a96a-cbda-11ed-bbc9-5324a1a8b94b","customer_id":"16d3fcf0-7f4c-11ec-a4c3-7bf43125108d","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"2 items",
			&sock.Request{
				URI:    "/v1/activeflows?page_token=2020-10-10%2003:30:17.000000&page_size=10&filter_customer_id=2457d824-7f4c-11ec-9489-b3552a7c9d63&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			"2020-10-10 03:30:17.000000",
			10,

			map[string]string{
				"customer_id": "2457d824-7f4c-11ec-9489-b3552a7c9d63",
				"deleted":     "false",
			},
			[]*activeflow.Activeflow{
				{
					ID: uuid.FromStringOrNil("ae3724b0-cbda-11ed-a44c-1be0474024bd"),
				},
				{
					ID: uuid.FromStringOrNil("ae66a7bc-cbda-11ed-81b3-17eb5a6f42b2"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ae3724b0-cbda-11ed-a44c-1be0474024bd","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"ae66a7bc-cbda-11ed-81b3-17eb5a6f42b2","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"empty",
			&sock.Request{
				URI:      "/v1/activeflows?page_token=2020-10-10%2003:30:17.000000&page_size=10&filter_customer_id=3ee14bee-7f4c-11ec-a1d8-a3a488ed5885&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			"2020-10-10 03:30:17.000000",
			10,

			map[string]string{
				"customer_id": "3ee14bee-7f4c-11ec-a1d8-a3a488ed5885",
				"deleted":     "false",
			},
			[]*activeflow.Activeflow{},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)
			mockActiveflowHandler := activeflowhandler.NewMockActiveflowHandler(mc)

			h := &listenHandler{
				utilHandler:       mockUtil,
				sockHandler:       mockSock,
				flowHandler:       mockFlowHandler,
				activeflowHandler: mockActiveflowHandler,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockActiveflowHandler.EXPECT().Gets(gomock.Any(), tt.pageToken, tt.pageSize, tt.responseFilters).Return(tt.responseActiveflows, nil)

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

func Test_v1ActiveflowsIDNextGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		callID          uuid.UUID
		currentActionID uuid.UUID

		responseAction action.Action
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/activeflows/cec5b926-06a7-11eb-967e-fb463343f0a5/next",
				Method:   sock.RequestMethodGet,
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)
			mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				flowHandler:       mockFlowHandler,
				activeflowHandler: mockActive,
			}

			mockActive.EXPECT().ExecuteNextAction(gomock.Any(), tt.callID, tt.currentActionID).Return(&tt.responseAction, nil)
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

func Test_v1ActiveflowsIDForwardActionIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		callID          uuid.UUID
		forwardActionID uuid.UUID
		forwardNow      bool

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/activeflows/6f14f3b8-5758-11ec-a413-772c32e3e51f/forward_action_id",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"forward_action_id": "6732dd5e-5758-11ec-92b1-bfe33ab190aa", "forward_now": true}`),
			},

			uuid.FromStringOrNil("6f14f3b8-5758-11ec-a413-772c32e3e51f"),
			uuid.FromStringOrNil("6732dd5e-5758-11ec-92b1-bfe33ab190aa"),
			true,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
		{
			"forward now false",
			&sock.Request{
				URI:      "/v1/activeflows/6f14f3b8-5758-11ec-a413-772c32e3e51f/forward_action_id",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"forward_action_id": "6732dd5e-5758-11ec-92b1-bfe33ab190aa", "forward_now": false}`),
			},

			uuid.FromStringOrNil("6f14f3b8-5758-11ec-a413-772c32e3e51f"),
			uuid.FromStringOrNil("6732dd5e-5758-11ec-92b1-bfe33ab190aa"),
			false,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)
			mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				flowHandler:       mockFlowHandler,
				activeflowHandler: mockActive,
			}

			mockActive.EXPECT().SetForwardActionID(gomock.Any(), tt.callID, tt.forwardActionID, tt.forwardNow).Return(nil)
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

func Test_v1ActiveflowsIDExecutePost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		activeflowID uuid.UUID

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/activeflows/07c60d7c-a7ae-11ec-ad69-c3e765668a2b/execute",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("07c60d7c-a7ae-11ec-ad69-c3e765668a2b"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)
			mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
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

func Test_v1ActiveflowsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		activeflowID   uuid.UUID
		responseDelete *activeflow.Activeflow

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/activeflows/343b6e40-cbdb-11ed-b13d-9f730017f25a",
				Method: sock.RequestMethodGet,
			},

			uuid.FromStringOrNil("343b6e40-cbdb-11ed-b13d-9f730017f25a"),
			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("343b6e40-cbdb-11ed-b13d-9f730017f25a"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"343b6e40-cbdb-11ed-b13d-9f730017f25a","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)
			mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				flowHandler:       mockFlowHandler,
				activeflowHandler: mockActive,
			}

			mockActive.EXPECT().Get(gomock.Any(), tt.activeflowID).Return(tt.responseDelete, nil)
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

func Test_v1ActiveflowsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		activeflowID   uuid.UUID
		responseDelete *activeflow.Activeflow

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/activeflows/4356d70a-adde-11ec-bff4-9fc5420b5bcb",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("4356d70a-adde-11ec-bff4-9fc5420b5bcb"),
			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("4356d70a-adde-11ec-bff4-9fc5420b5bcb"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4356d70a-adde-11ec-bff4-9fc5420b5bcb","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)
			mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
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

func Test_v1ActiveflowsIDStopPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseActiveflow *activeflow.Activeflow

		expectActiveflowID uuid.UUID
		expectRes          *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/activeflows/1ee42c4c-c8db-11ed-91cd-d30ce2e5c4b3/stop",
				Method: sock.RequestMethodPost,
			},

			responseActiveflow: &activeflow.Activeflow{
				ID: uuid.FromStringOrNil("1ee42c4c-c8db-11ed-91cd-d30ce2e5c4b3"),
			},

			expectActiveflowID: uuid.FromStringOrNil("1ee42c4c-c8db-11ed-91cd-d30ce2e5c4b3"),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1ee42c4c-c8db-11ed-91cd-d30ce2e5c4b3","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)
			mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				flowHandler:       mockFlowHandler,
				activeflowHandler: mockActive,
			}

			mockActive.EXPECT().Stop(gomock.Any(), tt.expectActiveflowID).Return(tt.responseActiveflow, nil)
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

func Test_v1ActiveflowsIDPushActionsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseActiveflow *activeflow.Activeflow

		expectActiveflowID uuid.UUID
		expectActions      []action.Action
		expectRes          *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/activeflows/636c6116-fb00-11ed-8c06-330f0ae26e9b/push_actions",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"actions":[{"type":"answer"}]}`),
			},

			responseActiveflow: &activeflow.Activeflow{
				ID: uuid.FromStringOrNil("636c6116-fb00-11ed-8c06-330f0ae26e9b"),
			},

			expectActiveflowID: uuid.FromStringOrNil("636c6116-fb00-11ed-8c06-330f0ae26e9b"),
			expectActions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"636c6116-fb00-11ed-8c06-330f0ae26e9b","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)
			mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				flowHandler:       mockFlowHandler,
				activeflowHandler: mockActive,
			}

			mockActive.EXPECT().PushActions(gomock.Any(), tt.expectActiveflowID, tt.expectActions).Return(tt.responseActiveflow, nil)
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
