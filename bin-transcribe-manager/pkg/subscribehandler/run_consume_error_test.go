package subscribehandler

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// consumeErrorHook signals on a buffered channel for every log entry that contains the text.
type consumeErrorHook struct {
	match string
	ch    chan struct{}
}

func (h *consumeErrorHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *consumeErrorHook) Fire(e *logrus.Entry) error {
	if strings.Contains(e.Message, h.match) {
		h.ch <- struct{}{}
	}
	return nil
}

// Test_Run_consumeErrorIsLogged checks that a failure of the consumer, which runs in its own
// goroutine and cannot be returned to the caller, is logged and does not fail Run.
func Test_Run_consumeErrorIsLogged(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := &consumeErrorHook{match: "Could not consume the subscribed evnet message correctly", ch: make(chan struct{}, 2)}
	logrus.AddHook(hook)
	t.Cleanup(func() {
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	})

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().QueueCreate(gomock.Any(), "normal").Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), "topic").Return(nil)
	// the list of binding patterns is internal to the handler
	mockSock.EXPECT().QueueBind(gomock.Any(), gomock.Any(), gomock.Any(), false, nil).Return(nil).AnyTimes()
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), gomock.Any(), gomock.Any(), false, false, false, 10, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgConsume) error {
			return fmt.Errorf("consume failed")
		})

	h := NewSubscribeHandler(mockSock, commonoutline.QueueNameTranscribeSubscribe, nil)
	if err := h.Run(); err != nil {
		t.Fatalf("Wrong match. expect: ok (the consumer failure is only logged), got: %v", err)
	}

	select {
	case <-hook.ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. expect: the consume error to be logged")
	}
}
