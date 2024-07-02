package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	mmmessage "monorepo/bin-message-manager/models/message"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_MessageGets(t *testing.T) {

	tests := []struct {
		name string

		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		response  []mmmessage.Message
		expectRes []*mmmessage.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
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

			mockReq.EXPECT().MessageV1MessageGets(ctx, tt.agent.CustomerID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.MessageGets(ctx, tt.agent, tt.pageSize, tt.pageToken)
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
		name  string
		agent *amagent.Agent
		id    uuid.UUID

		response  *mmmessage.Message
		expectRes *mmmessage.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("5d607ade-a2e4-11ec-b1b8-6fdc099c84f1"),

			&mmmessage.Message{
				ID:         uuid.FromStringOrNil("5d607ade-a2e4-11ec-b1b8-6fdc099c84f1"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			&mmmessage.WebhookMessage{
				ID:         uuid.FromStringOrNil("5d607ade-a2e4-11ec-b1b8-6fdc099c84f1"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			res, err := h.MessageGet(ctx, tt.agent, tt.id)
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
		name  string
		agent *amagent.Agent

		source       *commonaddress.Address
		destinations []commonaddress.Address
		text         string

		response  *mmmessage.Message
		expectRes *mmmessage.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
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

			mockReq.EXPECT().MessageV1MessageSend(ctx, uuid.Nil, tt.agent.CustomerID, tt.source, tt.destinations, tt.text).Return(tt.response, nil)
			res, err := h.MessageSend(ctx, tt.agent, tt.source, tt.destinations, tt.text)
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
		name  string
		agent *amagent.Agent
		id    uuid.UUID

		response  *mmmessage.Message
		expectRes *mmmessage.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("88c326c0-a2e6-11ec-84b4-7f4501f624df"),

			&mmmessage.Message{
				ID:         uuid.FromStringOrNil("88c326c0-a2e6-11ec-84b4-7f4501f624df"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			&mmmessage.WebhookMessage{
				ID:         uuid.FromStringOrNil("88c326c0-a2e6-11ec-84b4-7f4501f624df"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			res, err := h.MessageDelete(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
