package listenhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flowhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flowhandler/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/rabbitmq"
)

func TestFlowsGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	type test struct {
		name       string
		request    *rabbitmq.Request
		expectFlow *flow.Flow
	}

	tests := []test{
		{
			"empty actions",
			&rabbitmq.Request{
				URI:      "/v1/flows",
				Method:   rabbitmq.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"name":"test","detail":"test detail","actions":[]}`),
			},
			&flow.Flow{
				Name:    "test",
				Detail:  "test detail",
				Actions: []flow.Action{},
			},
		},
		{
			"has actions echo",
			&rabbitmq.Request{
				URI:      "/v1/flows",
				Method:   rabbitmq.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"name":"test","detail":"test detail","actions":[{"type":"echo"}]}`),
			},
			&flow.Flow{
				Name:   "test",
				Detail: "test detail",
				Actions: []flow.Action{
					{
						Type: flow.ActionTypeEcho,
					},
				},
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFlowHandler.EXPECT().FlowCreate(gomock.Any(), tt.expectFlow).Return(&flow.Flow{}, nil)
			h.processRequest(tt.request)
		})
	}
}

func TestV1FlowsIDActionsIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &listenHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		flowHandler: mockFlowHandler,
	}

	type test struct {
		name           string
		request        *rabbitmq.Request
		expectFlowID   uuid.UUID
		expectActionID uuid.UUID
		// expectFlow *flow.Flow
	}

	tests := []test{
		{
			"empty actions",
			&rabbitmq.Request{
				URI:      "/v1/flows/c71bba06-8a77-11ea-93c7-47dc226c8c31/actions/00000000-0000-0000-0000-000000000001",
				Method:   rabbitmq.RequestMethodGet,
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
