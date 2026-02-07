package agenthandler

import (
	"context"
	"fmt"
	"testing"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8a7bb5d0-f84f-4568-917c-14961a8a7141"),
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeAgent,
						Target: "0de675c4-d1e4-498c-81f7-01bd8ee9e656",
					},
				},
			},

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0de675c4-d1e4-498c-81f7-01bd8ee9e656"),
				},
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

		groupcall     *cmgroupcall.Groupcall
		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			groupcall: &cmgroupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("59e5b918-ac3e-4381-9894-f611cadeab93"),
					CustomerID: uuid.FromStringOrNil("2b0153e0-28e0-11ef-ac14-9b7259fa6ef3"),
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeAgent,
						Target: "e3eae3d0-8e4f-46a1-b6bd-5d36feae4749",
					},
				},
			},
			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e3eae3d0-8e4f-46a1-b6bd-5d36feae4749"),
				},
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
			// mockResource := resourcehandler.NewMockResourceHandler(mc)

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				// resourceHandler: mockResource,
			}
			ctx := context.Background()

			for _, destination := range tt.groupcall.Destinations {
				agentID := uuid.FromStringOrNil(destination.Target)
				mockDB.EXPECT().AgentGet(ctx, agentID).Return(tt.responseAgent, nil)
				mockDB.EXPECT().AgentSetStatus(ctx, tt.responseAgent.ID, agent.StatusBusy).Return(nil)
				mockDB.EXPECT().AgentGet(ctx, tt.responseAgent.ID).Return(tt.responseAgent, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentStatusUpdated, tt.responseAgent)
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

		expectFilter map[agent.Field]any
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("82ed53fa-ccca-11ee-be19-17f582a54cf4"),
			},
			responseAgents: []*agent.Agent{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e3722b4c-ccca-11ee-b18c-03025e4b324b"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e39bfb34-ccca-11ee-9c3e-2fba9dd3bf35"),
					},
				},
			},

			expectFilter: map[agent.Field]any{
				agent.FieldCustomerID: uuid.FromStringOrNil("82ed53fa-ccca-11ee-be19-17f582a54cf4"),
				agent.FieldDeleted:    false,
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
			mockDB.EXPECT().AgentList(ctx, uint64(1000), gomock.Any(), tt.expectFilter).Return(tt.responseAgents, nil)

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

func Test_EventCustomerCreated(t *testing.T) {

	tests := []struct {
		name string

		customer *cmcustomer.Customer

		responseUUID  uuid.UUID
		responseHash  string
		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID:    uuid.FromStringOrNil("9c0ea002-c8e4-11ef-bfbd-3316b71b50ac"),
				Email: "test@voipbin.net",
			},

			responseUUID: uuid.FromStringOrNil("38979028-c8e5-11ef-ab04-9b5ea42ae2be"),
			responseHash: "hash_string",
			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e3722b4c-ccca-11ee-b18c-03025e4b324b"),
				},
				Username: "test@voipbin.net",
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
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
				cache:         mockCache,
			}
			ctx := context.Background()

			// agent Create expectations
			mockUtil.EXPECT().EmailIsValid(tt.customer.Email).Return(true)
			mockDB.EXPECT().AgentGetByUsername(ctx, tt.customer.Email).Return(nil, fmt.Errorf(""))
			mockUtil.EXPECT().HashGenerate(gomock.Any(), defaultPasswordHashCost).Return(tt.responseHash, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().AgentCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.responseUUID).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentCreated, tt.responseAgent)

			// PasswordForgot expectations (welcome email)
			mockDB.EXPECT().AgentGetByUsername(ctx, tt.customer.Email).Return(tt.responseAgent, nil)
			mockCache.EXPECT().PasswordResetTokenSet(ctx, gomock.Any(), tt.responseAgent.ID, passwordResetTokenTTL).Return(nil)
			mockReq.EXPECT().EmailV1EmailSend(ctx, uuid.Nil, uuid.Nil, []commonaddress.Address{
				{Type: commonaddress.TypeEmail, Target: tt.customer.Email},
			}, "Welcome to VoIPBin - Set Your Password", gomock.Any(), gomock.Nil()).Return(nil, nil)

			if err := h.EventCustomerCreated(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventCustomerCreated_EmailFails(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &agentHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
		utilHandler:   mockUtil,
		cache:         mockCache,
	}
	ctx := context.Background()

	customer := &cmcustomer.Customer{
		ID:    uuid.FromStringOrNil("9c0ea002-c8e4-11ef-bfbd-3316b71b50ac"),
		Email: "test@voipbin.net",
	}

	responseUUID := uuid.FromStringOrNil("38979028-c8e5-11ef-ab04-9b5ea42ae2be")
	responseAgent := &agent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("e3722b4c-ccca-11ee-b18c-03025e4b324b"),
		},
		Username: "test@voipbin.net",
	}

	// agent Create expectations
	mockUtil.EXPECT().EmailIsValid(customer.Email).Return(true)
	mockDB.EXPECT().AgentGetByUsername(ctx, customer.Email).Return(nil, fmt.Errorf(""))
	mockUtil.EXPECT().HashGenerate(gomock.Any(), defaultPasswordHashCost).Return("hash_string", nil)
	mockUtil.EXPECT().UUIDCreate().Return(responseUUID)
	mockDB.EXPECT().AgentCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().AgentGet(ctx, responseUUID).Return(responseAgent, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, responseAgent.CustomerID, agent.EventTypeAgentCreated, responseAgent)

	// PasswordForgot fails (agent lookup fails for email)
	mockDB.EXPECT().AgentGetByUsername(ctx, customer.Email).Return(nil, fmt.Errorf("not found"))

	// EventCustomerCreated should still succeed even though PasswordForgot failed
	if err := h.EventCustomerCreated(ctx, customer); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
