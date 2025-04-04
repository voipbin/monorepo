package listenhandler

import (
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/flowhandler"
)

func Test_v1FlowsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		expectedCustomerID uuid.UUID
		expectedType       flow.Type
		expectedName       string
		expectedDetail     string
		expectedPersist    bool
		expectedActions    []action.Action
	}{
		{
			name: "empty actions",
			request: &sock.Request{
				URI:      "/v1/flows",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a356975a-8055-11ec-9c11-37c0ba53de51","type":"flow","name":"test","detail":"test detail","actions":[]}`),
			},

			expectedCustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			expectedType:       flow.TypeFlow,
			expectedName:       "test",
			expectedDetail:     "test detail",
			expectedPersist:    false,
			expectedActions:    []action.Action{},
		},
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/flows",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"flow","name":"test","detail":"test detail","actions":[{"type":"echo"},{"type":"talk","option":{"text":"hello world\nworld2"}}]}`),
			},
			expectedCustomerID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
			expectedType:       flow.TypeFlow,
			expectedName:       "test",
			expectedDetail:     "test detail",
			expectedPersist:    false,
			expectedActions: []action.Action{
				{
					Type: action.TypeEcho,
				},
				{
					Type: action.TypeTalk,
					Option: map[string]any{
						"text": "hello world\nworld2",
					},
				},
			},
		},
		{
			name: "has 2 actions",
			request: &sock.Request{
				URI:      "/v1/flows",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"flow","name":"test","detail":"test detail","actions":[{"type":"answer"},{"type":"echo"}]}`),
			},
			expectedCustomerID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
			expectedType:       flow.TypeFlow,
			expectedName:       "test",
			expectedDetail:     "test detail",
			expectedPersist:    false,
			expectedActions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type: action.TypeEcho,
				},
			},
		},
		{
			name: "has 2 actions with customer_id",
			request: &sock.Request{
				URI:      "/v1/flows",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"flow","name":"test","detail":"test detail","customer_id":"a356975a-8055-11ec-9c11-37c0ba53de51","actions":[{"type":"answer"},{"type":"echo"}]}`),
			},
			expectedCustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			expectedType:       flow.TypeFlow,
			expectedName:       "test",
			expectedDetail:     "test detail",
			expectedPersist:    false,
			expectedActions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type: action.TypeEcho,
				},
			},
		},
		{
			name: "type conference",
			request: &sock.Request{
				URI:      "/v1/flows",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"conference","name":"test","detail":"test detail","customer_id":"a356975a-8055-11ec-9c11-37c0ba53de51","actions":[{"type":"answer"},{"type":"echo"}]}`),
			},
			expectedCustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			expectedType:       flow.TypeConference,
			expectedName:       "test",
			expectedDetail:     "test detail",
			expectedPersist:    false,
			expectedActions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type: action.TypeEcho,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				flowHandler: mockFlowHandler,
			}

			mockFlowHandler.EXPECT().Create(gomock.Any(), tt.expectedCustomerID, tt.expectedType, tt.expectedName, tt.expectedDetail, tt.expectedPersist, tt.expectedActions).Return(&flow.Flow{}, nil)

			if _, err := h.processRequest(tt.request); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_v1FlowsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseFilters map[string]string
		responseFlows   []*flow.Flow

		expectedToken string
		expectedSize  uint64
		expectedRes   *sock.Response
	}{
		{
			name: "1 item",
			request: &sock.Request{
				URI:      "/v1/flows?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_customer_id=16d3fcf0-7f4c-11ec-a4c3-7bf43125108d&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			responseFilters: map[string]string{
				"customer_id": "16d3fcf0-7f4c-11ec-a4c3-7bf43125108d",
				"deleted":     "false",
			},

			responseFlows: []*flow.Flow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c64b621a-6c03-11ec-b44a-c7b5fb85cead"),
						CustomerID: uuid.FromStringOrNil("16d3fcf0-7f4c-11ec-a4c3-7bf43125108d"),
					},
					Type: flow.TypeFlow,
					Actions: []action.Action{
						{
							Type: action.TypeAnswer,
						},
					},
				},
			},

			expectedToken: "2020-10-10T03:30:17.000000",
			expectedSize:  10,
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c64b621a-6c03-11ec-b44a-c7b5fb85cead","customer_id":"16d3fcf0-7f4c-11ec-a4c3-7bf43125108d","type":"flow","name":"","detail":"","persist":false,"actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			name: "has various filters",
			request: &sock.Request{
				URI:      "/v1/flows?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_customer_id=16d3fcf0-7f4c-11ec-a4c3-7bf43125108d&filter_deleted=false&filter_type=flow",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			responseFilters: map[string]string{
				"customer_id": "16d3fcf0-7f4c-11ec-a4c3-7bf43125108d",
				"deleted":     "false",
				"type":        string(flow.TypeFlow),
			},

			responseFlows: []*flow.Flow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e1acb018-b099-11ee-b942-ebca8278ad69"),
						CustomerID: uuid.FromStringOrNil("16d3fcf0-7f4c-11ec-a4c3-7bf43125108d"),
					},
					Type: flow.TypeFlow,
					Actions: []action.Action{
						{
							Type: action.TypeAnswer,
						},
					},
				},
			},

			expectedToken: "2020-10-10T03:30:17.000000",
			expectedSize:  10,
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"e1acb018-b099-11ee-b942-ebca8278ad69","customer_id":"16d3fcf0-7f4c-11ec-a4c3-7bf43125108d","type":"flow","name":"","detail":"","persist":false,"actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			name: "2 items",
			request: &sock.Request{
				URI:      "/v1/flows?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_customer_id=2457d824-7f4c-11ec-9489-b3552a7c9d63",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			responseFilters: map[string]string{
				"customer_id": "2457d824-7f4c-11ec-9489-b3552a7c9d63",
			},

			responseFlows: []*flow.Flow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("13a7aeaa-0c4d-11eb-8210-073d8779e386"),
						CustomerID: uuid.FromStringOrNil("2457d824-7f4c-11ec-9489-b3552a7c9d63"),
					},
					Type: flow.TypeFlow,
					Actions: []action.Action{
						{
							Type: action.TypeAnswer,
						},
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("3645134e-0c4d-11eb-a2da-4bb8abe75c48"),
						CustomerID: uuid.FromStringOrNil("2457d824-7f4c-11ec-9489-b3552a7c9d63"),
					},
					Type: flow.TypeFlow,
					Actions: []action.Action{
						{
							Type: action.TypeEcho,
						},
					},
				},
			},

			expectedToken: "2020-10-10T03:30:17.000000",
			expectedSize:  10,
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"13a7aeaa-0c4d-11eb-8210-073d8779e386","customer_id":"2457d824-7f4c-11ec-9489-b3552a7c9d63","type":"flow","name":"","detail":"","persist":false,"actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""},{"id":"3645134e-0c4d-11eb-a2da-4bb8abe75c48","customer_id":"2457d824-7f4c-11ec-9489-b3552a7c9d63","type":"flow","name":"","detail":"","persist":false,"actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"echo"}],"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			name: "empty",
			request: &sock.Request{
				URI:      "/v1/flows?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_customer_id=3ee14bee-7f4c-11ec-a1d8-a3a488ed5885",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			responseFilters: map[string]string{
				"customer_id": "3ee14bee-7f4c-11ec-a1d8-a3a488ed5885",
			},
			responseFlows: []*flow.Flow{},

			expectedToken: "2020-10-10T03:30:17.000000",
			expectedSize:  10,
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
		{
			name: "type flow",
			request: &sock.Request{
				URI:      "/v1/flows?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_customer_id=49e66560-7f4c-11ec-9d15-2396902a0309&filter_type=flow",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			responseFilters: map[string]string{
				"customer_id": "49e66560-7f4c-11ec-9d15-2396902a0309",
				"type":        string(flow.TypeFlow),
			},
			responseFlows: []*flow.Flow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c64b621a-6c03-11ec-b44a-c7b5fb85cead"),
						CustomerID: uuid.FromStringOrNil("49e66560-7f4c-11ec-9d15-2396902a0309"),
					},
					Type: flow.TypeFlow,
					Actions: []action.Action{
						{
							Type: action.TypeAnswer,
						},
					},
				},
			},

			expectedToken: "2020-10-10T03:30:17.000000",
			expectedSize:  10,
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c64b621a-6c03-11ec-b44a-c7b5fb85cead","customer_id":"49e66560-7f4c-11ec-9d15-2396902a0309","type":"flow","name":"","detail":"","persist":false,"actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""}]`),
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

			h := &listenHandler{
				utilHandler: mockUtil,
				sockHandler: mockSock,
				flowHandler: mockFlowHandler,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockFlowHandler.EXPECT().Gets(gomock.Any(), tt.expectedToken, tt.expectedSize, tt.responseFilters).Return(tt.responseFlows, nil)

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

func Test_v1FlowsIDActionsIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseAction *action.Action

		expectedFlowID   uuid.UUID
		expectedActionID uuid.UUID
		expectedRes      *sock.Response
	}{
		{
			name: "empty actions",
			request: &sock.Request{
				URI:      "/v1/flows/c71bba06-8a77-11ea-93c7-47dc226c8c31/actions/67633cd6-0292-11f0-87f9-b37524aeeaea",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},

			responseAction: &action.Action{
				ID: uuid.FromStringOrNil("67633cd6-0292-11f0-87f9-b37524aeeaea"),
			},

			expectedFlowID:   uuid.FromStringOrNil("c71bba06-8a77-11ea-93c7-47dc226c8c31"),
			expectedActionID: uuid.FromStringOrNil("67633cd6-0292-11f0-87f9-b37524aeeaea"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"67633cd6-0292-11f0-87f9-b37524aeeaea","next_id":"00000000-0000-0000-0000-000000000000","type":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				flowHandler: mockFlowHandler,
			}

			mockFlowHandler.EXPECT().ActionGet(gomock.Any(), tt.expectedFlowID, tt.expectedActionID).Return(tt.responseAction, nil)
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

func Test_v1FlowsIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseFlow *flow.Flow

		expectedFlowID uuid.UUID
		expectedRes    *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/flows/01677a56-0c2d-11eb-96cb-eb2cd309ca81",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},

			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("01677a56-0c2d-11eb-96cb-eb2cd309ca81"),
				},
				Type: flow.TypeFlow,
				Actions: []action.Action{
					{
						Type: action.TypeAnswer,
					},
				},
			},

			expectedFlowID: uuid.FromStringOrNil("01677a56-0c2d-11eb-96cb-eb2cd309ca81"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"01677a56-0c2d-11eb-96cb-eb2cd309ca81","customer_id":"00000000-0000-0000-0000-000000000000","type":"flow","name":"","detail":"","persist":false,"actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			name: "persist true",
			request: &sock.Request{
				URI:      "/v1/flows/53b8aeb4-822b-11eb-82fe-a3c14b4e38de",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},

			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("53b8aeb4-822b-11eb-82fe-a3c14b4e38de"),
				},
				Type:    flow.TypeFlow,
				Persist: true,
				Actions: []action.Action{
					{
						Type: action.TypeAnswer,
					},
				},
			},

			expectedFlowID: uuid.FromStringOrNil("53b8aeb4-822b-11eb-82fe-a3c14b4e38de"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"53b8aeb4-822b-11eb-82fe-a3c14b4e38de","customer_id":"00000000-0000-0000-0000-000000000000","type":"flow","name":"","detail":"","persist":true,"actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				flowHandler: mockFlowHandler,
			}

			mockFlowHandler.EXPECT().Get(gomock.Any(), tt.responseFlow.ID).Return(tt.responseFlow, nil)

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

