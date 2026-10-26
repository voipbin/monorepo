package conversationhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	emmemail "monorepo/bin-email-manager/models/email"
	mmmessage "monorepo/bin-message-manager/models/message"
	mmtarget "monorepo/bin-message-manager/models/target"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
	"monorepo/bin-conversation-manager/pkg/whatsapphandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	nmnumber "monorepo/bin-number-manager/models/number"
	wcwidget "monorepo/bin-webchat-manager/models/widget"
)

// errDBDown is the injected database failure used to assert that the cause is propagated.
var errDBDown = errors.New("db down")

// Other injected failures, distinct from errDBDown so that a test can tell which one reached the caller.
var (
	errUpstreamDown = errors.New("upstream down")
	errCreateFailed = errors.New("create failed")
	errBadHook      = errors.New("bad hook payload")
	errFirst        = errors.New("first failure")
	errSecond       = errors.New("second failure")
)

// fullMocks bundles the mocks used by the error path tests of this file.
type fullMocks struct {
	util     *utilhandler.MockUtilHandler
	db       *dbhandler.MockDBHandler
	notify   *notifyhandler.MockNotifyHandler
	req      *requesthandler.MockRequestHandler
	account  *accounthandler.MockAccountHandler
	message  *messagehandler.MockMessageHandler
	line     *linehandler.MockLineHandler
	whatsapp *whatsapphandler.MockWhatsAppHandler
}

func newFullMocks(mc *gomock.Controller) (*conversationHandler, *fullMocks) {
	m := &fullMocks{
		util:     utilhandler.NewMockUtilHandler(mc),
		db:       dbhandler.NewMockDBHandler(mc),
		notify:   notifyhandler.NewMockNotifyHandler(mc),
		req:      requesthandler.NewMockRequestHandler(mc),
		account:  accounthandler.NewMockAccountHandler(mc),
		message:  messagehandler.NewMockMessageHandler(mc),
		line:     linehandler.NewMockLineHandler(mc),
		whatsapp: whatsapphandler.NewMockWhatsAppHandler(mc),
	}
	h := &conversationHandler{
		utilHandler:     m.util,
		db:              m.db,
		notifyHandler:   m.notify,
		reqHandler:      m.req,
		accountHandler:  m.account,
		messageHandler:  m.message,
		lineHandler:     m.line,
		whatsappHandler: m.whatsapp,
	}
	return h, m
}

// expectGetOrCreateFailure makes GetOrCreateBySelfAndPeer fail: the lookup misses and the create is rejected.
func expectGetOrCreateFailure(m *fullMocks, ctx context.Context) {
	m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("not found"))
	m.util.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"))
	m.db.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(errCreateFailed)
}

func Test_Hook_errors(t *testing.T) {
	accountID := uuid.FromStringOrNil("e8f5795a-e6eb-11ec-bb81-c3cec34bd99c")
	validURI := "https://hook.example.com/v1.0/conversation/accounts/" + accountID.String()

	tests := []struct {
		name string

		uri   string
		setup func(ctx context.Context, m *fullMocks)

		expectContains string
		expectIs       error
	}{
		{
			name:           "unparsable uri",
			uri:            "://bad",
			setup:          func(ctx context.Context, m *fullMocks) {},
			expectContains: "parse",
		},
		{
			name:           "path too short",
			uri:            "https://hook.example.com/v1.0",
			setup:          func(ctx context.Context, m *fullMocks) {},
			expectContains: "no customer info found",
		},
		{
			name: "account lookup failure",
			uri:  validURI,
			setup: func(ctx context.Context, m *fullMocks) {
				m.account.EXPECT().Get(ctx, accountID).Return(nil, errDBDown)
			},
			expectContains: "could not get account info",
			expectIs:       errDBDown,
		},
		{
			name: "line hook failure",
			uri:  validURI,
			setup: func(ctx context.Context, m *fullMocks) {
				ac := &account.Account{Identity: commonidentity.Identity{ID: accountID}, Type: account.TypeLine}
				m.account.EXPECT().Get(ctx, accountID).Return(ac, nil)
				m.line.EXPECT().Hook(ctx, ac, []byte("x")).Return(nil, errBadHook)
			},
			expectContains: "hook",
			expectIs:       errBadHook,
		},
		{
			name: "whatsapp hook failure",
			uri:  validURI,
			setup: func(ctx context.Context, m *fullMocks) {
				ac := &account.Account{Identity: commonidentity.Identity{ID: accountID}, Type: account.TypeWhatsApp}
				m.account.EXPECT().Get(ctx, accountID).Return(ac, nil)
				m.whatsapp.EXPECT().Hook(ctx, ac, []byte("x"), "sig").Return(nil, errBadHook)
			},
			expectContains: "hook",
			expectIs:       errBadHook,
		},
		{
			name: "unsupported account type",
			uri:  validURI,
			setup: func(ctx context.Context, m *fullMocks) {
				ac := &account.Account{Identity: commonidentity.Identity{ID: accountID}, Type: account.TypeSMS}
				m.account.EXPECT().Get(ctx, accountID).Return(ac, nil)
			},
			expectContains: "unsupported account type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newFullMocks(mc)
			ctx := context.Background()
			tt.setup(ctx, m)

			err := h.Hook(ctx, tt.uri, "POST", "sig", []byte("x"))
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("Wrong error. expect to contain: %q, got: %v", tt.expectContains, err)
			}
			if tt.expectIs != nil && !errors.Is(err, tt.expectIs) {
				t.Errorf("Wrong cause. expect: %v, got: %v", tt.expectIs, err)
			}
		})
	}
}

