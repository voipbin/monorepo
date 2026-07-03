package servicehandler

import (
	"context"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cminteraction "monorepo/bin-contact-manager/models/interaction"
	cmresolution "monorepo/bin-contact-manager/models/resolution"

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

		responseItems []*cminteraction.Interaction
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

			responseItems: []*cminteraction.Interaction{},
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

func Test_InteractionGet(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	interactionID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	tests := []struct {
		name string

		agent         *auth.AuthIdentity
		interactionID uuid.UUID

		responseInteraction *cminteraction.Interaction
		expectRes           *cminteraction.Interaction
		expectErr           bool
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
			interactionID: interactionID,

			responseInteraction: &cminteraction.Interaction{
				ID:         interactionID,
				CustomerID: customerID,
				Direction:  "incoming",
				PeerType:   "tel",
				PeerTarget: "+155****1111",
			},
			expectRes: &cminteraction.Interaction{
				ID:         interactionID,
				CustomerID: customerID,
				Direction:  "incoming",
				PeerType:   "tel",
				PeerTarget: "+155****1111",
			},
			expectErr: false,
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
			interactionID: interactionID,

			responseInteraction: &cminteraction.Interaction{
				ID:         interactionID,
				CustomerID: customerID,
				Direction:  "incoming",
				PeerType:   "tel",
				PeerTarget: "+155****1111",
			},
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
			interactionID: interactionID,

			responseInteraction: &cminteraction.Interaction{
				ID:         interactionID,
				CustomerID: customerID,
			},
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

			// interactionGet is always called (even on permission-denied path, since we fetch-first).
			if tt.responseInteraction != nil {
				mockReq.EXPECT().
					ContactV1InteractionGet(ctx, tt.agent.CustomerID, tt.interactionID).
					Return(tt.responseInteraction, nil)
			}

			res, err := h.InteractionGet(ctx, tt.agent, tt.interactionID)
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
			if res == nil {
				t.Errorf("Expected result but got nil")
			}
		})
	}
}

func Test_ResolutionCreate(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	interactionID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")
	resolvedByID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")

	tests := []struct {
		name string

		agent          *auth.AuthIdentity
		interactionID  uuid.UUID
		contactID      uuid.UUID
		resolutionType string
		resolvedByType string
		resolvedByID   uuid.UUID

		responseInteraction *cminteraction.Interaction
		responseResolution  *cmresolution.Resolution
		expectErr           bool
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
			interactionID:  interactionID,
			contactID:      contactID,
			resolutionType: "positive",
			resolvedByType: "agent",
			resolvedByID:   resolvedByID,

			responseInteraction: &cminteraction.Interaction{
				ID:         interactionID,
				CustomerID: customerID,
			},
			responseResolution: &cmresolution.Resolution{
				ID:             uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004"),
				CustomerID:     customerID,
				InteractionID:  interactionID,
				ContactID:      contactID,
				ResolutionType: "positive",
				ResolvedByType: "agent",
				ResolvedByID:   resolvedByID,
			},
			expectErr: false,
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
			interactionID:  interactionID,
			contactID:      contactID,
			resolutionType: "positive",
			resolvedByType: "agent",
			resolvedByID:   resolvedByID,

			responseInteraction: &cminteraction.Interaction{
				ID:         interactionID,
				CustomerID: customerID,
			},
			expectErr: true,
		},
		{
			name: "no permission",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			interactionID:  interactionID,
			contactID:      contactID,
			resolutionType: "positive",
			resolvedByType: "agent",
			resolvedByID:   resolvedByID,

			responseInteraction: &cminteraction.Interaction{
				ID:         interactionID,
				CustomerID: customerID,
			},
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

			// interactionGet is always called (ownership pre-check).
			if tt.responseInteraction != nil {
				mockReq.EXPECT().
					ContactV1InteractionGet(ctx, tt.agent.CustomerID, tt.interactionID).
					Return(tt.responseInteraction, nil)
			}

			if !tt.expectErr && tt.responseResolution != nil {
				mockReq.EXPECT().
					ContactV1ResolutionCreate(ctx, tt.agent.CustomerID, tt.contactID, tt.interactionID, tt.resolutionType, tt.resolvedByType, tt.resolvedByID).
					Return(tt.responseResolution, nil)
			}

			res, err := h.ResolutionCreate(ctx, tt.agent, tt.interactionID, tt.contactID, tt.resolutionType, tt.resolvedByType, tt.resolvedByID)
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
			if res == nil {
				t.Errorf("Expected result but got nil")
			}
		})
	}
}

func Test_ResolutionDelete(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	interactionID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	resolutionID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")

	tests := []struct {
		name string

		agent         *auth.AuthIdentity
		interactionID uuid.UUID
		resolutionID  uuid.UUID

		expectErr    bool
		expectErrVal error
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
			interactionID: interactionID,
			resolutionID:  resolutionID,
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
			interactionID: interactionID,
			resolutionID:  resolutionID,
			expectErr:     true,
			expectErrVal:  serviceerrors.ErrPermissionDenied,
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
			interactionID: interactionID,
			resolutionID:  resolutionID,
			expectErr:     true,
			expectErrVal:  serviceerrors.ErrPermissionDenied,
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
					ContactV1ResolutionDelete(ctx, tt.agent.CustomerID, tt.interactionID, tt.resolutionID).
					Return(nil)
			}

			err := h.ResolutionDelete(ctx, tt.agent, tt.interactionID, tt.resolutionID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func Test_InteractionListUnresolved(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")

	tests := []struct {
		name string

		agent *auth.AuthIdentity
		size  uint64
		token string
		since string

		responseItems []*cminteraction.Interaction
		responseToken string
		expectErr     bool
	}{
		{
			name: "normal - default since",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			size:          20,
			token:         "",
			since:         "",
			responseItems: []*cminteraction.Interaction{},
			responseToken: "",
			expectErr:     false,
		},
		{
			name: "normal - explicit since",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			size:          20,
			token:         "",
			since:         "7d",
			responseItems: []*cminteraction.Interaction{},
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
			size:      20,
			token:     "",
			since:     "",
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
			size:      20,
			token:     "",
			since:     "",
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
					ContactV1InteractionListUnresolved(ctx, tt.agent.CustomerID, tt.size, tt.token, tt.since).
					Return(tt.responseItems, tt.responseToken, nil)
			}

			items, _, err := h.InteractionListUnresolved(ctx, tt.agent, tt.size, tt.token, tt.since)
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
