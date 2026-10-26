package subscribehandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-schedule-manager/pkg/schedulehandler"
)

// messageHook forwards the message of every logged entry to a channel.
type messageHook struct {
	messages chan string
}

func (h *messageHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *messageHook) Fire(e *logrus.Entry) error {
	select {
	case h.messages <- e.Message:
	default:
	}
	return nil
}

// captureLogs swaps the hooks of the standard logger for the test and restores them afterwards.
func captureLogs(t *testing.T) *messageHook {
	t.Helper()

	hook := &messageHook{messages: make(chan string, 64)}
	old := logrus.StandardLogger().ReplaceHooks(logrus.LevelHooks{})
	logrus.StandardLogger().AddHook(hook)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(old) })

	return hook
}

func waitForLog(t *testing.T, hook *messageHook, contains string) {
	t.Helper()

	timeout := time.After(testWait)

	for {
		select {
		case msg := <-hook.messages:
			if strings.Contains(msg, contains) {
				return
			}
		case <-timeout:
			t.Fatalf("Wrong match. expect: a log containing %q, got: timeout", contains)
		}
	}
}

func Test_processEvent_logs_handler_failure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSchedule := schedulehandler.NewMockScheduleHandler(mc)

	customer := &cmcustomer.Customer{ID: uuid.FromStringOrNil("29c7a2be-6f23-11f0-b3f6-9f2a3ba4bd12")}
	data, err := json.Marshal(customer)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	mockSchedule.EXPECT().EventCustomerDeleted(gomock.Any(), customer).Return(stderrors.New("synthetic handler failure"))

	hook := captureLogs(t)
	h := &subscribeHandler{sockHandler: mockSock, scheduleHandler: mockSchedule}

	h.processEvent(&sock.Event{
		Publisher: publisherCustomerManager,
		Type:      string(cmcustomer.EventTypeCustomerDeleted),
		Data:      json.RawMessage(data),
	})

	waitForLog(t, hook, "Could not process the event correctly")
}

func Test_Run_consumer_failure_is_only_logged(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSchedule := schedulehandler.NewMockScheduleHandler(mc)

	queueName := string(commonoutline.QueueNameScheduleSubscribe)
	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, gomock.Any(), string(commonoutline.QueueNameEvent), false, nil).Return(nil).Times(len(topicPatterns))
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgConsume) error {
			return stderrors.New("synthetic consume failure")
		})

	hook := captureLogs(t)
	h := NewSubscribeHandler(mockSock, queueName, mockSchedule)

	if err := h.Run(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	waitForLog(t, hook, "Could not consume the request message correctly")
}