func Test_HookVerify_errors(t *testing.T) {
	accountID := uuid.FromStringOrNil("e8f5795a-e6eb-11ec-bb81-c3cec34bd99c")

	t.Run("unparsable uri", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, _ := newFullMocks(mc)

		if _, err := h.HookVerify(context.Background(), "://bad", "subscribe", "t", "c"); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("account lookup failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.account.EXPECT().Get(ctx, accountID).Return(nil, errDBDown)

		res, err := h.HookVerify(ctx, "https://hook.example.com/v1.0/conversation/accounts/"+accountID.String(), "subscribe", "t", "c")
		if !errors.Is(err, errDBDown) || !strings.Contains(err.Error(), "could not get account") {
			t.Errorf("Wrong match. expect: lookup error, got: %v", err)
		}
		if res != "" {
			t.Errorf("Wrong match. expect: empty challenge, got: %q", res)
		}
	})
}

// Test_hookResults_dispatch verifies how hooked results are dispatched by the owner of the conversation.
func Test_hookResults_dispatch(t *testing.T) {
	accountID := uuid.FromStringOrNil("e9eb2682-e6ed-11ec-a8f2-0b533280b1ae")
	agentID := uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777")
	convID := uuid.FromStringOrNil("f7f25d6c-e874-11ec-b140-3f088b887f43")
	custID := uuid.FromStringOrNil("a0a0a0a0-a0a0-a0a0-a0a0-a0a0a0a0a0a0")
	msgID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	ac := &account.Account{Identity: commonidentity.Identity{ID: accountID}}

	agentOwned := &conversation.Conversation{
		Identity: commonidentity.Identity{ID: convID, CustomerID: custID},
		Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: agentID},
		Type:     conversation.TypeWhatsApp,
		// AccountID is set so a wrongly routed flow dispatch would reach the account lookup
		// and fail the mock expectations.
		AccountID: accountID,
	}
	flowOwned := &conversation.Conversation{
		Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
		Type:      conversation.TypeWhatsApp,
		AccountID: accountID,
	}
	msg := &message.Message{Identity: commonidentity.Identity{ID: msgID, CustomerID: custID}, ConversationID: convID}

	t.Run("whatsapp agent owned conversation does not trigger a flow", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.whatsapp.EXPECT().Hook(ctx, ac, []byte("x"), "sig").Return([]*whatsapphandler.HookResult{{Conversation: agentOwned, Message: msg}}, nil)

		if err := h.hookWhatsApp(ctx, ac, []byte("x"), "sig"); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("whatsapp skips incomplete results", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.whatsapp.EXPECT().Hook(ctx, ac, []byte("x"), "sig").Return([]*whatsapphandler.HookResult{
			{Conversation: nil, Message: msg},
			{Conversation: flowOwned, Message: nil},
		}, nil)

		if err := h.hookWhatsApp(ctx, ac, []byte("x"), "sig"); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("whatsapp flow failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.whatsapp.EXPECT().Hook(ctx, ac, []byte("x"), "sig").Return([]*whatsapphandler.HookResult{{Conversation: flowOwned, Message: msg}}, nil)
		m.account.EXPECT().Get(ctx, accountID).Return(nil, errDBDown)

		err := h.hookWhatsApp(ctx, ac, []byte("x"), "sig")
		if !errors.Is(err, errDBDown) || !strings.Contains(err.Error(), "could not run flow mode") {
			t.Errorf("Wrong match. expect: flow mode error, got: %v", err)
		}
	})

	t.Run("line agent owned conversation does not trigger a flow", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		lineAgent := *agentOwned
		lineAgent.Type = conversation.TypeLine
		m.line.EXPECT().Hook(ctx, ac, []byte("x")).Return([]*linehandler.HookResult{{Conversation: &lineAgent, Message: msg}}, nil)

		if err := h.hookLine(ctx, ac, []byte("x")); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}

// Test_runExecuteModeFlow_activeflowFailure verifies that a failed activeflow creation is reported by every flow source.
func Test_runExecuteModeFlow_activeflowFailure(t *testing.T) {
	flowID := uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888")
	convID := uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999")
	custID := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	target := "+15550001111"
	waAccountID := uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc")

	m1 := &message.Message{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"), CustomerID: custID}, ConversationID: convID}

	tests := []struct {
		name string

		cv    *conversation.Conversation
		setup func(ctx context.Context, m *fullMocks)

		expectContains string
		expectIs       error
	}{
		{
			name: "message conversation",
			cv: &conversation.Conversation{
				Identity: commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:     conversation.TypeMessage,
				Self:     commonaddress.Address{Target: target},
			},
			setup: func(ctx context.Context, m *fullMocks) {
				m.req.EXPECT().NumberV1NumberList(ctx, "", uint64(1), gomock.Any()).Return([]nmnumber.Number{{MessageFlowID: flowID}}, nil)
			},
			expectContains: "could not execute activeflow. number_id",
			expectIs:       errUpstreamDown,
		},
		{
			name: "whatsapp conversation",
			cv: &conversation.Conversation{
				Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:      conversation.TypeWhatsApp,
				AccountID: waAccountID,
			},
			setup: func(ctx context.Context, m *fullMocks) {
				m.account.EXPECT().Get(ctx, waAccountID).Return(&account.Account{MessageFlowID: flowID}, nil)
			},
			expectContains: "could not execute activeflow. account_id",
			expectIs:       errUpstreamDown,
		},
		{
			name: "webchat conversation",
			cv: &conversation.Conversation{
				Identity: commonidentity.Identity{ID: convID, CustomerID: custID},
				Type:     conversation.TypeWebchat,
				Self:     commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()},
			},
			setup: func(ctx context.Context, m *fullMocks) {
				m.req.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(&wcwidget.Widget{MessageFlowID: flowID}, nil)
			},
			expectContains: "could not execute activeflow. widget_id",
			expectIs:       errUpstreamDown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, m := newFullMocks(mc)
			ctx := context.Background()

			tt.setup(ctx, m)
			m.req.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, custID, flowID, fmactiveflow.ReferenceTypeConversation, convID, uuid.Nil, nil, "", fmactiveflow.WebhookMethodNone).Return(nil, errUpstreamDown)

			err := h.runExecuteModeFlow(ctx, tt.cv, m1)
			if err == nil || !strings.Contains(err.Error(), tt.expectContains) || (tt.expectIs != nil && !errors.Is(err, tt.expectIs)) {
				t.Errorf("Wrong match. expect to contain: %q with cause: %v, got: %v", tt.expectContains, tt.expectIs, err)
			}
		})
	}
}

