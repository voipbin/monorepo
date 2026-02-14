package agenthandler

import (
	"context"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"
)

func Test_Get(t *testing.T) {
	tests := []struct {
		name string

		id            uuid.UUID
		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				Username: "test",
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

			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}

			if res.ID != tt.responseAgent.ID {
				t.Errorf("Wrong ID.\nexpect: %v\ngot: %v", tt.responseAgent.ID, res.ID)
			}
		})
	}
}

func Test_UpdateBasicInfo(t *testing.T) {
	tests := []struct {
		name string

		id         uuid.UUID
		agentName  string
		detail     string
		ringMethod agent.RingMethod

		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
			agentName:  "updated name",
			detail:     "updated detail",
			ringMethod: agent.RingMethodLinear,

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				Username:   "test",
				Name:       "updated name",
				Detail:     "updated detail",
				RingMethod: agent.RingMethodLinear,
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

			mockDB.EXPECT().AgentSetBasicInfo(ctx, tt.id, tt.agentName, tt.detail, tt.ringMethod).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentUpdated, tt.responseAgent)

			res, err := h.UpdateBasicInfo(ctx, tt.id, tt.agentName, tt.detail, tt.ringMethod)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}

			if res.Name != tt.agentName {
				t.Errorf("Wrong name.\nexpect: %v\ngot: %v", tt.agentName, res.Name)
			}
		})
	}
}

func Test_UpdatePassword(t *testing.T) {
	tests := []struct {
		name string

		id       uuid.UUID
		password string

		responseHash  string
		responseAgent *agent.Agent
		expectErr     bool
	}{
		{
			name: "normal",

			id:       uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
			password: "newpassword",

			responseHash: "newhash",
			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				Username:     "test",
				PasswordHash: "newhash",
			},
			expectErr: false,
		},
		{
			name: "guest agent error",

			id:       agent.GuestAgentID,
			password: "newpassword",

			responseHash:  "",
			responseAgent: nil,
			expectErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &agentHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			if !tt.expectErr {
				mockUtil.EXPECT().HashGenerate(tt.password, defaultPasswordHashCost).Return(tt.responseHash, nil)
				mockDB.EXPECT().AgentSetPasswordHash(ctx, tt.id, tt.responseHash).Return(nil)
				mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentUpdated, tt.responseAgent)
			}

			res, err := h.UpdatePassword(ctx, tt.id, tt.password)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong error match. expect error: %v, got error: %v", tt.expectErr, err)
			}

			if !tt.expectErr && res.PasswordHash != tt.responseHash {
				t.Errorf("Wrong password hash.\nexpect: %v\ngot: %v", tt.responseHash, res.PasswordHash)
			}
		})
	}
}

func Test_UpdateTagIDs(t *testing.T) {
	tests := []struct {
		name string

		id     uuid.UUID
		tagIDs []uuid.UUID

		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			id:     uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
			tagIDs: []uuid.UUID{uuid.FromStringOrNil("700c10b4-4b4e-11ec-959b-bb95248c693f")},

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				Username: "test",
				TagIDs:   []uuid.UUID{uuid.FromStringOrNil("700c10b4-4b4e-11ec-959b-bb95248c693f")},
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

			mockDB.EXPECT().AgentSetTagIDs(ctx, tt.id, tt.tagIDs).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishEvent(ctx, agent.EventTypeAgentUpdated, tt.responseAgent)

			res, err := h.UpdateTagIDs(ctx, tt.id, tt.tagIDs)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}

			if len(res.TagIDs) != len(tt.tagIDs) {
				t.Errorf("Wrong tag IDs length.\nexpect: %v\ngot: %v", len(tt.tagIDs), len(res.TagIDs))
			}
		})
	}
}

func Test_Delete_errors(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		mockFunc func(mockDB *dbhandler.MockDBHandler, id uuid.UUID)
	}{
		{
			name: "guest agent error",
			id:   agent.GuestAgentID,
			mockFunc: func(mockDB *dbhandler.MockDBHandler, id uuid.UUID) {
				// No mocks needed - should fail before database call
			},
		},
		{
			name: "only admin error",
			id:   uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
			mockFunc: func(mockDB *dbhandler.MockDBHandler, id uuid.UUID) {
				responseAgent := &agent.Agent{
					Identity: commonidentity.Identity{
						ID:         id,
						CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					},
					Permission: agent.PermissionCustomerAdmin,
				}
				mockDB.EXPECT().AgentGet(gomock.Any(), id).Return(responseAgent, nil)
				mockDB.EXPECT().AgentList(gomock.Any(), uint64(1000), "", map[agent.Field]any{
					agent.FieldCustomerID: responseAgent.CustomerID,
					agent.FieldDeleted:    false,
				}).Return([]*agent.Agent{responseAgent}, nil)
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

			tt.mockFunc(mockDB, tt.id)

			_, err := h.Delete(ctx, tt.id)
			if err == nil {
				t.Errorf("Wrong match. expect:error, got:nil")
			}
		})
	}
}

func Test_UpdateAddresses_errors(t *testing.T) {
	tests := []struct {
		name string

		id        uuid.UUID
		addresses []commonaddress.Address

		mockFunc func(mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, id uuid.UUID, addresses []commonaddress.Address)
	}{
		{
			name: "invalid extension id",
			id:   uuid.FromStringOrNil("464a277e-2d8d-11ef-8bc6-d7b95604d6f6"),
			addresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeExtension,
					Target: "invalid-uuid",
				},
			},
			mockFunc: func(mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, id uuid.UUID, addresses []commonaddress.Address) {
				responseAgent := &agent.Agent{
					Identity: commonidentity.Identity{
						ID:         id,
						CustomerID: uuid.FromStringOrNil("49d90a72-2d8d-11ef-b208-fb6caaa88ae9"),
					},
				}
				mockDB.EXPECT().AgentGet(gomock.Any(), id).Return(responseAgent, nil)
			},
		},
		{
			name: "invalid target for tel",
			id:   uuid.FromStringOrNil("464a277e-2d8d-11ef-8bc6-d7b95604d6f6"),
			addresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "",
				},
			},
			mockFunc: func(mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, id uuid.UUID, addresses []commonaddress.Address) {
				responseAgent := &agent.Agent{
					Identity: commonidentity.Identity{
						ID:         id,
						CustomerID: uuid.FromStringOrNil("49d90a72-2d8d-11ef-b208-fb6caaa88ae9"),
					},
				}
				mockDB.EXPECT().AgentGet(gomock.Any(), id).Return(responseAgent, nil)
			},
		},
		{
			name: "unknown address type",
			id:   uuid.FromStringOrNil("464a277e-2d8d-11ef-8bc6-d7b95604d6f6"),
			addresses: []commonaddress.Address{
				{
					Type:   "unknown",
					Target: "target",
				},
			},
			mockFunc: func(mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, id uuid.UUID, addresses []commonaddress.Address) {
				responseAgent := &agent.Agent{
					Identity: commonidentity.Identity{
						ID:         id,
						CustomerID: uuid.FromStringOrNil("49d90a72-2d8d-11ef-b208-fb6caaa88ae9"),
					},
				}
				mockDB.EXPECT().AgentGet(gomock.Any(), id).Return(responseAgent, nil)
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
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				cache:         mockCache,
			}
			ctx := context.Background()

			tt.mockFunc(mockDB, mockReq, tt.id, tt.addresses)

			_, err := h.UpdateAddresses(ctx, tt.id, tt.addresses)
			if err == nil {
				t.Errorf("Wrong match. expect:error, got:nil")
			}
		})
	}
}

