package queuecallhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

func Test_Execute(t *testing.T) {
	tests := []struct {
		name string

		id      uuid.UUID
		agentID uuid.UUID

		responseQueuecall *queuecall.Queuecall
		responseFlow      *fmflow.Flow

		expcetFlowActions  []fmaction.Action
		expectDestinations []commonaddress.Address
	}{
		{
			name: "normal",

			id:      uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),
			agentID: uuid.FromStringOrNil("624e1cd6-d1b0-11ec-8b3b-db12aa2e35f6"),

			responseQueuecall: &queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),
				QueueID:         uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("b658394e-5ee0-11ec-92ba-5f2f2eabf000"),
				ForwardActionID: uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
				ExitActionID:    uuid.FromStringOrNil("d708bbbe-5ee0-11ec-aca3-530babc708dd"),
				ConfbridgeID:    uuid.FromStringOrNil("d7357136-5ee0-11ec-abd0-a7463d258061"),
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821021656521",
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a9ca8282-5edf-11ec-a876-df977791e643"),
				},

				Status: queuecall.StatusWaiting,
			},
			responseFlow: &fmflow.Flow{
				ID: uuid.FromStringOrNil("af9486dc-d1b1-11ec-b34e-8fea9e29488f"),
			},

			expcetFlowActions: []fmaction.Action{
				{
					Type:   fmaction.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"d7357136-5ee0-11ec-abd0-a7463d258061"}`),
				},
			},
			expectDestinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeAgent,
					Target: "624e1cd6-d1b0-11ec-8b3b-db12aa2e35f6",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queuecallHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().QueuecallGet(ctx, tt.id).Return(tt.responseQueuecall, nil)

			// generateFlowForAgentCall
			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.responseQueuecall.CustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.expcetFlowActions, false).Return(tt.responseFlow, nil)

			mockReq.EXPECT().CallV1CallsCreate(ctx, tt.responseQueuecall.CustomerID, tt.responseFlow.ID, tt.responseQueuecall.ReferenceID, &tt.responseQueuecall.Source, tt.expectDestinations, false, false).Return([]*cmcall.Call{}, []*cmgroupcall.Groupcall{}, nil)

			//UpdateStatusConnecting
			mockDB.EXPECT().QueuecallSetStatusConnecting(ctx, tt.responseQueuecall.ID, tt.agentID).Return(nil)
			mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallConnecting, tt.responseQueuecall)
			mockReq.EXPECT().FlowV1ActiveflowUpdateForwardActionID(ctx, tt.responseQueuecall.ReferenceActiveflowID, tt.responseQueuecall.ForwardActionID, true).Return(nil)

			res, err := h.Execute(ctx, tt.id, tt.agentID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueuecall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseQueuecall, res)
			}
		})
	}
}

func Test_generateFlowForAgentCall(t *testing.T) {
	tests := []struct {
		name string

		customerID   uuid.UUID
		conferenceID uuid.UUID

		expectActions []fmaction.Action

		responseFlow *fmflow.Flow
	}{
		{
			"normal",

			uuid.FromStringOrNil("f3f276e4-d1b2-11ec-9c34-b7d7e11bf2e2"),
			uuid.FromStringOrNil("f42361d2-d1b2-11ec-8303-5baaf068dbab"),

			[]fmaction.Action{
				{
					Type:   fmaction.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"f42361d2-d1b2-11ec-8303-5baaf068dbab"}`),
				},
			},

			&fmflow.Flow{
				ID: uuid.FromStringOrNil("0420912c-d1b3-11ec-8366-c37da4c77e2a"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queuecallHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.customerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.expectActions, false).Return(tt.responseFlow, nil)

			res, err := h.generateFlowForAgentCall(ctx, tt.customerID, tt.conferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseFlow, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFlow, res)
			}
		})
	}
}
