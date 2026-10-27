package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-message-manager/internal/config"
	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/pkg/cachehandler"
	"monorepo/bin-message-manager/pkg/dbhandler"
)

const (
	testControlWait     = 5 * time.Second
	testRabbitAddress   = "test-rabbit-address"
	testMessageID       = "5c1d2a74-0b3e-11ef-9f3d-9b1c3e0a7a01"
	testMessageCustomer = "a3d4e5f6-0b3e-11ef-9f3d-9b1c3e0a7a02"
	testMessageText     = "synthetic message text"
	testSchemaPath      = "../../scripts/database_scripts_test/table_messages.sql"
)

var testControlDBSeq atomic.Int64

// controlEnv holds the test doubles that stand in for the database, the cache and the broker.
type controlEnv struct {
	db    *sql.DB
	redis *miniredis.Miniredis
	sock  *sockhandler.MockSockHandler

	connectErr error
	rabbitURI  atomic.Value // string
}

// replaceRegisterer gives the next handler construction its own metrics registry. The request handler and the
// circuit breaker register collectors without a guard, so a second construction in one process would panic.
func replaceRegisterer(t *testing.T) {
	t.Helper()

	orig := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = orig })
}

// newControlEnv installs the doubles through the setters of the common handler. Tests that use it must not run
// in parallel, because the setters, viper and the config are process globals.
func newControlEnv(t *testing.T) *controlEnv {
	t.Helper()

	resetControlState(t)
	replaceRegisterer(t)

	mr := miniredis.RunT(t)

	name := fmt.Sprintf("file:message_control_%d?mode=memory&cache=shared", testControlDBSeq.Add(1))
	db, err := sql.Open("sqlite3", name)
	if err != nil {
		t.Fatalf("Could not open the database. err: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	schema, err := os.ReadFile(testSchemaPath)
	if err != nil {
		t.Fatalf("Could not read the schema. err: %v", err)
	}
	if _, errExec := db.Exec(string(schema)); errExec != nil {
		t.Fatalf("Could not create the schema. err: %v", errExec)
	}

	mc := gomock.NewController(t)
	env := &controlEnv{
		db:    db,
		redis: mr,
		sock:  sockhandler.NewMockSockHandler(mc),
	}

	cfg := config.Get()
	cfg.DatabaseDSN = "test-database-dsn"
	cfg.RabbitMQAddress = testRabbitAddress
	cfg.RedisAddress = mr.Addr()
	cfg.RedisPassword = ""
	cfg.RedisDatabase = 0

	restoreConnect := commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) {
		if env.connectErr != nil {
			return nil, env.connectErr
		}
		return env.db, nil
	})
	t.Cleanup(restoreConnect)

	restoreSock := sockhandler.SetRabbitFactory(func(uri string) sockhandler.SockHandler {
		env.rabbitURI.Store(uri)
		return env.sock
	})
	t.Cleanup(restoreSock)

	return env
}

// expectHandler sets the broker calls of one handler construction.
func (e *controlEnv) expectHandler() {
	e.sock.EXPECT().Connect().Times(1)
	e.sock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil).Times(1)
}

// insertMessage stores a message through the database handler, with a cache that is dropped afterwards so that a
// later read comes from the database.
func (e *controlEnv) insertMessage(t *testing.T, customerID uuid.UUID) {
	t.Helper()

	e.insertMessageID(t, uuid.FromStringOrNil(testMessageID), customerID)
}

// insertMessageID stores a message with the given id, like insertMessage.
func (e *controlEnv) insertMessageID(t *testing.T, id uuid.UUID, customerID uuid.UUID) {
	t.Helper()

	h := dbhandler.NewHandler(e.db, cachehandler.NewHandler(e.redis.Addr(), "", 0))
	m := &message.Message{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Type:         message.TypeSMS,
		Source:       &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"},
		Targets:      nil,
		ProviderName: message.ProviderNameMessagebird,
		Text:         testMessageText,
		Medias:       []string{},
		Direction:    message.DirectionOutbound,
	}
	if err := h.MessageCreate(context.Background(), m); err != nil {
		t.Fatalf("Could not insert the message. err: %v", err)
	}
	e.redis.FlushAll()
}

