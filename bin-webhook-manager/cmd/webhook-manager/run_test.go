package main

import (
	"context"
	"database/sql"
	"fmt"
	"os/signal"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-webhook-manager/internal/config"
	"monorepo/bin-webhook-manager/models/account"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
)

const (
	consumeWait = 5 * time.Second
	testBroker  = "amqp-test"
)

// isolatePrometheus gives the test its own default registry. The request handler registers its
// collectors on every construction and would panic on the second one in a process.
func isolatePrometheus(t *testing.T) {
	t.Helper()

	orig := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = orig })
}

// useSockHandler makes the service create the given sock handler instead of a RabbitMQ one and
// checks that it is asked for the configured broker address.
func useSockHandler(t *testing.T, h sockhandler.SockHandler, expectAddr string) {
	t.Helper()

	restore := sockhandler.SetRabbitFactory(func(addr string) sockhandler.SockHandler {
		if addr != expectAddr {
			t.Errorf("Wrong match. expect broker address: %s, got: %s", expectAddr, addr)
		}
		return h
	})
	t.Cleanup(restore)
}

// useLazyDatabase makes the database connect function return a handle that never connects: no
// query is sent through it by these tests.
func useLazyDatabase(t *testing.T) {
	t.Helper()

	restore := commondatabasehandler.SetConnectFunc(func(dsn string) (*sql.DB, error) {
		return sql.Open("mysql", "tcp(127.0.0.1:1)/test")
	})
	t.Cleanup(restore)
}

// stopSignalHandling undoes what runDaemon installs and drains a left-over done signal.
func stopSignalHandling(t *testing.T) {
	t.Helper()

	drain := func() {
		select {
		case <-chDone:
		default:
		}
	}

	drain()
	t.Cleanup(func() {
		signal.Stop(chSigs)
		drain()
	})
}

// failPoint names the sock call that is made to fail. The empty value means no failure.
type failPoint string

const (
	failNone            failPoint = ""
	failEventTopic      failPoint = "event topic"
	failRequestQueue    failPoint = "request queue"
	failSubscribeQueue  failPoint = "subscribe queue"
	failGlobalExchange  failPoint = "global exchange"
	failSubscribeBind   failPoint = "subscribe bind"
	errSockCallInjected           = "injected sock failure"
)

// expectStartup sets the sock calls of run in the order of the code and returns channels that
// are closed when the consume goroutines are started. A call after the failing one is not
// expected, so gomock fails the test if the code goes on after an error.
func expectStartup(mock *sockhandler.MockSockHandler, fail failPoint) (rpcStarted, msgStarted chan struct{}) {
	rpcStarted = make(chan struct{})
	msgStarted = make(chan struct{})
	errInjected := fmt.Errorf("%s", errSockCallInjected)

	pick := func(point failPoint) error {
		if fail == point {
			return errInjected
		}
		return nil
	}

	mock.EXPECT().Connect()

	mock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameWebhookEventTopic), "topic").Return(pick(failEventTopic))
	if fail == failEventTopic {
		return rpcStarted, msgStarted
	}

	mock.EXPECT().QueueCreate(string(commonoutline.QueueNameWebhookRequest), "normal").Return(pick(failRequestQueue))
	if fail == failRequestQueue {
		return rpcStarted, msgStarted
	}

	mock.EXPECT().ConsumeRPC(gomock.Any(), string(commonoutline.QueueNameWebhookRequest), gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgRPC) error {
			close(rpcStarted)
			return nil
		})

	mock.EXPECT().QueueCreate(string(commonoutline.QueueNameWebhookSubscribe), "normal").Return(pick(failSubscribeQueue))
	if fail == failSubscribeQueue {
		return rpcStarted, msgStarted
	}

	mock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(pick(failGlobalExchange))
	if fail == failGlobalExchange {
		return rpcStarted, msgStarted
	}

	mock.EXPECT().QueueBind(string(commonoutline.QueueNameWebhookSubscribe), gomock.Any(), string(commonoutline.QueueNameEvent), false, gomock.Any()).
		Return(pick(failSubscribeBind)).MinTimes(1)
	if fail == failSubscribeBind {
		return rpcStarted, msgStarted
	}

	mock.EXPECT().ConsumeMessage(gomock.Any(), string(commonoutline.QueueNameWebhookSubscribe), gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgConsume) error {
			close(msgStarted)
			return nil
		})

	return rpcStarted, msgStarted
}

func waitStarted(t *testing.T, name string, ch chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(consumeWait):
		t.Fatalf("The %s consumer was not started", name)
	}
}

