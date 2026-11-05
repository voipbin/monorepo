package messagehandler

import (
	"context"
	stderrors "errors"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/linehandler"
)

func Test_sendSMS_medias(t *testing.T) {
	fileID := uuid.FromStringOrNil("c5e3f6a4-1234-11f0-eeee-eeeeeeeeeeee")
	medias := []media.Media{
		{Identity: commonidentity.Identity{ID: fileID}, Type: media.TypeImage, Filename: "a.jpg"},
	}

	t.Run("the message is created before it is sent and the media is handed over", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		cv := newSendTestConversation(conversation.TypeMessage)
		created := newSendTestMessage()

		// the event path of message-manager only updates the status of an existing record, so the
		// record has to exist before the send call.
		m.util.EXPECT().UUIDCreate().Return(testSendMessageID)
		gomock.InOrder(
			m.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(nil),
			m.db.EXPECT().MessageGet(gomock.Any(), testSendMessageID).Return(created, nil),
			m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), testSendCustomerID, message.EventTypeMessageCreated, created),
			m.sms.EXPECT().Send(gomock.Any(), cv, testSendMessageID, "hello", medias).Return(nil),
		)

		res, err := h.Send(context.Background(), cv, "hello", medias)
		if err != nil || res == nil {
			t.Errorf("expected ok, got: %v, %v", res, err)
		}
	})

	t.Run("a media without a file id is rejected before anything is created", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _ := newSendErrorHandler(mc)
		cv := newSendTestConversation(conversation.TypeMessage)

		// the valid media comes first, so a check that stops at the first media would let this pass
		res, err := h.Send(context.Background(), cv, "hello", []media.Media{medias[0], {Type: media.TypeFile}})
		var ve *cerrors.VoipbinError
		if !stderrors.As(err, &ve) || res != nil {
			t.Fatalf("expected a typed error and nil result, got: %v, %v", res, err)
		}
		if ve.Reason != "MEDIA_ID_REQUIRED" {
			t.Errorf("Wrong reason. got: %s", ve.Reason)
		}
	})

	t.Run("a media without a file id is rejected even when it comes first", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _ := newSendErrorHandler(mc)
		cv := newSendTestConversation(conversation.TypeMessage)

		_, err := h.Send(context.Background(), cv, "hello", []media.Media{{Type: media.TypeFile}, medias[0]})
		var ve *cerrors.VoipbinError
		if !stderrors.As(err, &ve) || ve.Reason != "MEDIA_ID_REQUIRED" {
			t.Errorf("expected MEDIA_ID_REQUIRED, got: %v", err)
		}
	})

	t.Run("a typed mms error marks the message failed and is returned unchanged", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		cv := newSendTestConversation(conversation.TypeMessage)
		sendErr := cerrors.InvalidArgument("message-manager", "MEDIA_NOT_FOUND", "not found")

		m.util.EXPECT().UUIDCreate().Return(testSendMessageID)
		expectCreateOK(m, testSendMessageID)
		m.sms.EXPECT().Send(gomock.Any(), cv, testSendMessageID, "hello", medias).Return(sendErr)

		failed := newSendTestMessage()
		m.db.EXPECT().MessageUpdate(gomock.Any(), testSendMessageID, failedStatus()).Return(nil)
		m.db.EXPECT().MessageGet(gomock.Any(), testSendMessageID).Return(failed, nil)
		m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), testSendCustomerID, message.EventTypeMessageUpdated, failed)

		res, err := h.Send(context.Background(), cv, "hello", medias)
		if !stderrors.Is(err, sendErr) || res != nil {
			t.Errorf("expected the typed error and nil result, got: %v, %v", res, err)
		}
	})
}

func Test_Send_mediasIgnoredForOtherTypes(t *testing.T) {
	fileID := uuid.FromStringOrNil("c5e3f6a4-1234-11f0-eeee-eeeeeeeeeeee")
	medias := []media.Media{{Identity: commonidentity.Identity{ID: fileID}}}

	t.Run("a line conversation stores and sends an empty media list", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		mockLine := linehandler.NewMockLineHandler(mc)
		h.lineHandler = mockLine
		cv := newSendTestConversation(conversation.TypeLine)
		ac := &account.Account{Identity: commonidentity.Identity{ID: testSendAccountID}}
		created := newSendTestMessage()

		m.account.EXPECT().Get(gomock.Any(), testSendAccountID).Return(ac, nil)
		m.util.EXPECT().UUIDCreate().Return(testSendMessageID)
		m.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, msg *message.Message) error {
			if len(msg.Medias) != 0 {
				t.Errorf("the record must not claim a media that is not sent. medias: %v", msg.Medias)
			}
			return nil
		})
		m.db.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(created, nil).AnyTimes()
		m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), testSendCustomerID, gomock.Any(), gomock.Any()).AnyTimes()
		mockLine.EXPECT().Send(gomock.Any(), cv, ac, "hello", []media.Media{}).Return(nil)
		m.db.EXPECT().MessageUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		if _, err := h.Send(context.Background(), cv, "hello", medias); err != nil {
			t.Errorf("expected ok, got: %v", err)
		}
	})

	t.Run("a line conversation without medias keeps the media list nil", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSendErrorHandler(mc)
		mockLine := linehandler.NewMockLineHandler(mc)
		h.lineHandler = mockLine
		cv := newSendTestConversation(conversation.TypeLine)
		ac := &account.Account{Identity: commonidentity.Identity{ID: testSendAccountID}}
		created := newSendTestMessage()

		m.account.EXPECT().Get(gomock.Any(), testSendAccountID).Return(ac, nil)
		m.util.EXPECT().UUIDCreate().Return(testSendMessageID)
		m.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, msg *message.Message) error {
			if msg.Medias != nil {
				t.Errorf("a nil list must stay nil. medias: %v", msg.Medias)
			}
			return nil
		})
		m.db.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(created, nil).AnyTimes()
		m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), testSendCustomerID, gomock.Any(), gomock.Any()).AnyTimes()
		mockLine.EXPECT().Send(gomock.Any(), cv, ac, "hello", gomock.Nil()).Return(nil)
		m.db.EXPECT().MessageUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		if _, err := h.Send(context.Background(), cv, "hello", nil); err != nil {
			t.Errorf("expected ok, got: %v", err)
		}
	})
}
