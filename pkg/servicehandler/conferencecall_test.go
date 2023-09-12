package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cfconferencecall "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_ConferencecallGet(t *testing.T) {

	tests := []struct {
		name             string
		customer         *cscustomer.Customer
		conferencecallID uuid.UUID

		responseConferencecall *cfconferencecall.Conferencecall

		expectRes *cfconferencecall.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("54df63a2-15ad-11ed-9309-b3fd99910cf5"),

			&cfconferencecall.Conferencecall{
				ID:         uuid.FromStringOrNil("54df63a2-15ad-11ed-9309-b3fd99910cf5"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},

			&cfconferencecall.WebhookMessage{
				ID:         uuid.FromStringOrNil("54df63a2-15ad-11ed-9309-b3fd99910cf5"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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

			mockReq.EXPECT().ConferenceV1ConferencecallGet(ctx, tt.conferencecallID).Return(tt.responseConferencecall, nil)

			res, err := h.ConferencecallGet(ctx, tt.customer, tt.conferencecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferencecallGets(t *testing.T) {

	tests := []struct {
		name      string
		customer  *cscustomer.Customer
		token     string
		limit     uint64
		response  []cfconferencecall.Conferencecall
		expectRes []*cfconferencecall.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			"2020-09-20T03:23:20.995000",
			10,
			[]cfconferencecall.Conferencecall{
				{
					ID:         uuid.FromStringOrNil("7fc9cf36-50ca-11ee-9003-c76f7d344275"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				},
			},
			[]*cfconferencecall.WebhookMessage{
				{
					ID:         uuid.FromStringOrNil("7fc9cf36-50ca-11ee-9003-c76f7d344275"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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

			mockReq.EXPECT().ConferenceV1ConferencecallGets(ctx, tt.customer.ID, tt.token, tt.limit).Return(tt.response, nil)
			res, err := h.ConferencecallGets(ctx, tt.customer, tt.limit, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferencecallKick(t *testing.T) {

	tests := []struct {
		name             string
		customer         *cscustomer.Customer
		conferencecallID uuid.UUID

		responseConferencecall *cfconferencecall.Conferencecall

		expectRes *cfconferencecall.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("c5291d7e-15ad-11ed-97e7-239ea4fba3e3"),

			&cfconferencecall.Conferencecall{
				ID:         uuid.FromStringOrNil("c5291d7e-15ad-11ed-97e7-239ea4fba3e3"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},

			&cfconferencecall.WebhookMessage{
				ID:         uuid.FromStringOrNil("c5291d7e-15ad-11ed-97e7-239ea4fba3e3"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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

			mockReq.EXPECT().ConferenceV1ConferencecallGet(ctx, tt.conferencecallID).Return(tt.responseConferencecall, nil)
			mockReq.EXPECT().ConferenceV1ConferencecallKick(ctx, tt.conferencecallID).Return(tt.responseConferencecall, nil)

			res, err := h.ConferencecallKick(ctx, tt.customer, tt.conferencecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
