package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler"
)

func TestFlowsPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		customerID uuid.UUID
		flowType   flow.Type
		flowName   string
		detail     string
		persist    bool
		webhookURI string
		actions    []action.Action
	}

	tests := []test{
		{
			"empty actions",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a356975a-8055-11ec-9c11-37c0ba53de51","type":"flow","name":"test","detail":"test detail","actions":[]}`),
			},

			uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			flow.TypeFlow,
			"test",
			"test detail",
			false,
			"",
			[]action.Action{},
		},
		{
			"has actions echo",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"flow","name":"test","detail":"test detail","actions":[{"type":"echo"}]}`),
			},
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
			flow.TypeFlow,
			"test",
			"test detail",
			false,
			"",
			[]action.Action{
				{
					Type: action.TypeEcho,
				},
			},
		},
		{
			"has 2 actions",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"flow","name":"test","detail":"test detail","actions":[{"type":"answer"},{"type":"echo"}]}`),
			},
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
			flow.TypeFlow,
			"test",
			"test detail",
			false,
			"",
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type: action.TypeEcho,
				},
			},
		},
		{
			"has 2 actions with customer_id",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"flow","name":"test","detail":"test detail","customer_id":"a356975a-8055-11ec-9c11-37c0ba53de51","actions":[{"type":"answer"},{"type":"echo"}]}`),
			},
			uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			flow.TypeFlow,
			"test",
			"test detail",
			false,
			"",
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type: action.TypeEcho,
				},
			},
		},
		{
			"type conference",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"conference","name":"test","detail":"test detail","customer_id":"a356975a-8055-11ec-9c11-37c0ba53de51","actions":[{"type":"answer"},{"type":"echo"}]}`),
			},
			uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			flow.TypeConference,
			"test",
			"test detail",
			false,
			"",
			[]action.Action{
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
			mockFlowHandler.EXPECT().FlowCreate(gomock.Any(), tt.customerID, tt.flowType, tt.flowName, tt.detail, tt.persist, tt.webhookURI, tt.actions).Return(&flow.Flow{}, nil)

			if _, err := h.processRequest(tt.request); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestV1FlowsGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	type test struct {
		name       string
		customerID uuid.UUID
		flowType   flow.Type
		pageToken  string
		pageSize   uint64
		request    *rabbitmqhandler.Request
		flows      []*flow.Flow

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"1 item",
			uuid.FromStringOrNil("16d3fcf0-7f4c-11ec-a4c3-7bf43125108d"),
			flow.TypeNone,
			"2020-10-10T03:30:17.000000",
			10,
			&rabbitmqhandler.Request{
				URI:      "/v1/flows?page_token=2020-10-10T03:30:17.000000&page_size=10&customer_id=16d3fcf0-7f4c-11ec-a4c3-7bf43125108d",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			[]*flow.Flow{
				{
					ID:         uuid.FromStringOrNil("c64b621a-6c03-11ec-b44a-c7b5fb85cead"),
					CustomerID: uuid.FromStringOrNil("16d3fcf0-7f4c-11ec-a4c3-7bf43125108d"),
					Type:       flow.TypeFlow,
					Actions: []action.Action{
						{
							Type: action.TypeAnswer,
						},
					},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c64b621a-6c03-11ec-b44a-c7b5fb85cead","customer_id":"16d3fcf0-7f4c-11ec-a4c3-7bf43125108d","type":"flow","name":"","detail":"","persist":false,"webhook_uri":"","actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"2 items",
			uuid.FromStringOrNil("2457d824-7f4c-11ec-9489-b3552a7c9d63"),
			flow.TypeNone,
			"2020-10-10T03:30:17.000000",
			10,
			&rabbitmqhandler.Request{
				URI:      "/v1/flows?page_token=2020-10-10T03:30:17.000000&page_size=10&customer_id=2457d824-7f4c-11ec-9489-b3552a7c9d63",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			[]*flow.Flow{
				{
					ID:         uuid.FromStringOrNil("13a7aeaa-0c4d-11eb-8210-073d8779e386"),
					CustomerID: uuid.FromStringOrNil("2457d824-7f4c-11ec-9489-b3552a7c9d63"),
					Type:       flow.TypeFlow,
					Actions: []action.Action{
						{
							Type: action.TypeAnswer,
						},
					},
				},
				{
					ID:         uuid.FromStringOrNil("3645134e-0c4d-11eb-a2da-4bb8abe75c48"),
					CustomerID: uuid.FromStringOrNil("2457d824-7f4c-11ec-9489-b3552a7c9d63"),
					Type:       flow.TypeFlow,
					Actions: []action.Action{
						{
							Type: action.TypeEcho,
						},
					},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"13a7aeaa-0c4d-11eb-8210-073d8779e386","customer_id":"2457d824-7f4c-11ec-9489-b3552a7c9d63","type":"flow","name":"","detail":"","persist":false,"webhook_uri":"","actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""},{"id":"3645134e-0c4d-11eb-a2da-4bb8abe75c48","customer_id":"2457d824-7f4c-11ec-9489-b3552a7c9d63","type":"flow","name":"","detail":"","persist":false,"webhook_uri":"","actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"echo"}],"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"empty",
			uuid.FromStringOrNil("3ee14bee-7f4c-11ec-a1d8-a3a488ed5885"),
			flow.TypeNone,
			"2020-10-10T03:30:17.000000",
			10,
			&rabbitmqhandler.Request{
				URI:      "/v1/flows?page_token=2020-10-10T03:30:17.000000&page_size=10&customer_id=3ee14bee-7f4c-11ec-a1d8-a3a488ed5885",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			[]*flow.Flow{},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
		{
			"type flow",
			uuid.FromStringOrNil("49e66560-7f4c-11ec-9d15-2396902a0309"),
			flow.TypeFlow,
			"2020-10-10T03:30:17.000000",
			10,
			&rabbitmqhandler.Request{
				URI:      "/v1/flows?page_token=2020-10-10T03:30:17.000000&page_size=10&customer_id=49e66560-7f4c-11ec-9d15-2396902a0309&type=flow",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			[]*flow.Flow{
				{
					ID:         uuid.FromStringOrNil("c64b621a-6c03-11ec-b44a-c7b5fb85cead"),
					CustomerID: uuid.FromStringOrNil("49e66560-7f4c-11ec-9d15-2396902a0309"),
					Type:       flow.TypeFlow,
					Actions: []action.Action{
						{
							Type: action.TypeAnswer,
						},
					},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c64b621a-6c03-11ec-b44a-c7b5fb85cead","customer_id":"49e66560-7f4c-11ec-9d15-2396902a0309","type":"flow","name":"","detail":"","persist":false,"webhook_uri":"","actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.flowType != flow.TypeNone {
				mockFlowHandler.EXPECT().FlowGetsByType(gomock.Any(), tt.customerID, tt.flowType, tt.pageToken, tt.pageSize).Return(tt.flows, nil)
			} else {
				mockFlowHandler.EXPECT().FlowGets(gomock.Any(), tt.customerID, tt.pageToken, tt.pageSize).Return(tt.flows, nil)
			}

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

func TestV1FlowsIDActionsIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	type test struct {
		name           string
		request        *rabbitmqhandler.Request
		expectFlowID   uuid.UUID
		expectActionID uuid.UUID
	}

	tests := []test{
		{
			"empty actions",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows/c71bba06-8a77-11ea-93c7-47dc226c8c31/actions/00000000-0000-0000-0000-000000000001",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},
			uuid.FromStringOrNil("c71bba06-8a77-11ea-93c7-47dc226c8c31"),
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFlowHandler.EXPECT().ActionGet(gomock.Any(), tt.expectFlowID, tt.expectActionID).Return(nil, nil)
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

func TestV1FlowsIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	tests := []struct {
		name      string
		request   *rabbitmqhandler.Request
		flow      *flow.Flow
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows/01677a56-0c2d-11eb-96cb-eb2cd309ca81",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},
			&flow.Flow{
				ID:   uuid.FromStringOrNil("01677a56-0c2d-11eb-96cb-eb2cd309ca81"),
				Type: flow.TypeFlow,
				Actions: []action.Action{
					{
						Type: action.TypeAnswer,
					},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"01677a56-0c2d-11eb-96cb-eb2cd309ca81","customer_id":"00000000-0000-0000-0000-000000000000","type":"flow","name":"","detail":"","persist":false,"webhook_uri":"","actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"persist true",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows/53b8aeb4-822b-11eb-82fe-a3c14b4e38de",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},
			&flow.Flow{
				ID:      uuid.FromStringOrNil("53b8aeb4-822b-11eb-82fe-a3c14b4e38de"),
				Type:    flow.TypeFlow,
				Persist: true,
				Actions: []action.Action{
					{
						Type: action.TypeAnswer,
					},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"53b8aeb4-822b-11eb-82fe-a3c14b4e38de","customer_id":"00000000-0000-0000-0000-000000000000","type":"flow","name":"","detail":"","persist":true,"webhook_uri":"","actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"webhook uri set",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows/53b8aeb4-822b-11eb-82fe-a3c14b4e38de",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},
			&flow.Flow{
				ID:         uuid.FromStringOrNil("53b8aeb4-822b-11eb-82fe-a3c14b4e38de"),
				Type:       flow.TypeFlow,
				Persist:    true,
				WebhookURI: "http://pchero21.com/test_webhook",
				Actions: []action.Action{
					{
						Type: action.TypeAnswer,
					},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"53b8aeb4-822b-11eb-82fe-a3c14b4e38de","customer_id":"00000000-0000-0000-0000-000000000000","type":"flow","name":"","detail":"","persist":true,"webhook_uri":"http://pchero21.com/test_webhook","actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFlowHandler.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(tt.flow, nil)

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

func TestV1FlowsIDPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	type test struct {
		name         string
		request      *rabbitmqhandler.Request
		requestFlow  *flow.Flow
		responseFlow *flow.Flow
		expectRes    *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows/b6768dd6-676f-11eb-8f00-7fb6aa43e2dc",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name","detail":"update detail","actions":[{"type":"answer"},{"type":"echo"}]}`),
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("b6768dd6-676f-11eb-8f00-7fb6aa43e2dc"),
				Name:   "update name",
				Detail: "update detail",
				Actions: []action.Action{
					{
						Type: action.TypeAnswer,
					},
					{
						Type: action.TypeEcho,
					},
				},
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("b6768dd6-676f-11eb-8f00-7fb6aa43e2dc"),
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b6768dd6-676f-11eb-8f00-7fb6aa43e2dc","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"update name","detail":"update detail","persist":false,"webhook_uri":"","actions":[{"id":"559d044e-6770-11eb-8c51-eb96d1c14b35","type":"answer"},{"id":"561fa020-6770-11eb-b8ff-ef78ac0df0fb","type":"echo"}],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"webhook uri update",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows/2fea2826-822d-11eb-8bcc-97bfc5739d38",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name","detail":"update detail","webhook_uri":"https://test.com/update_webhook","actions":[{"type":"answer"}]}`),
			},
			&flow.Flow{
				ID:         uuid.FromStringOrNil("2fea2826-822d-11eb-8bcc-97bfc5739d38"),
				Name:       "update name",
				Detail:     "update detail",
				WebhookURI: "https://test.com/update_webhook",
				Actions: []action.Action{
					{
						Type: action.TypeAnswer,
					},
				},
			},
			&flow.Flow{
				ID:         uuid.FromStringOrNil("2fea2826-822d-11eb-8bcc-97bfc5739d38"),
				Name:       "update name",
				Detail:     "update detail",
				WebhookURI: "https://test.com/update_webhook",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("3aa85b98-822d-11eb-9020-e7d103dc0571"),
						Type: action.TypeAnswer,
					},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2fea2826-822d-11eb-8bcc-97bfc5739d38","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"update name","detail":"update detail","persist":false,"webhook_uri":"https://test.com/update_webhook","actions":[{"id":"3aa85b98-822d-11eb-9020-e7d103dc0571","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFlowHandler.EXPECT().FlowUpdate(gomock.Any(), tt.requestFlow).Return(tt.responseFlow, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestV1FlowsIDDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	type test struct {
		name      string
		flowID    uuid.UUID
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("89ecd1f6-67c6-11eb-815a-a75d4cc3df3e"),
			&rabbitmqhandler.Request{
				URI:      "/v1/flows/89ecd1f6-67c6-11eb-815a-a75d4cc3df3e",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFlowHandler.EXPECT().FlowDelete(gomock.Any(), tt.flowID).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
