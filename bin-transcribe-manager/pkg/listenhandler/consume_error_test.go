package listenhandler

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// matchCountHook counts the log entries whose message contains the given text and signals every
// match on a buffered channel, so a test can wait for an exact number of matches.
type matchCountHook struct {
	match string
	count atomic.Int64
	ch    chan struct{}
}

func (h *matchCountHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *matchCountHook) Fire(e *logrus.Entry) error {
	if strings.Contains(e.Message, h.match) {
		h.count.Add(1)
		h.ch <- struct{}{}
	}
	return nil
}

// Test_Run_consumeErrorIsLogged checks that a failure of a queue consumer, which runs in its own
// goroutine and cannot be returned to the caller, is logged once per queue and does not fail Run.
func Test_Run_consumeErrorIsLogged(t *testing.T) {
	const queue = "test-queue"
	const queueVolatile = "test-queue-volatile"

	mc := gomock.NewController(t)
	defer mc.Finish()

	// both goroutines write the same message, so the hook counts the matches
	hook := &matchCountHook{match: "Could not consume the request message correctly", ch: make(chan struct{}, 4)}
	logrus.AddHook(hook)
	t.Cleanup(func() {
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	})

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &listenHandler{sockHandler: mockSock}

	consumed := make(chan string, 2)
	for _, name := range []string{queue, queueVolatile} {
		mockSock.EXPECT().QueueCreate(name, gomock.Any()).Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), name, constCosumerName, false, false, false, 10, gomock.Any()).DoAndReturn(
			func(_ context.Context, queueName string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
				consumed <- queueName
				return fmt.Errorf("consume failed for %s", queueName)
			})
	}

	if err := h.Run(queue, queueVolatile, ""); err != nil {
		t.Fatalf("Wrong match. expect: ok (the consumer failure is only logged), got: %v", err)
	}

	for i := 0; i < 2; i++ {
		select {
		case <-hook.ch:
		case <-time.After(5 * time.Second):
			t.Fatalf("Wrong match. expect: 2 consume error logs, got: %d", hook.count.Load())
		}
	}
	if got := len(consumed); got != 2 {
		t.Errorf("Wrong match. expect: both queues consumed, got: %d", got)
	}
}
