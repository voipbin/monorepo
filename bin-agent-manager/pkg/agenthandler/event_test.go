package agenthandler

import (
	"context"
	"testing"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/models/resource"
	"monorepo/bin-agent-manager/pkg/dbhandler"
	"monorepo/bin-agent-manager/pkg/resourcehandler"
)

func Test_EventGroupcallCreated(t *testing.T) {

	tests := []struct {
		name string

		groupcall *cmgroupcall.Groupcall

		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			groupcall: &cmgroupcall.Groupcall{
				ID: uuid.FromStringOrNil("8a7bb5d0-f84f-4568-917c-14961a8a7141"),
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeAgent,
						Target: "0de675c4-d1e4-498c-81f7-01bd8ee9e656",
					},
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
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			for _, destination := range tt.groupcall.Destinations {
				agentID := uuid.FromStringOrNil(destination.Target)
				mockDB.EXPECT().AgentGet(ctx, agentID).Return(tt.responseAgent, nil)
				mockDB.EXPECT().AgentSetStatus(ctx, tt.responseAgent.ID, agent.StatusRinging).Return(nil)
				mockDB.EXPECT().AgentGet(ctx, tt.responseAgent.ID).Return(tt.responseAgent, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentStatusUpdated, tt.responseAgent)
			}

			if err := h.EventGroupcallCreated(ctx, tt.groupcall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventGroupcallAnswered(t *testing.T) {

	tests := []struct {
		name string

		groupcall         *cmgroupcall.Groupcall
		responseAgent     *agent.Agent
		responseResources []*resource.Resource

		expectFilters map[string]string
	}{
		{
			name: "normal",

			groupcall: &cmgroupcall.Groupcall{
				ID:         uuid.FromStringOrNil("59e5b918-ac3e-4381-9894-f611cadeab93"),
				CustomerID: uuid.FromStringOrNil("2b0153e0-28e0-11ef-ac14-9b7259fa6ef3"),
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeAgent,
						Target: "e3eae3d0-8e4f-46a1-b6bd-5d36feae4749",
					},
				},
			},
			responseAgent: &agent.Agent{
				ID:     uuid.FromStringOrNil("e3eae3d0-8e4f-46a1-b6bd-5d36feae4749"),
				Status: agent.StatusAvailable,
			},
			responseResources: []*resource.Resource{
				{
					ID: uuid.FromStringOrNil("6fc07d9e-28e0-11ef-b511-0716834ef197"),
				},
			},

			expectFilters: map[string]string{
				"customer_id":    "2b0153e0-28e0-11ef-ac14-9b7259fa6ef3",
				"reference_type": string(resource.ReferenceTypeGroupcall),
				"reference_id":   "59e5b918-ac3e-4381-9894-f611cadeab93",
				"deleted":        "false",
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
			mockResource := resourcehandler.NewMockResourceHandler(mc)

			h := &agentHandler{
				reqHandler:      mockReq,
				db:              mockDB,
				notifyHandler:   mockNotify,
				resourceHandler: mockResource,
			}
			ctx := context.Background()

			for _, destination := range tt.groupcall.Destinations {
				agentID := uuid.FromStringOrNil(destination.Target)
				mockDB.EXPECT().AgentGet(ctx, agentID).Return(tt.responseAgent, nil)
				mockDB.EXPECT().AgentSetStatus(ctx, tt.responseAgent.ID, agent.StatusBusy).Return(nil)
				mockDB.EXPECT().AgentGet(ctx, tt.responseAgent.ID).Return(tt.responseAgent, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentStatusUpdated, tt.responseAgent)
			}

			mockResource.EXPECT().Gets(ctx, uint64(100), "", tt.expectFilters).Return(tt.responseResources, nil)
			for _, r := range tt.responseResources {
				mockResource.EXPECT().UpdateData(ctx, r.ID, tt.groupcall).Return(&resource.Resource{}, nil)
			}

			if err := h.EventGroupcallProgressing(ctx, tt.groupcall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventCustomerDeleted(t *testing.T) {

	tests := []struct {
		name string

		customer       *cmcustomer.Customer
		responseAgents []*agent.Agent

		expectFilter map[string]string
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("82ed53fa-ccca-11ee-be19-17f582a54cf4"),
			},
			responseAgents: []*agent.Agent{
				{
					ID: uuid.FromStringOrNil("e3722b4c-ccca-11ee-b18c-03025e4b324b"),
				},
				{
					ID: uuid.FromStringOrNil("e39bfb34-ccca-11ee-9c3e-2fba9dd3bf35"),
				},
			},

			expectFilter: map[string]string{
				"customer_id": "82ed53fa-ccca-11ee-be19-17f582a54cf4",
				"deleted":     "false",
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().AgentGets(ctx, uint64(1000), gomock.Any(), tt.expectFilter).Return(tt.responseAgents, nil)

			for _, ag := range tt.responseAgents {
				mockDB.EXPECT().AgentDelete(ctx, ag.ID).Return(nil)
				mockDB.EXPECT().AgentGet(ctx, ag.ID).Return(ag, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, ag.CustomerID, agent.EventTypeAgentDeleted, ag)
			}

			if err := h.EventCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
