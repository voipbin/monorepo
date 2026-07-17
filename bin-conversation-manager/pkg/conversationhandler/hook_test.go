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
	"monorepo/bin-conversation-manager/pkg/whatsapphandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
)

func Test_Hook(t *testing.T) {

	tests := []struct {
		name string

		uri       string
		method    string
		signature string
		data      []byte

		expectAccountID uuid.UUID

		responseAccount  *account.Account
		responseUUIDs    []uuid.UUID
		responseMessages []*message.Message
	}{
		{
			name: "normal line",

			uri:       "https://hook.voipbin.net/v1.0/conversation/accounts/e8f5795a-e6eb-11ec-bb81-c3cec34bd99c",
			method:    "POST",
			signature: "",
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
			mockWhatsApp := whatsapphandler.NewMockWhatsAppHandler(mc)
			h := &conversationHandler{
				utilHandler:     mocKUtil,
				db:              mockDB,
				notifyHandler:   mockNotify,
				accountHandler:  mockAccount,
				messageHandler:  mockMessage,
				lineHandler:     mockLine,
				whatsappHandler: mockWhatsApp,
			}
			ctx := context.Background()

			mockAccount.EXPECT().Get(ctx, tt.expectAccountID).Return(tt.responseAccount, nil)
			mockLine.EXPECT().Hook(ctx, tt.responseAccount, tt.data).Return([]*linehandler.HookResult{}, nil)

			if err := h.Hook(ctx, tt.uri, tt.method, tt.signature, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_hookLine(t *testing.T) {

	accountID := uuid.FromStringOrNil("e9eb2682-e6ed-11ec-a8f2-0b533280b1ae")
	flowID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	convID1 := uuid.FromStringOrNil("f7f25d6c-e874-11ec-b140-3f088b887f43")
	convID2 := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	custID1 := uuid.FromStringOrNil("a0a0a0a0-a0a0-a0a0-a0a0-a0a0a0a0a0a0")
	custID2 := uuid.FromStringOrNil("b0b0b0b0-b0b0-b0b0-b0b0-b0b0b0b0b0b0")
	msgID1 := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	msgID2 := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	afID := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")
	agentOwnerID := uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777")

	tests := []struct {
		name string

		account *account.Account
		data    []byte

		responseHookResults []*linehandler.HookResult
		// responseAccount is what accountHandler.Get(ctx, cv.AccountID) returns for
		// flow-mode dispatch. May be nil for cases that never fetch.
		responseAccount    *account.Account
		responseActiveflow *fmactiveflow.Activeflow

		// expectActiveflowCalls: 0 = none expected; >0 = activeflow create succeeds
		// for each valid result; -1 = activeflow create returns error for first result.
		expectActiveflowCalls int
		expectHookError       error
		expectError           bool
	}{
		{
			name: "account has nil message flow id -> no activeflow created",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
				},
			},
			data: []byte(`{}`),

			responseHookResults: []*linehandler.HookResult{
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID:         convID1,
							CustomerID: custID1,
						},
						Type:      conversation.TypeLine,
						AccountID: accountID,
					},
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID:         msgID1,
							CustomerID: custID1,
						},
						ConversationID: convID1,
					},
				},
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
				},
				MessageFlowID: uuid.Nil,
			},

			expectActiveflowCalls: 0,
			expectError:           false,
		},
		{
			name: "non-nil message flow id triggers activeflow for each result",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
				},
				MessageFlowID: flowID,
			},
			data: []byte(`{}`),

			responseHookResults: []*linehandler.HookResult{
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID:         convID1,
							CustomerID: custID1,
						},
						Type:      conversation.TypeLine,
						AccountID: accountID,
					},
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID:         msgID1,
							CustomerID: custID1,
						},
						ConversationID: convID1,
					},
				},
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID:         convID2,
							CustomerID: custID2,
						},
						Type:      conversation.TypeLine,
						AccountID: accountID,
					},
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID:         msgID2,
							CustomerID: custID2,
						},
						ConversationID: convID2,
					},
				},
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
				},
				MessageFlowID: flowID,
			},
			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: afID,
				},
			},

			expectActiveflowCalls: 2,
			expectError:           false,
		},
		{
			name: "skips results with nil conversation or message",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
				},
				MessageFlowID: flowID,
			},
			data: []byte(`{}`),

			responseHookResults: []*linehandler.HookResult{
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID:         convID1,
							CustomerID: custID1,
						},
						Type:      conversation.TypeLine,
						AccountID: accountID,
					},
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID:         msgID1,
							CustomerID: custID1,
						},
						ConversationID: convID1,
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
						Type:      conversation.TypeLine,
						AccountID: accountID,
					},
					Message: nil,
				},
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
				},
				MessageFlowID: flowID,
			},
			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: afID,
				},
			},

			expectActiveflowCalls: 1,
			expectError:           false,
		},
		{
			name: "assigned conversation (agent owner) -> agent mode no-op, no flow rpcs",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
				},
				MessageFlowID: flowID,
			},
			data: []byte(`{}`),

			responseHookResults: []*linehandler.HookResult{
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID:         convID1,
							CustomerID: custID1,
						},
						Owner: commonidentity.Owner{
							OwnerType: commonidentity.OwnerTypeAgent,
							OwnerID:   agentOwnerID,
						},
						Type:      conversation.TypeLine,
						AccountID: accountID,
					},
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID:         msgID1,
							CustomerID: custID1,
						},
						ConversationID: convID1,
					},
				},
			},

			// No accountHandler.Get, no FlowV1ActiveflowCreate, no FlowV1VariableSetVariable,
			// no FlowV1ActiveflowExecute — agent-owned conversations skip the flow path entirely.
			expectActiveflowCalls: 0,
			expectError:           false,
		},
		{
			name: "returns error when executeActiveflow fails",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
				},
				MessageFlowID: flowID,
			},
			data: []byte(`{}`),

			responseHookResults: []*linehandler.HookResult{
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID:         convID1,
							CustomerID: custID1,
						},
						Type:      conversation.TypeLine,
						AccountID: accountID,
					},
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID:         msgID1,
							CustomerID: custID1,
						},
						ConversationID: convID1,
					},
				},
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
				},
				MessageFlowID: flowID,
			},

			expectActiveflowCalls: -1, // special: activeflow create returns error
			expectError:           true,
		},
		{
			name: "returns error when Hook fails",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
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
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			mockWhatsApp := whatsapphandler.NewMockWhatsAppHandler(mc)
			h := &conversationHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				notifyHandler:   mockNotify,
				reqHandler:      mockReq,
				accountHandler:  mockAccount,
				messageHandler:  mockMessage,
				lineHandler:     mockLine,
				whatsappHandler: mockWhatsApp,
			}
			ctx := context.Background()

			// mock lineHandler.Hook
			if tt.expectHookError != nil {
				mockLine.EXPECT().Hook(ctx, tt.account, tt.data).Return(nil, tt.expectHookError)
			} else {
				mockLine.EXPECT().Hook(ctx, tt.account, tt.data).Return(tt.responseHookResults, nil)
			}

			// Collect valid flow-mode results: non-nil conversation, non-nil message,
			// AND not owned by an agent (agent-owned conversations skip the flow path).
			var flowResults []*linehandler.HookResult
			for _, r := range tt.responseHookResults {
				if r.Conversation == nil || r.Message == nil {
					continue
				}
				if r.Conversation.OwnerType == commonidentity.OwnerTypeAgent && r.Conversation.OwnerID != uuid.Nil {
					continue
				}
				flowResults = append(flowResults, r)
			}

			// Each flow-mode result triggers an accountHandler.Get for the cv.AccountID lookup
			// performed inside runExecuteModeFlowLine.
			for range flowResults {
				mockAccount.EXPECT().Get(ctx, accountID).Return(tt.responseAccount, nil)
			}

			// mock activeflow calls based on expectActiveflowCalls
			if tt.expectActiveflowCalls == -1 {
				// activeflow create returns error for first valid flow-mode result
				r := flowResults[0]
				mockReq.EXPECT().FlowV1ActiveflowCreate(
					ctx,
					uuid.Nil,
					r.Message.CustomerID,
					tt.responseAccount.MessageFlowID,
					fmactiveflow.ReferenceTypeConversation,
					r.Message.ConversationID,
					uuid.Nil,
					nil,
					gomock.Any(),
					gomock.Any(),
				).Return(nil, fmt.Errorf("activeflow create failed"))
			} else if tt.expectActiveflowCalls > 0 {
				for _, r := range flowResults {
					activeflowID := tt.responseActiveflow.ID
					mockReq.EXPECT().FlowV1ActiveflowCreate(
						ctx,
						uuid.Nil,
						r.Message.CustomerID,
						tt.responseAccount.MessageFlowID,
						fmactiveflow.ReferenceTypeConversation,
						r.Message.ConversationID,
						uuid.Nil,
						nil,
						gomock.Any(),
						gomock.Any(),
					).Return(tt.responseActiveflow, nil)
					mockReq.EXPECT().FlowV1VariableSetVariable(ctx, activeflowID, gomock.Any()).Times(2).Return(nil)
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

func Test_hookWhatsApp(t *testing.T) {

	accountID := uuid.FromStringOrNil("aaaaaaaa-1111-1111-1111-111111111111")
	convID := uuid.FromStringOrNil("bbbbbbbb-2222-2222-2222-222222222222")
	custID := uuid.FromStringOrNil("cccccccc-3333-3333-3333-333333333333")
	msgID := uuid.FromStringOrNil("dddddddd-4444-4444-4444-444444444444")

	tests := []struct {
		name string

		account   *account.Account
		data      []byte
		signature string

		responseHookResults []*whatsapphandler.HookResult

		setup           func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler)
		expectHookError error
		expectError     bool
	}{
		{
			name: "normal - whatsapp conversation type triggers flow dispatch via runExecuteModeFlowWhatsApp",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
				},
				Type: account.TypeWhatsApp,
			},
			data:      []byte(`{}`),
			signature: "sha256=abc",

			responseHookResults: []*whatsapphandler.HookResult{
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID:         convID,
							CustomerID: custID,
						},
						Type:      conversation.TypeWhatsApp,
						AccountID: accountID,
					},
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID:         msgID,
							CustomerID: custID,
						},
						ConversationID: convID,
					},
				},
			},

			// account has no MessageFlowID, so no activeflow RPC is triggered.
			setup: func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler) {
				mockAccount.EXPECT().Get(gomock.Any(), accountID).Return(&account.Account{
					Identity:      commonidentity.Identity{ID: accountID, CustomerID: custID},
					MessageFlowID: uuid.Nil,
				}, nil)
			},
			expectError: false,
		},
		{
			name: "normal - skips nil conversation or message",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
				},
				Type: account.TypeWhatsApp,
			},
			data:      []byte(`{}`),
			signature: "sha256=abc",

			responseHookResults: []*whatsapphandler.HookResult{
				{
					Conversation: nil,
					Message: &message.Message{
						Identity: commonidentity.Identity{
							ID: msgID,
						},
					},
				},
				{
					Conversation: &conversation.Conversation{
						Identity: commonidentity.Identity{
							ID: convID,
						},
					},
					Message: nil,
				},
			},

			expectError: false,
		},
		{
			name:            "hook returns error",
			account:         &account.Account{Identity: commonidentity.Identity{ID: accountID}, Type: account.TypeWhatsApp},
			data:            []byte(`{}`),
			signature:       "sha256=abc",
			expectHookError: fmt.Errorf("whatsapp hook failed"),
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
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			mockWhatsApp := whatsapphandler.NewMockWhatsAppHandler(mc)
			h := &conversationHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				notifyHandler:   mockNotify,
				reqHandler:      mockReq,
				accountHandler:  mockAccount,
				messageHandler:  mockMessage,
				lineHandler:     mockLine,
				whatsappHandler: mockWhatsApp,
			}
			ctx := context.Background()

			if tt.setup != nil {
				tt.setup(mockAccount, mockReq)
			}

			if tt.expectHookError != nil {
				mockWhatsApp.EXPECT().Hook(ctx, tt.account, tt.data, tt.signature).Return(nil, tt.expectHookError)
			} else {
				mockWhatsApp.EXPECT().Hook(ctx, tt.account, tt.data, tt.signature).Return(tt.responseHookResults, nil)
			}

			// Collect valid flow-mode results
			err := h.hookWhatsApp(ctx, tt.account, tt.data, tt.signature)
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

