package main

import (
	"errors"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-schedule-manager/internal/config"
	"monorepo/bin-schedule-manager/pkg/backuphandler"
	"monorepo/bin-schedule-manager/pkg/dispatchhandler"
	"monorepo/bin-schedule-manager/pkg/schedulehandler"
)

const testWait = 5 * time.Second

// useRedisConfig points the process-wide config at the given cache address for
// the test and restores it afterwards. The config singleton is loaded once
// first, so nothing later overwrites the test value.
func useRedisConfig(t *testing.T, addr string) {
	t.Helper()

	config.LoadGlobalConfig()
	old := *config.Get()
	*config.Get() = config.Config{RedisAddress: addr}

	t.Cleanup(func() { *config.Get() = old })
}

func Test_initCache_connects(t *testing.T) {
	mr := miniredis.RunT(t)
	useRedisConfig(t, mr.Addr())

	cache, err := initCache()
	if err != nil || cache == nil {
		t.Errorf("Wrong match. expect: a connected cache, got: %v, %v", cache, err)
	}
}

// Test_initCache_unreachable connects to the address of a server that was just
// stopped. Client retries make a failed connect slow, so this is the only
// failure case.
func Test_initCache_unreachable(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()
	useRedisConfig(t, addr)

	cache, err := initCache()
	if err == nil || cache != nil {
		t.Fatalf("Wrong match. expect: error and nil cache, got: %v, %v", cache, err)
	}
	if !strings.Contains(err.Error(), "cache connect error") {
		t.Errorf("Wrong match. expect: the cache connect error, got: %v", err)
	}
}

// Test_initSignal injects a signal value into the package channel. The helper
// goroutine of initSignal never exits on its own, so this is the only call in
// the package.
func Test_initSignal(t *testing.T) {
	t.Cleanup(func() { signal.Stop(chSigs) })

	initSignal()

	chSigs <- syscall.SIGTERM

	select {
	case <-chDone:
	case <-time.After(testWait):
		t.Fatalf("Wrong match. expect: the done channel signaled, got: timeout")
	}
}

func Test_runListen(t *testing.T) {
	errQueue := errors.New("synthetic queue failure")
	queue := string(commonoutline.QueueNameScheduleRequest)

	tests := []struct {
		name string

		queueErr error
	}{
		{name: "queue declared and consumer started"},
		{name: "queue declare fails", queueErr: errQueue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockSchedule := schedulehandler.NewMockScheduleHandler(mc)
			mockDispatch := dispatchhandler.NewMockDispatchHandler(mc)
			mockBackup := backuphandler.NewMockBackupHandler(mc)

			mockSock.EXPECT().QueueCreate(queue, "normal").Return(tt.queueErr)

			consumed := make(chan struct{})
			if tt.queueErr == nil {
				mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, gomock.Any(), false, false, false, 10, gomock.Any()).
					DoAndReturn(func(_ any, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
						close(consumed)
						return nil
					})
			}

			err := runListen(mockSock, mockSchedule, mockDispatch, mockBackup, 90)
			if tt.queueErr != nil {
				if !errors.Is(err, tt.queueErr) || !strings.Contains(err.Error(), "could not run the listenhandler") {
					t.Errorf("Wrong match. expect: wrapped queue error, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			select {
			case <-consumed:
			case <-time.After(testWait):
				t.Fatalf("Wrong match. expect: the consumer started, got: timeout")
			}
		})
	}
}

func Test_runSubscribe(t *testing.T) {
	errBind := errors.New("synthetic bind failure")
	queue := string(commonoutline.QueueNameScheduleSubscribe)
	exchange := string(commonoutline.QueueNameEvent)

	tests := []struct {
		name string

		bindErr error
	}{
		{name: "queue bound and consumer started"},
		{name: "bind fails", bindErr: errBind},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockSchedule := schedulehandler.NewMockScheduleHandler(mc)

			// the subscribed pattern list is private to the subscribe handler and
			// has one entry, so the pattern argument is not pinned here
			calls := []any{
				mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil),
				mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil),
				mockSock.EXPECT().QueueBind(queue, gomock.Any(), exchange, false, nil).Return(tt.bindErr).Times(1),
			}
			gomock.InOrder(calls...)

			consumed := make(chan struct{})
			if tt.bindErr == nil {
				mockSock.EXPECT().ConsumeMessage(gomock.Any(), queue, gomock.Any(), false, false, false, 10, gomock.Any()).
					DoAndReturn(func(_ any, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgConsume) error {
						close(consumed)
						return nil
					})
			}

			err := runSubscribe(mockSock, mockSchedule)
			if tt.bindErr != nil {
				if !errors.Is(err, tt.bindErr) {
					t.Errorf("Wrong match. expect: the bind error reachable, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			select {
			case <-consumed:
			case <-time.After(testWait):
				t.Fatalf("Wrong match. expect: the consumer started, got: timeout")
			}
		})
	}
}
