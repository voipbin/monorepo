package servicehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/requesthandler"

	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-common-handler/pkg/utilhandler"

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
				Permission: amagent.PermissionProjectSuperAdmin,
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
				Permission: amagent.PermissionCustomerAdmin, // not ProjectSuperAdmin
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
				Permission: amagent.PermissionProjectSuperAdmin,
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
				Permission: amagent.PermissionProjectSuperAdmin,
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
				Permission: amagent.PermissionCustomerAdmin, // not ProjectSuperAdmin
			}),
			id:      uuid.FromStringOrNil("b1b2b3b4-0000-0000-0000-000000000003"),
			wantErr: true,
		},
		{
			name: "backend error",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
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

			// Backend is called only if permission check passes (ProjectSuperAdmin).
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
				Permission: amagent.PermissionProjectSuperAdmin,
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
				Permission: amagent.PermissionProjectSuperAdmin,
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
				Permission: amagent.PermissionCustomerAdmin, // not ProjectSuperAdmin
			}),
			id:      uuid.FromStringOrNil("c1c2c3c4-0000-0000-0000-000000000003"),
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

			// Backend get is called only when permission check passes (ProjectSuperAdmin).
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

func Test_OutboundConfigSelfGet(t *testing.T) {
	t1 := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string

		agent *auth.AuthIdentity

		responseConfigs []cmoutboundconfig.OutboundConfig
		responseErr     error

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
			responseConfigs: []cmoutboundconfig.OutboundConfig{
				{
					ID:         uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					Name:       "my-config",
					TMCreate:   &t1,
				},
			},
			expectRes: &cmoutboundconfig.WebhookMessage{
				ID:         uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Name:       "my-config",
				TMCreate:   &t1,
			},
			wantErr: false,
		},
		{
			name: "not found (empty list)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			responseConfigs: []cmoutboundconfig.OutboundConfig{},
			wantErr:         true,
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			// Backend is only called if the permission check passes (CustomerAdmin).
			if tt.responseConfigs != nil || tt.responseErr != nil {
				mockUtil.EXPECT().TimeGetCurTime().Return("2024-01-15T10:30:00.000000Z")
				mockReq.EXPECT().
					CallV1OutboundConfigList(ctx, tt.agent.CustomerID, uint64(1), "2024-01-15T10:30:00.000000Z").
					Return(tt.responseConfigs, tt.responseErr)
			}

			res, err := h.OutboundConfigSelfGet(ctx, tt.agent)
			if (err != nil) != tt.wantErr {
				t.Errorf("OutboundConfigSelfGet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("OutboundConfigSelfGet() = %v, want %v", res, tt.expectRes)
			}
		})
	}
}

func Test_OutboundConfigSelfUpdate(t *testing.T) {
	name := "updated-config"
	t1 := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string

		agent *auth.AuthIdentity
		req   *cmoutboundconfig.UpdateRequest

		responseListConfigs []cmoutboundconfig.OutboundConfig
		responseListErr     error
		responseUpdated     *cmoutboundconfig.OutboundConfig
		responseUpdateErr   error

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
			responseListConfigs: []cmoutboundconfig.OutboundConfig{
				{
					ID:         uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					Name:       "old-config",
					TMCreate:   &t1,
				},
			},
			responseUpdated: &cmoutboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Name:       name,
				TMCreate:   &t1,
			},
			expectRes: &cmoutboundconfig.WebhookMessage{
				ID:         uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Name:       name,
				TMCreate:   &t1,
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
				Permission: amagent.PermissionCustomerManager,
			}),
			req:     &cmoutboundconfig.UpdateRequest{Name: &name},
			wantErr: true,
		},
		{
			name: "config not found",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			req:                 &cmoutboundconfig.UpdateRequest{Name: &name},
			responseListConfigs: []cmoutboundconfig.OutboundConfig{},
			wantErr:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			// List is called if permission check passes (CustomerAdmin).
			if tt.responseListConfigs != nil || tt.responseListErr != nil {
				mockUtil.EXPECT().TimeGetCurTime().Return("2024-01-15T10:30:00.000000Z")
				mockReq.EXPECT().
					CallV1OutboundConfigList(ctx, tt.agent.CustomerID, uint64(1), "2024-01-15T10:30:00.000000Z").
					Return(tt.responseListConfigs, tt.responseListErr)
			}

			// Update is called only when list succeeds and returns a config.
			if !tt.wantErr && tt.responseUpdated != nil {
				configID := uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001")
				mockReq.EXPECT().
					CallV1OutboundConfigUpdate(ctx, configID, tt.req).
					Return(tt.responseUpdated, tt.responseUpdateErr)
			}

			res, err := h.OutboundConfigSelfUpdate(ctx, tt.agent, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("OutboundConfigSelfUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("OutboundConfigSelfUpdate() = %v, want %v", res, tt.expectRes)
			}
		})
	}
}
