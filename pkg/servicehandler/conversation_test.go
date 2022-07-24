package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cvconversation "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	cvmedia "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	cvmessage "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_ConversationGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name      string
		customer  *cscustomer.Customer
		pageToken string
		pageSize  uint64

		response  []cvconversation.Conversation
		expectRes []*cvconversation.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]cvconversation.Conversation{
				{
					ID: uuid.FromStringOrNil("18965a18-ed21-11ec-89d2-b7e541377482"),
				},
				{
					ID: uuid.FromStringOrNil("18c13288-ed21-11ec-9d0f-c7be55dc87d7"),
				},
			},
			[]*cvconversation.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("18965a18-ed21-11ec-89d2-b7e541377482"),
				},
				{
					ID: uuid.FromStringOrNil("18c13288-ed21-11ec-9d0f-c7be55dc87d7"),
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

			mockReq.EXPECT().ConversationV1ConversationGetsByCustomerID(ctx, tt.customer.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)
			res, err := h.ConversationGetsByCustomerID(ctx, tt.customer, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationGet(t *testing.T) {

	tests := []struct {
		name           string
		customer       *cscustomer.Customer
		conversationID uuid.UUID

		response  *cvconversation.Conversation
		expectRes *cvconversation.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("828e75ba-ed24-11ec-bbf2-7f0e56ac76f1"),

			&cvconversation.Conversation{
				ID:         uuid.FromStringOrNil("828e75ba-ed24-11ec-bbf2-7f0e56ac76f1"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&cvconversation.WebhookMessage{
				ID: uuid.FromStringOrNil("828e75ba-ed24-11ec-bbf2-7f0e56ac76f1"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
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

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.response, nil)
			res, err := h.ConversationGet(ctx, tt.customer, tt.conversationID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationMessageGetsByConversationID(t *testing.T) {

	tests := []struct {
		name           string
		customer       *cscustomer.Customer
		conversationID uuid.UUID
		pageToken      string
		pageSize       uint64

		responseConversation *cvconversation.Conversation
		responseMessages     []cvmessage.Message
		expectRes            []*cvmessage.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("ee26103a-ed24-11ec-bfa1-7b247ecf7e93"),
			"2020-10-20T01:00:00.995000",
			10,

			&cvconversation.Conversation{
				ID:         uuid.FromStringOrNil("ee26103a-ed24-11ec-bfa1-7b247ecf7e93"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			[]cvmessage.Message{
				{
					ID: uuid.FromStringOrNil("13c78e5e-ed25-11ec-b924-b319c14e0209"),
				},
				{
					ID: uuid.FromStringOrNil("13e8436a-ed25-11ec-ba44-8b0716e4b2f0"),
				},
			},
			[]*cvmessage.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("13c78e5e-ed25-11ec-b924-b319c14e0209"),
				},
				{
					ID: uuid.FromStringOrNil("13e8436a-ed25-11ec-ba44-8b0716e4b2f0"),
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

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
			mockReq.EXPECT().ConversationV1ConversationMessageGetsByConversationID(ctx, tt.conversationID, tt.pageToken, tt.pageSize).Return(tt.responseMessages, nil)

			res, err := h.ConversationMessageGetsByConversationID(ctx, tt.customer, tt.conversationID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationMessageSend(t *testing.T) {

	tests := []struct {
		name string

		customer       *cscustomer.Customer
		conversationID uuid.UUID
		text           string
		medias         []cvmedia.Media

		responseConversation *cvconversation.Conversation
		responseMessage      *cvmessage.Message

		expectRes *cvmessage.WebhookMessage
	}{
		{
			"simple text message",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("8dd8eda0-ed25-11ec-9b1a-07913127a65a"),
			"hello world",
			[]cvmedia.Media{},

			&cvconversation.Conversation{
				ID:         uuid.FromStringOrNil("8dd8eda0-ed25-11ec-9b1a-07913127a65a"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&cvmessage.Message{
				ID: uuid.FromStringOrNil("c9bd73a4-ed25-11ec-8283-43aafea65e87"),
			},

			&cvmessage.WebhookMessage{
				ID: uuid.FromStringOrNil("c9bd73a4-ed25-11ec-8283-43aafea65e87"),
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

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
			mockReq.EXPECT().ConversationV1MessageSend(ctx, tt.conversationID, tt.text, tt.medias).Return(tt.responseMessage, nil)

			res, err := h.ConversationMessageSend(ctx, tt.customer, tt.conversationID, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
