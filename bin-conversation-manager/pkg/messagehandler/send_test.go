package messagehandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
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

	wcmessage "monorepo/bin-webchat-manager/models/message"
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
				// outbound: source = Self, destination = Peer (Peer is zero here).
				Source: commonaddress.Address{
					Target: "75a20d08-f1de-11ec-8eb1-97f517197fe2",
				},
				Destination: commonaddress.Address{},
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

func Test_Send_sendWebchat(t *testing.T) {

	tests := []struct {
		name string

		conversation *conversation.Conversation
		text         string

		responseWebchatMessage *wcmessage.Message

		expectSessionID uuid.UUID
		expectMessage   *message.Message
	}{
		{
			name: "webchat text type",

			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c1a9c2d0-1234-11f0-aaaa-aaaaaaaaaaaa"),
					CustomerID: uuid.FromStringOrNil("c2b0c3d1-1234-11f0-bbbb-bbbbbbbbbbbb"),
				},
				Type: conversation.TypeWebchat,
				Self: commonaddress.Address{
					Type:   commonaddress.TypeWebchat,
					Target: "widget-id",
				},
				Peer: commonaddress.Address{
					Type:   commonaddress.TypeWebchat,
					Target: "c3d1e4f2-1234-11f0-cccc-cccccccccccc",
				},
			},
			text: "Hello, welcome to voipbin world!",

			responseWebchatMessage: &wcmessage.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c4e2f5a3-1234-11f0-dddd-dddddddddddd"),
				},
			},

			expectSessionID: uuid.FromStringOrNil("c3d1e4f2-1234-11f0-cccc-cccccccccccc"),
			expectMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c4e2f5a3-1234-11f0-dddd-dddddddddddd"),
					CustomerID: uuid.FromStringOrNil("c2b0c3d1-1234-11f0-bbbb-bbbbbbbbbbbb"),
				},
				ConversationID: uuid.FromStringOrNil("c1a9c2d0-1234-11f0-aaaa-aaaaaaaaaaaa"),
				Direction:      message.DirectionOutgoing,
				Status:         message.StatusDone,
				ReferenceType:  message.ReferenceTypeWebchat,
				ReferenceID:    uuid.FromStringOrNil("c4e2f5a3-1234-11f0-dddd-dddddddddddd"),
				Text:           "Hello, welcome to voipbin world!",
				// outbound: source = Self, destination = Peer.
				Source: commonaddress.Address{
					Type:   commonaddress.TypeWebchat,
					Target: "widget-id",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeWebchat,
					Target: "c3d1e4f2-1234-11f0-cccc-cccccccccccc",
				},
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
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &messageHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}
			ctx := context.Background()

			mockReq.EXPECT().WebchatV1MessageCreate(ctx, tt.conversation.CustomerID, tt.expectSessionID, wcmessage.DirectionOutbound, uuid.Nil, tt.text).Return(tt.responseWebchatMessage, nil)

			// sendWebchat's race-guard Get: no row yet (event handler
			// hasn't won the race), proceed to Create below.
			mockDB.EXPECT().MessageGet(ctx, tt.responseWebchatMessage.ID).Return(nil, dbhandler.ErrNotFound)

			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.expectMessage.ID).Return(tt.expectMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectMessage.CustomerID, message.EventTypeMessageCreated, tt.expectMessage)

			res, err := h.Send(ctx, tt.conversation, tt.text, nil)
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
			name: "sms text type",

			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("063e96aa-1bc2-11f0-bf97-6b63f3f47bbd"),
					CustomerID: uuid.FromStringOrNil("06706478-1bc2-11f0-963f-67d210800b88"),
				},
				Type: conversation.TypeMessage,
				Self: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+123****6789",
				},
				Peer: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+987****4321",
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
				// outbound: source = Self, destination = Peer.
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+123****6789",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+987****4321",
				},
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

		responseAccount *account.Account
		responseUUID    uuid.UUID
		responseWamid   string

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
					Target: "+155****4567",
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
				// sendWhatsApp does not pass medias, so Medias is nil (not []).
				Medias: nil,
				// outbound: source = Self, destination = Peer (Peer is zero here).
				Source: commonaddress.Address{
					Target: "+155****4567",
				},
				Destination: commonaddress.Address{},
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
				Medias:         nil,
				Source: commonaddress.Address{
					Target: "+155****4567",
				},
				Destination: commonaddress.Address{},
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

func Test_Send_sendWebchat_InvalidSessionID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &messageHandler{}
	ctx := context.Background()

	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d1a9c2d0-1234-11f0-aaaa-aaaaaaaaaaaa"),
			CustomerID: uuid.FromStringOrNil("d2b0c3d1-1234-11f0-bbbb-bbbbbbbbbbbb"),
		},
		Type: conversation.TypeWebchat,
		Self: commonaddress.Address{
			Type:   commonaddress.TypeWebchat,
			Target: "widget-id",
		},
		Peer: commonaddress.Address{
			Type:   commonaddress.TypeWebchat,
			Target: "not-a-valid-uuid",
		},
	}

	// WebchatV1MessageCreate must NOT be called -- the session id parse
	// failure must short-circuit before any RPC is attempted. Absence
	// enforced by a nil reqHandler: a call would panic on nil deref.
	res, err := h.Send(ctx, cv, "hello", nil)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok (res: %v)", res)
	}
}

// Test_Send_sendWebchat_RaceEventHandlerWon simulates the
// messageEventSentWebchat subscribed-event path winning the race and
// persisting the same wm.ID before sendWebchat's own Create runs.
// sendWebchat's post-RPC Get must find that row and return it, rather
// than attempting -- and failing -- a duplicate insert.
func Test_Send_sendWebchat_RaceEventHandlerWon(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &messageHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("e2b0c3d1-1234-11f0-bbbb-bbbbbbbbbbbb")
	sessionID := uuid.FromStringOrNil("e3d1e4f2-1234-11f0-cccc-cccccccccccc")
	messageID := uuid.FromStringOrNil("e4e2f5a3-1234-11f0-dddd-dddddddddddd")

	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("e1a9c2d0-1234-11f0-aaaa-aaaaaaaaaaaa"),
			CustomerID: customerID,
		},
		Type: conversation.TypeWebchat,
		Self: commonaddress.Address{
			Type:   commonaddress.TypeWebchat,
			Target: "widget-id",
		},
		Peer: commonaddress.Address{
			Type:   commonaddress.TypeWebchat,
			Target: sessionID.String(),
		},
	}

	existing := &message.Message{
		Identity: commonidentity.Identity{
			ID:         messageID,
			CustomerID: customerID,
		},
		ConversationID: cv.ID,
		Direction:      message.DirectionOutgoing,
		Status:         message.StatusDone,
		ReferenceType:  message.ReferenceTypeWebchat,
		ReferenceID:    messageID,
		Text:           "hello",
	}

	mockReq.EXPECT().WebchatV1MessageCreate(ctx, customerID, sessionID, wcmessage.DirectionOutbound, uuid.Nil, "hello").Return(&wcmessage.Message{
		Identity: commonidentity.Identity{ID: messageID},
	}, nil)

	// The event handler already won the race and persisted this row --
	// sendWebchat's guard Get finds it and must NOT attempt Create.
	mockDB.EXPECT().MessageGet(ctx, messageID).Return(existing, nil)

	res, err := h.Send(ctx, cv, "hello", nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, existing) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", existing, res)
	}
}
