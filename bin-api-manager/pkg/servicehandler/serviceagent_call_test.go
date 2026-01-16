package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/pkg/dbhandler"
	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentCallGets(t *testing.T) {

	type test struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		responseCalls []cmcall.Call
		expectFilters map[cmcall.Field]any
		expectRes     []*cmcall.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			size:  100,
			token: "2021-03-01 01:00:00.995000",

			responseCalls: []cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("896d70be-3f9f-11ef-bfa4-532d59e85e8c"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8996f42a-3f9f-11ef-9c18-0b629aa668a1"),
					},
				},
			},
			expectFilters: map[cmcall.Field]any{
				cmcall.FieldOwnerID:    uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				cmcall.FieldCustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				cmcall.FieldDeleted:    false,
			},
			expectRes: []*cmcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("896d70be-3f9f-11ef-bfa4-532d59e85e8c"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8996f42a-3f9f-11ef-9c18-0b629aa668a1"),
					},
				},
			},
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

			mockReq.EXPECT().CallV1CallList(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseCalls, nil)

			res, err := h.ServiceAgentCallGets(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentCallGet(t *testing.T) {

	type test struct {
		name string

		agent  *amagent.Agent
		callID uuid.UUID

		responseCall *cmcall.Call
		expectRes    *cmcall.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			callID: uuid.FromStringOrNil("dbee2cd4-3f9f-11ef-bafc-33410c3df327"),

			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dbee2cd4-3f9f-11ef-bafc-33410c3df327"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
				TMDelete: defaultTimestamp,
			},
			expectRes: &cmcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dbee2cd4-3f9f-11ef-bafc-33410c3df327"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
				TMDelete: defaultTimestamp,
			},
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)

			res, err := h.ServiceAgentCallGet(ctx, tt.agent, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentCallDelete(t *testing.T) {

	type test struct {
		name string

		agent  *amagent.Agent
		callID uuid.UUID

		responseCall *cmcall.Call
		expectRes    *cmcall.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			callID: uuid.FromStringOrNil("4568ee74-3fa0-11ef-b393-9b1928f41f55"),

			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4568ee74-3fa0-11ef-b393-9b1928f41f55"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
				TMDelete: defaultTimestamp,
			},
			expectRes: &cmcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4568ee74-3fa0-11ef-b393-9b1928f41f55"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
				TMDelete: defaultTimestamp,
			},
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)
			mockReq.EXPECT().CallV1CallDelete(ctx, tt.callID).Return(tt.responseCall, nil)

			res, err := h.ServiceAgentCallDelete(ctx, tt.agent, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
