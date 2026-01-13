package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	rmextension "monorepo/bin-registrar-manager/models/extension"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ServiceAgentExtensionGets(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent

		responseAgent      *amagent.Agent
		responseExtensions []*rmextension.Extension
		expectedRes        []*rmextension.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2ddf1c90-bbc1-11ef-b991-1bd6ee52cbc5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeExtension,
						Target: "2e3b3c3c-bbc1-11ef-93c0-17537443cb56",
					},
					{
						Type:   commonaddress.TypeExtension,
						Target: "2e67d562-bbc1-11ef-b531-a3248f6d1477",
					},
				},
			},

			responseAgent: &amagent.Agent{
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeExtension,
						Target: "2e3b3c3c-bbc1-11ef-93c0-17537443cb56",
					},
					{
						Type:   commonaddress.TypeExtension,
						Target: "2e67d562-bbc1-11ef-b531-a3248f6d1477",
					},
				},
			},
			responseExtensions: []*rmextension.Extension{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2e3b3c3c-bbc1-11ef-93c0-17537443cb56"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2e67d562-bbc1-11ef-b531-a3248f6d1477"),
					},
				},
			},
			expectedRes: []*rmextension.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2e3b3c3c-bbc1-11ef-93c0-17537443cb56"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2e67d562-bbc1-11ef-b531-a3248f6d1477"),
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

			for i, address := range tt.agent.Addresses {
				extensionID := uuid.FromStringOrNil(address.Target)
				mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, extensionID.Return(tt.responseExtensions[i], nil)
			}

			res, err := h.ServiceAgentExtensionGets(ctx, tt.agent)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}
