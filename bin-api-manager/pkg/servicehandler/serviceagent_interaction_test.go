package servicehandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-common-handler/pkg/requesthandler"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-api-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentInteractionList(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	fixedSince := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		agent      *auth.AuthIdentity
		size       uint64
		token      string
		peerType   string
		peerTarget string
		contactID  uuid.UUID
		addressID  uuid.UUID
		since      time.Time

		responseItems []*tmpeerevent.PeerEvent
		responseToken string
		expectErr     bool
	}{
		{
			name: "normal - plain agent permission is sufficient",
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
			since:      time.Time{},

			responseItems: []*tmpeerevent.PeerEvent{},
			responseToken: "",
			expectErr:     false,
		},
		{
			name: "normal - no filter, unfiltered mode with since forwarded to RPC",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			size:       20,
			token:      "",
			peerType:   "",
			peerTarget: "",
			contactID:  uuid.Nil,
			addressID:  uuid.Nil,
			since:      fixedSince,

			responseItems: []*tmpeerevent.PeerEvent{},
			responseToken: "",
			expectErr:     false,
		},
		{
			name:       "permission denied - direct access not supported",
			agent:      auth.NewDirectIdentity(&auth.DirectScope{CustomerID: customerID}),
			size:       20,
			token:      "",
			peerType:   "tel",
			peerTarget: "+155****1111",

			expectErr: true,
		},
		{
			// Round 2 PR review finding: the RPC-failure path was
			// previously untested for this servicehandler.
			name: "rpc error propagates",
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
			since:      time.Time{},

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
					ContactV1InteractionList(ctx, tt.agent.CustomerID, tt.size, tt.token, tt.peerType, tt.peerTarget, tt.contactID, tt.addressID, tt.since).
					Return(tt.responseItems, tt.responseToken, nil)
			} else if tt.name == "rpc error propagates" {
				mockReq.EXPECT().
					ContactV1InteractionList(ctx, tt.agent.CustomerID, tt.size, tt.token, tt.peerType, tt.peerTarget, tt.contactID, tt.addressID, tt.since).
					Return(nil, "", fmt.Errorf("rpc timeout"))
			}

			items, _, err := h.ServiceAgentInteractionList(ctx, tt.agent, tt.size, tt.token, tt.peerType, tt.peerTarget, tt.contactID, tt.addressID, tt.since)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: err, got: ok")
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if items == nil {
				t.Errorf("Wrong match. expect: non-nil, got: nil")
			}
		})
	}
}
