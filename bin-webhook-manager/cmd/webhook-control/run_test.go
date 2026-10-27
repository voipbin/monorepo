package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"go.uber.org/mock/gomock"

	cscustomer "monorepo/bin-customer-manager/models/customer"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-webhook-manager/internal/config"
	"monorepo/bin-webhook-manager/models/account"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
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

const testBroker = "amqp-test"

// useLazyDatabase makes the database connect function return a handle that never connects: no
// query is sent through it by these tests.
func useLazyDatabase(t *testing.T) {
	t.Helper()

	restore := commondatabasehandler.SetConnectFunc(func(dsn string) (*sql.DB, error) {
		return sql.Open("mysql", "tcp(127.0.0.1:1)/test")
	})
	t.Cleanup(restore)
}

// expectHandlerInit sets the sock calls of initWebhookHandler. With failTopic the event topic
// declaration fails and nothing is expected after it.
func expectHandlerInit(mock *sockhandler.MockSockHandler, failTopic bool) {
	var errTopic error
	if failTopic {
		errTopic = fmt.Errorf("injected sock failure")
	}

	mock.EXPECT().Connect()
	mock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameWebhookEventTopic), "topic").Return(errTopic)
}

func Test_initWebhookHandler(t *testing.T) {
	tests := []struct {
		name string

		failTopic bool

		expectErr string
	}{
		{
			name:      "handler is created",
			failTopic: false,
		},
		{
			name:      "event topic exchange cannot be declared",
			failTopic: true,
			expectErr: "could not declare the topic exchange: injected sock failure",
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
			expectHandlerInit(mockSock, tt.failTopic)

			db, err := commondatabasehandler.Connect("test")
			if err != nil {
				t.Fatalf("Could not create the database handle. err: %v", err)
			}
			defer commondatabasehandler.Close(db)

			res, err := initWebhookHandler(db, mockCache)
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil handler, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil {
				t.Errorf("Wrong match. expect: a handler, got: nil")
			}
		})
	}
}

func Test_initHandler(t *testing.T) {
	t.Run("handler is created", func(t *testing.T) {
		isolatePrometheus(t)

		mr := miniredis.RunT(t)
		setConfig(t, config.Config{RedisAddress: mr.Addr(), RabbitMQAddress: testBroker})

		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		useSockHandler(t, mockSock, testBroker)
		useLazyDatabase(t)
		expectHandlerInit(mockSock, false)

		res, err := initHandler()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res == nil {
			t.Errorf("Wrong match. expect: a handler, got: nil")
		}
	})

	t.Run("cache is unreachable", func(t *testing.T) {
		setConfig(t, config.Config{RedisAddress: closedAddr(t)})
		useLazyDatabase(t)

		res, err := initHandler()
		if err == nil || !strings.Contains(err.Error(), "could not initialize the cache: could not connect to the cache") {
			t.Errorf("Wrong match. expect error containing: could not initialize the cache: could not connect to the cache, got: %v", err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil handler, got: %v", res)
		}
	})
}

// Test_runSend_success runs both send commands to the end. The customer command reads an account
// without a webhook address from the cache, the URI command uses the system customer and an
// address that URL validation rejects before any connection. The payload is a routing-keyed event,
// so the delivery is observed as one publish with the payload unchanged.
func Test_runSend_success(t *testing.T) {
	const (
		ownerID = "7c1f4e52-9d0b-11ee-8f4f-0b0e2b1c2d3e"
		eventID = "9e2a6b34-9d0b-11ee-8f4f-0b0e2b1c2d3e"
	)
	inner := `{"id":"` + eventID + `","customer_id":"` + ownerID + `"}`
	payload := `{"type":"call_updated","data":` + inner + `}`
	expectKey := "customer_id." + ownerID + ".call.call_updated." + eventID

	tests := []struct {
		name string

		run func() error
		set map[string]string

		expectOut map[string]string
	}{
		{
			name: "send to customer",
			run:  func() error { return runSendToCustomer(nil, nil) },
			set: map[string]string{
				"customer-id": ownerID,
				"data":        payload,
				"data-type":   "application/json",
			},
			expectOut: map[string]string{
				"status":      "success",
				"customer-id": ownerID,
				"data-type":   "application/json",
				"message":     "Webhook sent to customer successfully",
			},
		},
		{
			name: "send to uri",
			run:  func() error { return runSendToURI(nil, nil) },
			set: map[string]string{
				"customer-id": cscustomer.IDSystem.String(),
				"uri":         "http://127.0.0.1:1/hook",
				"method":      "PUT",
				"data":        payload,
				"data-type":   "application/json",
			},
			expectOut: map[string]string{
				"status":      "success",
				"customer-id": cscustomer.IDSystem.String(),
				"uri":         "http://127.0.0.1:1/hook",
				"method":      "PUT",
				"data-type":   "application/json",
				"message":     "Webhook sent to URI successfully",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			isolatePrometheus(t)

			mr := miniredis.RunT(t)
			setConfig(t, config.Config{RedisAddress: mr.Addr(), RabbitMQAddress: testBroker})

			seed := cachehandler.NewHandler(mr.Addr(), "", 0)
			if errSeed := seed.AccountSet(context.Background(), &account.Account{ID: uuid.FromStringOrNil(tt.set["customer-id"])}); errSeed != nil {
				t.Fatalf("Could not seed the account. err: %v", errSeed)
			}

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			useSockHandler(t, mockSock, testBroker)
			useLazyDatabase(t)
			expectHandlerInit(mockSock, false)
			mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameWebhookEventTopic), expectKey, gomock.Any()).
				DoAndReturn(func(_ string, _ string, evt *sock.Event) error {
					if evt.Type != "call_updated" || evt.Publisher != string(commonoutline.ServiceNameWebhookManager) || evt.DataType != "application/json" || string(evt.Data) != inner {
						t.Errorf("Wrong match. got event: type: %s, publisher: %s, data type: %s, data: %s", evt.Type, evt.Publisher, evt.DataType, string(evt.Data))
					}
					return nil
				})

			for k, v := range tt.set {
				viper.Set(k, v)
			}

			var err error
			out := captureStdout(t, func() { err = tt.run() })
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			res := map[string]string{}
			if errUnmarshal := json.Unmarshal([]byte(out), &res); errUnmarshal != nil {
				t.Fatalf("Wrong match. expect: json output, got: %q, err: %v", out, errUnmarshal)
			}

			if len(res) != len(tt.expectOut) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectOut, res)
			}
			for k, v := range tt.expectOut {
				if res[k] != v {
					t.Errorf("Wrong match. field: %s, expect: %s, got: %s", k, v, res[k])
				}
			}
		})
	}
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