func Test_setVariables_errors(t *testing.T) {
	afID := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")
	cv := &conversation.Conversation{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999")}}
	msg := &message.Message{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")}, Text: "hi", Direction: message.DirectionIncoming}

	t.Run("conversation variables failure stops before message variables", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		// a single call is expected: the message variables must not be attempted.
		m.req.EXPECT().FlowV1VariableSetVariable(ctx, afID, gomock.Any()).Return(errUpstreamDown)

		err := h.setVariables(ctx, afID, cv, msg)
		if !errors.Is(err, errUpstreamDown) || !strings.Contains(err.Error(), "conversation variable") {
			t.Errorf("Wrong match. expect: conversation variable error, got: %v", err)
		}
	})

	t.Run("message variables failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		gomock.InOrder(
			m.req.EXPECT().FlowV1VariableSetVariable(ctx, afID, gomock.Any()).Return(nil),
			m.req.EXPECT().FlowV1VariableSetVariable(ctx, afID, gomock.Any()).Return(errUpstreamDown),
		)

		err := h.setVariables(ctx, afID, cv, msg)
		if !errors.Is(err, errUpstreamDown) || !strings.Contains(err.Error(), "message variable") {
			t.Errorf("Wrong match. expect: message variable error, got: %v", err)
		}
	})
}

