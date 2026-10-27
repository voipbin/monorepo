package listenhandler

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_listenRun_consumeError(t *testing.T) {
	logs := captureLogs(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &listenHandler{
		sockHandler:                       mockSock,
		rabbitQueueListenRequestPermanent: "perm-q",
		rabbitQueueListenRequestVolatile:  "vol-q",
	}

	mockSock.EXPECT().QueueCreate("perm-q", "normal").Return(nil)
	mockSock.EXPECT().QueueCreate("vol-q", "volatile").Return(nil)
	for _, q := range []string{"perm-q", "vol-q"} {
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), q, "rtpengine-proxy", false, false, false, 10, gomock.Any()).DoAndReturn(
			func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgRPC) error {
				return errors.New("consume failed")
			}).Times(1)
	}

	if err := h.listenRun(); err != nil {
		t.Fatalf("Wrong match. expected: ok, got: %v", err)
	}

	// each queue has its own goroutine: wait for both logs before the controller finishes
	for _, q := range []string{"perm-q", "vol-q"} {
		logs.waitFor(t, "consume error for "+q, msgIs(logrus.ErrorLevel, "ConsumeRPC failed for queue "+q))
	}
}
