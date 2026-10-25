package messagehandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
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
	"monorepo/bin-conversation-manager/pkg/smshandler"
	"monorepo/bin-conversation-manager/pkg/whatsapphandler"

	wcmessage "monorepo/bin-webchat-manager/models/message"
)

type sendMocks struct {
	util     *utilhandler.MockUtilHandler
	db       *dbhandler.MockDBHandler
	notify   *notifyhandler.MockNotifyHandler
	req      *requesthandler.MockRequestHandler
	account  *accounthandler.MockAccountHandler
	sms      *smshandler.MockSMSHandler
	whatsapp *whatsapphandler.MockWhatsAppHandler
}

func newSendErrorHandler(mc *gomock.Controller) (*messageHandler, *sendMocks) {
	m := &sendMocks{
		util:     utilhandler.NewMockUtilHandler(mc),
		db:       dbhandler.NewMockDBHandler(mc),
		notify:   notifyhandler.NewMockNotifyHandler(mc),
		req:      requesthandler.NewMockRequestHandler(mc),
		account:  accounthandler.NewMockAccountHandler(mc),
		sms:      smshandler.NewMockSMSHandler(mc),
		whatsapp: whatsapphandler.NewMockWhatsAppHandler(mc),
	}
	h := &messageHandler{
		utilHandler:     m.util,
		db:              m.db,
		notifyHandler:   m.notify,
		reqHandler:      m.req,
		accountHandler:  m.account,
		smsHandler:      m.sms,
		whatsappHandler: m.whatsapp,
	}
	return h, m
}

var (
	testSendCustomerID = uuid.FromStringOrNil("c2b0c3d1-1234-11f0-bbbb-bbbbbbbbbbbb")
	testSendConvID     = uuid.FromStringOrNil("c1a9c2d0-1234-11f0-aaaa-aaaaaaaaaaaa")
	testSendAccountID  = uuid.FromStringOrNil("c3c1d4e2-1234-11f0-cccc-cccccccccccc")
	testSendMessageID  = uuid.FromStringOrNil("c4d2e5f3-1234-11f0-dddd-dddddddddddd")
)

func newSendTestConversation(t conversation.Type) *conversation.Conversation {
	return &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         testSendConvID,
			CustomerID: testSendCustomerID,
		},
		AccountID: testSendAccountID,
		Type:      t,
		Self:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"},
		Peer:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100002"},
	}
}

func newSendTestMessage() *message.Message {
	return &message.Message{
		Identity: commonidentity.Identity{
			ID:         testSendMessageID,
			CustomerID: testSendCustomerID,
		},
		ConversationID: testSendConvID,
	}
}

// expectCreateOK sets up a successful h.Create sequence for the given message id.
func expectCreateOK(m *sendMocks, id uuid.UUID) {
	res := newSendTestMessage()
	res.ID = id
	m.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(nil)
	m.db.EXPECT().MessageGet(gomock.Any(), id).Return(res, nil)
	m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), testSendCustomerID, message.EventTypeMessageCreated, res)
}

func failedStatus() map[message.Field]any {
	return map[message.Field]any{message.FieldStatus: message.StatusFailed}
}

func Test_sendSMS_errors(t *testing.T) {
	sendErr := fmt.Errorf("provider rejected")

	t.Run("create failure does not hand the message to the provider", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		m.util.EXPECT().UUIDCreate().Return(testSendMessageID)
		m.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(fmt.Errorf("insert failed"))

		res, err := h.Send(context.Background(), newSendTestConversation(conversation.TypeMessage), "hello", nil)
		if err == nil || res != nil {
			t.Errorf("expected error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("provider failure marks the message failed and returns the provider error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		cv := newSendTestConversation(conversation.TypeMessage)

		m.util.EXPECT().UUIDCreate().Return(testSendMessageID)
		expectCreateOK(m, testSendMessageID)
		m.sms.EXPECT().Send(gomock.Any(), cv, testSendMessageID, "hello").Return(sendErr)

		failed := newSendTestMessage()
		m.db.EXPECT().MessageUpdate(gomock.Any(), testSendMessageID, failedStatus()).Return(nil)
		m.db.EXPECT().MessageGet(gomock.Any(), testSendMessageID).Return(failed, nil)
		m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), testSendCustomerID, message.EventTypeMessageUpdated, failed)

		res, err := h.Send(context.Background(), cv, "hello", nil)
		if !stderrors.Is(err, sendErr) || res != nil {
			t.Errorf("expected the provider error and nil result, got: %v, %v", res, err)
		}
	})
}