func Test_dbBacked_errors(t *testing.T) {
	id := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")
	custID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	self := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550001111"}
	peer := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550002222"}

	t.Run("List returns the db failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.db.EXPECT().ConversationList(ctx, uint64(10), "token", gomock.Any()).Return(nil, errDBDown)

		res, err := h.List(ctx, "token", 10, nil)
		if !errors.Is(err, errDBDown) || res != nil {
			t.Errorf("Wrong match. expect: the db error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("Create fails when the created row cannot be read back", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.util.EXPECT().UUIDCreate().Return(id)
		m.db.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(nil)
		m.db.EXPECT().ConversationGet(ctx, id).Return(nil, errDBDown)

		// no webhook event may be published for a conversation that could not be read back.
		res, err := h.Create(ctx, custID, "n", "d", conversation.TypeMessage, "", self, peer)
		if !errors.Is(err, errDBDown) || res != nil {
			t.Errorf("Wrong match. expect: the db error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("Update fails when the db update fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		fields := map[conversation.Field]any{conversation.FieldName: "new name"}
		m.db.EXPECT().ConversationUpdate(ctx, id, fields).Return(errDBDown)

		res, err := h.Update(ctx, id, fields)
		if !errors.Is(err, errDBDown) || res != nil {
			t.Errorf("Wrong match. expect: the db error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("Update fails when the updated row cannot be read back", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		fields := map[conversation.Field]any{conversation.FieldName: "new name"}
		m.db.EXPECT().ConversationUpdate(ctx, id, fields).Return(nil)
		m.db.EXPECT().ConversationGet(ctx, id).Return(nil, errDBDown)

		res, err := h.Update(ctx, id, fields)
		if !errors.Is(err, errDBDown) || res != nil {
			t.Errorf("Wrong match. expect: the db error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("UpdateMetadata fails when the db update fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.db.EXPECT().ConversationUpdate(ctx, id, gomock.Any()).Return(errDBDown)

		res, err := h.UpdateMetadata(ctx, id, conversation.Metadata{})
		if !errors.Is(err, errDBDown) || res != nil {
			t.Errorf("Wrong match. expect: the db error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("UpdateMetadata fails when the updated row cannot be read back", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.db.EXPECT().ConversationUpdate(ctx, id, gomock.Any()).Return(nil)
		m.db.EXPECT().ConversationGet(ctx, id).Return(nil, errDBDown)

		res, err := h.UpdateMetadata(ctx, id, conversation.Metadata{})
		if !errors.Is(err, errDBDown) || res != nil {
			t.Errorf("Wrong match. expect: the db error and nil result, got: %v, %v", res, err)
		}
	})
}

func Test_CreateAndExecuteFlow_errors(t *testing.T) {
	id := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")
	afID := uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")
	custID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	flowID := uuid.FromStringOrNil("2b5bc824-2066-11f0-81b0-672de53dec30")
	self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: "aa847807-6cc4-4713-9dec-53a42840e74c"}
	peer := commonaddress.Address{Type: commonaddress.TypeWebSession, Target: "876defde-ad5e-11ed-a8c3-7bc19647b03f"}
	created := &conversation.Conversation{Identity: commonidentity.Identity{ID: id, CustomerID: custID}, Type: conversation.TypeWebchat, Self: self, Peer: peer}

	t.Run("create failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.util.EXPECT().UUIDCreate().Return(id)
		m.db.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(errDBDown)

		res, err := h.CreateAndExecuteFlow(ctx, custID, flowID, conversation.TypeWebchat, "", self, peer)
		if !errors.Is(err, errDBDown) || res != nil {
			t.Errorf("Wrong match. expect: the db error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("read back failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.util.EXPECT().UUIDCreate().Return(id)
		m.db.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(nil)
		m.db.EXPECT().ConversationGet(ctx, id).Return(nil, errDBDown)

		res, err := h.CreateAndExecuteFlow(ctx, custID, flowID, conversation.TypeWebchat, "", self, peer)
		if !errors.Is(err, errDBDown) || res != nil {
			t.Errorf("Wrong match. expect: the db error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("execute failure keeps the created conversation", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.util.EXPECT().UUIDCreate().Return(id)
		m.db.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(nil)
		m.db.EXPECT().ConversationGet(ctx, id).Return(created, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, custID, conversation.EventTypeConversationCreated, created)
		m.req.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, custID, flowID, fmactiveflow.ReferenceTypeConversation, id, uuid.Nil, nil, "", fmactiveflow.WebhookMethodNone).Return(&fmactiveflow.Activeflow{Identity: commonidentity.Identity{ID: afID}}, nil)
		m.req.EXPECT().FlowV1VariableSetVariable(ctx, afID, gomock.Any()).Return(nil)
		m.req.EXPECT().FlowV1ActiveflowExecute(ctx, afID).Return(errUpstreamDown)

		res, err := h.CreateAndExecuteFlow(ctx, custID, flowID, conversation.TypeWebchat, "", self, peer)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
		if res != created {
			t.Errorf("Wrong match. expect: the created conversation, got: %v", res)
		}
	})
}

func Test_Email_errors(t *testing.T) {
	emailID := uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")
	e := &emmemail.Email{
		Identity:     commonidentity.Identity{ID: emailID, CustomerID: uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")},
		Source:       &commonaddress.Address{Type: commonaddress.TypeEmail, Target: "sender@example.com"},
		Destinations: []commonaddress.Address{{Type: commonaddress.TypeEmail, Target: "a@example.com"}, {Type: commonaddress.TypeEmail, Target: "b@example.com"}},
		Status:       emmemail.StatusDelivered,
	}

	// the dedup key and the conversation addresses are derived from the normalized targets.
	normalized := func(a commonaddress.Address) commonaddress.Address {
		a.Target, _ = commonaddress.NormalizeTarget(a.Type, a.Target)
		return a
	}
	txIDOf := func(dst commonaddress.Address) string {
		return e.ID.String() + ":" + normalized(dst).Target
	}
	self := normalized(*e.Source)
	dstA, dstB := normalized(e.Destinations[0]), normalized(e.Destinations[1])
	txA, txB := txIDOf(e.Destinations[0]), txIDOf(e.Destinations[1])

	t.Run("sent: duplicate lookup failure does not stop the other destination and the first error is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		gomock.InOrder(
			m.db.EXPECT().MessageGetsByTransactionID(ctx, txA, "", uint64(1)).Return(nil, errFirst),
			// the second destination is still looked up after the first failure and fails differently.
			m.db.EXPECT().MessageGetsByTransactionID(ctx, txB, "", uint64(1)).Return(nil, errSecond),
		)

		err := h.EmailEventSent(ctx, e)
		if !errors.Is(err, errFirst) || errors.Is(err, errSecond) || !strings.Contains(err.Error(), "could not check existing message") {
			t.Errorf("Wrong match. expect: the first lookup error only, got: %v", err)
		}
	})

	t.Run("sent: conversation failure does not stop the other destination and the first error is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.db.EXPECT().MessageGetsByTransactionID(ctx, txA, "", uint64(1)).Return(nil, nil)
		m.db.EXPECT().MessageGetsByTransactionID(ctx, txB, "", uint64(1)).Return(nil, nil)
		m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, self, dstA).Return(nil, fmt.Errorf("not found"))
		m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, self, dstB).Return(nil, fmt.Errorf("not found"))
		m.util.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")).Times(2)
		gomock.InOrder(
			m.db.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(errFirst),
			m.db.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(errSecond),
		)

		err := h.EmailEventSent(ctx, e)
		if !errors.Is(err, errFirst) || errors.Is(err, errSecond) || !strings.Contains(err.Error(), "could not get conversation") {
			t.Errorf("Wrong match. expect: the first conversation error only, got: %v", err)
		}
	})

	t.Run("sent: message create failure on the first destination keeps the second one and returns the first error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		cvA := &conversation.Conversation{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001")}}
		cvB := &conversation.Conversation{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000002")}}

		m.db.EXPECT().MessageGetsByTransactionID(ctx, txA, "", uint64(1)).Return(nil, nil)
		m.db.EXPECT().MessageGetsByTransactionID(ctx, txB, "", uint64(1)).Return(nil, nil)
		m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, self, dstA).Return(cvA, nil)
		m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, self, dstB).Return(cvB, nil)
		// both creates fail; the second destination must still be attempted and the first error is returned.
		gomock.InOrder(
			m.message.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, args messagehandler.MessageCreateArgs) (*message.Message, error) {
				if args.ConversationID != cvA.ID {
					t.Errorf("Wrong match. expect first destination conversation: %s, got: %s", cvA.ID, args.ConversationID)
				}
				return nil, errFirst
			}),
			m.message.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, args messagehandler.MessageCreateArgs) (*message.Message, error) {
				if args.ConversationID != cvB.ID {
					t.Errorf("Wrong match. expect second destination conversation: %s, got: %s", cvB.ID, args.ConversationID)
				}
				return nil, errSecond
			}),
		)

		err := h.EmailEventSent(ctx, e)
		if !errors.Is(err, errFirst) || errors.Is(err, errSecond) {
			t.Errorf("Wrong match. expect: the first error only, got: %v", err)
		}
	})

	t.Run("updated: lookup failure on every destination returns the first error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.db.EXPECT().MessageGetsByTransactionID(ctx, txA, "", uint64(1)).Return(nil, errFirst)
		m.db.EXPECT().MessageGetsByTransactionID(ctx, txB, "", uint64(1)).Return(nil, errSecond)

		err := h.EmailEventUpdated(ctx, e)
		if !errors.Is(err, errFirst) || errors.Is(err, errSecond) || !strings.Contains(err.Error(), "could not find the message") {
			t.Errorf("Wrong match. expect: lookup error, got: %v", err)
		}
	})
}

