package msgreceiver

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	dbhandler "gitlab.com/voipbin/bin-manager/flow-manager/pkg/db_handler"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flow"
	flowhandler "gitlab.com/voipbin/bin-manager/flow-manager/pkg/flow_handler"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/rabbitmq"
)

func TestFlowsGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &msgReceiver{
		db:          mockDB,
		sock:        mockSock,
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
				Data:     `{"name":"test","detail":"test detail","actions":[]}`,
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
				Data:     `{"name":"test","detail":"test detail","actions":[{"type":"echo"}]}`,
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
			h.process(tt.request)
		})
	}
}

func TestV1FlowsIDActionsIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

	h := &msgReceiver{
		db:          mockDB,
		sock:        mockSock,
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
				Data:     `{}`,
			},
			uuid.FromStringOrNil("c71bba06-8a77-11ea-93c7-47dc226c8c31"),
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFlowHandler.EXPECT().ActionGet(gomock.Any(), tt.expectFlowID, gomock.Any(), tt.expectActionID).Return(nil, nil)
			res, err := h.process(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Wrong match. expect: 200, got: %d", res.StatusCode)
			}
		})
	}
}