// listMessages runs runList once. Every run builds the handler again, so it gets its own metrics registry and broker calls.
func (e *controlEnv) listMessages(t *testing.T) []message.Message {
	t.Helper()

	replaceRegisterer(t)
	e.expectHandler()

	var err error
	out := captureStdout(t, func() { err = runList(nil, nil) })
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var res []message.Message
	decodeOutput(t, out, &res)
	return res
}

// markDeleted sets the delete timestamp of a stored message and drops the cache.
func (e *controlEnv) markDeleted(t *testing.T, id uuid.UUID) {
	t.Helper()

	if _, err := e.db.Exec("update message_messages set tm_delete = ? where id = ?", time.Now().UTC(), id.Bytes()); err != nil {
		t.Fatalf("Could not mark the message deleted. err: %v", err)
	}
	e.redis.FlushAll()
}

// decodeOutput decodes the JSON printed by a command.
func decodeOutput(t *testing.T, out string, v any) {
	t.Helper()

	if err := json.Unmarshal([]byte(out), v); err != nil {
		t.Fatalf("Wrong output. expect: json, got: %q, err: %v", out, err)
	}
}

func Test_initHandler(t *testing.T) {
	t.Run("builds the handler and connects the broker", func(t *testing.T) {
		env := newControlEnv(t)
		env.expectHandler()

		res, err := initHandler()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res == nil {
			t.Fatalf("Wrong match. expect: handler, got: nil")
		}
		if got := env.rabbitURI.Load(); got != testRabbitAddress {
			t.Errorf("Wrong broker address. expect: %s, got: %v", testRabbitAddress, got)
		}
	})

	t.Run("database connection fails", func(t *testing.T) {
		env := newControlEnv(t)
		env.connectErr = errors.New("injected failure")

		res, err := initHandler()
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
		if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
			t.Errorf("Wrong match. expect: could not connect to the database, got: %v", err)
		}
	})

	t.Run("cache connection fails", func(t *testing.T) {
		newControlEnv(t)
		config.Get().RedisAddress = "127.0.0.1:0"

		res, err := initHandler()
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
		if err == nil || !strings.Contains(err.Error(), "could not initialize the cache") {
			t.Errorf("Wrong match. expect: could not initialize the cache, got: %v", err)
		}
	})
}

func Test_runGet(t *testing.T) {
	t.Run("prints the stored message", func(t *testing.T) {
		env := newControlEnv(t)
		env.expectHandler()
		env.insertMessage(t, uuid.FromStringOrNil(testMessageCustomer))
		viper.Set("id", testMessageID)

		var err error
		out := captureStdout(t, func() { err = runGet(nil, nil) })
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		var got message.Message
		decodeOutput(t, out, &got)
		if got.ID.String() != testMessageID || got.CustomerID.String() != testMessageCustomer || got.Text != testMessageText {
			t.Errorf("Wrong message. expect: %s %s %s, got: %s %s %s", testMessageID, testMessageCustomer, testMessageText, got.ID, got.CustomerID, got.Text)
		}

		// the read filled the cache, so the cache holds the same message now.
		if keys := env.redis.Keys(); len(keys) != 1 {
			t.Errorf("Wrong cache keys. expect: 1, got: %v", keys)
		}
	})

	t.Run("message does not exist", func(t *testing.T) {
		env := newControlEnv(t)
		env.expectHandler()
		viper.Set("id", testMessageID)

		var err error
		out := captureStdout(t, func() { err = runGet(nil, nil) })
		if err == nil || !strings.Contains(err.Error(), "failed to retrieve message") {
			t.Fatalf("Wrong match. expect: failed to retrieve message, got: %v", err)
		}
		if out != "" {
			t.Errorf("Wrong match. expect: no output, got: %q", out)
		}
	})

	t.Run("id is malformed", func(t *testing.T) {
		env := newControlEnv(t)
		env.expectHandler()
		viper.Set("id", "not-a-uuid")

		err := runGet(nil, nil)
		if err == nil || !strings.Contains(err.Error(), "failed to resolve message ID") {
			t.Fatalf("Wrong match. expect: failed to resolve message ID, got: %v", err)
		}
	})
}