func Test_MessageEvent_errors(t *testing.T) {
	custID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	msgID := uuid.FromStringOrNil("db596422-07f5-11f0-9afe-e7cd6b75aeac")
	source := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550001111"}
	mm := &mmmessage.Message{
		Identity: commonidentity.Identity{ID: msgID, CustomerID: custID},
		Source:   &source,
		Targets:  []mmtarget.Target{{Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550002222"}}},
	}

	t.Run("received: conversation failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		expectGetOrCreateFailure(m, ctx)

		err := h.MessageEventReceived(ctx, mm)
		if !errors.Is(err, errCreateFailed) || !strings.Contains(err.Error(), "Could not get conversation") {
			t.Errorf("Wrong match. expect: conversation error, got: %v", err)
		}
	})

	t.Run("sent: conversation failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		expectGetOrCreateFailure(m, ctx)

		err := h.MessageEventSent(ctx, mm)
		if !errors.Is(err, errCreateFailed) || !strings.Contains(err.Error(), "Could not get conversation") {
			t.Errorf("Wrong match. expect: conversation error, got: %v", err)
		}
	})

	t.Run("received: flow failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		cv := &conversation.Conversation{
			Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"), CustomerID: custID},
			Type:     conversation.TypeMessage,
			Self:     mm.Targets[0].Destination,
			Peer:     source,
		}
		m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, gomock.Any(), gomock.Any()).Return(cv, nil)
		m.message.EXPECT().Create(ctx, gomock.Any()).Return(&message.Message{Identity: commonidentity.Identity{ID: msgID}}, nil)
		m.req.EXPECT().NumberV1NumberList(ctx, "", uint64(1), gomock.Any()).Return(nil, errUpstreamDown)

		err := h.MessageEventReceived(ctx, mm)
		if !errors.Is(err, errUpstreamDown) || !strings.Contains(err.Error(), "could not run flow mode") {
			t.Errorf("Wrong match. expect: flow mode error, got: %v", err)
		}
	})

	t.Run("received: message create failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		cv := &conversation.Conversation{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"), CustomerID: custID}}
		m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, gomock.Any(), gomock.Any()).Return(cv, nil)
		m.message.EXPECT().Create(ctx, gomock.Any()).Return(nil, errDBDown)

		err := h.MessageEventReceived(ctx, mm)
		if !errors.Is(err, errDBDown) || !strings.Contains(err.Error(), "Could not create a message") {
			t.Errorf("Wrong match. expect: create error, got: %v", err)
		}
	})

	t.Run("sent: message create failure for an unknown message", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		cv := &conversation.Conversation{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"), CustomerID: custID}}
		m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, gomock.Any(), gomock.Any()).Return(cv, nil)
		m.message.EXPECT().Get(ctx, msgID).Return(nil, fmt.Errorf("not found"))
		m.message.EXPECT().Create(ctx, gomock.Any()).Return(nil, errDBDown)

		err := h.MessageEventSent(ctx, mm)
		if !errors.Is(err, errDBDown) || !strings.Contains(err.Error(), "Could not create a message") {
			t.Errorf("Wrong match. expect: create error, got: %v", err)
		}
	})

	t.Run("sent: update failure of an existing message", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		cv := &conversation.Conversation{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"), CustomerID: custID}}
		m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, gomock.Any(), gomock.Any()).Return(cv, nil)
		m.message.EXPECT().Get(ctx, msgID).Return(&message.Message{Identity: commonidentity.Identity{ID: msgID}}, nil)
		m.message.EXPECT().UpdateStatus(ctx, msgID, message.StatusDone).Return(nil, errDBDown)

		err := h.MessageEventSent(ctx, mm)
		if !errors.Is(err, errDBDown) || !strings.Contains(err.Error(), "Could not update the message") {
			t.Errorf("Wrong match. expect: update error, got: %v", err)
		}
	})
}

