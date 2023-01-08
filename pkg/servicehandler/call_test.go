package servicehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_callGet(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		callID   uuid.UUID

		responseCall *cmcall.Call
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a"),
			&cmcall.Call{
				ID:         uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				TMDelete:   defaultTimestamp,
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

			res, err := h.callGet(ctx, tt.customer, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseCall) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.responseCall, res)
			}
		})
	}
}

func Test_callGet_error(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		callID   uuid.UUID

		responseCall      *cmcall.Call
		responseCallError error
	}{
		{
			name: "call get returns an error",
			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			callID: uuid.FromStringOrNil("7b7e58de-8f37-11ed-8852-0f407ad6849f"),

			responseCallError: fmt.Errorf(""),
		},
		{
			name: "deleted call info",
			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			callID: uuid.FromStringOrNil("7b7e58de-8f37-11ed-8852-0f407ad6849f"),

			responseCall: &cmcall.Call{
				ID:         uuid.FromStringOrNil("7b7e58de-8f37-11ed-8852-0f407ad6849f"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				TMDelete:   "2020-09-20 03:23:20.995000",
			},
		},
		{
			name: "user has no permission",
			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("bf255b00-8f37-11ed-8505-ebf5b5e2e761"),
			},
			callID: uuid.FromStringOrNil("bf41540e-8f37-11ed-8355-4be7200818a7"),

			responseCall: &cmcall.Call{
				ID:         uuid.FromStringOrNil("bf41540e-8f37-11ed-8355-4be7200818a7"),
				CustomerID: uuid.FromStringOrNil("d4c81cfe-8f37-11ed-9504-9f06f39cf4f0"),
				TMDelete:   defaultTimestamp,
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, tt.responseCallError)

			_, err := h.callGet(ctx, tt.customer, tt.callID)
			if err == nil {
				t.Error("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_CallCreate(t *testing.T) {

	tests := []struct {
		name         string
		customer     *cscustomer.Customer
		flowID       uuid.UUID
		actions      []fmaction.Action
		source       *commonaddress.Address
		destinations []commonaddress.Address

		responseCall []cmcall.Call
		expectRes    []*cmcall.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("2c45d0b8-efc4-11ea-9a45-4f30fc2e0b02"),
			[]fmaction.Action{},
			&commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "testsource@test.com",
			},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeSIP,
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
			&commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "testsource@test.com",
			},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeSIP,
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
			&commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "testsource@test.com",
			},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeSIP,
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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			targetFlowID := tt.flowID
			if targetFlowID == uuid.Nil {
				targetFlowID = uuid.Must(uuid.NewV4())
				mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.customer.ID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.actions, false).Return(&fmflow.Flow{ID: targetFlowID}, nil)
			}
			mockReq.EXPECT().CallV1CallsCreate(ctx, tt.customer.ID, targetFlowID, uuid.Nil, tt.source, tt.destinations).Return(tt.responseCall, nil)

			res, err := h.CallCreate(ctx, tt.customer, tt.flowID, tt.actions, tt.source, tt.destinations)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}

}

func Test_CallDelete(t *testing.T) {

	tests := []struct {
		name     string
		customer *cscustomer.Customer
		callID   uuid.UUID
		call     *cmcall.Call

		expectRes *cmcall.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("eccc7bf4-8926-11ed-b638-0fcef48a97d2"),
			&cmcall.Call{
				ID:         uuid.FromStringOrNil("eccc7bf4-8926-11ed-b638-0fcef48a97d2"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				TMDelete:   defaultTimestamp,
			},

			&cmcall.WebhookMessage{
				ID:         uuid.FromStringOrNil("eccc7bf4-8926-11ed-b638-0fcef48a97d2"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				TMDelete:   defaultTimestamp,
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.call, nil)
			mockReq.EXPECT().CallV1CallDelete(ctx, tt.callID).Return(tt.call, nil)

			res, err := h.CallDelete(ctx, tt.customer, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallHangup(t *testing.T) {

	tests := []struct {
		name     string
		customer *cscustomer.Customer
		callID   uuid.UUID
		call     *cmcall.Call

		expectRes *cmcall.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("9e9ed0b6-6791-11eb-9810-87fda8377194"),
			&cmcall.Call{
				ID:         uuid.FromStringOrNil("9e9ed0b6-6791-11eb-9810-87fda8377194"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				TMDelete:   defaultTimestamp,
			},

			&cmcall.WebhookMessage{
				ID:         uuid.FromStringOrNil("9e9ed0b6-6791-11eb-9810-87fda8377194"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				TMDelete:   defaultTimestamp,
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.call, nil)
			mockReq.EXPECT().CallV1CallHangup(ctx, tt.callID).Return(tt.call, nil)

			res, err := h.CallHangup(ctx, tt.customer, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
