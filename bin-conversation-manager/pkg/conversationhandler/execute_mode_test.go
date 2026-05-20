package conversationhandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	nmnumber "monorepo/bin-number-manager/models/number"
)

func Test_getExecuteMode(t *testing.T) {
	tests := []struct {
		name string
		cv   *conversation.Conversation
		want ExecuteMode
	}{
		{
			name: "unassigned conversation -> flow mode",
			cv: &conversation.Conversation{
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeNone,
					OwnerID:   uuid.Nil,
				},
			},
			want: ExecuteModeFlow,
		},
		{
			name: "agent owner with non-nil owner id -> agent mode",
			cv: &conversation.Conversation{
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				},
			},
			want: ExecuteModeAgent,
		},
		{
			name: "agent owner with nil owner id -> flow mode (defensive against malformed state)",
			cv: &conversation.Conversation{
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.Nil,
				},
			},
			want: ExecuteModeFlow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &conversationHandler{}
			got := h.getExecuteMode(tt.cv)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_runExecuteModeAgent_isNoop(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &conversationHandler{reqHandler: mockReq}

	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			CustomerID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		},
	}
	m := &message.Message{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		},
	}

	// mockReq.EXPECT() — no expectations: any RPC call will fail the test.

	err := h.runExecuteModeAgent(context.Background(), cv, m)
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
}

