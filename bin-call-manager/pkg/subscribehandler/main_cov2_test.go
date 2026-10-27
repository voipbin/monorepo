package subscribehandler

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// signalHook notifies via channel when a log entry containing the keyword is fired.
type signalHook struct {
	keyword string
	ch      chan string
}

func (s *signalHook) Levels() []logrus.Level { return logrus.AllLevels }

func (s *signalHook) Fire(e *logrus.Entry) error {
	if strings.Contains(e.Message, s.keyword) {
		select {
		case s.ch <- e.Message:
		default:
		}
	}
	return nil
}

func Test_Run_cov2_consumeError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &subscribeHandler{sockHandler: mockSock, subscribeQueue: commonoutline.QueueName("test-queue")}

	hook := &signalHook{keyword: "Could not consume the subscribed", ch: make(chan string, 1)}
	hooks := make(logrus.LevelHooks)
	hooks.Add(hook)
	old := logrus.StandardLogger().ReplaceHooks(hooks)
	defer logrus.StandardLogger().ReplaceHooks(old)

	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
	mockSock.EXPECT().QueueSubscribe("test-queue", string(commonoutline.QueueNameAsteriskEventAll)).Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind("test-queue", gomock.Any(), string(commonoutline.QueueNameEvent), false, nil).Return(nil).Times(len(topicPatterns))
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), "test-queue", gomock.Any(), false, false, false, 20, gomock.Any()).Return(errors.New("consume failed"))

	if err := h.Run(); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}

	var msg string
	select {
	case msg = <-hook.ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("Timed out waiting for the expected log message")
	}
	if !strings.Contains(msg, "consume failed") {
		t.Errorf("Wrong match. got: %s", msg)
	}
}
