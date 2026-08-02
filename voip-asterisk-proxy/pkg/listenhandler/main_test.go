package listenhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// waitTimeout is the max time a test waits for a consumer goroutine to
// signal completion via its done channel, or for a value on chErr.
const waitTimeout = 2 * time.Second

func Test_Run_validation_error(t *testing.T) {

	tests := []struct {
		name string

		rabbitQueueListenRequestPermanent string
		rabbitQueueListenRequestVolatile  string
	}{
		{
			name: "empty element from trailing comma in permanent list",

			rabbitQueueListenRequestPermanent: "asterisk.call.request,",
			rabbitQueueListenRequestVolatile:  "asterisk.11:22:33:44:55:66.request",
		},
		{
			name: "empty element from trailing comma in volatile list",

			rabbitQueueListenRequestPermanent: "asterisk.call.request",
			rabbitQueueListenRequestVolatile:  "",
		},
		{
			name: "duplicate element within permanent list",

			rabbitQueueListenRequestPermanent: "asterisk.call.request,asterisk.call.request",
			rabbitQueueListenRequestVolatile:  "asterisk.11:22:33:44:55:66.request",
		},
		{
			name: "volatile queue name collides with permanent queue name",

			rabbitQueueListenRequestPermanent: "asterisk.11:22:33:44:55:66.request",
			rabbitQueueListenRequestVolatile:  "asterisk.11:22:33:44:55:66.request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			// no QueueCreate call is expected for any of these cases -- the
			// controller will fail the test if QueueCreate is called since
			// no expectation was set.

			h := &listenHandler{
				sockHandler:                       mockSock,
				rabbitQueueListenRequestPermanent: tt.rabbitQueueListenRequestPermanent,
				rabbitQueueListenRequestVolatile:  tt.rabbitQueueListenRequestVolatile,
			}

			chErr, err := h.Run()
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if chErr != nil {
				t.Errorf("Wrong match. expect: nil channel, got: %v", chErr)
			}
		})
	}
}

func Test_Run_permanent_declare_failure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &listenHandler{
		sockHandler:                       mockSock,
		rabbitQueueListenRequestPermanent: "asterisk.call.request",
		rabbitQueueListenRequestVolatile:  "asterisk.11:22:33:44:55:66.request",
	}

	mockSock.EXPECT().QueueCreate("asterisk.call.request", "normal").Return(fmt.Errorf("could not declare exchange"))
	// no other QueueCreate or ConsumeRPC calls are expected -- the strict
	// mock controller fails the test if they occur.

	chErr, err := h.Run()
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
	if chErr != nil {
		t.Errorf("Wrong match. expect: nil channel, got: %v", chErr)
	}
}

func Test_Run_volatile_declare_failure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &listenHandler{
		sockHandler:                       mockSock,
		rabbitQueueListenRequestPermanent: "asterisk.call.request",
		rabbitQueueListenRequestVolatile:  "asterisk.11:22:33:44:55:66.request",
	}

	mockSock.EXPECT().QueueCreate("asterisk.call.request", "normal").Return(nil)
	mockSock.EXPECT().QueueCreate("asterisk.11:22:33:44:55:66.request", "volatile").Return(fmt.Errorf("could not declare exchange"))
	// no ConsumeRPC call is expected for either queue.

	chErr, err := h.Run()
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
	if chErr != nil {
		t.Errorf("Wrong match. expect: nil channel, got: %v", chErr)
	}
}

func Test_Run_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &listenHandler{
		sockHandler:                       mockSock,
		rabbitQueueListenRequestPermanent: "asterisk.call.request",
		rabbitQueueListenRequestVolatile:  "asterisk.11:22:33:44:55:66.request",
	}

	donePermanent := make(chan struct{})
	doneVolatile := make(chan struct{})

	mockSock.EXPECT().QueueCreate("asterisk.call.request", "normal").Return(nil)
	mockSock.EXPECT().QueueCreate("asterisk.11:22:33:44:55:66.request", "volatile").Return(nil)

	mockSock.EXPECT().ConsumeRPC(gomock.Any(), "asterisk.call.request", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cbConsume sock.CbMsgRPC) error {
			defer close(donePermanent)
			return nil
		},
	)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), "asterisk.11:22:33:44:55:66.request", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cbConsume sock.CbMsgRPC) error {
			defer close(doneVolatile)
			return nil
		},
	)

	chErr, err := h.Run()
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if chErr == nil {
		t.Errorf("Wrong match. expect: non-nil channel, got: nil")
	}

	waitDone(t, donePermanent, "permanent consumer")
	waitDone(t, doneVolatile, "volatile consumer")
}