func Test_sendWhatsApp_errors(t *testing.T) {
	sendErr := fmt.Errorf("graph api rejected")
	cv := newSendTestConversation(conversation.TypeWhatsApp)
	ac := &account.Account{Identity: commonidentity.Identity{ID: testSendAccountID}}

	t.Run("account lookup failure stops before creating a message", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		m.account.EXPECT().Get(gomock.Any(), testSendAccountID).Return(nil, fmt.Errorf("lookup failed"))

		res, err := h.Send(context.Background(), cv, "hello", nil)
		if err == nil || res != nil {
			t.Errorf("expected error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("create failure does not call the provider", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		m.account.EXPECT().Get(gomock.Any(), testSendAccountID).Return(ac, nil)
		m.util.EXPECT().UUIDCreate().Return(testSendMessageID)
		m.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(fmt.Errorf("insert failed"))

		res, err := h.Send(context.Background(), cv, "hello", nil)
		if err == nil || res != nil {
			t.Errorf("expected error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("provider failure marks the message failed and returns the provider error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		m.account.EXPECT().Get(gomock.Any(), testSendAccountID).Return(ac, nil)
		m.util.EXPECT().UUIDCreate().Return(testSendMessageID)
		expectCreateOK(m, testSendMessageID)
		m.whatsapp.EXPECT().Send(gomock.Any(), cv, ac, "hello").Return("", sendErr)

		failed := newSendTestMessage()
		m.db.EXPECT().MessageUpdate(gomock.Any(), testSendMessageID, failedStatus()).Return(nil)
		m.db.EXPECT().MessageGet(gomock.Any(), testSendMessageID).Return(failed, nil)
		m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), testSendCustomerID, message.EventTypeMessageUpdated, failed)

		res, err := h.Send(context.Background(), cv, "hello", nil)
		if !stderrors.Is(err, sendErr) || res != nil {
			t.Errorf("expected the provider error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("done status update failure is returned and the transaction id is not stored", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		m.account.EXPECT().Get(gomock.Any(), testSendAccountID).Return(ac, nil)
		m.util.EXPECT().UUIDCreate().Return(testSendMessageID)
		expectCreateOK(m, testSendMessageID)
		m.whatsapp.EXPECT().Send(gomock.Any(), cv, ac, "hello").Return("wamid.test", nil)

		done := map[message.Field]any{message.FieldStatus: message.StatusDone}
		m.db.EXPECT().MessageUpdate(gomock.Any(), testSendMessageID, done).Return(fmt.Errorf("update failed"))

		res, err := h.Send(context.Background(), cv, "hello", nil)
		if err == nil || res != nil {
			t.Errorf("expected error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("failing to store the wamid does not fail the send", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		m.account.EXPECT().Get(gomock.Any(), testSendAccountID).Return(ac, nil)
		m.util.EXPECT().UUIDCreate().Return(testSendMessageID)
		expectCreateOK(m, testSendMessageID)
		m.whatsapp.EXPECT().Send(gomock.Any(), cv, ac, "hello").Return("wamid.test", nil)

		done := newSendTestMessage()
		m.db.EXPECT().MessageUpdate(gomock.Any(), testSendMessageID, map[message.Field]any{message.FieldStatus: message.StatusDone}).Return(nil)
		m.db.EXPECT().MessageGet(gomock.Any(), testSendMessageID).Return(done, nil)
		m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), testSendCustomerID, message.EventTypeMessageUpdated, done)
		m.db.EXPECT().MessageUpdate(gomock.Any(), testSendMessageID, map[message.Field]any{message.FieldTransactionID: "wamid.test"}).Return(fmt.Errorf("persist failed"))

		res, err := h.Send(context.Background(), cv, "hello", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != done {
			t.Errorf("expected the status-updated message, got: %v", res)
		}
	})
}

func Test_sendWebchat_errors(t *testing.T) {
	sessionID := uuid.FromStringOrNil("c5e3f6a4-1234-11f0-eeee-eeeeeeeeeeee")

	newCV := func() *conversation.Conversation {
		cv := newSendTestConversation(conversation.TypeWebchat)
		cv.Peer = commonaddress.Address{Type: commonaddress.TypeWebchat, Target: sessionID.String()}
		return cv
	}
	wm := &wcmessage.Message{Identity: commonidentity.Identity{ID: testSendMessageID}}

	t.Run("webchat service failure is returned before anything is stored", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		rpcErr := fmt.Errorf("rpc timeout")
		m.req.EXPECT().WebchatV1MessageCreate(gomock.Any(), testSendCustomerID, sessionID, wcmessage.DirectionOutbound, uuid.Nil, "hello").Return(nil, rpcErr)

		res, err := h.Send(context.Background(), newCV(), "hello", nil)
		if !stderrors.Is(err, rpcErr) || res != nil {
			t.Errorf("expected the rpc error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("unrelated create failure is returned without a second lookup", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		m.req.EXPECT().WebchatV1MessageCreate(gomock.Any(), testSendCustomerID, sessionID, wcmessage.DirectionOutbound, uuid.Nil, "hello").Return(wm, nil)
		// first lookup: not stored yet; exactly one lookup is expected
		m.db.EXPECT().MessageGet(gomock.Any(), testSendMessageID).Return(nil, dbhandler.ErrNotFound).Times(1)
		m.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(fmt.Errorf("connection reset"))

		res, err := h.Send(context.Background(), newCV(), "hello", nil)
		if err == nil || res != nil {
			t.Errorf("expected error and nil result, got: %v, %v", res, err)
		}
	})

	for _, dupErr := range []string{
		"Error 1062: Duplicate entry 'x' for key 'PRIMARY'",
		"UNIQUE constraint failed: conversation_messages.id",
	} {
		t.Run("duplicate insert falls back to the stored row: "+dupErr[:12], func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newSendErrorHandler(mc)
			stored := newSendTestMessage()

			m.req.EXPECT().WebchatV1MessageCreate(gomock.Any(), testSendCustomerID, sessionID, wcmessage.DirectionOutbound, uuid.Nil, "hello").Return(wm, nil)
			gomock.InOrder(
				m.db.EXPECT().MessageGet(gomock.Any(), testSendMessageID).Return(nil, dbhandler.ErrNotFound),
				m.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(fmt.Errorf("%s", dupErr)),
				m.db.EXPECT().MessageGet(gomock.Any(), testSendMessageID).Return(stored, nil),
			)

			res, err := h.Send(context.Background(), newCV(), "hello", nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res != stored {
				t.Errorf("expected the stored message, got: %v", res)
			}
		})
	}

	t.Run("duplicate insert with a failing re-lookup returns the insert error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		m.req.EXPECT().WebchatV1MessageCreate(gomock.Any(), testSendCustomerID, sessionID, wcmessage.DirectionOutbound, uuid.Nil, "hello").Return(wm, nil)
		gomock.InOrder(
			m.db.EXPECT().MessageGet(gomock.Any(), testSendMessageID).Return(nil, dbhandler.ErrNotFound),
			m.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(fmt.Errorf("Error 1062: Duplicate entry 'x' for key 'PRIMARY'")),
			m.db.EXPECT().MessageGet(gomock.Any(), testSendMessageID).Return(nil, dbhandler.ErrNotFound),
		)

		res, err := h.Send(context.Background(), newCV(), "hello", nil)
		if err == nil || res != nil {
			t.Errorf("expected error and nil result, got: %v, %v", res, err)
		}
	})
}
