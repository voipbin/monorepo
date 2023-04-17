package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/transfer-manager.git/models/transfer"
	"gitlab.com/voipbin/bin-manager/transfer-manager.git/pkg/transferhandler"
)

func Test_processV1TransfersPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		responseTransfer *transfer.Transfer

		expectType                transfer.Type
		expectTransfererCallID    uuid.UUID
		expectTransfereeAddresses []commonaddress.Address
		expectRes                 *rabbitmqhandler.Response
	}{
		{
			name: "type blind",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/transfers",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"blind","transferer_call_id":"9d6ae370-dc73-11ed-a491-aff35612aed5","transferee_addresses":[{"type":"tel","target":"+821100000001"}]}`),
			},

			responseTransfer: &transfer.Transfer{
				ID: uuid.FromStringOrNil("9e44cd92-dc73-11ed-ab06-c7042aaaba15"),
			},

			expectType:             transfer.TypeBlind,
			expectTransfererCallID: uuid.FromStringOrNil("9d6ae370-dc73-11ed-a491-aff35612aed5"),
			expectTransfereeAddresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
			},
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9e44cd92-dc73-11ed-ab06-c7042aaaba15","customer_id":"00000000-0000-0000-0000-000000000000","type":"","transferer_call_id":"00000000-0000-0000-0000-000000000000","transferee_addresses":null,"transferee_call_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockTransfer := transferhandler.NewMockTransferHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				transferHandler: mockTransfer,
			}

			mockTransfer.EXPECT().ServiceStart(gomock.Any(), tt.expectType, tt.expectTransfererCallID, tt.expectTransfereeAddresses).Return(tt.responseTransfer, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
