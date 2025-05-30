package requesthandler

import (
	"context"
	"reflect"
	"testing"

	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_TransferV1TransferStart(t *testing.T) {

	type test struct {
		name string

		transferType        tmtransfer.Type
		transfererCallID    uuid.UUID
		transfereeAddresses []commonaddress.Address

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *tmtransfer.Transfer
	}

	tests := []test{
		{
			name: "normal",

			transferType:     tmtransfer.TypeAttended,
			transfererCallID: uuid.FromStringOrNil("47ca8e80-dd35-11ed-8213-bf37002f55ef"),
			transfereeAddresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
			},

			expectTarget: "bin-manager.transfer-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/transfers",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"type":"attended","transferer_call_id":"47ca8e80-dd35-11ed-8213-bf37002f55ef","transferee_addresses":[{"type":"tel","target":"+821100000001"}]}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"926ea4c6-dd35-11ed-8414-27310fdd3d82"}`),
			},
			expectRes: &tmtransfer.Transfer{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("926ea4c6-dd35-11ed-8414-27310fdd3d82"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.TransferV1TransferStart(ctx, tt.transferType, tt.transfererCallID, tt.transfereeAddresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
