package messagehandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/smshandler"
	"monorepo/bin-conversation-manager/pkg/whatsapphandler"
)

func Test_Send_sendLine(t *testing.T) {

	tests := []struct {
		name string

		conversation *conversation.Conversation
		text         string
		medias       []media.Media

		responseAccount *account.Account
		responseUUID    uuid.UUID

		expectUpdateFields map[message.Field]any
		expectMessage      *message.Message
	}{
		{
			name: "line text type",

			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7b1034a8-e6ef-11ec-9e9d-c3f3e36741ac"),
					CustomerID: uuid.FromStringOrNil("e54ded88-e6ef-11ec-83af-7fac5b21e9aa"),
				},
				AccountID: uuid.FromStringOrNil("086b4920-fe3f-11ed-b570-bf801ec89642"),
				Type:      conversation.TypeLine,
				DialogID:  "18a7a0e8-e6f0-11ec-8cee-47dd7e7164e3",
				Self: commonaddress.Address{
					Target: "75a20d08-f1de-11ec-8eb1-97f517197fe2",
				},
			},
			text:   "hello, this is test message.",
			medias: []media.Media{},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("086b4920-fe3f-11ed-b570-bf801ec89642"),
				},
			},
			responseUUID: uuid.FromStringOrNil("a0a1478e-0246-11ee-9a48-57bb5aa639c8"),

			expectUpdateFields: map[message.Field]any{
				message.FieldStatus: message.StatusDone,
			},
			expectMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a0a1478e-0246-11ee-9a48-57bb5aa639c8"),
					CustomerID: uuid.FromStringOrNil("e54ded88-e6ef-11ec-83af-7fac5b21e9aa"),
				},
				ConversationID: uuid.FromStringOrNil("7b1034a8-e6ef-11ec-9e9d-c3f3e36741ac"),
				Direction:      message.DirectionOutgoing,
				Status:         message.StatusProgressing,
				ReferenceType:  message.ReferenceTypeLine,
				ReferenceID:    uuid.Nil,
				TransactionID:  "",
				Text:           "hello, this is test message.",
				Medias:         []media.Media{},
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
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &messageHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				notifyHandler:  mockNotify,
				accountHandler: mockAccount,
				lineHandler:    mockLine,
			}
			ctx := context.Background()

			mockAccount.EXPECT().Get(ctx, tt.conversation.AccountID).Return(tt.responseAccount, nil)

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, gomock.Any()).Return(tt.expectMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectMessage.CustomerID, message.EventTypeMessageCreated, tt.expectMessage)

			mockLine.EXPECT().Send(ctx, tt.conversation, tt.responseAccount, tt.text, tt.medias).Return(nil)

			// update
			mockDB.EXPECT().MessageUpdate(ctx, tt.expectMessage.ID, tt.expectUpdateFields).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.expectMessage.ID).Return(tt.expectMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectMessage.CustomerID, message.EventTypeMessageUpdated, tt.expectMessage)

			res, err := h.Send(ctx, tt.conversation, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectMessage, res)
			}
		})
	}
}

func Test_Send_sendSMS(t *testing.T) {

	tests := []struct {
		name string

		conversation *conversation.Conversation
		text         string
		medias       []media.Media

		responseUUIDSmsID uuid.UUID

		expectMessage *message.Message
	}{
		{
			name: "line text type",

			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("063e96aa-1bc2-11f0-bf97-6b63f3f47bbd"),
					CustomerID: uuid.FromStringOrNil("06706478-1bc2-11f0-963f-67d210800b88"),
				},
				Type: conversation.TypeMessage,
				Self: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+123456789",
				},
				Peer: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+987654321",
				},
			},
			text:   "hello, this is test message.",
			medias: []media.Media{},

			responseUUIDSmsID: uuid.FromStringOrNil("06d11af2-1bc2-11f0-bf96-2f9dcd281889"),

			expectMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("06d11af2-1bc2-11f0-bf96-2f9dcd281889"),
					CustomerID: uuid.FromStringOrNil("06706478-1bc2-11f0-963f-67d210800b88"),
				},
				ConversationID: uuid.FromStringOrNil("063e96aa-1bc2-11f0-bf97-6b63f3f47bbd"),
				Direction:      message.DirectionOutgoing,
				Status:         message.StatusProgressing,
				ReferenceType:  message.ReferenceTypeMessage,
				ReferenceID:    uuid.FromStringOrNil("06d11af2-1bc2-11f0-bf96-2f9dcd281889"),
				TransactionID:  "",
				Text:           "hello, this is test message.",
				Medias:         []media.Media{},
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
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			mockSms := smshandler.NewMockSMSHandler(mc)
			h := &messageHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				notifyHandler:  mockNotify,
				accountHandler: mockAccount,
				lineHandler:    mockLine,
				smsHandler:     mockSms,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDSmsID)
			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.expectMessage.ID).Return(tt.expectMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectMessage.CustomerID, message.EventTypeMessageCreated, tt.expectMessage)

			mockSms.EXPECT().Send(ctx, tt.conversation, tt.responseUUIDSmsID, tt.text).Return(nil)

			res, err := h.Send(ctx, tt.conversation, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectMessage, res)
			}
		})
	}
}

