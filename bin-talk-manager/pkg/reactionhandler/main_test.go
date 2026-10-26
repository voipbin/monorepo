package reactionhandler

import (
	"testing"

	gomock "go.uber.org/mock/gomock"

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

	h, ok := New(mockDB, mockSock, mockNotify, mockUtil).(*reactionHandler)
	if !ok {
		t.Fatalf("New() did not return a *reactionHandler")
	}

	if h.dbHandler != mockDB || h.sockHandler != mockSock || h.notifyHandler != mockNotify || h.utilHandler != mockUtil {
		t.Errorf("New() did not keep its collaborators")
	}
}