func Test_runList(t *testing.T) {
	t.Run("prints the messages of the customer", func(t *testing.T) {
		env := newControlEnv(t)
		env.expectHandler()
		env.insertMessage(t, uuid.FromStringOrNil(testMessageCustomer))
		viper.Set("customer-id", testMessageCustomer)
		viper.Set("limit", 10)

		var err error
		out := captureStdout(t, func() { err = runList(nil, nil) })
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		var got []message.Message
		decodeOutput(t, out, &got)
		if len(got) != 1 || got[0].ID.String() != testMessageID {
			t.Errorf("Wrong messages. expect: 1 message %s, got: %v", testMessageID, got)
		}
	})

	t.Run("other customer gets an empty list", func(t *testing.T) {
		env := newControlEnv(t)
		env.expectHandler()
		env.insertMessage(t, uuid.FromStringOrNil(testMessageCustomer))
		viper.Set("customer-id", "b4e5f6a7-0b3e-11ef-9f3d-9b1c3e0a7a03")
		viper.Set("limit", 10)

		var err error
		out := captureStdout(t, func() { err = runList(nil, nil) })
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		var got []message.Message
		decodeOutput(t, out, &got)
		if len(got) != 0 {
			t.Errorf("Wrong messages. expect: none, got: %v", got)
		}
	})

	t.Run("leaves out deleted messages and applies the limit", func(t *testing.T) {
		env := newControlEnv(t)
		customerID := uuid.FromStringOrNil(testMessageCustomer)
		idA := uuid.FromStringOrNil("c1000000-0b3e-11ef-9f3d-9b1c3e0a7a0a")
		idB := uuid.FromStringOrNil("c2000000-0b3e-11ef-9f3d-9b1c3e0a7a0b")
		idDeleted := uuid.FromStringOrNil("c3000000-0b3e-11ef-9f3d-9b1c3e0a7a0c")
		env.insertMessageID(t, idA, customerID)
		env.insertMessageID(t, idB, customerID)
		env.insertMessageID(t, idDeleted, customerID)
		env.markDeleted(t, idDeleted)
		viper.Set("customer-id", testMessageCustomer)
		viper.Set("limit", 10)

		got := env.listMessages(t)
		ids := map[string]bool{}
		for _, m := range got {
			ids[m.ID.String()] = true
		}
		if len(got) != 2 || !ids[idA.String()] || !ids[idB.String()] {
			t.Errorf("Wrong messages. expect: %s and %s only, got: %v", idA, idB, ids)
		}

		// the limit caps the number of printed messages.
		viper.Set("limit", 1)
		if got = env.listMessages(t); len(got) != 1 {
			t.Errorf("Wrong message count. expect: 1, got: %d", len(got))
		}
	})

	t.Run("token selects the messages created before it", func(t *testing.T) {
		env := newControlEnv(t)
		env.insertMessage(t, uuid.FromStringOrNil(testMessageCustomer))
		viper.Set("customer-id", testMessageCustomer)
		viper.Set("limit", 10)

		tests := []struct {
			name        string
			token       string
			expectCount int
		}{
			{name: "token before the message", token: utilhandler.TimeGetCurTimeAdd(-24 * time.Hour), expectCount: 0},
			{name: "token after the message", token: utilhandler.TimeGetCurTimeAdd(24 * time.Hour), expectCount: 1},
		}
		for _, tt := range tests {
			viper.Set("token", tt.token)
			if got := env.listMessages(t); len(got) != tt.expectCount {
				t.Errorf("Wrong message count. case: %s, expect: %d, got: %d", tt.name, tt.expectCount, len(got))
			}
		}
	})

	t.Run("database error", func(t *testing.T) {
		env := newControlEnv(t)
		env.expectHandler()
		viper.Set("customer-id", testMessageCustomer)
		viper.Set("limit", 10)
		// a closed database makes the query fail.
		_ = env.db.Close()

		err := runList(nil, nil)
		if err == nil || !strings.Contains(err.Error(), "failed to retrieve messages") {
			t.Fatalf("Wrong match. expect: failed to retrieve messages, got: %v", err)
		}
	})

	t.Run("customer id is missing", func(t *testing.T) {
		env := newControlEnv(t)
		env.expectHandler()

		err := runList(nil, nil)
		if err == nil || !strings.Contains(err.Error(), "failed to resolve customer ID") {
			t.Fatalf("Wrong match. expect: failed to resolve customer ID, got: %v", err)
		}
	})
}