func Test_Run_single_consumer_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &listenHandler{
		sockHandler:                       mockSock,
		rabbitQueueListenRequestPermanent: "asterisk.call.request",
		rabbitQueueListenRequestVolatile:  "asterisk.11:22:33:44:55:66.request",
	}

	done := make(chan struct{})
	expectedErr := fmt.Errorf("consumer died")

	mockSock.EXPECT().QueueCreate("asterisk.call.request", "normal").Return(nil)
	mockSock.EXPECT().QueueCreate("asterisk.11:22:33:44:55:66.request", "volatile").Return(nil)

	mockSock.EXPECT().ConsumeRPC(gomock.Any(), "asterisk.call.request", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cbConsume sock.CbMsgRPC) error {
			defer close(done)
			return expectedErr
		},
	)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), "asterisk.11:22:33:44:55:66.request", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	chErr, err := h.Run()
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	waitDone(t, done, "permanent consumer")

	select {
	case gotErr := <-chErr:
		if gotErr != expectedErr {
			t.Errorf("Wrong match. expect: %v, got: %v", expectedErr, gotErr)
		}
	case <-time.After(waitTimeout):
		t.Errorf("Timed out waiting for the consumer error on chErr")
	}
}

func Test_Run_concurrent_consumer_errors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &listenHandler{
		sockHandler:                       mockSock,
		rabbitQueueListenRequestPermanent: "asterisk.call.request",
		rabbitQueueListenRequestVolatile:  "asterisk.11:22:33:44:55:66.request",
	}

	donePermanent := make(chan struct{})
	doneVolatile := make(chan struct{})
	errPermanent := fmt.Errorf("permanent consumer died")
	errVolatile := fmt.Errorf("volatile consumer died")

	mockSock.EXPECT().QueueCreate("asterisk.call.request", "normal").Return(nil)
	mockSock.EXPECT().QueueCreate("asterisk.11:22:33:44:55:66.request", "volatile").Return(nil)

	mockSock.EXPECT().ConsumeRPC(gomock.Any(), "asterisk.call.request", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cbConsume sock.CbMsgRPC) error {
			defer close(donePermanent)
			return errPermanent
		},
	)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), "asterisk.11:22:33:44:55:66.request", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cbConsume sock.CbMsgRPC) error {
			defer close(doneVolatile)
			return errVolatile
		},
	)

	chErr, err := h.Run()
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	// both goroutines must actually invoke ConsumeRPC (and therefore have
	// sent, or be about to send, on chErr) before we read anything off the
	// channel -- otherwise this test would pass even if the channel were
	// sized 1 instead of len(listenQueues), since one goroutine could
	// simply not have run yet.
	waitDone(t, donePermanent, "permanent consumer")
	waitDone(t, doneVolatile, "volatile consumer")

	got := map[error]bool{}
	for i := range 2 {
		select {
		case gotErr := <-chErr:
			got[gotErr] = true
		case <-time.After(waitTimeout):
			t.Fatalf("Timed out waiting for consumer error #%d on chErr", i+1)
		}
	}

	if !got[errPermanent] || !got[errVolatile] {
		t.Errorf("Wrong match. expect both errors delivered on chErr. got: %v", got)
	}
}

// waitDone waits for the given channel to be closed, failing the test if it
// isn't closed within waitTimeout.
func waitDone(t *testing.T, done chan struct{}, label string) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(waitTimeout):
		t.Fatalf("Timed out waiting for %s to run", label)
	}
}