func Test_EventWebchat_errors(t *testing.T) {
	custID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	msgID := uuid.FromStringOrNil("db596422-07f5-11f0-9afe-e7cd6b75aeac")
	convID := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")

	payload := func(t *testing.T, direction string) []byte {
		data, err := json.Marshal(webchatMessage{ID: msgID, CustomerID: custID, WidgetID: widgetID, SessionID: sessionID, Direction: direction, Text: "hi"})
		if err != nil {
			t.Fatalf("could not marshal test fixture: %v", err)
		}
		return data
	}
	selfAddr := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()}
	peerAddr := commonaddress.Address{Type: commonaddress.TypeWebSession, Target: sessionID.String()}
	cvFlow := &conversation.Conversation{Identity: commonidentity.Identity{ID: convID, CustomerID: custID}, Type: conversation.TypeWebchat, Self: selfAddr, Peer: peerAddr}
	cvAgent := *cvFlow
	cvAgent.Owner = commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777")}

	tests := []struct {
		name string

		direction string
		setup     func(ctx context.Context, m *fullMocks)

		expectContains string
		expectIs       error
	}{
		{
			name:      "inbound: conversation failure",
			direction: webchatDirectionInbound,
			setup: func(ctx context.Context, m *fullMocks) {
				expectGetOrCreateFailure(m, ctx)
			},
			expectContains: "Could not get conversation",
			expectIs:       errCreateFailed,
		},
		{
			name:      "inbound: message create failure",
			direction: webchatDirectionInbound,
			setup: func(ctx context.Context, m *fullMocks) {
				m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, selfAddr, peerAddr).Return(cvFlow, nil)
				m.message.EXPECT().Create(ctx, gomock.Any()).Return(nil, errDBDown)
			},
			expectContains: "Could not create a message",
			expectIs:       errDBDown,
		},
		{
			name:      "inbound: flow failure",
			direction: webchatDirectionInbound,
			setup: func(ctx context.Context, m *fullMocks) {
				m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, selfAddr, peerAddr).Return(cvFlow, nil)
				m.message.EXPECT().Create(ctx, gomock.Any()).Return(&message.Message{Identity: commonidentity.Identity{ID: msgID}}, nil)
				m.req.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(nil, errUpstreamDown)
			},
			expectContains: "could not run flow mode",
			expectIs:       errUpstreamDown,
		},
		{
			name:      "outbound: conversation failure",
			direction: webchatDirectionOutbound,
			setup: func(ctx context.Context, m *fullMocks) {
				expectGetOrCreateFailure(m, ctx)
			},
			expectContains: "Could not get conversation",
			expectIs:       errCreateFailed,
		},
		{
			name:      "outbound: message create failure",
			direction: webchatDirectionOutbound,
			setup: func(ctx context.Context, m *fullMocks) {
				m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, selfAddr, peerAddr).Return(cvFlow, nil)
				m.message.EXPECT().Get(ctx, msgID).Return(nil, fmt.Errorf("not found"))
				m.message.EXPECT().Create(ctx, gomock.Any()).Return(nil, errDBDown)
			},
			expectContains: "Could not create a message",
			expectIs:       errDBDown,
		},
		{
			name:      "outbound: update failure of an existing message",
			direction: webchatDirectionOutbound,
			setup: func(ctx context.Context, m *fullMocks) {
				m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, selfAddr, peerAddr).Return(cvFlow, nil)
				m.message.EXPECT().Get(ctx, msgID).Return(&message.Message{Identity: commonidentity.Identity{ID: msgID}}, nil)
				m.message.EXPECT().UpdateStatus(ctx, msgID, message.StatusDone).Return(nil, errDBDown)
			},
			expectContains: "Could not update the message",
			expectIs:       errDBDown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, m := newFullMocks(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			err := h.eventWebchat(ctx, payload(t, tt.direction))
			if err == nil || !strings.Contains(err.Error(), tt.expectContains) || (tt.expectIs != nil && !errors.Is(err, tt.expectIs)) {
				t.Errorf("Wrong match. expect to contain: %q with cause: %v, got: %v", tt.expectContains, tt.expectIs, err)
			}
		})
	}

	t.Run("outbound: existing message is marked done", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, selfAddr, peerAddr).Return(cvFlow, nil)
		m.message.EXPECT().Get(ctx, msgID).Return(&message.Message{Identity: commonidentity.Identity{ID: msgID}}, nil)
		m.message.EXPECT().UpdateStatus(ctx, msgID, message.StatusDone).Return(&message.Message{Identity: commonidentity.Identity{ID: msgID}, Status: message.StatusDone}, nil)

		if err := h.eventWebchat(ctx, payload(t, webchatDirectionOutbound)); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("inbound: agent owned conversation does not trigger a flow", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, m := newFullMocks(mc)
		ctx := context.Background()

		m.db.EXPECT().ConversationGetBySelfAndPeer(ctx, selfAddr, peerAddr).Return(&cvAgent, nil)
		m.message.EXPECT().Create(ctx, gomock.Any()).Return(&message.Message{Identity: commonidentity.Identity{ID: msgID}}, nil)
		// no widget lookup and no activeflow call are expected.

		if err := h.eventWebchat(ctx, payload(t, webchatDirectionInbound)); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}

func Test_eventSMS_handlerFailures(t *testing.T) {
	custID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	source := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550001111"}
	mm := &mmmessage.Message{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("db596422-07f5-11f0-9afe-e7cd6b75aeac"), CustomerID: custID},
		Source:   &source,
		Targets:  []mmtarget.Target{{Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550002222"}}},
	}

	for _, direction := range []mmmessage.Direction{mmmessage.DirectionInbound, mmmessage.DirectionOutbound} {
		t.Run(string(direction), func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, m := newFullMocks(mc)
			ctx := context.Background()

			expectGetOrCreateFailure(m, ctx)

			tmp := *mm
			tmp.Direction = direction
			data, err := json.Marshal(&tmp)
			if err != nil {
				t.Fatalf("could not marshal test fixture: %v", err)
			}

			err = h.eventSMS(ctx, data)
			if !errors.Is(err, errCreateFailed) || !strings.Contains(err.Error(), "Could not handle the event correctly") {
				t.Errorf("Wrong match. expect: wrapped handler error, got: %v", err)
			}
		})
	}
}
