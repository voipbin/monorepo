package listenhandler

import (
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
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

		responseActiveflow *activeflow.Activeflow

		expectedID                    uuid.UUID
		expectedCustomerID            uuid.UUID
		expectedRefereceType          activeflow.ReferenceType
		expectedRefereceID            uuid.UUID
		expectedReferenceActiveflowID uuid.UUID
		expectedFlowID                uuid.UUID
		expectedRes                   *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/activeflows",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"a508739b-d98d-40fb-8a47-61e9a70958cd","customer_id":"23858d7a-049a-11f0-aa02-8f574d89004b","reference_type": "call", "reference_id": "b66c4922-a7a4-11ec-8e1b-6765ceec0323", "reference_activeflow_id": "b38d3d0e-07d4-11f0-9884-dbff88cf1fea", "flow_id": "24092c98-05ee-11eb-a410-17d716ff3d61"}`),
			},

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a508739b-d98d-40fb-8a47-61e9a70958cd"),
					CustomerID: uuid.FromStringOrNil("cd607242-7f4b-11ec-a34f-bb861637ee36"),
				},

				FlowID: uuid.FromStringOrNil("24092c98-05ee-11eb-a410-17d716ff3d61"),

				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b66c4922-a7a4-11ec-8e1b-6765ceec0323"),

				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ExecuteCount:    0,
				ForwardActionID: action.IDEmpty,
			},

			expectedID:                    uuid.FromStringOrNil("a508739b-d98d-40fb-8a47-61e9a70958cd"),
			expectedCustomerID:            uuid.FromStringOrNil("23858d7a-049a-11f0-aa02-8f574d89004b"),
			expectedRefereceType:          activeflow.ReferenceTypeCall,
			expectedRefereceID:            uuid.FromStringOrNil("b66c4922-a7a4-11ec-8e1b-6765ceec0323"),
			expectedReferenceActiveflowID: uuid.FromStringOrNil("b38d3d0e-07d4-11f0-9884-dbff88cf1fea"),
			expectedFlowID:                uuid.FromStringOrNil("24092c98-05ee-11eb-a410-17d716ff3d61"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a508739b-d98d-40fb-8a47-61e9a70958cd","customer_id":"cd607242-7f4b-11ec-a34f-bb861637ee36","flow_id":"24092c98-05ee-11eb-a410-17d716ff3d61","status":"","reference_type":"call","reference_id":"b66c4922-a7a4-11ec-8e1b-6765ceec0323","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000001","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			name: "empty id",
			request: &sock.Request{
				URI:      "/v1/activeflows",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"2400ed44-049a-11f0-9b34-f37b0677df6b","reference_type": "call", "reference_id": "b66c4922-a7a4-11ec-8e1b-6765ceec0323", "flow_id": "24092c98-05ee-11eb-a410-17d716ff3d61"}`),
			},

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7a18e7e0-7d67-44f2-9591-58cc7d8f5610"),
					CustomerID: uuid.FromStringOrNil("cd607242-7f4b-11ec-a34f-bb861637ee36"),
				},

				FlowID: uuid.FromStringOrNil("24092c98-05ee-11eb-a410-17d716ff3d61"),

				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b66c4922-a7a4-11ec-8e1b-6765ceec0323"),

				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ExecuteCount:    0,
				ForwardActionID: action.IDEmpty,
			},

			expectedID:                    uuid.Nil,
			expectedCustomerID:            uuid.FromStringOrNil("2400ed44-049a-11f0-9b34-f37b0677df6b"),
			expectedRefereceType:          activeflow.ReferenceTypeCall,
			expectedRefereceID:            uuid.FromStringOrNil("b66c4922-a7a4-11ec-8e1b-6765ceec0323"),
			expectedReferenceActiveflowID: uuid.Nil,
			expectedFlowID:                uuid.FromStringOrNil("24092c98-05ee-11eb-a410-17d716ff3d61"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7a18e7e0-7d67-44f2-9591-58cc7d8f5610","customer_id":"cd607242-7f4b-11ec-a34f-bb861637ee36","flow_id":"24092c98-05ee-11eb-a410-17d716ff3d61","status":"","reference_type":"call","reference_id":"b66c4922-a7a4-11ec-8e1b-6765ceec0323","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000001","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockActive.EXPECT().Create(gomock.Any(), tt.expectedID, tt.expectedCustomerID, tt.expectedRefereceType, tt.expectedRefereceID, tt.expectedReferenceActiveflowID, tt.expectedFlowID).Return(tt.responseActiveflow, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1ActiveflowsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseFilters     map[string]string
		responseActiveflows []*activeflow.Activeflow

		expectedToken string
		expectedSize  uint64
		expectedRes   *sock.Response
	}{
		{
			name: "1 item",
			request: &sock.Request{
				URI:    "/v1/activeflows?page_token=2020-10-10%2003:30:17.000000&page_size=10&filter_customer_id=16d3fcf0-7f4c-11ec-a4c3-7bf43125108d&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			responseFilters: map[string]string{
				"customer_id": "16d3fcf0-7f4c-11ec-a4c3-7bf43125108d",
				"deleted":     "false",
			},
			responseActiveflows: []*activeflow.Activeflow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ae07a96a-cbda-11ed-bbc9-5324a1a8b94b"),
						CustomerID: uuid.FromStringOrNil("16d3fcf0-7f4c-11ec-a4c3-7bf43125108d"),
					},
				},
			},

			expectedToken: "2020-10-10 03:30:17.000000",
			expectedSize:  10,
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ae07a96a-cbda-11ed-bbc9-5324a1a8b94b","customer_id":"16d3fcf0-7f4c-11ec-a4c3-7bf43125108d","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			name: "2 items",
			request: &sock.Request{
				URI:    "/v1/activeflows?page_token=2020-10-10%2003:30:17.000000&page_size=10&filter_customer_id=2457d824-7f4c-11ec-9489-b3552a7c9d63&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			responseFilters: map[string]string{
				"customer_id": "2457d824-7f4c-11ec-9489-b3552a7c9d63",
				"deleted":     "false",
			},
			responseActiveflows: []*activeflow.Activeflow{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ae3724b0-cbda-11ed-a44c-1be0474024bd"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ae66a7bc-cbda-11ed-81b3-17eb5a6f42b2"),
					},
				},
			},

			expectedToken: "2020-10-10 03:30:17.000000",
			expectedSize:  10,
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ae3724b0-cbda-11ed-a44c-1be0474024bd","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"ae66a7bc-cbda-11ed-81b3-17eb5a6f42b2","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			name: "empty",
			request: &sock.Request{
				URI:      "/v1/activeflows?page_token=2020-10-10%2003:30:17.000000&page_size=10&filter_customer_id=3ee14bee-7f4c-11ec-a1d8-a3a488ed5885&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			responseFilters: map[string]string{
				"customer_id": "3ee14bee-7f4c-11ec-a1d8-a3a488ed5885",
				"deleted":     "false",
			},
			responseActiveflows: []*activeflow.Activeflow{},

			expectedToken: "2020-10-10 03:30:17.000000",
			expectedSize:  10,
			expectedRes: &sock.Response{
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
			mockActiveflowHandler.EXPECT().Gets(gomock.Any(), tt.expectedToken, tt.expectedSize, tt.responseFilters).Return(tt.responseActiveflows, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1ActiveflowsIDNextGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseAction action.Action

		expectedCallID          uuid.UUID
		expectedCurrentActionID uuid.UUID
		expectedRes             *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/activeflows/cec5b926-06a7-11eb-967e-fb463343f0a5/next",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"current_action_id": "6a1ce642-06a8-11eb-a632-978be835f982"}`),
			},

			responseAction: action.Action{
				ID:   uuid.FromStringOrNil("63698276-06ab-11eb-9cbf-c771a09c1619"),
				Type: action.TypeEcho,
			},

			expectedCallID:          uuid.FromStringOrNil("cec5b926-06a7-11eb-967e-fb463343f0a5"),
			expectedCurrentActionID: uuid.FromStringOrNil("6a1ce642-06a8-11eb-a632-978be835f982"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"63698276-06ab-11eb-9cbf-c771a09c1619","next_id":"00000000-0000-0000-0000-000000000000","type":"echo"}`),
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

			mockActive.EXPECT().ExecuteNextAction(gomock.Any(), tt.expectedCallID, tt.expectedCurrentActionID).Return(&tt.responseAction, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1ActiveflowsIDForwardActionIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		expectedCallID          uuid.UUID
		expectedForwardActionID uuid.UUID
		expectedForwardNow      bool

		expectedRes *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/activeflows/6f14f3b8-5758-11ec-a413-772c32e3e51f/forward_action_id",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"forward_action_id": "6732dd5e-5758-11ec-92b1-bfe33ab190aa", "forward_now": true}`),
			},

			expectedCallID:          uuid.FromStringOrNil("6f14f3b8-5758-11ec-a413-772c32e3e51f"),
			expectedForwardActionID: uuid.FromStringOrNil("6732dd5e-5758-11ec-92b1-bfe33ab190aa"),
			expectedForwardNow:      true,
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
		{
			name: "forward now false",

			request: &sock.Request{
				URI:      "/v1/activeflows/6f14f3b8-5758-11ec-a413-772c32e3e51f/forward_action_id",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"forward_action_id": "6732dd5e-5758-11ec-92b1-bfe33ab190aa", "forward_now": false}`),
			},

			expectedCallID:          uuid.FromStringOrNil("6f14f3b8-5758-11ec-a413-772c32e3e51f"),
			expectedForwardActionID: uuid.FromStringOrNil("6732dd5e-5758-11ec-92b1-bfe33ab190aa"),
			expectedForwardNow:      false,
			expectedRes: &sock.Response{
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

			mockActive.EXPECT().SetForwardActionID(gomock.Any(), tt.expectedCallID, tt.expectedForwardActionID, tt.expectedForwardNow).Return(nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1ActiveflowsIDExecutePost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		activeflowID uuid.UUID

		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/activeflows/07c60d7c-a7ae-11ec-ad69-c3e765668a2b/execute",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},

			activeflowID: uuid.FromStringOrNil("07c60d7c-a7ae-11ec-ad69-c3e765668a2b"),

			expectedRes: &sock.Response{
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

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1ActiveflowsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseActiveflow *activeflow.Activeflow

		expectedActiveflowID uuid.UUID
		expectedRes          *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/activeflows/343b6e40-cbdb-11ed-b13d-9f730017f25a",
				Method: sock.RequestMethodGet,
			},

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("343b6e40-cbdb-11ed-b13d-9f730017f25a"),
				},
			},

			expectedActiveflowID: uuid.FromStringOrNil("343b6e40-cbdb-11ed-b13d-9f730017f25a"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"343b6e40-cbdb-11ed-b13d-9f730017f25a","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockActive.EXPECT().Get(gomock.Any(), tt.expectedActiveflowID).Return(tt.responseActiveflow, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1ActiveflowsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseActiveflow *activeflow.Activeflow

		expectedActiveflowID uuid.UUID
		expectedRes          *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/activeflows/4356d70a-adde-11ec-bff4-9fc5420b5bcb",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4356d70a-adde-11ec-bff4-9fc5420b5bcb"),
				},
			},

			expectedActiveflowID: uuid.FromStringOrNil("4356d70a-adde-11ec-bff4-9fc5420b5bcb"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4356d70a-adde-11ec-bff4-9fc5420b5bcb","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockActive.EXPECT().Delete(gomock.Any(), tt.expectedActiveflowID).Return(tt.responseActiveflow, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1ActiveflowsIDStopPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseActiveflow *activeflow.Activeflow

		expectedActiveflowID uuid.UUID
		expectedRes          *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/activeflows/1ee42c4c-c8db-11ed-91cd-d30ce2e5c4b3/stop",
				Method: sock.RequestMethodPost,
			},

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1ee42c4c-c8db-11ed-91cd-d30ce2e5c4b3"),
				},
			},

			expectedActiveflowID: uuid.FromStringOrNil("1ee42c4c-c8db-11ed-91cd-d30ce2e5c4b3"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1ee42c4c-c8db-11ed-91cd-d30ce2e5c4b3","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockActive.EXPECT().Stop(gomock.Any(), tt.expectedActiveflowID).Return(tt.responseActiveflow, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1ActiveflowsIDAddActionsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseActiveflow *activeflow.Activeflow

		expectedActiveflowID uuid.UUID
		expectedActions      []action.Action
		expectedRes          *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/activeflows/28c9a8b8-03f8-11f0-89f4-cbeb7b84f217/add_actions",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"actions":[{"type":"answer"}]}`),
			},

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("28c9a8b8-03f8-11f0-89f4-cbeb7b84f217"),
				},
			},

			expectedActiveflowID: uuid.FromStringOrNil("28c9a8b8-03f8-11f0-89f4-cbeb7b84f217"),
			expectedActions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
			},
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"28c9a8b8-03f8-11f0-89f4-cbeb7b84f217","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockActive.EXPECT().AddActions(gomock.Any(), tt.expectedActiveflowID, tt.expectedActions).Return(tt.responseActiveflow, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1ActiveflowsIDPushActionsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseActiveflow *activeflow.Activeflow

		expectedActiveflowID uuid.UUID
		expectedActions      []action.Action
		expectedRes          *sock.Response
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("636c6116-fb00-11ed-8c06-330f0ae26e9b"),
				},
			},

			expectedActiveflowID: uuid.FromStringOrNil("636c6116-fb00-11ed-8c06-330f0ae26e9b"),
			expectedActions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
			},
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"636c6116-fb00-11ed-8c06-330f0ae26e9b","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","reference_activeflow_id":"00000000-0000-0000-0000-000000000000","stack_map":null,"current_stack_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_stack_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","execute_count":0,"executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockActive.EXPECT().PushActions(gomock.Any(), tt.expectedActiveflowID, tt.expectedActions).Return(tt.responseActiveflow, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1ActiveflowsIDServiceStop(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		expectedActiveflowID uuid.UUID
		expectedServiceID    uuid.UUID
		expectedRes          *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/activeflows/b25c4a40-f9e9-11ef-93bd-cf01bb7d261a/service_stop",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"service_id":"b2bc0b7e-f9e9-11ef-81d3-430e6fb7f082"}`),
			},

			expectedActiveflowID: uuid.FromStringOrNil("b25c4a40-f9e9-11ef-93bd-cf01bb7d261a"),
			expectedServiceID:    uuid.FromStringOrNil("b2bc0b7e-f9e9-11ef-81d3-430e6fb7f082"),
			expectedRes: &sock.Response{
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

			mockActive.EXPECT().ServiceStop(gomock.Any(), tt.expectedActiveflowID, tt.expectedServiceID).Return(nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}
