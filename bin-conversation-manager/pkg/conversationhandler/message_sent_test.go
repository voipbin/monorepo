package conversationhandler

import (
	"context"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
	mmmessage "monorepo/bin-message-manager/models/message"
	mmtarget "monorepo/bin-message-manager/models/target"
)

func Test_MessageEventSent(t *testing.T) {
	customerID := uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000001")
	smsID := uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000002")
	caseID := uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000003")

	selfAddr := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"}
	peerAddr1 := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000002"}
	peerAddr2 := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000003"}

	newConversation := func(idSuffix string, peer commonaddress.Address, metadata *conversation.Metadata) *conversation.Conversation {
		return &conversation.Conversation{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("d2000000-0000-0000-0000-" + idSuffix),
				CustomerID: customerID,
			},
			Type:     conversation.TypeMessage,
			Self:     selfAddr,
			Peer:     peer,
			Metadata: metadata,
		}
	}

	newIncoming := func(targets ...commonaddress.Address) *mmmessage.Message {
		res := &mmmessage.Message{
			Identity: commonidentity.Identity{ID: smsID, CustomerID: customerID},
			Source:   &selfAddr,
			Text:     "sent text",
		}
		for _, d := range targets {
			res.Targets = append(res.Targets, mmtarget.Target{Destination: d})
		}
		return res
	}

	tests := []struct {
		name string

		incoming *mmmessage.Message

		// per target: the conversation returned by the lookup
		conversations []*conversation.Conversation
		// per target: nil means "message does not exist yet" and a Create call is expected
		existing []*message.Message

		expectCaseID *uuid.UUID
	}{
		{
			name:     "message not stored yet creates an outgoing message with self as source and peer as destination",
			incoming: newIncoming(peerAddr1),
			conversations: []*conversation.Conversation{
				newConversation("000000000001", peerAddr1, nil),
			},
			existing: []*message.Message{nil},
		},
		{
			name:     "case hint of the conversation is passed to the created message",
			incoming: newIncoming(peerAddr1),
			conversations: []*conversation.Conversation{
				newConversation("000000000002", peerAddr1, &conversation.Metadata{ContactCaseID: &caseID}),
			},
			existing:     []*message.Message{nil},
			expectCaseID: &caseID,
		},
		{
			name:     "message already stored only gets its status updated",
			incoming: newIncoming(peerAddr1),
			conversations: []*conversation.Conversation{
				newConversation("000000000003", peerAddr1, nil),
			},
			existing: []*message.Message{
				{Identity: commonidentity.Identity{ID: smsID, CustomerID: customerID}},
			},
		},
		{
			name:     "two targets are handled independently with their own destination",
			incoming: newIncoming(peerAddr1, peerAddr2),
			conversations: []*conversation.Conversation{
				newConversation("000000000004", peerAddr1, nil),
				newConversation("000000000005", peerAddr2, nil),
			},
			existing: []*message.Message{nil, nil},
		},
		{
			name:          "no target does nothing",
			incoming:      newIncoming(),
			conversations: nil,
			existing:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			h := &conversationHandler{
				db:             mockDB,
				messageHandler: mockMessage,
			}

			ctx := context.Background()

			for i, cv := range tt.conversations {
				peer := tt.incoming.Targets[i].Destination

				mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, selfAddr, peer).Return(cv, nil)

				if tt.existing[i] == nil {
					mockMessage.EXPECT().Get(ctx, smsID).Return(nil, dbhandler.ErrNotFound)
					mockMessage.EXPECT().Create(ctx, messagehandler.MessageCreateArgs{
						ID:             smsID,
						CustomerID:     customerID,
						ConversationID: cv.ID,
						Direction:      message.DirectionOutgoing,
						Status:         message.StatusDone,
						ReferenceType:  message.ReferenceTypeMessage,
						ReferenceID:    smsID,
						Text:           "sent text",
						Medias:         []media.Media{},
						Source:         selfAddr,
						Destination:    peer,
						CaseID:         tt.expectCaseID,
					}).Return(&message.Message{Identity: commonidentity.Identity{ID: smsID}}, nil)
				} else {
					mockMessage.EXPECT().Get(ctx, smsID).Return(tt.existing[i], nil)
					mockMessage.EXPECT().UpdateStatus(ctx, smsID, message.StatusDone).Return(tt.existing[i], nil)
				}
			}

			if err := h.MessageEventSent(ctx, tt.incoming); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_MessageEventSent_errors(t *testing.T) {
	customerID := uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000011")
	smsID := uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000012")

	selfAddr := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000011"}
	peerAddr := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000012"}

	incoming := &mmmessage.Message{
		Identity: commonidentity.Identity{ID: smsID, CustomerID: customerID},
		Source:   &selfAddr,
		Targets:  []mmtarget.Target{{Destination: peerAddr}},
		Text:     "sent text",
	}
	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000011"),
			CustomerID: customerID,
		},
		Type: conversation.TypeMessage,
		Self: selfAddr,
		Peer: peerAddr,
	}

	tests := []struct {
		name  string
		setup func(mockDB *dbhandler.MockDBHandler, mockMessage *messagehandler.MockMessageHandler, ctx context.Context)
	}{
		{
			name: "conversation lookup and creation both fail",
			setup: func(mockDB *dbhandler.MockDBHandler, mockMessage *messagehandler.MockMessageHandler, ctx context.Context) {
				mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, selfAddr, peerAddr).Return(nil, dbhandler.ErrNotFound)
				mockDB.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(fmt.Errorf("create failed"))
			},
		},
		{
			name: "message create fails",
			setup: func(mockDB *dbhandler.MockDBHandler, mockMessage *messagehandler.MockMessageHandler, ctx context.Context) {
				mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, selfAddr, peerAddr).Return(cv, nil)
				mockMessage.EXPECT().Get(ctx, smsID).Return(nil, dbhandler.ErrNotFound)
				mockMessage.EXPECT().Create(ctx, gomock.Any()).Return(nil, fmt.Errorf("create failed"))
			},
		},
		{
			name: "status update of the stored message fails",
			setup: func(mockDB *dbhandler.MockDBHandler, mockMessage *messagehandler.MockMessageHandler, ctx context.Context) {
				mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, selfAddr, peerAddr).Return(cv, nil)
				mockMessage.EXPECT().Get(ctx, smsID).Return(&message.Message{Identity: commonidentity.Identity{ID: smsID}}, nil)
				mockMessage.EXPECT().UpdateStatus(ctx, smsID, message.StatusDone).Return(nil, fmt.Errorf("update failed"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &conversationHandler{
				utilHandler:    utilhandler.NewUtilHandler(),
				db:             mockDB,
				notifyHandler:  mockNotify,
				messageHandler: mockMessage,
			}

			ctx := context.Background()
			tt.setup(mockDB, mockMessage, ctx)

			if err := h.MessageEventSent(ctx, incoming); err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func Test_MessageSend_errors(t *testing.T) {
	conversationID := uuid.FromStringOrNil("d3000000-0000-0000-0000-000000000001")
	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{ID: conversationID},
		Type:     conversation.TypeLine,
	}

	tests := []struct {
		name  string
		setup func(mockDB *dbhandler.MockDBHandler, mockMessage *messagehandler.MockMessageHandler, ctx context.Context)
	}{
		{
			name: "conversation not found",
			setup: func(mockDB *dbhandler.MockDBHandler, mockMessage *messagehandler.MockMessageHandler, ctx context.Context) {
				mockDB.EXPECT().ConversationGet(ctx, conversationID).Return(nil, dbhandler.ErrNotFound)
			},
		},
		{
			name: "send fails",
			setup: func(mockDB *dbhandler.MockDBHandler, mockMessage *messagehandler.MockMessageHandler, ctx context.Context) {
				mockDB.EXPECT().ConversationGet(ctx, conversationID).Return(cv, nil)
				mockMessage.EXPECT().Send(ctx, cv, "text", []media.Media{}).Return(nil, fmt.Errorf("send failed"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			h := &conversationHandler{
				db:             mockDB,
				messageHandler: mockMessage,
			}

			ctx := context.Background()
			tt.setup(mockDB, mockMessage, ctx)

			res, err := h.MessageSend(ctx, conversationID, "text", []media.Media{})
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}
