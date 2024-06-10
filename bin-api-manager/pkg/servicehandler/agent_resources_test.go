package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	amresource "monorepo/bin-agent-manager/models/resource"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_AgentResourceGet(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
		resourceID uuid.UUID

		response  *amresource.Resource
		expectRes *amresource.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("b7c1f5a6-2669-11ef-9813-d76588980672"),

			&amresource.Resource{
				ID:      uuid.FromStringOrNil("b7c1f5a6-2669-11ef-9813-d76588980672"),
				OwnerID: uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
			},
			&amresource.WebhookMessage{
				ID:      uuid.FromStringOrNil("b7c1f5a6-2669-11ef-9813-d76588980672"),
				OwnerID: uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1ResourceGet(ctx, tt.resourceID).Return(tt.response, nil)

			res, err := h.AgentResourceGet(ctx, tt.agent, tt.resourceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentResourceGets(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		size    uint64
		token   string
		filters map[string]string

		response  []amresource.Resource
		expectRes []*amresource.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			10,
			"2020-09-20 03:23:20.995000",
			map[string]string{
				"deleted": "false",
			},

			[]amresource.Resource{
				{
					ID:      uuid.FromStringOrNil("abe5de5e-266a-11ef-b030-efc73b89ab11"),
					OwnerID: uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				},
			},
			[]*amresource.WebhookMessage{
				{
					ID:      uuid.FromStringOrNil("abe5de5e-266a-11ef-b030-efc73b89ab11"),
					OwnerID: uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				},
			},
		},
		{
			"2 agents",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			10,
			"2020-09-20 03:23:20.995000",
			map[string]string{
				"deleted": "false",
			},

			[]amresource.Resource{
				{
					ID:      uuid.FromStringOrNil("ac326666-266a-11ef-84bf-af740a91b4fc"),
					OwnerID: uuid.FromStringOrNil("d68fe618-0e78-11ef-a017-876e16634556"),
				},
				{
					ID:      uuid.FromStringOrNil("ac88ca2e-266a-11ef-b781-9f0d19d62efd"),
					OwnerID: uuid.FromStringOrNil("d68fe618-0e78-11ef-a017-876e16634556"),
				},
			},
			[]*amresource.WebhookMessage{
				{
					ID:      uuid.FromStringOrNil("ac326666-266a-11ef-84bf-af740a91b4fc"),
					OwnerID: uuid.FromStringOrNil("d68fe618-0e78-11ef-a017-876e16634556"),
				},
				{
					ID:      uuid.FromStringOrNil("ac88ca2e-266a-11ef-b781-9f0d19d62efd"),
					OwnerID: uuid.FromStringOrNil("d68fe618-0e78-11ef-a017-876e16634556"),
				},
			},
		},
		{
			"normal, agent has the same customer id but has nonepermission",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d68fe618-0e78-11ef-a017-876e16634556"),
				CustomerID: uuid.FromStringOrNil("51639bbe-8e5e-11ee-afc4-4fbef5d3d983"),
				Permission: amagent.PermissionNone,
			},
			10,
			"2020-09-20 03:23:20.995000",
			map[string]string{
				"deleted": "false",
			},

			[]amresource.Resource{
				{
					ID:      uuid.FromStringOrNil("cf06af30-266a-11ef-ad49-630e85b02911"),
					OwnerID: uuid.FromStringOrNil("d68fe618-0e78-11ef-a017-876e16634556"),
				},
			},
			[]*amresource.WebhookMessage{
				{
					ID:      uuid.FromStringOrNil("cf06af30-266a-11ef-ad49-630e85b02911"),
					OwnerID: uuid.FromStringOrNil("d68fe618-0e78-11ef-a017-876e16634556"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AgentV1ResourceGets(ctx, tt.token, tt.size, tt.filters).Return(tt.response, nil)

			res, err := h.AgentResourceGets(ctx, tt.agent, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
