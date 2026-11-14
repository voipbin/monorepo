package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/internal/config"
	"monorepo/bin-billing-manager/models/failedevent"
	"monorepo/bin-billing-manager/pkg/dbhandler"
	"monorepo/bin-billing-manager/pkg/subscribehandler"
)

var errTestCause = errors.New("injected failure")

const waitLimit = 5 * time.Second

func Test_signalHandler(t *testing.T) {
	origSigs, origDone := chSigs, chDone
	t.Cleanup(func() { chSigs, chDone = origSigs, origDone })

	chSigs = make(chan os.Signal, 1)
	chDone = make(chan bool, 1)

	select {
	case <-chDone:
		t.Fatalf("the done channel must be open before a signal arrives")
	default:
	}

	chSigs <- syscall.SIGTERM
	signalHandler()

	select {
	case _, ok := <-chDone:
		if ok {
			t.Errorf("the done channel must be closed, not written to")
		}
	case <-time.After(waitLimit):
		t.Errorf("the done channel was not closed after the signal")
	}
}

func Test_runListen(t *testing.T) {
	queue := string(commonoutline.QueueNameBillingRequest)

	t.Run("queue declare failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errTestCause)

		err := runListen(mockSock, nil, nil, nil, nil)
		if !errors.Is(err, errTestCause) {
			t.Errorf("wrong error. expect the queue failure, got: %v", err)
		}
	})

	t.Run("consumer started", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		started := make(chan struct{})

		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, gomock.Any(), false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ any, _ string, _ string, _ bool, _ bool, _ bool, _ int, cb sock.CbMsgRPC) error {
				if cb == nil {
					t.Errorf("the request callback must be set")
				}
				close(started)
				return nil
			})

		if err := runListen(mockSock, nil, nil, nil, nil); err != nil {
			t.Fatalf("expected ok, got: %v", err)
		}

		select {
		case <-started:
		case <-time.After(waitLimit):
			t.Errorf("the request consumer was not started")
		}
	})
}

func Test_runSubscribe(t *testing.T) {
	tests := []struct {
		name      string
		runErr    error
		expectErr error
	}{
		{"started", nil, nil},
		{"run failure", errTestCause, errTestCause},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSub := subscribehandler.NewMockSubscribeHandler(mc)
			mockSub.EXPECT().Run().Return(tt.runErr)

			err := runSubscribe(mockSub)
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("wrong error. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

// Test_buildFailedEventHandler verifies that the subscribe handler persists an event it cannot
// process through the failed event handler that was built together with it.
func Test_buildFailedEventHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := sockhandler.NewMockSockHandler(mc)

	queue := string(commonoutline.QueueNameBillingSubscribe)
	consumer := make(chan sock.CbMsgConsume, 1)

	mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queue, gomock.Any(), string(commonoutline.QueueNameEvent), false, nil).Return(nil).AnyTimes()
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queue, string(commonoutline.ServiceNameBillingManager), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ any, _ string, _ string, _ bool, _ bool, _ bool, _ int, cb sock.CbMsgConsume) error {
			consumer <- cb
			return nil
		})

	subHandler, failedHandler := buildFailedEventHandler(mockDB, mockSock, nil, nil)
	if subHandler == nil || failedHandler == nil {
		t.Fatalf("both handlers must be built. got: %v, %v", subHandler, failedHandler)
	}

	if err := subHandler.Run(); err != nil {
		t.Fatalf("could not start the subscribe handler: %v", err)
	}

	var cb sock.CbMsgConsume
	select {
	case cb = <-consumer:
	case <-time.After(waitLimit):
		t.Fatalf("the event consumer was not started")
	}

	// the data is not a call, so the processor fails and the event must be kept for a retry
	event := &sock.Event{
		Type:      "call_progressing",
		Publisher: string(commonoutline.ServiceNameCallManager),
		DataType:  "application/json",
		Data:      json.RawMessage(`"not a call"`),
	}
	mockDB.EXPECT().FailedEventCreate(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, fe *failedevent.FailedEvent) error {
			if fe.EventType != "call_progressing" || fe.EventPublisher != string(commonoutline.ServiceNameCallManager) {
				t.Errorf("wrong failed event. type: %s, publisher: %s", fe.EventType, fe.EventPublisher)
			}
			if fe.Status != failedevent.StatusPending || fe.RetryCount != 0 {
				t.Errorf("a new failed event must be pending without retries. got: %s, %d", fe.Status, fe.RetryCount)
			}
			if fe.ErrorMessage == "" {
				t.Errorf("the processing error must be kept")
			}
			return nil
		})

	if err := cb(event); err != nil {
		t.Errorf("a saved failed event must be acknowledged. got: %v", err)
	}
}

// unreachableDSN points at a unix socket that does not exist, so a connection attempt fails at once
// without any network access.
const unreachableDSN = "unix(/nonexistent/billing-test.sock)/billing_test?timeout=1s"

// Test_runDaemon_databaseUnavailable verifies that the daemon stops with the connection failure
// before it touches the migrations, the cache, or the message broker.
func Test_runDaemon_databaseUnavailable(t *testing.T) {
	origSigs, origDone := chSigs, chDone
	savedConfig := *config.Get()
	chSigs = make(chan os.Signal, 1)
	chDone = make(chan bool, 1)
	testSigs, testDone := chSigs, chDone
	t.Cleanup(func() {
		signal.Stop(testSigs)
		chSigs, chDone = origSigs, origDone
		*config.Get() = savedConfig
	})

	*config.Get() = config.Config{DatabaseDSN: unreachableDSN}

	err := runDaemon()
	if err == nil {
		t.Fatalf("expected the connection failure, got nil")
	}

	// the signal goroutine started by the daemon must still react to its channel
	testSigs <- syscall.SIGTERM
	select {
	case _, ok := <-testDone:
		if ok {
			t.Errorf("the done channel must be closed, not written to")
		}
	case <-time.After(waitLimit):
		t.Errorf("the signal goroutine did not close the done channel")
	}
}

// Test_main_help runs the binary entry point with the help flag, which finishes without starting
// the daemon.
func Test_main_help(t *testing.T) {
	origArgs := os.Args
	origFormatter := logrus.StandardLogger().Formatter
	origLevel := logrus.GetLevel()
	t.Cleanup(func() {
		os.Args = origArgs
		logrus.SetFormatter(origFormatter)
		logrus.SetLevel(origLevel)
		viper.Reset()
	})

	os.Args = []string{"billing-manager", "--help"}
	main()
}
