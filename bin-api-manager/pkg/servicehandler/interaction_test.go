package servicehandler

import (
	"context"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/requesthandler"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-api-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_InteractionList(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")

	tests := []struct {
		name string

		agent      *auth.AuthIdentity
		size       uint64
		token      string
		peerType   string
		peerTarget string
		contactID  uuid.UUID
		addressID  uuid.UUID

		responseItems []*tmpeerevent.PeerEvent
		responseToken string
		expectErr     bool
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			size:       20,
			token:      "",
			peerType:   "tel",
			peerTarget: "+155****1111",
			contactID:  uuid.Nil,
			addressID:  uuid.Nil,

			responseItems: []*tmpeerevent.PeerEvent{
				{
					CustomerID: customerID,
					Publisher:  "call",
					Direction:  "incoming",
					Peer:       commonaddress.Address{Type: "tel", Target: "+155****1111"},
				},
			},
			responseToken: "",
			expectErr:     false,
		},
		{
			name: "agent permission is insufficient",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			size:       20,
			token:      "",
			peerType:   "tel",
			peerTarget: "+155****1111",
			contactID:  uuid.Nil,
			addressID:  uuid.Nil,

			expectErr: true,
		},
		{
			name: "permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			size:       20,
			token:      "",
			peerType:   "tel",
			peerTarget: "+155****1111",

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			if !tt.expectErr {
				mockReq.EXPECT().
					ContactV1InteractionList(ctx, tt.agent.CustomerID, tt.size, tt.token, tt.peerType, tt.peerTarget, tt.contactID, tt.addressID, time.Time{}).
					Return(tt.responseItems, tt.responseToken, nil)
			}

			items, _, err := h.InteractionList(ctx, tt.agent, tt.size, tt.token, tt.peerType, tt.peerTarget, tt.contactID, tt.addressID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if items == nil {
				t.Errorf("Expected result but got nil")
			}
		})
	}
}