func Test_Send_sendWhatsApp(t *testing.T) {

	tests := []struct {
		name string

		conversation *conversation.Conversation
		text         string
		medias       []media.Media

		responseAccount         *account.Account
		responseUUID            uuid.UUID
		responseWamid           string

		expectCreateMessage     *message.Message
		expectUpdateFields      map[message.Field]any
		expectUpdateWamidFields map[message.Field]any
		expectMessage           *message.Message
	}{
		{
			name: "whatsapp text type",

			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1a9c2d0-1234-11f0-aaaa-aaaaaaaaaaaa"),
					CustomerID: uuid.FromStringOrNil("b2b0c3d1-1234-11f0-bbbb-bbbbbbbbbbbb"),
				},
				AccountID: uuid.FromStringOrNil("b3c1d4e2-1234-11f0-cccc-cccccccccccc"),
				Type:      conversation.TypeWhatsApp,
				DialogID:  "12345678901234",
				Self: commonaddress.Address{
					Target: "+15551234567",
				},
			},
			text:   "hello from whatsapp",
			medias: []media.Media{},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b3c1d4e2-1234-11f0-cccc-cccccccccccc"),
				},
			},
			responseUUID:  uuid.FromStringOrNil("b4d2e5f3-1234-11f0-dddd-dddddddddddd"),
			responseWamid: "wamid.abc123xyz",

			expectCreateMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b4d2e5f3-1234-11f0-dddd-dddddddddddd"),
					CustomerID: uuid.FromStringOrNil("b2b0c3d1-1234-11f0-bbbb-bbbbbbbbbbbb"),
				},
				ConversationID: uuid.FromStringOrNil("b1a9c2d0-1234-11f0-aaaa-aaaaaaaaaaaa"),
				Direction:      message.DirectionOutgoing,
				Status:         message.StatusProgressing,
				ReferenceType:  message.ReferenceTypeWhatsApp,
				ReferenceID:    uuid.Nil,
				TransactionID:  "",
				Text:           "hello from whatsapp",
				Medias:         []media.Media{},
			},
			expectUpdateFields: map[message.Field]any{
				message.FieldStatus: message.StatusDone,
			},
			expectUpdateWamidFields: map[message.Field]any{
				message.FieldTransactionID: "wamid.abc123xyz",
			},
			expectMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b4d2e5f3-1234-11f0-dddd-dddddddddddd"),
					CustomerID: uuid.FromStringOrNil("b2b0c3d1-1234-11f0-bbbb-bbbbbbbbbbbb"),
				},
				ConversationID: uuid.FromStringOrNil("b1a9c2d0-1234-11f0-aaaa-aaaaaaaaaaaa"),
				Direction:      message.DirectionOutgoing,
				Status:         message.StatusProgressing,
				ReferenceType:  message.ReferenceTypeWhatsApp,
				ReferenceID:    uuid.Nil,
				TransactionID:  "",
				Text:           "hello from whatsapp",
				Medias:         []media.Media{},
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
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockWhatsapp := whatsapphandler.NewMockWhatsAppHandler(mc)
			h := &messageHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				notifyHandler:   mockNotify,
				accountHandler:  mockAccount,
				whatsappHandler: mockWhatsapp,
			}
			ctx := context.Background()

			mockAccount.EXPECT().Get(ctx, tt.conversation.AccountID).Return(tt.responseAccount, nil)

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().MessageCreate(ctx, tt.expectCreateMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.responseUUID).Return(tt.expectCreateMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectCreateMessage.CustomerID, message.EventTypeMessageCreated, tt.expectCreateMessage)

			mockWhatsapp.EXPECT().Send(ctx, tt.conversation, tt.responseAccount, tt.text).Return(tt.responseWamid, nil)

			// update status to done
			mockDB.EXPECT().MessageUpdate(ctx, tt.expectCreateMessage.ID, tt.expectUpdateFields).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.expectCreateMessage.ID).Return(tt.expectMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectMessage.CustomerID, message.EventTypeMessageUpdated, tt.expectMessage)

			// persist wamid
			mockDB.EXPECT().MessageUpdate(ctx, tt.expectMessage.ID, tt.expectUpdateWamidFields).Return(nil)

			res, err := h.Send(ctx, tt.conversation, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectMessage, res)
			}
		})
	}
}
