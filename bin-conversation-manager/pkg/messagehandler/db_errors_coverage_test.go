package messagehandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/smshandler"
	"monorepo/bin-conversation-manager/pkg/whatsapphandler"
)

var errStoreCoverage = errors.New("store failure")

func Test_NewMessageHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)
	mockLine := linehandler.NewMockLineHandler(mc)
	mockSMS := smshandler.NewMockSMSHandler(mc)
	mockWhatsApp := whatsapphandler.NewMockWhatsAppHandler(mc)

	res, ok := NewMessageHandler(mockDB, mockNotify, mockReq, mockAccount, mockLine, mockSMS, mockWhatsApp).(*messageHandler)
	if !ok {
		t.Fatalf("NewMessageHandler() did not return a *messageHandler")
	}

	if res.db != mockDB || res.notifyHandler != mockNotify || res.reqHandler != mockReq || res.accountHandler != mockAccount ||
		res.lineHandler != mockLine || res.smsHandler != mockSMS || res.whatsappHandler != mockWhatsApp {
		t.Errorf("NewMessageHandler() did not keep the given handlers")
	}
	if res.utilHandler == nil {
		t.Errorf("NewMessageHandler() did not create the util handler")
	}
}

// Every failure of the store is returned with the step that failed and the cause, and no event is published.
func Test_db_storeFailures(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("6a1d3c40-2b5e-4f78-9c0d-1e2f3a4b5c70")

	tests := []struct {
		name string

		setup func(m *sendMocks)
		call  func(h *messageHandler) error

		expectContext string
	}{
		{
			name: "create reads the created message back and that fails",
			setup: func(m *sendMocks) {
				m.db.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().MessageGet(ctx, id).Return(nil, errStoreCoverage)
			},
			call: func(h *messageHandler) error {
				_, err := h.Create(ctx, MessageCreateArgs{ID: id})
				return err
			},
			expectContext: "Could not get created message.",
		},
		{
			name: "delete fails",
			setup: func(m *sendMocks) {
				m.db.EXPECT().MessageDelete(ctx, id).Return(errStoreCoverage)
			},
			call: func(h *messageHandler) error {
				_, err := h.Delete(ctx, id)
				return err
			},
			expectContext: "Could not delete the message.",
		},
		{
			name: "delete reads the deleted message back and that fails",
			setup: func(m *sendMocks) {
				m.db.EXPECT().MessageDelete(ctx, id).Return(nil)
				m.db.EXPECT().MessageGet(ctx, id).Return(nil, errStoreCoverage)
			},
			call: func(h *messageHandler) error {
				_, err := h.Delete(ctx, id)
				return err
			},
			expectContext: "Could not get deleted message.",
		},
		{
			name: "list fails",
			setup: func(m *sendMocks) {
				m.db.EXPECT().MessageList(ctx, "token", uint64(10), gomock.Any()).Return(nil, errStoreCoverage)
			},
			call: func(h *messageHandler) error {
				_, err := h.List(ctx, "token", 10, map[message.Field]any{})
				return err
			},
			expectContext: "Could not get messages.",
		},
		{
			name: "update status reads the updated message back and that fails",
			setup: func(m *sendMocks) {
				m.db.EXPECT().MessageUpdate(ctx, id, map[message.Field]any{message.FieldStatus: message.StatusDone}).Return(nil)
				m.db.EXPECT().MessageGet(ctx, id).Return(nil, errStoreCoverage)
			},
			call: func(h *messageHandler) error {
				_, err := h.UpdateStatus(ctx, id, message.StatusDone)
				return err
			},
			expectContext: "Could not get updated message.",
		},
		{
			name: "get fails with an error that is not a missing record",
			setup: func(m *sendMocks) {
				m.db.EXPECT().MessageGet(ctx, id).Return(nil, errStoreCoverage)
			},
			call: func(h *messageHandler) error {
				_, err := h.Get(ctx, id)
				return err
			},
			expectContext: "Could not get message.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the notify mock has no declared call, so a published event fails the test
			h, m := newSendErrorHandler(mc)
			tt.setup(m)

			err := tt.call(h)
			if !errors.Is(err, errStoreCoverage) {
				t.Errorf("error = %v, want the cause %v kept", err, errStoreCoverage)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectContext) {
				t.Errorf("error = %v, want the context %q", err, tt.expectContext)
			}
		})
	}
}