func Test_HookVerify(t *testing.T) {

	accountID := uuid.FromStringOrNil("e8f5795a-e6eb-11ec-bb81-c3cec34bd99c")

	tests := []struct {
		name string

		uri         string
		mode        string
		verifyToken string
		challenge   string

		expectAccountID uuid.UUID

		responseAccount         *account.Account
		responseVerifyWebhook   string
		responseVerifyWebhookOK bool

		expectResult string
		expectError  bool
	}{
		{
			name: "normal whatsapp account",

			uri:         "https://hook.voipbin.net/v1.0/conversation/accounts/e8f5795a-e6eb-11ec-bb81-c3cec34bd99c",
			mode:        "subscribe",
			verifyToken: "mytoken",
			challenge:   "challenge123",

			expectAccountID: accountID,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
				},
				Type: account.TypeWhatsApp,
			},
			responseVerifyWebhook:   "challenge123",
			responseVerifyWebhookOK: true,

			expectResult: "challenge123",
			expectError:  false,
		},
		{
			name: "non-whatsapp account returns error",

			uri:         "https://hook.voipbin.net/v1.0/conversation/accounts/e8f5795a-e6eb-11ec-bb81-c3cec34bd99c",
			mode:        "subscribe",
			verifyToken: "mytoken",
			challenge:   "challenge123",

			expectAccountID: accountID,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: accountID,
				},
				Type: account.TypeLine,
			},

			expectResult: "",
			expectError:  true,
		},
		{
			name: "short uri path returns error",

			uri:         "https://hook.voipbin.net/v1.0",
			mode:        "subscribe",
			verifyToken: "mytoken",
			challenge:   "challenge123",

			expectResult: "",
			expectError:  true,
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
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			mockWhatsApp := whatsapphandler.NewMockWhatsAppHandler(mc)
			h := &conversationHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				notifyHandler:   mockNotify,
				accountHandler:  mockAccount,
				messageHandler:  mockMessage,
				lineHandler:     mockLine,
				whatsappHandler: mockWhatsApp,
			}
			ctx := context.Background()

			if tt.responseAccount != nil {
				mockAccount.EXPECT().Get(ctx, tt.expectAccountID).Return(tt.responseAccount, nil)

				if tt.responseAccount.Type == account.TypeWhatsApp {
					mockWhatsApp.EXPECT().VerifyWebhook(ctx, tt.responseAccount, tt.mode, tt.verifyToken, tt.challenge).Return(tt.responseVerifyWebhook, nil)
				}
			}

			result, err := h.HookVerify(ctx, tt.uri, tt.mode, tt.verifyToken, tt.challenge)
			if tt.expectError {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
			} else {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				if result != tt.expectResult {
					t.Errorf("Wrong match. expect: %s, got: %s", tt.expectResult, result)
				}
			}
		})
	}
}
