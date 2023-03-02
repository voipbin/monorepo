package agenthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/dbhandler"
	cmgroupdial "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
)

func Test_EventGroupdialCreated(t *testing.T) {

	tests := []struct {
		name string

		groupdial *cmgroupdial.Groupdial

		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			groupdial: &cmgroupdial.Groupdial{
				ID: uuid.FromStringOrNil("8a7bb5d0-f84f-4568-917c-14961a8a7141"),
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeAgent,
					Target: "0de675c4-d1e4-498c-81f7-01bd8ee9e656",
				},
			},

			responseAgent: &agent.Agent{
				ID:     uuid.FromStringOrNil("0de675c4-d1e4-498c-81f7-01bd8ee9e656"),
				Status: agent.StatusAvailable,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AgentGet(ctx, tt.responseAgent.ID).Return(tt.responseAgent, nil)
			mockDB.EXPECT().AgentSetStatus(ctx, tt.responseAgent.ID, agent.StatusRinging).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.responseAgent.ID).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentStatusUpdated, tt.responseAgent)

			if err := h.EventGroupdialCreated(ctx, tt.groupdial); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventGroupdialAnswered(t *testing.T) {

	tests := []struct {
		name string

		groupdial     *cmgroupdial.Groupdial
		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			groupdial: &cmgroupdial.Groupdial{
				ID: uuid.FromStringOrNil("59e5b918-ac3e-4381-9894-f611cadeab93"),
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeAgent,
					Target: "e3eae3d0-8e4f-46a1-b6bd-5d36feae4749",
				},
			},
			responseAgent: &agent.Agent{
				ID:     uuid.FromStringOrNil("e3eae3d0-8e4f-46a1-b6bd-5d36feae4749"),
				Status: agent.StatusAvailable,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AgentGet(ctx, tt.responseAgent.ID).Return(tt.responseAgent, nil)
			mockDB.EXPECT().AgentSetStatus(ctx, tt.responseAgent.ID, agent.StatusBusy).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.responseAgent.ID).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentStatusUpdated, tt.responseAgent)

			if err := h.EventGroupdialAnswered(ctx, tt.groupdial); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
