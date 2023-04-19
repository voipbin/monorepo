package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	tmtransfer "gitlab.com/voipbin/bin-manager/transfer-manager.git/models/transfer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_TransferStart(t *testing.T) {

	type test struct {
		name string

		customer            *cscustomer.Customer
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

			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("9e84e358-8284-11ed-b722-2fa228151282"),
			},
			transferType:     tmtransfer.TypeAttended,
			transfererCallID: uuid.FromStringOrNil("00d773d4-dd3b-11ed-bcad-d3c44f5b7491"),
			transfereeAddresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
			},

			responseTransfererCall: &cmcall.Call{
				ID:         uuid.FromStringOrNil("00d773d4-dd3b-11ed-bcad-d3c44f5b7491"),
				CustomerID: uuid.FromStringOrNil("9e84e358-8284-11ed-b722-2fa228151282"),
				TMDelete:   defaultTimestamp,
			},
			responseTransfer: &tmtransfer.Transfer{
				ID: uuid.FromStringOrNil("00ff06ba-dd3b-11ed-944c-bf71648b5aaa"),
			},

			expectRes: &tmtransfer.WebhookMessage{
				ID: uuid.FromStringOrNil("00ff06ba-dd3b-11ed-944c-bf71648b5aaa"),
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

			res, err := h.TransferStart(ctx, tt.customer, tt.transferType, tt.transfererCallID, tt.transfereeAddresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
