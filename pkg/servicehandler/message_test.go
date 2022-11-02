package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_MessageGets(t *testing.T) {

	tests := []struct {
		name string

		customer  *cscustomer.Customer
		pageToken string
		pageSize  uint64

		response  []mmmessage.Message
		expectRes []*mmmessage.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			"2021-03-01 01:00:00.995000",
			10,

			[]mmmessage.Message{
				{
					ID: uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
				},
			},
			[]*mmmessage.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
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

			mockReq.EXPECT().MessageV1MessageGets(ctx, tt.customer.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.MessageGets(ctx, tt.customer, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, num := range res {
				num.TMCreate = ""
				num.TMUpdate = ""
				num.TMDelete = ""
			}

			if !reflect.DeepEqual(res[0], tt.expectRes[0]) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_MessageGet(t *testing.T) {

	tests := []struct {
		name     string
		customer *cscustomer.Customer
		id       uuid.UUID

		response  *mmmessage.Message
		expectRes *mmmessage.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("5d4cca48-a2e4-11ec-b285-abf68e05b01d"),
			},
			uuid.FromStringOrNil("5d607ade-a2e4-11ec-b1b8-6fdc099c84f1"),

			&mmmessage.Message{
				ID:         uuid.FromStringOrNil("5d607ade-a2e4-11ec-b1b8-6fdc099c84f1"),
				CustomerID: uuid.FromStringOrNil("5d4cca48-a2e4-11ec-b285-abf68e05b01d"),
				TMDelete:   defaultTimestamp,
			},
			&mmmessage.WebhookMessage{
				ID:         uuid.FromStringOrNil("5d607ade-a2e4-11ec-b1b8-6fdc099c84f1"),
				CustomerID: uuid.FromStringOrNil("5d4cca48-a2e4-11ec-b285-abf68e05b01d"),
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

			mockReq.EXPECT().MessageV1MessageGet(ctx, tt.id).Return(tt.response, nil)

			res, err := h.MessageGet(ctx, tt.customer, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_MessageSend(t *testing.T) {

	tests := []struct {
		name     string
		customer *cscustomer.Customer

		source       *commonaddress.Address
		destinations []commonaddress.Address
		text         string

		response  *mmmessage.Message
		expectRes *mmmessage.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			"hello world",

			&mmmessage.Message{
				ID: uuid.FromStringOrNil("14e68482-a2e5-11ec-8d92-6bc6fec62487"),
			},
			&mmmessage.WebhookMessage{
				ID: uuid.FromStringOrNil("14e68482-a2e5-11ec-8d92-6bc6fec62487"),
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

			mockReq.EXPECT().MessageV1MessageSend(ctx, uuid.Nil, tt.customer.ID, tt.source, tt.destinations, tt.text).Return(tt.response, nil)
			res, err := h.MessageSend(ctx, tt.customer, tt.source, tt.destinations, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_MessageDelete(t *testing.T) {

	tests := []struct {
		name     string
		customer *cscustomer.Customer
		id       uuid.UUID

		response  *mmmessage.Message
		expectRes *mmmessage.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("8866d636-a2e6-11ec-88f1-b76cfda0af43"),
			},
			uuid.FromStringOrNil("88c326c0-a2e6-11ec-84b4-7f4501f624df"),

			&mmmessage.Message{
				ID:         uuid.FromStringOrNil("88c326c0-a2e6-11ec-84b4-7f4501f624df"),
				CustomerID: uuid.FromStringOrNil("8866d636-a2e6-11ec-88f1-b76cfda0af43"),
				TMDelete:   defaultTimestamp,
			},
			&mmmessage.WebhookMessage{
				ID:         uuid.FromStringOrNil("88c326c0-a2e6-11ec-84b4-7f4501f624df"),
				CustomerID: uuid.FromStringOrNil("8866d636-a2e6-11ec-88f1-b76cfda0af43"),
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

			mockReq.EXPECT().MessageV1MessageGet(ctx, tt.id).Return(tt.response, nil)
			mockReq.EXPECT().MessageV1MessageDelete(ctx, tt.id).Return(tt.response, nil)

			res, err := h.MessageDelete(ctx, tt.customer, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