func Test_runDelete(t *testing.T) {
	t.Run("marks the message deleted and publishes the deleted event", func(t *testing.T) {
		env := newControlEnv(t)
		env.expectHandler()
		// The stored message has no customer, so the webhook goroutine returns before it uses the request handler.
		// A webhook request would keep using the request handler metrics after the call returns, and the next
		// handler construction replaces them; the event goroutine only calls the mock below.
		env.insertMessage(t, uuid.Nil)
		viper.Set("id", testMessageID)

		events := make(chan *sock.Event, 1)
		env.sock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ string, _ string, evt *sock.Event) error {
				events <- evt
				return nil
			}).Times(1)

		var err error
		out := captureStdout(t, func() { err = runDelete(nil, nil) })
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		var got message.Message
		decodeOutput(t, out, &got)
		if got.ID.String() != testMessageID || got.TMDelete == nil {
			t.Errorf("Wrong message. expect: deleted %s, got: %+v", testMessageID, got)
		}

		select {
		case evt := <-events:
			if evt.Type != message.EventTypeMessageDeleted {
				t.Errorf("Wrong event type. expect: %s, got: %s", message.EventTypeMessageDeleted, evt.Type)
			}
		case <-time.After(testControlWait):
			t.Fatalf("Wrong match. expect: event published, got: timeout")
		}
	})

	t.Run("message does not exist", func(t *testing.T) {
		env := newControlEnv(t)
		env.expectHandler()
		viper.Set("id", testMessageID)

		err := runDelete(nil, nil)
		if err == nil || !strings.Contains(err.Error(), "failed to delete message") {
			t.Fatalf("Wrong match. expect: failed to delete message, got: %v", err)
		}
	})

	t.Run("id is malformed", func(t *testing.T) {
		env := newControlEnv(t)
		env.expectHandler()
		viper.Set("id", "not-a-uuid")

		err := runDelete(nil, nil)
		if err == nil || !strings.Contains(err.Error(), "failed to resolve message ID") {
			t.Fatalf("Wrong match. expect: failed to resolve message ID, got: %v", err)
		}
	})
}

func Test_runSend_sendFails(t *testing.T) {
	tests := []struct {
		name string

		billing func() (*sock.Response, error)

		expectErr string
	}{
		{
			name: "balance is insufficient",
			billing: func() (*sock.Response, error) {
				return &sock.Response{StatusCode: 200, DataType: "application/json", Data: json.RawMessage(`{"valid":false}`)}, nil
			},
			expectErr: "insufficient balance for message",
		},
		{
			name: "balance check fails",
			billing: func() (*sock.Response, error) {
				return nil, errors.New("injected failure")
			},
			expectErr: "could not validate the customer's balance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newControlEnv(t)
			env.expectHandler()

			viper.Set("customer-id", testMessageCustomer)
			viper.Set("source", "+15550100001")
			viper.Set("destinations", []string{"+15550100002"})
			viper.Set("text", testMessageText)

			var billingCalls atomic.Int32
			env.sock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, req *sock.Request) (*sock.Response, error) {
					if strings.Contains(req.URI, "is_valid_balance") {
						billingCalls.Add(1)
						return tt.billing()
					}
					// the customer lookup fails, which lets the send continue to the balance check.
					return nil, errors.New("customer service unavailable")
				}).AnyTimes()

			var err error
			out := captureStdout(t, func() { err = runSend(nil, nil) })
			if err == nil || !strings.Contains(err.Error(), "failed to send message") || !strings.Contains(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect: failed to send message: %s, got: %v", tt.expectErr, err)
			}
			if out != "" {
				t.Errorf("Wrong match. expect: no output, got: %q", out)
			}
			if got := billingCalls.Load(); got != 1 {
				t.Errorf("Wrong balance check count. expect: 1, got: %d", got)
			}

			// nothing was stored for a rejected send.
			var count int
			if errCount := env.db.QueryRow("select count(*) from message_messages").Scan(&count); errCount != nil || count != 0 {
				t.Errorf("Wrong stored count. expect: 0, got: %d, err: %v", count, errCount)
			}
		})
	}
}
