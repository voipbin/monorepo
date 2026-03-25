package agenthandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/dbhandler"
)

func Test_DirectHashRegenerate_ExistingDirect(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseAgent        *agent.Agent
		responseDirect       *dmdirect.Direct
		responseUpdatedAgent *agent.Agent

		expectUpdateFields map[agent.Field]any
	}

	tests := []test{
		{
			name: "agent with existing direct_id regenerates",

			id: uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				DirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				DirectHash: "oldhash123",
			},
			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				ResourceType: "agent",
				ResourceID:   uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
				Hash:         "newhash456",
			},
			responseUpdatedAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				DirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				DirectHash: "newhash456",
			},

			expectUpdateFields: map[agent.Field]any{
				agent.FieldDirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				agent.FieldDirectHash: "newhash456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &agentHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockReq.EXPECT().DirectV1DirectRegenerate(ctx, tt.responseAgent.DirectID).Return(tt.responseDirect, nil)
			mockDB.EXPECT().AgentUpdate(ctx, tt.id, tt.expectUpdateFields).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseUpdatedAgent, nil)

			res, err := h.DirectHashRegenerate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseUpdatedAgent) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedAgent, res)
			}
		})
	}
}

func Test_DirectHashRegenerate_NoDirect(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseAgent        *agent.Agent
		responseDirect       *dmdirect.Direct
		responseUpdatedAgent *agent.Agent

		expectUpdateFields map[agent.Field]any
	}

	tests := []test{
		{
			name: "agent with no direct_id creates new direct",

			id: uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222"),
				},
				DirectID:   uuid.Nil,
				DirectHash: "",
			},
			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222"),
				},
				ResourceType: "agent",
				ResourceID:   uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
				Hash:         "createdhash789",
			},
			responseUpdatedAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222"),
				},
				DirectID:   uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
				DirectHash: "createdhash789",
			},

			expectUpdateFields: map[agent.Field]any{
				agent.FieldDirectID:   uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
				agent.FieldDirectHash: "createdhash789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &agentHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockReq.EXPECT().DirectV1DirectCreate(ctx, tt.responseAgent.CustomerID, "agent", tt.id).Return(tt.responseDirect, nil)
			mockDB.EXPECT().AgentUpdate(ctx, tt.id, tt.expectUpdateFields).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseUpdatedAgent, nil)

			res, err := h.DirectHashRegenerate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseUpdatedAgent) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedAgent, res)
			}
		})
	}
}
