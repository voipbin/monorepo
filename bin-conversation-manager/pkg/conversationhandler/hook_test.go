package conversationhandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
)

func Test_Hook(t *testing.T) {

	tests := []struct {
		name string

		uri  string
		data []byte

		expectAccountID uuid.UUID

		responseAccount  *account.Account
		responseUUIDs    []uuid.UUID
		responseMessages []*message.Message
	}{
		{
			name: "normal",

			uri: "https://hook.voipbin.net/v1.0/conversation/accounts/e8f5795a-e6eb-11ec-bb81-c3cec34bd99c",
			data: []byte(`{
				"destination": "U11298214116e3afbad432b5794a6d3a0",
				"events": [
					{
						"type": "message",
						"message": {
							"type": "text",
							"id": "16173792131295",
							"text": "Hello"
						},
						"webhookEventId": "01G49KHTWA1D2WF05D0VHEMGZE",
						"deliveryContext": {
							"isRedelivery": false
						},
						"timestamp": 1653884906096,
						"source": {
							"type": "user",
							"userId": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
						},
						"replyToken": "4bdd674a22cc479b8e9e429465396b76",
						"mode": "active"
					}
				]
			}`),

			expectAccountID: uuid.FromStringOrNil("e8f5795a-e6eb-11ec-bb81-c3cec34bd99c"),

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e8f5795a-e6eb-11ec-bb81-c3cec34bd99c"),
				},
				Type: account.TypeLine,
			},
			responseUUIDs: []uuid.UUID{
				uuid.FromStringOrNil("cb285f42-0075-11ee-ad73-0fae8c027ffc"),
			},
			responseMessages: []*message.Message{
				{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mocKUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &conversationHandler{
				utilHandler:    mocKUtil,
				db:             mockDB,
				notifyHandler:  mockNotify,
				accountHandler: mockAccount,
				messageHandler: mockMessage,
				lineHandler:    mockLine,
			}
			ctx := context.Background()

			mockAccount.EXPECT().Get(ctx, tt.expectAccountID).Return(tt.responseAccount, nil)
			mockLine.EXPECT().Hook(ctx, tt.responseAccount, tt.data).Return([]*linehandler.HookResult{}, nil)

			if err := h.Hook(ctx, tt.uri, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_hookLine(t *testing.T) {

	tests := []struct {
		name string

		account *account.Account
		data    []byte

		responseHookResults []*linehandler.HookResult
		responseActiveflow  *fmactiveflow.Activeflow

		expectActiveflowCalls int
		expectHookError       error
		expectError           bool
	}{
		{
			name: "nil message flow id returns early without calling activeflow",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e9eb2682-e6ed-11ec-a8f2-0b533280b1ae"),
				},
			},
			data: []byte(`{}`),

			responseHookResults: []*linehandler.HookResult{
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("f7f25d6c-e874-11ec-b140-3f088b887f43"),
							CustomerID: uuid.FromStringOrNil("a0a0a0a0-a0a0-a0a0-a0a0-a0a0a0a0a0a0"),
						},
					},
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
							CustomerID: uuid.FromStringOrNil("a0a0a0a0-a0a0-a0a0-a0a0-a0a0a0a0a0a0"),
						},
						ConversationID: uuid.FromStringOrNil("f7f25d6c-e874-11ec-b140-3f088b887f43"),
					},
				},
			},

			expectActiveflowCalls: 0,
			expectError:           false,
		},
		{
			name: "non-nil message flow id triggers activeflow for each result",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e9eb2682-e6ed-11ec-a8f2-0b533280b1ae"),
				},
				MessageFlowID: uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
			},
			data: []byte(`{}`),

			responseHookResults: []*linehandler.HookResult{
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("f7f25d6c-e874-11ec-b140-3f088b887f43"),
							CustomerID: uuid.FromStringOrNil("a0a0a0a0-a0a0-a0a0-a0a0-a0a0a0a0a0a0"),
						},
					},
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
							CustomerID: uuid.FromStringOrNil("a0a0a0a0-a0a0-a0a0-a0a0-a0a0a0a0a0a0"),
						},
						ConversationID: uuid.FromStringOrNil("f7f25d6c-e874-11ec-b140-3f088b887f43"),
					},
				},
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
							CustomerID: uuid.FromStringOrNil("b0b0b0b0-b0b0-b0b0-b0b0-b0b0b0b0b0b0"),
						},
					},
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
							CustomerID: uuid.FromStringOrNil("b0b0b0b0-b0b0-b0b0-b0b0-b0b0b0b0b0b0"),
						},
						ConversationID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
					},
				},
			},
			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
				},
			},

			expectActiveflowCalls: 2,
			expectError:           false,
		},
		{
			name: "skips results with nil conversation or message",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e9eb2682-e6ed-11ec-a8f2-0b533280b1ae"),
				},
				MessageFlowID: uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
			},
			data: []byte(`{}`),

			responseHookResults: []*linehandler.HookResult{
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("f7f25d6c-e874-11ec-b140-3f088b887f43"),
							CustomerID: uuid.FromStringOrNil("a0a0a0a0-a0a0-a0a0-a0a0-a0a0a0a0a0a0"),
						},
					},
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
							CustomerID: uuid.FromStringOrNil("a0a0a0a0-a0a0-a0a0-a0a0-a0a0a0a0a0a0"),
						},
						ConversationID: uuid.FromStringOrNil("f7f25d6c-e874-11ec-b140-3f088b887f43"),
					},
				},
				{
					Conversation: nil,
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
						},
					},
				},
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
						},
					},
					Message: nil,
				},
			},
			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
				},
			},

			expectActiveflowCalls: 1,
			expectError:           false,
		},
		{
			name: "returns error when executeActiveflow fails",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e9eb2682-e6ed-11ec-a8f2-0b533280b1ae"),
				},
				MessageFlowID: uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
			},
			data: []byte(`{}`),

			responseHookResults: []*linehandler.HookResult{
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("f7f25d6c-e874-11ec-b140-3f088b887f43"),
							CustomerID: uuid.FromStringOrNil("a0a0a0a0-a0a0-a0a0-a0a0-a0a0a0a0a0a0"),
						},
					},
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
							CustomerID: uuid.FromStringOrNil("a0a0a0a0-a0a0-a0a0-a0a0-a0a0a0a0a0a0"),
						},
						ConversationID: uuid.FromStringOrNil("f7f25d6c-e874-11ec-b140-3f088b887f43"),
					},
				},
			},

			expectActiveflowCalls: -1, // special: activeflow create returns error
			expectError:           true,
		},
		{
			name: "returns error when Hook fails",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e9eb2682-e6ed-11ec-a8f2-0b533280b1ae"),
				},
			},
			data: []byte(`{}`),

			expectHookError: fmt.Errorf("hook failed"),
			expectError:     true,
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
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &conversationHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				notifyHandler:  mockNotify,
				reqHandler:     mockReq,
				messageHandler: mockMessage,
				lineHandler:    mockLine,
			}
			ctx := context.Background()

			// mock lineHandler.Hook
			if tt.expectHookError != nil {
				mockLine.EXPECT().Hook(ctx, tt.account, tt.data).Return(nil, tt.expectHookError)
			} else {
				mockLine.EXPECT().Hook(ctx, tt.account, tt.data).Return(tt.responseHookResults, nil)
			}

			// mock activeflow calls based on expectActiveflowCalls
			if tt.expectActiveflowCalls == -1 {
				// activeflow create returns error
				r := tt.responseHookResults[0]
				mockReq.EXPECT().FlowV1ActiveflowCreate(
					ctx,
					uuid.Nil,
					r.Message.CustomerID,
					tt.account.MessageFlowID,
					fmactiveflow.ReferenceTypeConversation,
					r.Message.ConversationID,
					uuid.Nil,
				).Return(nil, fmt.Errorf("activeflow create failed"))
			} else if tt.expectActiveflowCalls > 0 {
				// collect valid results (non-nil conversation and message)
				var validResults []*linehandler.HookResult
				for _, r := range tt.responseHookResults {
					if r.Conversation != nil && r.Message != nil {
						validResults = append(validResults, r)
					}
				}

				for _, r := range validResults {
					activeflowID := tt.responseActiveflow.ID
					mockReq.EXPECT().FlowV1ActiveflowCreate(
						ctx,
						uuid.Nil,
						r.Message.CustomerID,
						tt.account.MessageFlowID,
						fmactiveflow.ReferenceTypeConversation,
						r.Message.ConversationID,
						uuid.Nil,
					).Return(tt.responseActiveflow, nil)
					mockReq.EXPECT().FlowV1VariableSetVariable(ctx, activeflowID, gomock.Any()).Return(nil)
					mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, activeflowID).Return(nil)
				}
			}

			err := h.hookLine(ctx, tt.account, tt.data)
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
