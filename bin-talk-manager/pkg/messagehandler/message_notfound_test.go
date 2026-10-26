package messagehandler

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-talk-manager/pkg/dbhandler"
)

func Test_New(t *testing.T) {
	mc := gomock.NewController(t)

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := sockhandler.NewMockSockHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := commonutil.NewMockUtilHandler(mc)

	h, ok := New(mockDB, mockSock, mockNotify, mockUtil).(*messageHandler)
	if !ok {
		t.Fatalf("New() did not return a *messageHandler")
	}

	if h.dbHandler != mockDB || h.sockHandler != mockSock || h.notifyHandler != mockNotify || h.utilHandler != mockUtil {
		t.Errorf("New() did not keep its collaborators")
	}
}

func Test_MessageGet_notFound_error(t *testing.T) {
	mc := gomock.NewController(t)

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &messageHandler{dbHandler: mockDB}

	ctx := context.Background()
	id := uuid.FromStringOrNil("7c1d5e60-9a7f-11ef-8c2b-0f6f3c1a7d41")

	mockDB.EXPECT().MessageGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

	res, err := h.MessageGet(ctx, id)
	if res != nil {
		t.Errorf("MessageGet() result = %v, want none", res)
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) || ve.Status != cerrors.StatusNotFound || ve.Reason != "MESSAGE_NOT_FOUND" {
		t.Fatalf("MessageGet() error = %v, want a MESSAGE_NOT_FOUND not found error", err)
	}
	if !stderrors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("MessageGet() error = %v, want it to keep the database error as its cause", err)
	}
}
