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
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestCallCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	type test struct {
		name         string
		customer     *cscustomer.Customer
		flowID       uuid.UUID
		actions      []fmaction.Action
		source       *cmaddress.Address
		destinations []cmaddress.Address

		responseCall []cmcall.Call
		expectRes    []*cmcall.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("2c45d0b8-efc4-11ea-9a45-4f30fc2e0b02"),
			[]fmaction.Action{},
			&cmaddress.Address{
				Type:   cmaddress.TypeSIP,
				Target: "testsource@test.com",
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeSIP,
					Target: "testdestination@test.com",
				},
			},
			[]cmcall.Call{
				{
					ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
				},
			},
			[]*cmcall.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
				},
			},
		},
		{
			"with actions only",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.Nil,
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeSIP,
				Target: "testsource@test.com",
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeSIP,
					Target: "testdestination@test.com",
				},
			},
			[]cmcall.Call{
				{
					ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
				},
			},
			[]*cmcall.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
				},
			},
		},
		{
			"if both has given, flowid has more priority",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("2ca43d36-8df9-11ec-846a-ebf271da36c8"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeSIP,
				Target: "testsource@test.com",
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeSIP,
					Target: "testdestination@test.com",
				},
			},
			[]cmcall.Call{
				{
					ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
				},
			},
			[]*cmcall.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			targetFlowID := tt.flowID
			if targetFlowID == uuid.Nil {
				targetFlowID = uuid.Must(uuid.NewV4())
				mockReq.EXPECT().FMV1FlowCreate(gomock.Any(), tt.customer.ID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.actions, false).Return(&fmflow.Flow{ID: targetFlowID}, nil)
			}
			mockReq.EXPECT().CMV1CallsCreate(gomock.Any(), tt.customer.ID, targetFlowID, uuid.Nil, tt.source, tt.destinations).Return(tt.responseCall, nil)

			res, err := h.CallCreate(tt.customer, tt.flowID, tt.actions, tt.source, tt.destinations)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
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