func Test_run(t *testing.T) {
	tests := []struct {
		name string

		fail failPoint

		expectErr []string
	}{
		{
			name:      "all handlers start",
			fail:      failNone,
			expectErr: nil,
		},
		{
			name:      "event topic exchange cannot be declared",
			fail:      failEventTopic,
			expectErr: []string{"could not declare the topic exchange: " + errSockCallInjected},
		},
		{
			name:      "request queue cannot be created",
			fail:      failRequestQueue,
			expectErr: []string{"could not run listen handler: could not run the listen handler correctly: could not declare the queue for listenHandler. err: " + errSockCallInjected},
		},
		{
			name:      "subscribe queue cannot be created",
			fail:      failSubscribeQueue,
			expectErr: []string{"could not run subscribe handler: could not run the subscribe handler correctly: could not declare the queue for subscribeHandler. err: " + errSockCallInjected},
		},
		{
			name:      "global event exchange cannot be declared",
			fail:      failGlobalExchange,
			expectErr: []string{"could not run subscribe handler: could not run the subscribe handler correctly: could not declare the global topic exchange. err: " + errSockCallInjected},
		},
		{
			name:      "subscribe pattern cannot be bound",
			fail:      failSubscribeBind,
			expectErr: []string{"could not bind the topic pattern. pattern: ", errSockCallInjected},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolatePrometheus(t)
			setConfig(t, config.Config{RabbitMQAddress: testBroker})

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			useSockHandler(t, mockSock, testBroker)
			useLazyDatabase(t)

			rpcStarted, msgStarted := expectStartup(mockSock, tt.fail)

			db, err := commondatabasehandler.Connect("test")
			if err != nil {
				t.Fatalf("Could not create the database handle. err: %v", err)
			}
			defer commondatabasehandler.Close(db)

			err = run(db, mockCache)
			if len(tt.expectErr) == 0 {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
			} else {
				for _, expect := range tt.expectErr {
					if err == nil || !strings.Contains(err.Error(), expect) {
						t.Errorf("Wrong match.\nexpect error containing: %s\ngot: %v", expect, err)
					}
				}
			}

			// the listen consumer starts once its queue exists, the subscribe consumer only on success.
			if tt.fail != failEventTopic && tt.fail != failRequestQueue {
				waitStarted(t, "listen", rpcStarted)
			}
			if tt.fail == failNone {
				waitStarted(t, "subscribe", msgStarted)
			}
		})
	}
}

func Test_runDaemon(t *testing.T) {
	t.Run("starts and stops on the done signal", func(t *testing.T) {
		isolatePrometheus(t)
		stopSignalHandling(t)

		mr := miniredis.RunT(t)
		setConfig(t, config.Config{RedisAddress: mr.Addr(), RabbitMQAddress: testBroker})

		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		useSockHandler(t, mockSock, testBroker)
		useLazyDatabase(t)
		rpcStarted, msgStarted := expectStartup(mockSock, failNone)

		// the done signal is sent once both consumers are started: runDaemon must wait for it.
		var signaled atomic.Bool
		go func() {
			<-rpcStarted
			<-msgStarted
			signaled.Store(true)
			chDone <- true
		}()

		if err := runDaemon(); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if !signaled.Load() {
			t.Errorf("Wrong match. expect: runDaemon returns after the done signal, got: returned before it")
		}
	})

	t.Run("cache is unreachable", func(t *testing.T) {
		stopSignalHandling(t)

		setConfig(t, config.Config{RedisAddress: closedAddr(t)})
		useLazyDatabase(t)

		err := runDaemon()
		if err == nil || !strings.Contains(err.Error(), "could not initialize the cache: cache connect error") {
			t.Errorf("Wrong match. expect error containing: could not initialize the cache: cache connect error, got: %v", err)
		}
	})

	t.Run("handlers cannot start", func(t *testing.T) {
		isolatePrometheus(t)
		stopSignalHandling(t)

		mr := miniredis.RunT(t)
		setConfig(t, config.Config{RedisAddress: mr.Addr(), RabbitMQAddress: testBroker})

		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		useSockHandler(t, mockSock, testBroker)
		useLazyDatabase(t)
		expectStartup(mockSock, failEventTopic)

		err := runDaemon()
		if err == nil || !strings.Contains(err.Error(), "could not run webhook-manager: could not declare the topic exchange: "+errSockCallInjected) {
			t.Errorf("Wrong match. expect error containing: could not run webhook-manager: could not declare the topic exchange, got: %v", err)
		}
	})
}

func Test_initCache_database(t *testing.T) {
	mr := miniredis.RunT(t)
	setConfig(t, config.Config{RedisAddress: mr.Addr(), RedisDatabase: 3})

	res, err := initCache()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	id := uuid.FromStringOrNil("5a0b3a3c-9d0b-11ee-8f4f-0b0e2b1c2d3e")
	if errSet := res.AccountSet(context.Background(), &account.Account{ID: id}); errSet != nil {
		t.Fatalf("Could not write to the cache. err: %v", errSet)
	}

	if got := len(mr.DB(3).Keys()); got != 1 {
		t.Errorf("Wrong match. expect keys in the configured database: 1, got: %d", got)
	}
	if got := len(mr.DB(0).Keys()); got != 0 {
		t.Errorf("Wrong match. expect keys in the default database: 0, got: %d", got)
	}
}
