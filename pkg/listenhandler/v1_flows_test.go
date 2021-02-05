package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/flow"
)

func TestFlowsPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	type test struct {
		name       string
		request    *rabbitmqhandler.Request
		expectFlow *flow.Flow
	}

	tests := []test{
		{
			"empty actions",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"name":"test","detail":"test detail","actions":[]}`),
			},
			&flow.Flow{
				Name:    "test",
				Detail:  "test detail",
				Actions: []action.Action{},
			},
		},
		{
			"has actions echo",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"name":"test","detail":"test detail","actions":[{"type":"echo"}]}`),
			},
			&flow.Flow{
				Name:   "test",
				Detail: "test detail",
				Actions: []action.Action{
					{
						Type: action.TypeEcho,
					},
				},
			},
		},
		{
			"has 2 actions",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"name":"test","detail":"test detail","actions":[{"type":"answer"},{"type":"echo"}]}`),
			},
			&flow.Flow{
				Name:   "test",
				Detail: "test detail",
				Actions: []action.Action{
					{
						Type: action.TypeAnswer,
					},
					{
						Type: action.TypeEcho,
					},
				},
			},
		},
		{
			"has 2 actions with user_id",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"name":"test","detail":"test detail","user_id":1,"actions":[{"type":"answer"},{"type":"echo"}]}`),
			},
			&flow.Flow{
				Name:   "test",
				Detail: "test detail",
				UserID: 1,
				Actions: []action.Action{
					{
						Type: action.TypeAnswer,
					},
					{
						Type: action.TypeEcho,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFlowHandler.EXPECT().FlowCreate(gomock.Any(), tt.expectFlow, false).Return(&flow.Flow{}, nil)

			h.processRequest(tt.request)
		})
	}
}

func TestV1FlowsIDActionsIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	type test struct {
		name           string
		request        *rabbitmqhandler.Request
		expectFlowID   uuid.UUID
		expectActionID uuid.UUID
		// expectFlow *flow.Flow
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

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	type test struct {
		name      string
		request   *rabbitmqhandler.Request
		flow      *flow.Flow
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows/01677a56-0c2d-11eb-96cb-eb2cd309ca81",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},
			&flow.Flow{
				ID: uuid.FromStringOrNil("01677a56-0c2d-11eb-96cb-eb2cd309ca81"),
				Actions: []action.Action{
					{
						Type: action.TypeAnswer,
					},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"01677a56-0c2d-11eb-96cb-eb2cd309ca81","user_id":0,"name":"","detail":"","persist":false,"actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFlowHandler.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(tt.flow, nil)

			// mockFlowHandler.EXPECT().ActionGet(gomock.Any(), tt.expectFlowID, tt.expectActionID).Return(nil, nil)
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

func TestV1FlowsIDPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	type test struct {
		name         string
		request      *rabbitmqhandler.Request
		flowID       uuid.UUID
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
			uuid.FromStringOrNil("b6768dd6-676f-11eb-8f00-7fb6aa43e2dc"),
			&flow.Flow{
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
				Data:       []byte(`{"id":"b6768dd6-676f-11eb-8f00-7fb6aa43e2dc","user_id":0,"name":"update name","detail":"update detail","persist":false,"actions":[{"id":"559d044e-6770-11eb-8c51-eb96d1c14b35","type":"answer"},{"id":"561fa020-6770-11eb-b8ff-ef78ac0df0fb","type":"echo"}],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFlowHandler.EXPECT().FlowUpdate(gomock.Any(), tt.flowID, tt.requestFlow).Return(tt.responseFlow, nil)

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

func TestV1FlowsGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	type test struct {
		name      string
		userID    uint64
		pageToken string
		pageSize  uint64
		request   *rabbitmqhandler.Request
		flows     []*flow.Flow

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			1,
			"2020-10-10T03:30:17.000000",
			10,
			&rabbitmqhandler.Request{
				URI:      "/v1/flows?page_token=2020-10-10T03:30:17.000000&page_size=10&user_id=1",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			[]*flow.Flow{
				{
					ID: uuid.FromStringOrNil("13a7aeaa-0c4d-11eb-8210-073d8779e386"),
					Actions: []action.Action{
						{
							Type: action.TypeAnswer,
						},
					},
				},
				{
					ID: uuid.FromStringOrNil("3645134e-0c4d-11eb-a2da-4bb8abe75c48"),
					Actions: []action.Action{
						{
							Type: action.TypeEcho,
						},
					},
				}},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"13a7aeaa-0c4d-11eb-8210-073d8779e386","user_id":0,"name":"","detail":"","persist":false,"actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""},{"id":"3645134e-0c4d-11eb-a2da-4bb8abe75c48","user_id":0,"name":"","detail":"","persist":false,"actions":[{"id":"00000000-0000-0000-0000-000000000000","type":"echo"}],"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"empty",
			1,
			"2020-10-10T03:30:17.000000",
			10,
			&rabbitmqhandler.Request{
				URI:      "/v1/flows?page_token=2020-10-10T03:30:17.000000&page_size=10&user_id=1",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFlowHandler.EXPECT().FlowGetsByUserID(gomock.Any(), tt.userID, tt.pageToken, tt.pageSize).Return(tt.flows, nil)

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
