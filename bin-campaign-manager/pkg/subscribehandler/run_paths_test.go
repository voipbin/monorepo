package subscribehandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_processEventRun_dispatchesAsynchronously(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
	h := &subscribeHandler{campaigncallHandler: mockCampaigncall}

	activeflowID := uuid.FromStringOrNil("5b0e9a3c-0000-4000-8000-000000000001")
	entered := make(chan uuid.UUID, 1)
	release := make(chan struct{})
	finished := make(chan struct{})
	mockCampaigncall.EXPECT().GetByActiveflowID(gomock.Any(), activeflowID).
		DoAndReturn(func(_ context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error) {
			defer close(finished)
			entered <- id
			// the handler is held here until the test has seen processEventRun return
			<-release
			return nil, errors.New("not found")
		})

	event := &sock.Event{
		Publisher: "flow-manager",
		Type:      "activeflow_deleted",
		DataType:  "application/json",
		Data:      []byte(`{"id":"5b0e9a3c-0000-4000-8000-000000000001"}`),
	}

	returned := make(chan error, 1)
	go func() { returned <- h.processEventRun(event) }()

	select {
	case err := <-returned:
		if err != nil {
			t.Fatalf("Wrong match. expect: nil, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("processEventRun waited for the event handler instead of returning at once")
	}

	select {
	case got := <-entered:
		if got != activeflowID {
			t.Errorf("Wrong match. expect: %s, got: %s", activeflowID, got)
		}
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("the event was not dispatched to the campaigncall handler")
	}

	close(release)
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("the event handler did not finish")
	}
}

func Test_Run_consumeFailureIsLogged(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := logrustest.NewGlobal()
	t.Cleanup(func() { hook.Reset(); logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) })

	mockSock := sockhandler.NewMockSockHandler(mc)
	queueName := string(commonoutline.QueueNameCampaignSubscribe)

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, gomock.Any(), string(commonoutline.QueueNameEvent), false, nil).Return(nil).Times(len(topicPatterns))
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameCampaignManager), false, false, false, 10, gomock.Any()).
		Return(errors.New("simulated consume failure"))

	h := NewSubscribeHandler(mockSock, queueName, nil, nil, nil)
	if err := h.Run(); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}

	deadline := time.After(5 * time.Second)
	for {
		for _, e := range hook.AllEntries() {
			if e.Level == logrus.ErrorLevel && e.Message == "Could not consume the request message correctly. err: simulated consume failure" {
				return
			}
		}
		select {
		case <-deadline:
			t.Fatal("the consume failure was not logged")
		case <-time.After(10 * time.Millisecond):
		}
	}
}
