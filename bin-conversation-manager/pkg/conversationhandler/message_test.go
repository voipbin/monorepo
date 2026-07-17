package conversationhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	mmmessage "monorepo/bin-message-manager/models/message"
	mmtarget "monorepo/bin-message-manager/models/target"
)

func Test_MessageSend(t *testing.T) {

	tests := []struct {
		name string

		conversationID uuid.UUID
		text           string
		medias         []media.Media

		responseConversation *conversation.Conversation
		responseMessage      *message.Message
	}{
		{
			"line text type",

			uuid.FromStringOrNil("7b1034a8-e6ef-11ec-9e9d-c3f3e36741ac"),
			"hello, this is test message.",
			[]media.Media{},

			&conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7b1034a8-e6ef-11ec-9e9d-c3f3e36741ac"),
					CustomerID: uuid.FromStringOrNil("e54ded88-e6ef-11ec-83af-7fac5b21e9aa"),
				},
				Type:     conversation.TypeLine,
				DialogID: "18a7a0e8-e6f0-11ec-8cee-47dd7e7164e3",
			},

			&message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9d11dae8-e870-11ec-b319-fb0d0b15716f"),
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
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &conversationHandler{
				db:             mockDB,
				notifyHandler:  mockNotify,
				messageHandler: mockMessage,
				lineHandler:    mockLine,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
			mockMessage.EXPECT().Send(ctx, tt.responseConversation, tt.text, tt.medias).Return(tt.responseMessage, nil)
			res, err := h.MessageSend(ctx, tt.conversationID, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseMessage, res)
			}
		})
	}
}

