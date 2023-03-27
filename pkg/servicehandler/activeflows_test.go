package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_activeflowGet(t *testing.T) {

	tests := []struct {
		name string

		customer     *cscustomer.Customer
		activeflowID uuid.UUID

		responseActiveflow *fmactiveflow.Activeflow
	}{
		{
			name: "normal",

			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			activeflowID: uuid.FromStringOrNil("306d40a4-cb22-11ed-a796-4776eeb9578e"),

			responseActiveflow: &fmactiveflow.Activeflow{
				ID:         uuid.FromStringOrNil("306d40a4-cb22-11ed-a796-4776eeb9578e"),
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

			mockReq.EXPECT().FlowV1ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)

			res, err := h.activeflowGet(ctx, tt.customer, tt.activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseActiveflow) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.responseActiveflow, res)
			}
		})
	}
}

func Test_ActiveflowGets(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		size     uint64
		token    string

		response  []fmactiveflow.Activeflow
		expectRes []*fmactiveflow.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("040422b6-3771-11ed-801b-27518c703c82"),
			},
			10,
			"2020-09-20 03:23:20.995000",

			[]fmactiveflow.Activeflow{
				{
					ID: uuid.FromStringOrNil("23dc5a36-cb23-11ed-8a25-8f48bd8c19bf"),
				},
			},
			[]*fmactiveflow.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("23dc5a36-cb23-11ed-8a25-8f48bd8c19bf"),
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

			mockReq.EXPECT().FlowV1ActiveflowGets(ctx, tt.customer.ID, tt.token, tt.size).Return(tt.response, nil)

			res, err := h.ActiveflowGets(ctx, tt.customer, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ActiveflowStop(t *testing.T) {

	tests := []struct {
		name               string
		customer           *cscustomer.Customer
		activeflowID       uuid.UUID
		responseActiveflow *fmactiveflow.Activeflow

		expectRes *fmactiveflow.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("2b4b10f4-cb24-11ed-ad87-0fe018a49bcd"),
			&fmactiveflow.Activeflow{
				ID:         uuid.FromStringOrNil("2b4b10f4-cb24-11ed-ad87-0fe018a49bcd"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				TMDelete:   defaultTimestamp,
			},

			&fmactiveflow.WebhookMessage{
				ID:         uuid.FromStringOrNil("2b4b10f4-cb24-11ed-ad87-0fe018a49bcd"),
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

			mockReq.EXPECT().FlowV1ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)
			mockReq.EXPECT().FlowV1ActiveflowStop(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)

			res, err := h.ActiveflowStop(ctx, tt.customer, tt.activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ActiveflowDelete(t *testing.T) {

	tests := []struct {
		name               string
		customer           *cscustomer.Customer
		activeflowID       uuid.UUID
		responseActiveflow *fmactiveflow.Activeflow

		expectRes *fmactiveflow.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("73161b68-cb24-11ed-8253-2f25bfb9d81b"),
			&fmactiveflow.Activeflow{
				ID:         uuid.FromStringOrNil("73161b68-cb24-11ed-8253-2f25bfb9d81b"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				TMDelete:   defaultTimestamp,
			},

			&fmactiveflow.WebhookMessage{
				ID:         uuid.FromStringOrNil("73161b68-cb24-11ed-8253-2f25bfb9d81b"),
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

			mockReq.EXPECT().FlowV1ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)
			mockReq.EXPECT().FlowV1ActiveflowDelete(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)

			res, err := h.ActiveflowDelete(ctx, tt.customer, tt.activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
