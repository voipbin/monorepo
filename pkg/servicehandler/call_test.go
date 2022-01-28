package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestCallCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	type test struct {
		name        string
		customer    *cscustomer.Customer
		flowID      uuid.UUID
		source      *address.Address
		destination *address.Address
		cmCall      *cmcall.Call
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("2c45d0b8-efc4-11ea-9a45-4f30fc2e0b02"),
			&address.Address{
				Type:   address.TypeSIP,
				Target: "testsource@test.com",
			},
			&address.Address{
				Type:   address.TypeSIP,
				Target: "testdestination@test.com",
			},
			&cmcall.Call{
				ID:         uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
				AsteriskID: "02:42:5d:f3:a7:05",
				ChannelID:  "d66d7c02-efc5-11ea-9f77-6fe9fae57afd",
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				FlowID:     uuid.FromStringOrNil("2c45d0b8-efc4-11ea-9a45-4f30fc2e0b02"),
				Type:       cmcall.TypeFlow,

				Source: cmaddress.Address{
					Type:   cmaddress.TypeSIP,
					Target: "testsource@test.com",
				},
				Destination: cmaddress.Address{
					Type:   cmaddress.TypeSIP,
					Target: "testdestination@test.com",
				},

				Status:       cmcall.StatusDialing,
				Data:         map[string]string{},
				Direction:    cmcall.DirectionIncoming,
				HangupBy:     "",
				HangupReason: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			mockReq.EXPECT().CMV1CallCreate(gomock.Any(), tt.customer.ID, tt.flowID, &tt.cmCall.Source, &tt.cmCall.Destination).Return(tt.cmCall, nil)

			res, err := h.CallCreate(tt.customer, tt.flowID, tt.source, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.cmCall.ConvertWebhookMessage()) != true {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.cmCall.ConvertWebhookMessage(), res)
			}
		})
	}

}

func TestCallDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name     string
		customer *cscustomer.Customer
		callID   uuid.UUID
		call     *cmcall.Call
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("9e9ed0b6-6791-11eb-9810-87fda8377194"),
			&cmcall.Call{
				ID:         uuid.FromStringOrNil("9e9ed0b6-6791-11eb-9810-87fda8377194"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().CMV1CallGet(gomock.Any(), tt.callID).Return(tt.call, nil)
			mockReq.EXPECT().CMV1CallHangup(gomock.Any(), tt.callID).Return(nil, nil)

			if err := h.CallDelete(tt.customer, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
