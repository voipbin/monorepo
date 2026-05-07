package servicehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"

	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_OutboundConfigCreate(t *testing.T) {
	name := "my-config"

	tests := []struct {
		name string

		agent *auth.AuthIdentity
		req   *cmoutboundconfig.UpdateRequest

		responseConfig *cmoutboundconfig.OutboundConfig
		responseErr    error

		expectRes *cmoutboundconfig.WebhookMessage
		wantErr   bool
	}{
		{
			name: "success",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			req: &cmoutboundconfig.UpdateRequest{Name: &name},
			responseConfig: &cmoutboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Name:       name,
			},
			expectRes: &cmoutboundconfig.WebhookMessage{
				ID:         uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Name:       name,
			},
			wantErr: false,
		},
		{
			name: "no permission",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerManager, // not Admin
			}),
			req:     &cmoutboundconfig.UpdateRequest{Name: &name},
			wantErr: true,
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

			if !tt.wantErr {
				mockReq.EXPECT().
					CallV1OutboundConfigCreate(ctx, tt.agent.CustomerID, tt.req).
					Return(tt.responseConfig, tt.responseErr)
			}

			res, err := h.OutboundConfigCreate(ctx, tt.agent, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("OutboundConfigCreate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("OutboundConfigCreate() = %v, want %v", res, tt.expectRes)
			}
		})
	}
}

func Test_OutboundConfigGet(t *testing.T) {
	tests := []struct {
		name string

		agent *auth.AuthIdentity
		id    uuid.UUID

		responseConfig *cmoutboundconfig.OutboundConfig
		responseErr    error

		expectRes *cmoutboundconfig.WebhookMessage
		wantErr   bool
	}{
		{
			name: "success",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			id: uuid.FromStringOrNil("b1b2b3b4-0000-0000-0000-000000000001"),
			responseConfig: &cmoutboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("b1b2b3b4-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Name:       "cfg",
			},
			expectRes: &cmoutboundconfig.WebhookMessage{
				ID:         uuid.FromStringOrNil("b1b2b3b4-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Name:       "cfg",
			},
			wantErr: false,
		},
		{
			name: "not found (Nil ID)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			id: uuid.FromStringOrNil("b1b2b3b4-0000-0000-0000-000000000002"),
			// backend returns a struct with Nil ID → outboundConfigGet treats it as not found
			responseConfig: &cmoutboundconfig.OutboundConfig{},
			wantErr:        true,
		},
		{
			name: "no permission",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerManager,
			}),
			id: uuid.FromStringOrNil("b1b2b3b4-0000-0000-0000-000000000003"),
			responseConfig: &cmoutboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("b1b2b3b4-0000-0000-0000-000000000003"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			wantErr: true,
		},
		{
			name: "backend error",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			id:          uuid.FromStringOrNil("b1b2b3b4-0000-0000-0000-000000000004"),
			responseErr: fmt.Errorf("rpc failure"),
			wantErr:     true,
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

			// outboundConfigGet calls the backend when there's a response or error to return
			if tt.responseConfig != nil || tt.responseErr != nil {
				mockReq.EXPECT().
					CallV1OutboundConfigGet(ctx, tt.id).
					Return(tt.responseConfig, tt.responseErr)
			}

			res, err := h.OutboundConfigGet(ctx, tt.agent, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("OutboundConfigGet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("OutboundConfigGet() = %v, want %v", res, tt.expectRes)
			}
		})
	}
}

func Test_OutboundConfigDelete(t *testing.T) {
	tests := []struct {
		name string

		agent *auth.AuthIdentity
		id    uuid.UUID

		responseConfig  *cmoutboundconfig.OutboundConfig
		responseGetErr  error
		responseDeleted *cmoutboundconfig.OutboundConfig
		responseDelErr  error

		expectRes *cmoutboundconfig.WebhookMessage
		wantErr   bool
	}{
		{
			name: "success",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			id: uuid.FromStringOrNil("c1c2c3c4-0000-0000-0000-000000000001"),
			responseConfig: &cmoutboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("c1c2c3c4-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Name:       "cfg",
			},
			responseDeleted: &cmoutboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("c1c2c3c4-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Name:       "cfg",
			},
			expectRes: &cmoutboundconfig.WebhookMessage{
				ID:         uuid.FromStringOrNil("c1c2c3c4-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Name:       "cfg",
			},
			wantErr: false,
		},
		{
			name: "not found (Nil ID)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			id:             uuid.FromStringOrNil("c1c2c3c4-0000-0000-0000-000000000002"),
			responseConfig: &cmoutboundconfig.OutboundConfig{}, // Nil ID → not found
			wantErr:        true,
		},
		{
			name: "no permission",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerManager,
			}),
			id: uuid.FromStringOrNil("c1c2c3c4-0000-0000-0000-000000000003"),
			responseConfig: &cmoutboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("c1c2c3c4-0000-0000-0000-000000000003"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			wantErr: true,
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

			// outboundConfigGet always calls the backend
			if tt.responseConfig != nil || tt.responseGetErr != nil {
				mockReq.EXPECT().
					CallV1OutboundConfigGet(ctx, tt.id).
					Return(tt.responseConfig, tt.responseGetErr)
			}

			// delete RPC is called only when get succeeds and permission passes
			if !tt.wantErr && tt.responseDeleted != nil {
				mockReq.EXPECT().
					CallV1OutboundConfigDelete(ctx, tt.id).
					Return(tt.responseDeleted, tt.responseDelErr)
			}

			res, err := h.OutboundConfigDelete(ctx, tt.agent, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("OutboundConfigDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("OutboundConfigDelete() = %v, want %v", res, tt.expectRes)
			}
		})
	}
}