func Test_executeActiveflow(t *testing.T) {

	tests := []struct {
		name string

		conversation *conversation.Conversation
		message      *message.Message
		flowID       uuid.UUID

		responseActiveflow *fmactiveflow.Activeflow

		// failureStage selects which downstream RPC fails ("" = happy path / no-flow)
		failureStage string

		expectError bool
	}{
		{
			name: "happy path: creates and executes activeflow",

			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				},
			},
			message: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				},
				ConversationID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			},
			flowID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),

			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
				},
			},

			expectError: false,
		},
		{
			name: "no flow configured: flowID == uuid.Nil returns nil without RPC calls",

			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
					CustomerID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
				},
			},
			message: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
					CustomerID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
				},
				ConversationID: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
			},
			flowID: uuid.Nil,

			expectError: false,
		},
		{
			name: "FlowV1ActiveflowCreate fails: error is propagated",

			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777"),
					CustomerID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
				},
			},
			message: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888"),
					CustomerID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
				},
				ConversationID: uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777"),
			},
			flowID: uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999"),

			failureStage: "create",
			expectError:  true,
		},
		{
			name: "FlowV1VariableSetVariable fails: error is propagated",

			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aaaaaaaa-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd"),
				},
			},
			message: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bbbbbbbb-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd"),
				},
				ConversationID: uuid.FromStringOrNil("aaaaaaaa-1111-1111-1111-111111111111"),
			},
			flowID: uuid.FromStringOrNil("cccccccc-3333-3333-3333-333333333333"),

			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dddddddd-4444-4444-4444-444444444444"),
				},
			},

			failureStage: "variable",
			expectError:  true,
		},
		{
			name: "FlowV1ActiveflowExecute fails: error is propagated",

			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("eeeeeeee-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("ffffffff-ffff-ffff-ffff-ffffffffffff"),
				},
			},
			message: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("eeeeeeee-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("ffffffff-ffff-ffff-ffff-ffffffffffff"),
				},
				ConversationID: uuid.FromStringOrNil("eeeeeeee-1111-1111-1111-111111111111"),
			},
			flowID: uuid.FromStringOrNil("eeeeeeee-3333-3333-3333-333333333333"),

			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("eeeeeeee-4444-4444-4444-444444444444"),
				},
			},

			failureStage: "execute",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &conversationHandler{
				db:             mockDB,
				notifyHandler:  mockNotify,
				reqHandler:     mockReq,
				messageHandler: mockMessage,
				lineHandler:    mockLine,
			}

			ctx := context.Background()

			// flowID == uuid.Nil: no expectations at all
			if tt.flowID != uuid.Nil {
				switch tt.failureStage {
				case "create":
					mockReq.EXPECT().FlowV1ActiveflowCreate(
						ctx,
						uuid.Nil,
						tt.message.CustomerID,
						tt.flowID,
						fmactiveflow.ReferenceTypeConversation,
						tt.message.ConversationID,
						uuid.Nil,
						nil,
						gomock.Any(),
						gomock.Any(),
					).Return(nil, fmt.Errorf("create failed"))

				case "variable":
					mockReq.EXPECT().FlowV1ActiveflowCreate(
						ctx,
						uuid.Nil,
						tt.message.CustomerID,
						tt.flowID,
						fmactiveflow.ReferenceTypeConversation,
						tt.message.ConversationID,
						uuid.Nil,
						nil,
						gomock.Any(),
						gomock.Any(),
					).Return(tt.responseActiveflow, nil)
					mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.responseActiveflow.ID, gomock.Any()).Return(fmt.Errorf("variable failed"))

				case "execute":
					mockReq.EXPECT().FlowV1ActiveflowCreate(
						ctx,
						uuid.Nil,
						tt.message.CustomerID,
						tt.flowID,
						fmactiveflow.ReferenceTypeConversation,
						tt.message.ConversationID,
						uuid.Nil,
						nil,
						gomock.Any(),
						gomock.Any(),
					).Return(tt.responseActiveflow, nil)
					mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.responseActiveflow.ID, gomock.Any()).Times(2).Return(nil)
					mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, tt.responseActiveflow.ID).Return(fmt.Errorf("execute failed"))

				default:
					// happy path
					mockReq.EXPECT().FlowV1ActiveflowCreate(
						ctx,
						uuid.Nil,
						tt.message.CustomerID,
						tt.flowID,
						fmactiveflow.ReferenceTypeConversation,
						tt.message.ConversationID,
						uuid.Nil,
						nil,
						gomock.Any(),
						gomock.Any(),
					).Return(tt.responseActiveflow, nil)
					mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.responseActiveflow.ID, gomock.Any()).Times(2).Return(nil)
					mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, tt.responseActiveflow.ID).Return(nil)
				}
			}

			err := h.executeActiveflow(ctx, tt.conversation, tt.message, tt.flowID)
			if tt.expectError {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
			} else {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}
		})
	}
}

func Test_MessageEventReceived(t *testing.T) {

	customerID := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	convID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	msgID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	convMsgID := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	agentOwnerID := uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777")

	selfAddr := commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: "+886987654321",
	}
	peerAddr := commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: "+886912345678",
	}

	tests := []struct {
		name string

		incoming *mmmessage.Message

		responseConversation *conversation.Conversation
		responseConvMessage  *message.Message
	}{
		{
			name: "assigned conversation (agent owner) -> agent mode no-op, no flow rpcs",

			incoming: &mmmessage.Message{
				Identity: commonidentity.Identity{
					ID:         msgID,
					CustomerID: customerID,
				},
				Source: &peerAddr,
				Targets: []mmtarget.Target{
					{Destination: selfAddr},
				},
				Direction: mmmessage.DirectionInbound,
				Text:      "hello, this is an assigned conversation message",
			},

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         convID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   agentOwnerID,
				},
				Type: conversation.TypeMessage,
				Self: selfAddr,
				Peer: peerAddr,
			},
			responseConvMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         convMsgID,
					CustomerID: customerID,
				},
				ConversationID: convID,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &conversationHandler{
				db:             mockDB,
				notifyHandler:  mockNotify,
				reqHandler:     mockReq,
				messageHandler: mockMessage,
				lineHandler:    mockLine,
			}

			ctx := context.Background()

			// GetOrCreateBySelfAndPeer returns the existing conversation directly
			// when ConversationGetBySelfAndPeer succeeds.
			mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, selfAddr, peerAddr).Return(tt.responseConversation, nil)

			mockMessage.EXPECT().Create(
				ctx,
				messagehandler.MessageCreateArgs{
					ID:             tt.incoming.ID,
					CustomerID:     tt.responseConversation.CustomerID,
					ConversationID: tt.responseConversation.ID,
					Direction:      message.DirectionIncoming,
					Status:         message.StatusDone,
					ReferenceType:  message.ReferenceTypeMessage,
					ReferenceID:    tt.incoming.ID,
					Text:           tt.incoming.Text,
					Medias:         []media.Media{},
					Source:         peerAddr,
					Destination:    selfAddr,
				},
			).Return(tt.responseConvMessage, nil)

			// Agent-owned conversations skip the flow path entirely. No flow RPCs are
			// expected on mockReq; gomock will fail if any unexpected call happens.

			if err := h.MessageEventReceived(ctx, tt.incoming); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_MessageEventReceived_CaseIDHint verifies that when the resolved
