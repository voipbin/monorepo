package listenhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/pkg/conferencecallhandler"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
)

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)
	mockCC := conferencecallhandler.NewMockConferencecallHandler(mc)

	res := NewListenHandler(mockSock, "test-queue", "100", mockConf, mockCC)
	h, ok := res.(*listenHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.queueListen != "test-queue" || h.exchangeDelay != "100" {
		t.Errorf("Wrong fields. queue: %s, delay: %s", h.queueListen, h.exchangeDelay)
	}
	if h.sockHandler != mockSock || h.conferenceHandler != mockConf || h.conferencecallHandler != mockCC {
		t.Errorf("Wrong handler wiring.")
	}
	if h.utilHandler == nil {
		t.Errorf("utilHandler must be set.")
	}
}

func Test_Run_queueCreateFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &listenHandler{sockHandler: mockSock, queueListen: "test-queue"}

	errQueue := errors.New("simulated queue failure")
	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(errQueue)
	// ConsumeRPC must not be called when the queue could not be created.

	err := h.Run()
	if !errors.Is(err, errQueue) {
		t.Errorf("Wrong match. expect: wrapped queue error, got: %v", err)
	}
}

func Test_Run_consumesRequests(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &listenHandler{sockHandler: mockSock, queueListen: "test-queue"}

	done := make(chan struct{})
	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", constCosumerName, false, false, false, 10, gomock.Any()).DoAndReturn(
		func(_ context.Context, _, _ string, _, _, _ bool, _ int, _ sock.CbMsgRPC) error {
			defer close(done)
			return errors.New("simulated consume failure")
		})

	if err := h.Run(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for ConsumeRPC to be called")
	}
}
