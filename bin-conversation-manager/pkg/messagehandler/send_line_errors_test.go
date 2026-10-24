package messagehandler

import (
	"context"
	"fmt"
	"testing"

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
)

func Test_Send_sendLine_errors(t *testing.T) {
	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000002"),
		},
		AccountID: uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000003"),
		Type:      conversation.TypeLine,
		DialogID:  "dialog-1",
	}
	ac := &account.Account{
		Identity: commonidentity.Identity{ID: cv.AccountID},
	}
	messageID := uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000004")
	created := &message.Message{
		Identity: commonidentity.Identity{
			ID:         messageID,
			CustomerID: cv.CustomerID,
		},
		ConversationID: cv.ID,
		Status:         message.StatusProgressing,
	}
	failed := &message.Message{
		Identity: commonidentity.Identity{
			ID:         messageID,
			CustomerID: cv.CustomerID,
		},
		ConversationID: cv.ID,
		Status:         message.StatusFailed,
	}
	text := "hello"
	medias := []media.Media{}

	tests := []struct {
		name  string
		setup func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockNotify *notifyhandler.MockNotifyHandler, mockAccount *accounthandler.MockAccountHandler, mockLine *linehandler.MockLineHandler)
	}{
		{
			name: "account lookup fails before any message is stored",
			setup: func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockNotify *notifyhandler.MockNotifyHandler, mockAccount *accounthandler.MockAccountHandler, mockLine *linehandler.MockLineHandler) {
				mockAccount.EXPECT().Get(ctx, cv.AccountID).Return(nil, fmt.Errorf("account failed"))
			},
		},
		{
			name: "message creation fails and nothing is sent",
			setup: func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockNotify *notifyhandler.MockNotifyHandler, mockAccount *accounthandler.MockAccountHandler, mockLine *linehandler.MockLineHandler) {
				mockAccount.EXPECT().Get(ctx, cv.AccountID).Return(ac, nil)
				mockUtil.EXPECT().UUIDCreate().Return(messageID)
				mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(fmt.Errorf("create failed"))
			},
		},
		{
			name: "line send failure marks the stored message as failed",
			setup: func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockNotify *notifyhandler.MockNotifyHandler, mockAccount *accounthandler.MockAccountHandler, mockLine *linehandler.MockLineHandler) {
				mockAccount.EXPECT().Get(ctx, cv.AccountID).Return(ac, nil)
				mockUtil.EXPECT().UUIDCreate().Return(messageID)
				mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().MessageGet(ctx, messageID).Return(created, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, cv.CustomerID, message.EventTypeMessageCreated, created)

				mockLine.EXPECT().Send(ctx, cv, ac, text, medias).Return(fmt.Errorf("line failed"))

				mockDB.EXPECT().MessageUpdate(ctx, messageID, map[message.Field]any{message.FieldStatus: message.StatusFailed}).Return(nil)
				mockDB.EXPECT().MessageGet(ctx, messageID).Return(failed, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, cv.CustomerID, message.EventTypeMessageUpdated, failed)
			},
		},
		{
			name: "status update after a successful send fails",
			setup: func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockNotify *notifyhandler.MockNotifyHandler, mockAccount *accounthandler.MockAccountHandler, mockLine *linehandler.MockLineHandler) {
				mockAccount.EXPECT().Get(ctx, cv.AccountID).Return(ac, nil)
				mockUtil.EXPECT().UUIDCreate().Return(messageID)
				mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().MessageGet(ctx, messageID).Return(created, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, cv.CustomerID, message.EventTypeMessageCreated, created)

				mockLine.EXPECT().Send(ctx, cv, ac, text, medias).Return(nil)

				mockDB.EXPECT().MessageUpdate(ctx, messageID, map[message.Field]any{message.FieldStatus: message.StatusDone}).Return(fmt.Errorf("update failed"))
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

			tt.setup(ctx, mockUtil, mockDB, mockNotify, mockAccount, mockLine)

			res, err := h.Send(ctx, cv, text, medias)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_Send_unsupportedType(t *testing.T) {
	// no handler is configured: an unknown conversation type must not reach any of them.
	h := &messageHandler{}

	res, err := h.Send(context.Background(), &conversation.Conversation{Type: conversation.Type("unknown")}, "hello", nil)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}
