package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	chatbotchatbotcall "gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_ChatbotcallGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		size     uint64
		token    string

		response  []chatbotchatbotcall.Chatbotcall
		expectRes []*chatbotchatbotcall.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("040422b6-3771-11ed-801b-27518c703c82"),
			},
			10,
			"2020-09-20 03:23:20.995000",

			[]chatbotchatbotcall.Chatbotcall{
				{
					ID: uuid.FromStringOrNil("78b58aef-2fcf-4a88-81e2-054f4e4c37d4"),
				},
			},
			[]*chatbotchatbotcall.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("78b58aef-2fcf-4a88-81e2-054f4e4c37d4"),
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

			mockReq.EXPECT().ChatbotV1ChatbotcallGetsByCustomerID(ctx, tt.customer.ID, tt.token, tt.size).Return(tt.response, nil)

			res, err := h.ChatbotcallGetsByCustomerID(ctx, tt.customer, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotcallGet(t *testing.T) {

	tests := []struct {
		name string

		customer      *cscustomer.Customer
		chatbotcallID uuid.UUID

		response  *chatbotchatbotcall.Chatbotcall
		expectRes *chatbotchatbotcall.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("539c78aa-3771-11ed-ab19-379f45ca7efc"),
			},
			uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),

			&chatbotchatbotcall.Chatbotcall{
				ID:         uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),
				CustomerID: uuid.FromStringOrNil("539c78aa-3771-11ed-ab19-379f45ca7efc"),
			},
			&chatbotchatbotcall.WebhookMessage{
				ID:         uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),
				CustomerID: uuid.FromStringOrNil("539c78aa-3771-11ed-ab19-379f45ca7efc"),
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

			mockReq.EXPECT().ChatbotV1ChatbotcallGet(ctx, tt.chatbotcallID).Return(tt.response, nil)

			res, err := h.ChatbotcallGet(ctx, tt.customer, tt.chatbotcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotcallDelete(t *testing.T) {

	tests := []struct {
		name string

		customer      *cscustomer.Customer
		chatbotcallID uuid.UUID

		responseChat *chatbotchatbotcall.Chatbotcall
		expectRes    *chatbotchatbotcall.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			&chatbotchatbotcall.Chatbotcall{
				ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			&chatbotchatbotcall.WebhookMessage{
				ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
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

			mockReq.EXPECT().ChatbotV1ChatbotcallGet(ctx, tt.chatbotcallID).Return(tt.responseChat, nil)
			mockReq.EXPECT().ChatbotV1ChatbotcallDelete(ctx, tt.chatbotcallID).Return(tt.responseChat, nil)

			res, err := h.ChatbotcallDelete(ctx, tt.customer, tt.chatbotcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