func Test_runExecuteModeFlowLine(t *testing.T) {
	accountID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	flowID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	convID := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	custID := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")
	msgID := uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")
	afID := uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666")

	tests := []struct {
		name    string
		cv      *conversation.Conversation
		m       *message.Message
		setup   func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler)
		wantErr bool
	}{
		{
			name: "valid line conversation with flow id -> executeActiveflow called",
			cv: &conversation.Conversation{
				Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:      conversation.TypeLine,
				AccountID: accountID,
			},
			m: &message.Message{
				Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
				ConversationID: convID,
			},
			setup: func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler) {
				mockAccount.EXPECT().Get(gomock.Any(), accountID).Return(&account.Account{
					Identity:      commonidentity.Identity{ID: accountID, CustomerID: custID},
					MessageFlowID: flowID,
				}, nil)
				mockReq.EXPECT().FlowV1ActiveflowCreate(
					gomock.Any(), uuid.Nil, custID, flowID,
					fmactiveflow.ReferenceTypeConversation, convID, uuid.Nil,
				).Return(&fmactiveflow.Activeflow{Identity: commonidentity.Identity{ID: afID}}, nil)
				mockReq.EXPECT().FlowV1VariableSetVariable(gomock.Any(), afID, gomock.Any()).Return(nil)
				mockReq.EXPECT().FlowV1ActiveflowExecute(gomock.Any(), afID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "line conversation with nil account id -> short-circuit, no fetch",
			cv: &conversation.Conversation{
				Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:      conversation.TypeLine,
				AccountID: uuid.Nil,
			},
			m: &message.Message{
				Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
				ConversationID: convID,
			},
			setup:   func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler) {},
			wantErr: false,
		},
		{
			name: "line conversation, account fetch fails -> error wrapped and returned",
			cv: &conversation.Conversation{
				Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:      conversation.TypeLine,
				AccountID: accountID,
			},
			m: &message.Message{
				Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
				ConversationID: convID,
			},
			setup: func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler) {
				mockAccount.EXPECT().Get(gomock.Any(), accountID).Return(nil, errors.New("db down"))
			},
			wantErr: true,
		},
		{
			name: "line conversation, account has no flow id -> no activeflow created, no error",
			cv: &conversation.Conversation{
				Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:      conversation.TypeLine,
				AccountID: accountID,
			},
			m: &message.Message{
				Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
				ConversationID: convID,
			},
			setup: func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler) {
				mockAccount.EXPECT().Get(gomock.Any(), accountID).Return(&account.Account{
					Identity:      commonidentity.Identity{ID: accountID, CustomerID: custID},
					MessageFlowID: uuid.Nil,
				}, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &conversationHandler{
				accountHandler: mockAccount,
				reqHandler:     mockReq,
			}
			tt.setup(mockAccount, mockReq)

			err := h.runExecuteModeFlowLine(context.Background(), tt.cv, tt.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("got err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func Test_runExecuteModeFlowMessage(t *testing.T) {
	numberID := uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777")
	flowID := uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888")
	convID := uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999")
	custID := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	msgID := uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	afID := uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc")
	target := "+15551234567"

	tests := []struct {
		name    string
		cv      *conversation.Conversation
		m       *message.Message
		setup   func(mockReq *requesthandler.MockRequestHandler)
		wantErr bool
	}{
		{
			name: "valid message conversation with flow id -> executeActiveflow called",
			cv: &conversation.Conversation{
				Identity: commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:     conversation.TypeMessage,
				Self:     commonaddress.Address{Target: target},
			},
			m: &message.Message{
				Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
				ConversationID: convID,
			},
			setup: func(mockReq *requesthandler.MockRequestHandler) {
				filters := map[nmnumber.Field]any{
					nmnumber.FieldNumber:  target,
					nmnumber.FieldDeleted: false,
				}
				mockReq.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), filters).Return([]nmnumber.Number{
					{
						Identity:      commonidentity.Identity{ID: numberID, CustomerID: custID},
						Number:        target,
						MessageFlowID: flowID,
					},
				}, nil)
				mockReq.EXPECT().FlowV1ActiveflowCreate(
					gomock.Any(), uuid.Nil, custID, flowID,
					fmactiveflow.ReferenceTypeConversation, convID, uuid.Nil,
				).Return(&fmactiveflow.Activeflow{Identity: commonidentity.Identity{ID: afID}}, nil)
				mockReq.EXPECT().FlowV1VariableSetVariable(gomock.Any(), afID, gomock.Any()).Return(nil)
				mockReq.EXPECT().FlowV1ActiveflowExecute(gomock.Any(), afID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "message conversation, number fetch fails -> error wrapped and returned",
			cv: &conversation.Conversation{
				Identity: commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:     conversation.TypeMessage,
				Self:     commonaddress.Address{Target: target},
			},
			m: &message.Message{
				Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
				ConversationID: convID,
			},
			setup: func(mockReq *requesthandler.MockRequestHandler) {
				filters := map[nmnumber.Field]any{
					nmnumber.FieldNumber:  target,
					nmnumber.FieldDeleted: false,
				}
				mockReq.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), filters).Return(nil, errors.New("number lookup failed"))
			},
			wantErr: true,
		},
		{
			name: "message conversation, number has no flow id -> no activeflow created, no error",
			cv: &conversation.Conversation{
				Identity: commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:     conversation.TypeMessage,
				Self:     commonaddress.Address{Target: target},
			},
			m: &message.Message{
				Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
				ConversationID: convID,
			},
			setup: func(mockReq *requesthandler.MockRequestHandler) {
				filters := map[nmnumber.Field]any{
					nmnumber.FieldNumber:  target,
					nmnumber.FieldDeleted: false,
				}
				mockReq.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), filters).Return([]nmnumber.Number{
					{
						Identity:      commonidentity.Identity{ID: numberID, CustomerID: custID},
						Number:        target,
						MessageFlowID: uuid.Nil,
					},
				}, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &conversationHandler{
				reqHandler: mockReq,
			}
			tt.setup(mockReq)

			err := h.runExecuteModeFlowMessage(context.Background(), tt.cv, tt.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("got err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// Test_runExecuteModeFlow_unsupportedTypeIsNoop verifies the default arm of the
// dispatcher: a conversation with an unsupported type returns nil without
// invoking any per-type runner. The TypeLine, TypeMessage, and TypeWhatsApp paths are covered
// by their respective Test_runExecuteModeFlow* functions.
func Test_runExecuteModeFlow_unsupportedTypeIsNoop(t *testing.T) {
	h := &conversationHandler{}
	cv := &conversation.Conversation{Type: "unknown"}
	m := &message.Message{}
	if err := h.runExecuteModeFlow(context.Background(), cv, m); err != nil {
		t.Errorf("expected nil for unsupported type, got: %v", err)
	}
}

func Test_runExecuteModeFlowWhatsApp(t *testing.T) {
	accountID := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	flowID := uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	convID := uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc")
	custID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd")
	msgID := uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	afID := uuid.FromStringOrNil("ffffffff-ffff-ffff-ffff-ffffffffffff")

	tests := []struct {
		name    string
		cv      *conversation.Conversation
		m       *message.Message
		setup   func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler)
		wantErr bool
	}{
		{
			name: "valid whatsapp conversation with flow id -> executeActiveflow called",
			cv: &conversation.Conversation{
				Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:      conversation.TypeWhatsApp,
				AccountID: accountID,
			},
			m: &message.Message{
				Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
				ConversationID: convID,
			},
			setup: func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler) {
				mockAccount.EXPECT().Get(gomock.Any(), accountID).Return(&account.Account{
					Identity:      commonidentity.Identity{ID: accountID, CustomerID: custID},
					MessageFlowID: flowID,
				}, nil)
				mockReq.EXPECT().FlowV1ActiveflowCreate(
					gomock.Any(), uuid.Nil, custID, flowID,
					fmactiveflow.ReferenceTypeConversation, convID, uuid.Nil,
				).Return(&fmactiveflow.Activeflow{Identity: commonidentity.Identity{ID: afID}}, nil)
				mockReq.EXPECT().FlowV1VariableSetVariable(gomock.Any(), afID, gomock.Any()).Return(nil)
				mockReq.EXPECT().FlowV1ActiveflowExecute(gomock.Any(), afID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "whatsapp conversation with nil account id -> short-circuit, no fetch",
			cv: &conversation.Conversation{
				Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:      conversation.TypeWhatsApp,
				AccountID: uuid.Nil,
			},
			m: &message.Message{
				Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
				ConversationID: convID,
			},
			setup:   func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler) {},
			wantErr: false,
		},
		{
			name: "whatsapp conversation, account fetch fails -> error returned",
			cv: &conversation.Conversation{
				Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:      conversation.TypeWhatsApp,
				AccountID: accountID,
			},
			m: &message.Message{
				Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
				ConversationID: convID,
			},
			setup: func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler) {
				mockAccount.EXPECT().Get(gomock.Any(), accountID).Return(nil, errors.New("db down"))
			},
			wantErr: true,
		},
		{
			name: "whatsapp conversation, account has no flow id -> no activeflow created, no error",
			cv: &conversation.Conversation{
				Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:      conversation.TypeWhatsApp,
				AccountID: accountID,
			},
			m: &message.Message{
				Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
				ConversationID: convID,
			},
			setup: func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler) {
				mockAccount.EXPECT().Get(gomock.Any(), accountID).Return(&account.Account{
					Identity:      commonidentity.Identity{ID: accountID, CustomerID: custID},
					MessageFlowID: uuid.Nil,
				}, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &conversationHandler{
				accountHandler: mockAccount,
				reqHandler:     mockReq,
			}
			tt.setup(mockAccount, mockReq)

			err := h.runExecuteModeFlowWhatsApp(context.Background(), tt.cv, tt.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("got err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
