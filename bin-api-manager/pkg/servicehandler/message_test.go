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

	"monorepo/bin-api-manager/models/auth"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_MessageList(t *testing.T) {

	tests := []struct {
		name string

		agent     *auth.AuthIdentity
		pageToken string
		pageSize  uint64

		response  []mmmessage.Message
		expectRes []*mmmessage.WebhookMessage
	}{
		{
			"normal",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			"2021-03-01T01:00:00.995000Z",
			10,

			[]mmmessage.Message{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
					},
				},
			},
			[]*mmmessage.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
					},
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

			mockReq.EXPECT().MessageV1MessageList(ctx, tt.pageToken, tt.pageSize, gomock.Any()).Return(tt.response, nil)

			res, err := h.MessageList(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, num := range res {
				num.TMCreate = nil
				num.TMUpdate = nil
				num.TMDelete = nil
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
		agent *auth.AuthIdentity
		id    uuid.UUID

		response  *mmmessage.Message
		expectRes *mmmessage.WebhookMessage
	}{
		{
			"normal",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			uuid.FromStringOrNil("5d607ade-a2e4-11ec-b1b8-6fdc099c84f1"),

			&mmmessage.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5d607ade-a2e4-11ec-b1b8-6fdc099c84f1"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
			},
			&mmmessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5d607ade-a2e4-11ec-b1b8-6fdc099c84f1"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
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
		agent *auth.AuthIdentity

		source       *commonaddress.Address
		destinations []commonaddress.Address
		text         string

		response  *mmmessage.Message
		expectRes *mmmessage.WebhookMessage
	}{
		{
			"normal",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("14e68482-a2e5-11ec-8d92-6bc6fec62487"),
				},
			},
			&mmmessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("14e68482-a2e5-11ec-8d92-6bc6fec62487"),
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

func Test_MessageSendMMS(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821****0001"}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+821****0002"}}
	medias := []uuid.UUID{uuid.FromStringOrNil("3f2b1c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d")}

	admin := auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"), CustomerID: customerID},
		Permission: amagent.PermissionCustomerAdmin,
	})
	noPermission := auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"), CustomerID: customerID},
		Permission: amagent.PermissionNone,
	})

	t.Run("normal", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}
		ctx := context.Background()

		response := &mmmessage.Message{
			Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("14e68482-a2e5-11ec-8d92-6bc6fec62487")},
			Type:     mmmessage.TypeMMS,
			Medias:   []string{medias[0].String()},
		}
		mockReq.EXPECT().MessageV1MessageSendMMS(ctx, uuid.Nil, customerID, source, destinations, "hello", medias).Return(response, nil)

		res, err := h.MessageSendMMS(ctx, admin, source, destinations, "hello", medias)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if !reflect.DeepEqual(res, response.ConvertWebhookMessage()) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", response.ConvertWebhookMessage(), res)
		}
		if res.Type != mmmessage.TypeMMS || len(res.Medias) != 1 {
			t.Errorf("The response must expose the type and the medias. got: %v", res)
		}
	})

	t.Run("no permission never reaches the message-manager", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}
		_, err := h.MessageSendMMS(context.Background(), noPermission, source, destinations, "hello", medias)
		if err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("no destination", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}
		_, err := h.MessageSendMMS(context.Background(), admin, source, nil, "hello", medias)
		if err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})
}

func Test_MessageDelete(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity
		id    uuid.UUID

		response  *mmmessage.Message
		expectRes *mmmessage.WebhookMessage
	}{
		{
			"normal",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			uuid.FromStringOrNil("88c326c0-a2e6-11ec-84b4-7f4501f624df"),

			&mmmessage.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("88c326c0-a2e6-11ec-84b4-7f4501f624df"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
			},
			&mmmessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("88c326c0-a2e6-11ec-84b4-7f4501f624df"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
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