// Conversation carries Metadata.ContactCaseID (contact-case-management
// design §4.3/§4.4), MessageEventReceived passes it through as
// MessageCreateArgs.CaseID so it survives onto the
// conversation_message_created event. A Conversation with nil Metadata
// (the far more common case) must pass a nil CaseID -- not a zero UUID.
func Test_MessageEventReceived_CaseIDHint(t *testing.T) {
	customerID := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	convID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111112")
	msgID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222223")
	convMsgID := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333334")
	caseID := uuid.FromStringOrNil("f1b2c3d4-000e-000e-000e-000000000001")

	selfAddr := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110000"}
	peerAddr := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15552220000"}

	incoming := &mmmessage.Message{
		Identity: commonidentity.Identity{
			ID:         msgID,
			CustomerID: customerID,
		},
		Source: &peerAddr,
		Targets: []mmtarget.Target{
			{Destination: selfAddr},
		},
		Direction: mmmessage.DirectionInbound,
		Text:      "hello, this conversation has an open case linked",
	}

	responseConversation := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         convID,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeAgent,
			OwnerID:   uuid.FromStringOrNil("f1b2c3d4-000e-000e-000e-000000000002"),
		},
		Type:     conversation.TypeMessage,
		Self:     selfAddr,
		Peer:     peerAddr,
		Metadata: &conversation.Metadata{ContactCaseID: &caseID},
	}
	responseConvMessage := &message.Message{
		Identity: commonidentity.Identity{
			ID:         convMsgID,
			CustomerID: customerID,
		},
		ConversationID: convID,
		CaseID:         &caseID,
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)
	mockLine := linehandler.NewMockLineHandler(mc)
	h := &conversationHandler{
		db:             mockDB,
		notifyHandler:  mockNotify,
		reqHandler:     mockReq,
		messageHandler: mockMessage,
		lineHandler:    mockLine,
	}
	ctx := context.Background()

	mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, selfAddr, peerAddr).Return(responseConversation, nil)

	mockMessage.EXPECT().Create(
		ctx,
		messagehandler.MessageCreateArgs{
			ID:             incoming.ID,
			CustomerID:     responseConversation.CustomerID,
			ConversationID: responseConversation.ID,
			Direction:      message.DirectionIncoming,
			Status:         message.StatusDone,
			ReferenceType:  message.ReferenceTypeMessage,
			ReferenceID:    incoming.ID,
			Text:           incoming.Text,
			Medias:         []media.Media{},
			Source:         peerAddr,
			Destination:    selfAddr,
			CaseID:         &caseID,
		},
	).Return(responseConvMessage, nil)

	if err := h.MessageEventReceived(ctx, incoming); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
