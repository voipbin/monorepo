package subscribehandler

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	call "monorepo/bin-call-manager/models/call"

	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

// failureSignalHook forwards the error entries of the standard logger to a buffered channel without
// blocking, so a test can wait for a log line written inside a goroutine.
type failureSignalHook struct {
	ch chan string
}

func (h *failureSignalHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.ErrorLevel}
}

func (h *failureSignalHook) Fire(entry *logrus.Entry) error {
	select {
	case h.ch <- entry.Message:
	default:
	}

	return nil
}

func installFailureSignalHook(t *testing.T) *failureSignalHook {
	t.Helper()

	hook := &failureSignalHook{ch: make(chan string, 100)}
	logrus.AddHook(hook)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) })

	return hook
}

// waitMessage waits for an error log message containing every part.
func (h *failureSignalHook) waitMessage(t *testing.T, parts ...string) {
	t.Helper()

	deadline := time.After(5 * time.Second)
	for {
		select {
		case msg := <-h.ch:
			matched := true
			for _, p := range parts {
				if !strings.Contains(msg, p) {
					matched = false
				}
			}
			if matched {
				return
			}
		case <-deadline:
			t.Fatalf("Wrong match. expect an error log containing %v, got: none", parts)
		}
	}
}

func Test_Run_queue_create_failure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	mockSock.EXPECT().QueueCreate(string(commonoutline.QueueNameTimelineSubscribe), "normal").Return(fmt.Errorf("simulated queue failure"))

	h := NewSubscribeHandler(mockSock, mockDB)

	doneCh, err := h.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "could not declare the queue for subscribeHandler") || !strings.Contains(err.Error(), "simulated queue failure") {
		t.Errorf("Wrong match. expect the wrapped queue failure, got: %v", err)
	}
	if doneCh != nil {
		t.Errorf("Wrong match. expect: nil channel, got: non-nil")
	}
}

func Test_Run_consume_failure_is_logged(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := installFailureSignalHook(t)

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)
	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().QueueSubscribe(queueName, string(commonoutline.QueueNameAsteriskEventAll)).Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).Return(nil)
	mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, gomock.Any(), string(commonoutline.QueueNameEvent), false, nil).Return(nil).AnyTimes()
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, "timeline-manager", false, false, false, 10, gomock.Any()).DoAndReturn(
		func(_ context.Context, _, _ string, _, _, _ bool, _ int, _ sock.CbMsgConsume) error {
			return fmt.Errorf("simulated consume failure")
		},
	)

	h := NewSubscribeHandler(mockSock, mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCh, err := h.Run(ctx)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	hook.waitMessage(t, "Could not consume subscribe events", "simulated consume failure")

	cancel()
	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. expect the flush worker to finish after the cancel, got: still running")
	}
}

func Test_flushBatch_peer_event_insert_failure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := installFailureSignalHook(t)

	mockDB := dbhandler.NewMockDBHandler(mc)

	callID := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	customerID := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")
	msg := call.WebhookMessage{
		Identity:    commonidentity.Identity{ID: callID, CustomerID: customerID},
		Source:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"},
		Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100002"},
		Direction:   call.DirectionIncoming,
	}

	entries := []eventEntry{
		{
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      call.EventTypeCallHangup,
				Data:      mustJSON(t, msg),
			},
			receivedAt: time.Now(),
		},
		{
			// an event that is not projected into the peer table, so the two batches differ in size
			event: &sock.Event{
				Publisher: "flow-manager",
				Type:      "activeflow_created",
				Data:      []byte(`{"id":"55555555-5555-5555-5555-555555555555"}`),
			},
			receivedAt: time.Now(),
		},
	}

	// the primary insert succeeds, the secondary peer projection fails
	mockDB.EXPECT().EventBatchInsert(gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().PeerEventBatchInsert(gomock.Any(), gomock.Any()).Return(fmt.Errorf("simulated peer insert failure"))

	h := &subscribeHandler{dbHandler: mockDB}
	h.flushBatch(entries)

	hook.waitMessage(t, "Could not batch insert peer events into ClickHouse. count: 1", "simulated peer insert failure")
}
