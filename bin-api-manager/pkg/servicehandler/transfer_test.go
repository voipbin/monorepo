package servicehandler

import (
	"context"
	"reflect"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_TransferStart(t *testing.T) {

	type test struct {
		name string

		agent               *auth.AuthIdentity
		transferType        tmtransfer.Type
		transfererCallID    uuid.UUID
		transfereeAddresses []commonaddress.Address

		responseTransfererCall *cmcall.Call
		responseTransfer       *tmtransfer.Transfer

		expectRes *tmtransfer.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			transferType:     tmtransfer.TypeAttended,
			transfererCallID: uuid.FromStringOrNil("00d773d4-dd3b-11ed-bcad-d3c44f5b7491"),
			transfereeAddresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
			},

			responseTransfererCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("00d773d4-dd3b-11ed-bcad-d3c44f5b7491"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
			},
			responseTransfer: &tmtransfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("00ff06ba-dd3b-11ed-944c-bf71648b5aaa"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			expectRes: &tmtransfer.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("00ff06ba-dd3b-11ed-944c-bf71648b5aaa"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.transfererCallID).Return(tt.responseTransfererCall, nil)
			mockReq.EXPECT().TransferV1TransferStart(ctx, tt.transferType, tt.transfererCallID, tt.transfereeAddresses).Return(tt.responseTransfer, nil)

			res, err := h.TransferStart(ctx, tt.agent, tt.transferType, tt.transfererCallID, tt.transfereeAddresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

// Test_TransferStart_agentPermissionDenied pins that the admin transfer path stays admin/manager only,
// the service agent path is a separate function.
func Test_TransferStart_agentPermissionDenied(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	callID := uuid.FromStringOrNil("00d773d4-dd3b-11ed-bcad-d3c44f5b7491")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"), CustomerID: customerID},
		Permission: amagent.PermissionCustomerAgent,
	})

	mockReq.EXPECT().CallV1CallGet(ctx, callID).Return(&cmcall.Call{
		Identity: commonidentity.Identity{ID: callID, CustomerID: customerID},
	}, nil)

	res, err := h.TransferStart(ctx, a, tmtransfer.TypeBlind, callID, []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550000001"}})
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: permission denied, got: %v, %v", res, err)
	}
}
