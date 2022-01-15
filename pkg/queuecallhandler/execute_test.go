package queuecallhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/notifyhandler"
)

func TestExecute(t *testing.T) {
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

	tests := []struct {
		name string

		queueCallID uuid.UUID

		queuecall *queuecall.Queuecall
		agents    []amagent.Agent

		responseQueuecall *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),
				UserID:          1,
				QueueID:         uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("b658394e-5ee0-11ec-92ba-5f2f2eabf000"),
				ForwardActionID: uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
				ExitActionID:    uuid.FromStringOrNil("d708bbbe-5ee0-11ec-aca3-530babc708dd"),
				ConfbridgeID:    uuid.FromStringOrNil("d7357136-5ee0-11ec-abd0-a7463d258061"),
				WebhookURI:      "test.com",
				WebhookMethod:   "",
				Source: cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a9ca8282-5edf-11ec-a876-df977791e643"),
				},

				Status: queuecall.StatusWait,
			},
			[]amagent.Agent{
				{
					ID:     uuid.FromStringOrNil("2e5a88c0-5ee1-11ec-b59f-cf1c32e0c3ea"),
					UserID: 1,
					Status: amagent.StatusAvailable,
				},
			},
			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),
				UserID:          1,
				QueueID:         uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("b658394e-5ee0-11ec-92ba-5f2f2eabf000"),
				ForwardActionID: uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
				ExitActionID:    uuid.FromStringOrNil("d708bbbe-5ee0-11ec-aca3-530babc708dd"),
				ConfbridgeID:    uuid.FromStringOrNil("d7357136-5ee0-11ec-abd0-a7463d258061"),
				WebhookURI:      "test.com",
				WebhookMethod:   "",
				Source: cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a9ca8282-5edf-11ec-a876-df977791e643"),
				},

				Status:    queuecall.StatusWait,
				TMCreate:  "2021-04-18 03:22:17.994000",
				TMService: "2021-04-18 03:52:17.994000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().QueuecallGet(gomock.Any(), tt.queueCallID).Return(tt.queuecall, nil)
			mockReq.EXPECT().AMV1AgentGetsByTagIDsAndStatus(gomock.Any(), tt.queuecall.UserID, tt.queuecall.TagIDs, amagent.StatusAvailable).Return(tt.agents, nil)
			mockReq.EXPECT().AMV1AgentDial(gomock.Any(), gomock.Any(), &tt.queuecall.Source, tt.queuecall.ConfbridgeID).Return(nil)
			mockReq.EXPECT().FMV1ActvieFlowUpdateForwardActionID(gomock.Any(), tt.queuecall.ReferenceID, tt.queuecall.ForwardActionID, true).Return(nil)
			mockDB.EXPECT().QueuecallSetServiceAgentID(gomock.Any(), tt.queuecall.ID, gomock.Any()).Return(nil)
			mockDB.EXPECT().QueuecallGet(gomock.Any(), tt.queueCallID).Return(tt.responseQueuecall, nil)
			mockNotify.EXPECT().NotifyEvent(gomock.Any(), notifyhandler.EventTypeQueuecallEntering, tt.responseQueuecall.WebhookURI, tt.responseQueuecall)

			h.Execute(ctx, tt.queueCallID)
		})
	}
}

func TestExecuteWithWrongStatus(t *testing.T) {
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

	tests := []struct {
		name string

		queueCallID uuid.UUID

		queuecall *queuecall.Queuecall
	}{
		{
			"status done",

			uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

				Status: queuecall.StatusDone,
			},
		},
		{
			"status abandoned",

			uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

				Status: queuecall.StatusAbandoned,
			},
		},
		{
			"status service",

			uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

				Status: queuecall.StatusService,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().QueuecallGet(gomock.Any(), tt.queueCallID).Return(tt.queuecall, nil)
			h.Execute(ctx, tt.queueCallID)
		})
	}
}

func TestExecuteWithWrongAgents(t *testing.T) {
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

	tests := []struct {
		name string

		queueCallID uuid.UUID

		queuecall  *queuecall.Queuecall
		agents     []amagent.Agent
		agentError error
	}{
		{
			"empty agents",

			uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

				Status: queuecall.StatusWait,
			},
			[]amagent.Agent{},
			nil,
		},
		{
			"error return",

			uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

				Status: queuecall.StatusWait,
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("f96eb0b4-5ef4-11ec-bd14-1f1d930e49d6"),
				},
			},
			fmt.Errorf(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().QueuecallGet(gomock.Any(), tt.queueCallID).Return(tt.queuecall, nil)
			mockReq.EXPECT().AMV1AgentGetsByTagIDsAndStatus(gomock.Any(), tt.queuecall.UserID, tt.queuecall.TagIDs, amagent.StatusAvailable).Return(tt.agents, tt.agentError)
			mockReq.EXPECT().QMV1QueuecallExecute(gomock.Any(), tt.queueCallID, 1000)

			h.Execute(ctx, tt.queueCallID)
		})
	}
}

func TestExecuteWithWrongRoutingMethod(t *testing.T) {
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

	tests := []struct {
		name string

		queueCallID uuid.UUID

		queuecall *queuecall.Queuecall
		agents    []amagent.Agent
	}{
		{
			"method none",

			uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

				Status:        queuecall.StatusWait,
				RoutingMethod: queue.RoutingMethodNone,
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("f96eb0b4-5ef4-11ec-bd14-1f1d930e49d6"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().QueuecallGet(gomock.Any(), tt.queueCallID).Return(tt.queuecall, nil)
			mockReq.EXPECT().AMV1AgentGetsByTagIDsAndStatus(gomock.Any(), tt.queuecall.UserID, tt.queuecall.TagIDs, amagent.StatusAvailable).Return(tt.agents, nil)
			mockReq.EXPECT().FMV1ActvieFlowUpdateForwardActionID(gomock.Any(), tt.queuecall.ReferenceID, tt.queuecall.ExitActionID, true).Return(nil)

			h.Execute(ctx, tt.queueCallID)
		})
	}
}

func TestExecuteWithAgentDialFailure(t *testing.T) {
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

	tests := []struct {
		name string

		queueCallID uuid.UUID

		queuecall *queuecall.Queuecall
		agents    []amagent.Agent
	}{
		{
			"method none",

			uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("b1c49460-5ede-11ec-9090-e3dad697e408"),

				Status:        queuecall.StatusWait,
				RoutingMethod: queue.RoutingMethodRandom,
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("f96eb0b4-5ef4-11ec-bd14-1f1d930e49d6"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().QueuecallGet(gomock.Any(), tt.queueCallID).Return(tt.queuecall, nil)
			mockReq.EXPECT().AMV1AgentGetsByTagIDsAndStatus(gomock.Any(), tt.queuecall.UserID, tt.queuecall.TagIDs, amagent.StatusAvailable).Return(tt.agents, nil)
			mockReq.EXPECT().AMV1AgentDial(gomock.Any(), tt.agents[0].ID, &tt.queuecall.Source, tt.queuecall.ConfbridgeID).Return(fmt.Errorf(""))

			mockReq.EXPECT().QMV1QueuecallExecute(gomock.Any(), tt.queuecall.ID, 1000)

			h.Execute(ctx, tt.queueCallID)
		})
	}
}