func Test_v1FlowsIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		expectedflowID  uuid.UUID
		expectedName    string
		expectedDetail  string
		expectedActions []action.Action

		responseFlow *flow.Flow
		expectedRes  *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/flows/b6768dd6-676f-11eb-8f00-7fb6aa43e2dc",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name","detail":"update detail","actions":[{"type":"answer"},{"type":"echo"}]}`),
			},

			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b6768dd6-676f-11eb-8f00-7fb6aa43e2dc"),
				},
				Name:   "update name",
				Detail: "update detail",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("559d044e-6770-11eb-8c51-eb96d1c14b35"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("561fa020-6770-11eb-b8ff-ef78ac0df0fb"),
						Type: action.TypeEcho,
					},
				},
			},

			expectedflowID: uuid.FromStringOrNil("b6768dd6-676f-11eb-8f00-7fb6aa43e2dc"),
			expectedName:   "update name",
			expectedDetail: "update detail",
			expectedActions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type: action.TypeEcho,
				},
			},
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b6768dd6-676f-11eb-8f00-7fb6aa43e2dc","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"update name","detail":"update detail","persist":false,"actions":[{"id":"559d044e-6770-11eb-8c51-eb96d1c14b35","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"},{"id":"561fa020-6770-11eb-b8ff-ef78ac0df0fb","next_id":"00000000-0000-0000-0000-000000000000","type":"echo"}],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				flowHandler: mockFlowHandler,
			}

			mockFlowHandler.EXPECT().Update(gomock.Any(), tt.expectedflowID, tt.expectedName, tt.expectedDetail, tt.expectedActions).Return(tt.responseFlow, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1FlowsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseFlow *flow.Flow

		expectedFlowID uuid.UUID
		expectedRes    *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/flows/89ecd1f6-67c6-11eb-815a-a75d4cc3df3e",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},

			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("89ecd1f6-67c6-11eb-815a-a75d4cc3df3e"),
				},
			},

			expectedFlowID: uuid.FromStringOrNil("89ecd1f6-67c6-11eb-815a-a75d4cc3df3e"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"89ecd1f6-67c6-11eb-815a-a75d4cc3df3e","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","persist":false,"actions":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				flowHandler: mockFlowHandler,
			}

			mockFlowHandler.EXPECT().Delete(gomock.Any(), tt.expectedFlowID).Return(tt.responseFlow, nil)

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

func Test_v1FlowsIDActionsPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseFlow *flow.Flow

		expectedFlowID  uuid.UUID
		expectedActions []action.Action
		expectedRes     *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/flows/b6768dd6-676f-11eb-8f00-7fb6aa43e2dc/actions",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"actions":[{"type":"answer"},{"type":"echo"}]}`),
			},

			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b6768dd6-676f-11eb-8f00-7fb6aa43e2dc"),
				},
				Name:   "update name",
				Detail: "update detail",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("559d044e-6770-11eb-8c51-eb96d1c14b35"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("561fa020-6770-11eb-b8ff-ef78ac0df0fb"),
						Type: action.TypeEcho,
					},
				},
			},

			expectedFlowID: uuid.FromStringOrNil("b6768dd6-676f-11eb-8f00-7fb6aa43e2dc"),
			expectedActions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type: action.TypeEcho,
				},
			},
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b6768dd6-676f-11eb-8f00-7fb6aa43e2dc","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"update name","detail":"update detail","persist":false,"actions":[{"id":"559d044e-6770-11eb-8c51-eb96d1c14b35","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"},{"id":"561fa020-6770-11eb-b8ff-ef78ac0df0fb","next_id":"00000000-0000-0000-0000-000000000000","type":"echo"}],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				flowHandler: mockFlowHandler,
			}

			mockFlowHandler.EXPECT().UpdateActions(gomock.Any(), tt.expectedFlowID, tt.expectedActions).Return(tt.responseFlow, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}
