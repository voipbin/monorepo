package messagehandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/linehandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		conversationID uuid.UUID
		direction      message.Direction
		status         message.Status
		referenceType  conversation.ReferenceType
		referenceID    string
		transactionID  string
		source         *commonaddress.Address
		text           string
		medias         []media.Media

		responseUUID    uuid.UUID
		responseMessage *message.Message

		expectMessage *message.Message
	}{
		{
			name: "normal",

			customerID:     uuid.FromStringOrNil("8db56df0-e86e-11ec-b6d7-1fa3ca565837"),
			conversationID: uuid.FromStringOrNil("8e0e1dce-e86e-11ec-9537-77df0d80af26"),
			direction:      message.DirectionIncoming,
			status:         message.StatusSent,
			referenceType:  conversation.ReferenceTypeLine,
			referenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
			transactionID:  "59946c7c-f1d5-11ec-bdad-2323294b508e",
			source: &commonaddress.Address{
				Type:   commonaddress.TypeLine,
				Target: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
			},
			text:   "Hello world",
			medias: []media.Media{},

			responseUUID: uuid.FromStringOrNil("f6834112-0240-11ee-8146-2fb17ae9ef0a"),
			responseMessage: &message.Message{
				ID:         uuid.FromStringOrNil("03b6a572-e870-11ec-a4a7-0bf92e5d8985"),
				CustomerID: uuid.FromStringOrNil("8db56df0-e86e-11ec-b6d7-1fa3ca565837"),
			},

			expectMessage: &message.Message{
				ID:             uuid.FromStringOrNil("f6834112-0240-11ee-8146-2fb17ae9ef0a"),
				CustomerID:     uuid.FromStringOrNil("8db56df0-e86e-11ec-b6d7-1fa3ca565837"),
				ConversationID: uuid.FromStringOrNil("8e0e1dce-e86e-11ec-9537-77df0d80af26"),
				Direction:      message.DirectionIncoming,
				Status:         message.StatusSent,
				ReferenceType:  conversation.ReferenceTypeLine,
				ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				TransactionID:  "59946c7c-f1d5-11ec-bdad-2323294b508e",
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeLine,
					Target: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				},
				Text:   "Hello world",
				Medias: []media.Media{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &messageHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.responseUUID).Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessage.CustomerID, message.EventTypeMessageCreated, tt.responseMessage)
			res, err := h.Create(ctx, tt.customerID, tt.conversationID, tt.direction, tt.status, tt.referenceType, tt.referenceID, tt.transactionID, tt.source, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseMessage, res)
			}

		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseMessage *message.Message
	}{
		{
			"normal",

			uuid.FromStringOrNil("af7c3310-f1d8-11ec-a2f1-db31b43cade8"),

			&message.Message{
				ID:         uuid.FromStringOrNil("af7c3310-f1d8-11ec-a2f1-db31b43cade8"),
				CustomerID: uuid.FromStringOrNil("8db56df0-e86e-11ec-b6d7-1fa3ca565837"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}
			ctx := context.Background()

			mockDB.EXPECT().MessageDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessage.CustomerID, message.EventTypeMessageDeleted, tt.responseMessage)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseMessage, res)
			}
		})
	}
}

func Test_GetsByConversationID(t *testing.T) {

	tests := []struct {
		name string

		conversationID uuid.UUID
		pageToken      string
		pageSize       uint64

		responseMessages []*message.Message
	}{
		{
			"normal",

			uuid.FromStringOrNil("7b1034a8-e6ef-11ec-9e9d-c3f3e36741ac"),
			"2022-04-18 03:22:17.995000",
			100,

			[]*message.Message{
				{
					ID: uuid.FromStringOrNil("ead67924-e7bb-11ec-9f65-a7aafd81f40b"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}
			ctx := context.Background()

			mockDB.EXPECT().MessageGetsByConversationID(ctx, tt.conversationID, tt.pageToken, tt.pageSize).Return(tt.responseMessages, nil)

			res, err := h.GetsByConversationID(ctx, tt.conversationID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessages) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseMessages, res)
			}

		})
	}
}

func Test_GetsByTransactionID(t *testing.T) {

	tests := []struct {
		name string

		transactionID string
		pageToken     string
		pageSize      uint64

		responseMessages []*message.Message
	}{
		{
			"normal",

			"2c4b24a4-f2ac-11ec-979b-5f7a7f205308",
			"2022-04-18 03:22:17.995000",
			100,

			[]*message.Message{
				{
					ID: uuid.FromStringOrNil("ead67924-e7bb-11ec-9f65-a7aafd81f40b"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}
			ctx := context.Background()

			mockDB.EXPECT().MessageGetsByTransactionID(ctx, tt.transactionID, tt.pageToken, tt.pageSize).Return(tt.responseMessages, nil)

			res, err := h.GetsByTransactionID(ctx, tt.transactionID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessages) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseMessages, res)
			}

		})
	}
}
