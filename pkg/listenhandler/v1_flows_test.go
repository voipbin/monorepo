package listenhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/flow"
)

func TestFlowsGet(t *testing.T) {
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
