package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"io/fs"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/smotes/purse"
	"github.com/spf13/viper"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-conversation-manager/internal/config"
)

const (
	seamCustomerA = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a01"
	seamCustomerB = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a02"

	seamAccountSMS   = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a11"
	seamAccountLine  = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a12"
	seamAccountOther = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a13"

	// rows that were deleted before the test; no listing may return them
	seamAccountDeleted      = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a14"
	seamConversationDeleted = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a24"
	seamMessageDeleted      = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a35"

	seamConversationMessage = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a21"
	seamConversationLine    = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a22"
	seamConversationOther   = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a23"

	seamMessageDone        = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a31"
	seamMessageFailed      = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a32"
	seamMessageProgressing = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a33"
	seamMessageOther       = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2a34"

	seamMissingID = "6c2e0b52-3a53-4c0e-8f2e-1f1d9f4b2aff"

	// the histogram that the request handler fills after a request is published
	seamRequestTimeMetric = "conversation_manager_request_process_time"
)

// seamEnv replaces the database, the cache and the message broker of the control commands. The
// process wide state it touches is restored by t.Cleanup, so the tests that use it must not run in
// parallel.
type seamEnv struct {
	db *sql.DB
	mr *miniredis.Miniredis

	mu       sync.Mutex
	evTypes  []string
	evKeys   []string
	requests int

	factoryCalls atomic.Int32
}

// newSeamEnv builds an isolated environment: a fresh in-memory database with the test schema and
// the seed rows, a cache server, and a mock broker that accepts only the calls that the control
// commands make.
func newSeamEnv(t *testing.T, mc *gomock.Controller) *seamEnv {
	t.Helper()

	// the global configuration is loaded once per process; a later load does not overwrite the
	// fields that this test sets
	config.LoadGlobalConfig()
	resetState(t)

	env := &seamEnv{mr: miniredis.RunT(t)}
	config.Get().RedisAddress = env.mr.Addr()
	config.Get().DatabaseDSN = "seam"

	env.db = newSeamDB(t)
	t.Cleanup(func() { _ = env.db.Close() })
	seedSeamDB(t, env.db)

	t.Cleanup(databasehandler.SetConnectFunc(func(string) (*sql.DB, error) { return env.db, nil }))

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().Connect().AnyTimes()
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), "topic").Return(nil).AnyTimes()
	mockSock.EXPECT().EventPublish(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ string, key string, evt *sock.Event) error {
			env.mu.Lock()
			defer env.mu.Unlock()
			env.evKeys = append(env.evKeys, key)
			env.evTypes = append(env.evTypes, evt.Type)
			return nil
		},
	).AnyTimes()
	mockSock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, _ *sock.Request) (*sock.Response, error) {
			env.mu.Lock()
			defer env.mu.Unlock()
			env.requests++
			return nil, nil
		},
	).AnyTimes()
	t.Cleanup(sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler {
		env.factoryCalls.Add(1)
		return mockSock
	}))

	return env
}

func newSeamDB(t *testing.T) *sql.DB {
	t.Helper()

	// every connection of ":memory:" is a separate database, so the pool is limited to one
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("could not open the database: %v", err)
	}
	db.SetMaxOpenConns(1)

	ps, err := purse.New(filepath.Join("../../scripts/database_scripts_test"))
	if err != nil {
		t.Fatalf("could not read the test schema: %v", err)
	}
	for _, name := range ps.Files() {
		contents, ok := ps.Get(name)
		if !ok {
			t.Fatalf("could not read the schema file: %s", name)
		}
		if _, errExec := db.Exec(contents); errExec != nil {
			t.Fatalf("could not apply the schema file %s: %v", name, errExec)
		}
	}

	return db
}

func seedSeamDB(t *testing.T, db *sql.DB) {
	t.Helper()

	id := func(s string) []byte { return uuid.FromStringOrNil(s).Bytes() }

	const deletedAt = "2020-01-02 00:00:00.000000"

	accounts := []struct{ id, customer, typ, name, created, deleted string }{
		{seamAccountSMS, seamCustomerA, "sms", "alpha", "2020-01-01 00:00:01.000000", ""},
		{seamAccountLine, seamCustomerA, "line", "beta", "2020-01-01 00:00:02.000000", ""},
		{seamAccountOther, seamCustomerB, "sms", "gamma", "2020-01-01 00:00:03.000000", ""},
		{seamAccountDeleted, seamCustomerA, "sms", "removed", "2020-01-01 00:00:04.000000", deletedAt},
	}
	for _, a := range accounts {
		if _, err := db.Exec(
			`insert into conversation_accounts(id, customer_id, type, name, detail, secret, token, message_flow_id, tm_create, tm_delete) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id(a.id), id(a.customer), a.typ, a.name, "detail", "fake-secret", "fake-token", uuid.Nil.Bytes(), a.created, nullIfEmpty(a.deleted),
		); err != nil {
			t.Fatalf("could not seed the account %s: %v", a.name, err)
		}
	}

	conversations := []struct{ id, customer, account, typ, created, deleted string }{
		{seamConversationMessage, seamCustomerA, seamAccountSMS, "message", "2020-01-01 00:00:01.000000", ""},
		{seamConversationLine, seamCustomerA, seamAccountLine, "line", "2020-01-01 00:00:02.000000", ""},
		{seamConversationOther, seamCustomerB, seamAccountOther, "message", "2020-01-01 00:00:03.000000", ""},
		{seamConversationDeleted, seamCustomerA, seamAccountSMS, "line", "2020-01-01 00:00:04.000000", deletedAt},
	}
	for _, c := range conversations {
		if _, err := db.Exec(
			`insert into conversation_conversations(id, customer_id, account_id, type, name, self, peer, metadata, tm_create, tm_delete) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id(c.id), id(c.customer), id(c.account), c.typ, "name", "{}", "{}", "{}", c.created, nullIfEmpty(c.deleted),
		); err != nil {
			t.Fatalf("could not seed the conversation %s: %v", c.id, err)
		}
	}

	messages := []struct{ id, customer, conversation, direction, status, created, deleted string }{
		{seamMessageDone, seamCustomerA, seamConversationMessage, "outgoing", "done", "2020-01-01 00:00:01.000000", ""},
		{seamMessageFailed, seamCustomerA, seamConversationMessage, "incoming", "failed", "2020-01-01 00:00:02.000000", ""},
		{seamMessageProgressing, seamCustomerA, seamConversationLine, "outgoing", "progressing", "2020-01-01 00:00:03.000000", ""},
		{seamMessageOther, seamCustomerB, seamConversationOther, "outgoing", "done", "2020-01-01 00:00:04.000000", ""},
		{seamMessageDeleted, seamCustomerA, seamConversationMessage, "outgoing", "done", "2020-01-01 00:00:05.000000", deletedAt},
	}
	for _, m := range messages {
		if _, err := db.Exec(
			`insert into conversation_messages(id, customer_id, conversation_id, direction, status, source, destination, medias, tm_create, tm_delete) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id(m.id), id(m.customer), id(m.conversation), m.direction, m.status, "{}", "{}", "[]", m.created, nullIfEmpty(m.deleted),
		); err != nil {
			t.Fatalf("could not seed the message %s: %v", m.id, err)
		}
	}
}

// count returns the number of rows of a query that selects one number.
func (e *seamEnv) count(t *testing.T, query string, args ...any) int {
	t.Helper()

	var n int
	if err := e.db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("could not run the query %q: %v", query, err)
	}

	return n
}

func (e *seamEnv) publishes() (events []string, keys []string, requests int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	return append([]string(nil), e.evTypes...), append([]string(nil), e.evKeys...), e.requests
}

// run executes one command with a fresh metrics registry, because every handler initialization
// registers the same collectors. When wantPublish is true it waits until the command has published
// exactly one event and one webhook request, and until the request handler has recorded the
// request time, so that no background goroutine outlives the call.
func (e *seamEnv) run(t *testing.T, wantPublish bool, fn func() error) (string, error) {
	t.Helper()

	reg := prometheus.NewRegistry()
	prevRegisterer, prevGatherer := prometheus.DefaultRegisterer, prometheus.DefaultGatherer
	prometheus.DefaultRegisterer, prometheus.DefaultGatherer = reg, reg
	defer func() { prometheus.DefaultRegisterer, prometheus.DefaultGatherer = prevRegisterer, prevGatherer }()

	events0, _, requests0 := e.publishes()

	var err error
	out := captureStdout(t, func() { err = fn() })

	events1, _, requests1 := e.publishes()
	if !wantPublish {
		if len(events1) != len(events0) || requests1 != requests0 {
			t.Errorf("the command published unexpectedly. events: %d -> %d, requests: %d -> %d", len(events0), len(events1), requests0, requests1)
		}
		return out, err
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		events, _, requests := e.publishes()
		if len(events) == len(events0)+1 && requests == requests0+1 && seamRequestTimeObserved(reg) {
			return out, err
		}
		if time.Now().After(deadline) {
			t.Fatalf("the publishes did not finish. events: %d -> %d, requests: %d -> %d", len(events0), len(events), requests0, requests)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func seamRequestTimeObserved(reg *prometheus.Registry) bool {
	families, err := reg.Gather()
	if err != nil {
		return false
	}
	for _, family := range families {
		if family.GetName() != seamRequestTimeMetric {
			continue
		}
		for _, metric := range family.GetMetric() {
			if metric.GetHistogram().GetSampleCount() >= 1 {
				return true
			}
		}
	}

	return false
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}

	return s
}

func seamIDs(t *testing.T, out string) []string {
	t.Helper()

	var items []map[string]any
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		t.Fatalf("the output is not a JSON list: %v. output: %q", err, out)
	}

	res := []string{}
	for _, item := range items {
		res = append(res, item["id"].(string))
	}
	sort.Strings(res)

	return res
}

func seamObject(t *testing.T, out string) map[string]any {
	t.Helper()

	var item map[string]any
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		t.Fatalf("the output is not a JSON object: %v. output: %q", err, out)
	}

	return item
}

func seamSorted(ids ...string) []string {
	res := append([]string{}, ids...)
	sort.Strings(res)

	return res
}

func seamSetFlags(flags map[string]any) {
	viper.Reset()
	for k, v := range flags {
		viper.Set(k, v)
	}
}

func Test_runConversationGet(t *testing.T) {
	tests := []struct {
		name string

		flags map[string]any

		expectErr  string
		expectType string
	}{
		{name: "found", flags: map[string]any{"id": seamConversationLine}, expectType: "line"},
		{name: "not found", flags: map[string]any{"id": seamMissingID}, expectErr: "failed to retrieve conversation: "},
		{name: "invalid id", flags: map[string]any{"id": "x"}, expectErr: "failed to resolve conversation ID: invalid format for Conversation ID: 'x' is not a valid UUID"},
		{name: "missing id", flags: map[string]any{}, expectErr: "failed to resolve conversation ID: Conversation ID is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			env := newSeamEnv(t, mc)
			seamSetFlags(tt.flags)

			out, err := env.run(t, false, func() error { return runConversationGet(nil, nil) })
			if tt.expectErr != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect: error prefix %q, got: %v", tt.expectErr, err)
				}
				if out != "" {
					t.Errorf("Wrong match. expect: no output, got: %q", out)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			item := seamObject(t, out)
			if item["id"] != tt.flags["id"] || item["type"] != tt.expectType || item["customer_id"] != seamCustomerA {
				t.Errorf("Wrong match. expect: id %v type %v customer %v, got: %v", tt.flags["id"], tt.expectType, seamCustomerA, item)
			}
		})
	}
}

func Test_runConversationList(t *testing.T) {
	tests := []struct {
		name string

		flags map[string]any

		expectErr string
		expectIDs []string
	}{
		{
			name:      "all types of the customer",
			flags:     map[string]any{"customer-id": seamCustomerA, "limit": 10},
			expectIDs: seamSorted(seamConversationMessage, seamConversationLine),
		},
		{
			name:      "type filter keeps only the matching type",
			flags:     map[string]any{"customer-id": seamCustomerA, "limit": 10, "type": "line"},
			expectIDs: seamSorted(seamConversationLine),
		},
		{
			name:      "the limit is applied",
			flags:     map[string]any{"customer-id": seamCustomerA, "limit": 1},
			expectIDs: seamSorted(seamConversationLine),
		},
		{
			name:      "another customer sees only its own rows",
			flags:     map[string]any{"customer-id": seamCustomerB, "limit": 10},
			expectIDs: seamSorted(seamConversationOther),
		},
		{
			name:      "no row matches",
			flags:     map[string]any{"customer-id": seamCustomerA, "limit": 10, "type": "whatsapp"},
			expectIDs: []string{},
		},
		{name: "customer id is required", flags: map[string]any{}, expectErr: "failed to resolve customer ID: Customer ID is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			env := newSeamEnv(t, mc)
			seamSetFlags(tt.flags)

			out, err := env.run(t, false, func() error { return runConversationList(nil, nil) })
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if got := seamIDs(t, out); !reflect.DeepEqual(got, tt.expectIDs) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectIDs, got)
			}
		})
	}
}

func Test_runAccountCreate(t *testing.T) {
	valid := func(overrides map[string]any) map[string]any {
		res := map[string]any{"customer-id": seamCustomerA, "type": "sms", "name": "created", "secret": "fake-secret-1", "token": "fake-token-1"}
		for k, v := range overrides {
			res[k] = v
		}
		return res
	}

	tests := []struct {
		name string

		flags map[string]any

		expectErr string
	}{
		{name: "sms account", flags: valid(nil)},
		{name: "customer id is required", flags: valid(map[string]any{"customer-id": ""}), expectErr: "failed to resolve customer ID: Customer ID is required"},
		{name: "invalid customer id", flags: valid(map[string]any{"customer-id": "x"}), expectErr: "failed to resolve customer ID: invalid format for Customer ID: 'x' is not a valid UUID"},
		{name: "type is required", flags: valid(map[string]any{"type": ""}), expectErr: "type is required"},
		{name: "secret is required", flags: valid(map[string]any{"secret": ""}), expectErr: "secret is required"},
		{name: "token is required", flags: valid(map[string]any{"token": ""}), expectErr: "token is required"},
		{name: "unsupported type", flags: valid(map[string]any{"type": "fax"}), expectErr: "failed to create account: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			env := newSeamEnv(t, mc)
			seamSetFlags(tt.flags)
			before := env.count(t, `select count(*) from conversation_accounts`)

			wantPublish := tt.expectErr == ""
			out, err := env.run(t, wantPublish, func() error { return runAccountCreate(nil, nil) })

			after := env.count(t, `select count(*) from conversation_accounts`)
			if tt.expectErr != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect: error prefix %q, got: %v", tt.expectErr, err)
				}
				if after != before {
					t.Errorf("Wrong match. expect: %d rows, got: %d", before, after)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if after != before+1 {
				t.Errorf("Wrong match. expect: %d rows, got: %d", before+1, after)
			}

			item := seamObject(t, out)
			if item["customer_id"] != seamCustomerA || item["type"] != "sms" || item["name"] != "created" {
				t.Errorf("Wrong match. expect: customer %s, type sms, name created, got: %v", seamCustomerA, item)
			}

			var secret, token string
			if errScan := env.db.QueryRow(`select secret, token from conversation_accounts where id = ?`, uuid.FromStringOrNil(item["id"].(string)).Bytes()).Scan(&secret, &token); errScan != nil {
				t.Fatalf("could not read the created account: %v", errScan)
			}
			if secret != "fake-secret-1" || token != "fake-token-1" {
				t.Errorf("Wrong match. expect: the given secret and token to be stored, got: %q %q", secret, token)
			}

			events, keys, _ := env.publishes()
			if len(events) != 1 || !strings.HasSuffix(events[0], "created") || !strings.Contains(keys[0], item["id"].(string)) {
				t.Errorf("Wrong match. expect: one created event for %v, got: %v %v", item["id"], events, keys)
			}
		})
	}
}

func Test_runAccountGet(t *testing.T) {
	tests := []struct {
		name string

		flags map[string]any

		expectErr  string
		expectName string
	}{
		{name: "found", flags: map[string]any{"id": seamAccountLine}, expectName: "beta"},
		{name: "not found", flags: map[string]any{"id": seamMissingID}, expectErr: "failed to retrieve account: "},
		{name: "missing id", flags: map[string]any{}, expectErr: "failed to resolve account ID: Account ID is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			env := newSeamEnv(t, mc)
			seamSetFlags(tt.flags)

			out, err := env.run(t, false, func() error { return runAccountGet(nil, nil) })
			if tt.expectErr != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect: error prefix %q, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if item := seamObject(t, out); item["name"] != tt.expectName || item["id"] != tt.flags["id"] {
				t.Errorf("Wrong match. expect: name %s, got: %v", tt.expectName, item)
			}
		})
	}
}

func Test_runAccountList(t *testing.T) {
	tests := []struct {
		name string

		flags map[string]any

		expectErr string
		expectIDs []string
	}{
		{name: "all types of the customer", flags: map[string]any{"customer-id": seamCustomerA, "limit": 10}, expectIDs: seamSorted(seamAccountSMS, seamAccountLine)},
		{name: "type filter", flags: map[string]any{"customer-id": seamCustomerA, "limit": 10, "type": "sms"}, expectIDs: seamSorted(seamAccountSMS)},
		{name: "the limit is applied", flags: map[string]any{"customer-id": seamCustomerA, "limit": 1}, expectIDs: seamSorted(seamAccountLine)},
		{name: "another customer", flags: map[string]any{"customer-id": seamCustomerB, "limit": 10}, expectIDs: seamSorted(seamAccountOther)},
		{name: "customer id is required", flags: map[string]any{}, expectErr: "failed to resolve customer ID: Customer ID is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			env := newSeamEnv(t, mc)
			seamSetFlags(tt.flags)

			out, err := env.run(t, false, func() error { return runAccountList(nil, nil) })
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if got := seamIDs(t, out); !reflect.DeepEqual(got, tt.expectIDs) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectIDs, got)
			}
		})
	}
}

func Test_runAccountUpdate(t *testing.T) {
	// the stored values of the seeded account when nothing is changed
	unchanged := [4]string{"alpha", "detail", "fake-secret", "fake-token"} // name, detail, secret, token

	tests := []struct {
		name string

		flags map[string]any

		expectErr     string
		expectStored  [4]string
		expectPublish bool
	}{
		{name: "name only", flags: map[string]any{"id": seamAccountSMS, "name": "renamed"}, expectStored: [4]string{"renamed", "detail", "fake-secret", "fake-token"}, expectPublish: true},
		{name: "detail only", flags: map[string]any{"id": seamAccountSMS, "detail": "new detail"}, expectStored: [4]string{"alpha", "new detail", "fake-secret", "fake-token"}, expectPublish: true},
		{name: "secret only", flags: map[string]any{"id": seamAccountSMS, "secret": "fake-secret-2"}, expectStored: [4]string{"alpha", "detail", "fake-secret-2", "fake-token"}, expectPublish: true},
		{name: "token only", flags: map[string]any{"id": seamAccountSMS, "token": "fake-token-2"}, expectStored: [4]string{"alpha", "detail", "fake-secret", "fake-token-2"}, expectPublish: true},
		{name: "no field to update", flags: map[string]any{"id": seamAccountSMS}, expectErr: "at least one field to update is required (name, detail, secret, token)", expectStored: unchanged},
		{name: "missing id", flags: map[string]any{"name": "renamed"}, expectErr: "failed to resolve account ID: Account ID is required", expectStored: unchanged},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			env := newSeamEnv(t, mc)
			seamSetFlags(tt.flags)

			out, err := env.run(t, tt.expectPublish, func() error { return runAccountUpdate(nil, nil) })
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			} else {
				if item := seamObject(t, out); item["name"] != tt.expectStored[0] || item["detail"] != tt.expectStored[1] {
					t.Errorf("Wrong match. expect: name %s detail %s, got: %v", tt.expectStored[0], tt.expectStored[1], item)
				}
				events, keys, _ := env.publishes()
				if len(events) != 1 || !strings.HasSuffix(events[0], "updated") || !strings.Contains(keys[0], seamAccountSMS) {
					t.Errorf("Wrong match. expect: one updated event for the account, got: %v %v", events, keys)
				}
			}

			var stored [4]string
			if errScan := env.db.QueryRow(`select name, detail, secret, token from conversation_accounts where id = ?`, uuid.FromStringOrNil(seamAccountSMS).Bytes()).Scan(&stored[0], &stored[1], &stored[2], &stored[3]); errScan != nil {
				t.Fatalf("could not read the account: %v", errScan)
			}
			if stored != tt.expectStored {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStored, stored)
			}
		})
	}
}

func Test_runAccountDelete(t *testing.T) {
	tests := []struct {
		name string

		flags map[string]any

		expectErr     string
		expectDeleted int
	}{
		{name: "delete", flags: map[string]any{"id": seamAccountSMS}, expectDeleted: 1},
		{name: "not found", flags: map[string]any{"id": seamMissingID}, expectErr: "failed to delete account: "},
		{name: "invalid id", flags: map[string]any{"id": "x"}, expectErr: "failed to resolve account ID: invalid format for Account ID: 'x' is not a valid UUID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			env := newSeamEnv(t, mc)
			seamSetFlags(tt.flags)

			deletedBefore := env.count(t, `select count(*) from conversation_accounts where tm_delete is not null`)
			out, err := env.run(t, tt.expectErr == "", func() error { return runAccountDelete(nil, nil) })
			deleted := env.count(t, `select count(*) from conversation_accounts where tm_delete is not null`) - deletedBefore
			if deleted != tt.expectDeleted {
				t.Errorf("Wrong match. expect: %d newly deleted rows, got: %d", tt.expectDeleted, deleted)
			}
			if tt.expectErr != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect: error prefix %q, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			item := seamObject(t, out)
			if item["id"] != seamAccountSMS || item["tm_delete"] == nil {
				t.Errorf("Wrong match. expect: the deleted account with a delete time, got: %v", item)
			}
			events, keys, _ := env.publishes()
			if len(events) != 1 || !strings.HasSuffix(events[0], "deleted") || !strings.Contains(keys[0], seamAccountSMS) {
				t.Errorf("Wrong match. expect: one deleted event for the account, got: %v %v", events, keys)
			}
		})
	}
}

func Test_runMessageGet(t *testing.T) {
	tests := []struct {
		name string

		flags map[string]any

		expectErr    string
		expectStatus string
	}{
		{name: "found", flags: map[string]any{"id": seamMessageFailed}, expectStatus: "failed"},
		{name: "not found", flags: map[string]any{"id": seamMissingID}, expectErr: "failed to retrieve message: "},
		{name: "missing id", flags: map[string]any{}, expectErr: "failed to resolve message ID: Message ID is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			env := newSeamEnv(t, mc)
			seamSetFlags(tt.flags)

			out, err := env.run(t, false, func() error { return runMessageGet(nil, nil) })
			if tt.expectErr != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect: error prefix %q, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if item := seamObject(t, out); item["status"] != tt.expectStatus || item["direction"] != "incoming" {
				t.Errorf("Wrong match. expect: status %s direction incoming, got: %v", tt.expectStatus, item)
			}
		})
	}
}

func Test_runMessageList(t *testing.T) {
	tests := []struct {
		name string

		flags map[string]any

		expectErr string
		expectIDs []string
	}{
		{name: "all messages of the customer", flags: map[string]any{"customer-id": seamCustomerA, "limit": 10}, expectIDs: seamSorted(seamMessageDone, seamMessageFailed, seamMessageProgressing)},
		{name: "the limit is applied", flags: map[string]any{"customer-id": seamCustomerA, "limit": 1}, expectIDs: seamSorted(seamMessageProgressing)},
		{name: "conversation filter", flags: map[string]any{"customer-id": seamCustomerA, "limit": 10, "conversation-id": seamConversationMessage}, expectIDs: seamSorted(seamMessageDone, seamMessageFailed)},
		{name: "direction filter", flags: map[string]any{"customer-id": seamCustomerA, "limit": 10, "direction": "outgoing"}, expectIDs: seamSorted(seamMessageDone, seamMessageProgressing)},
		{name: "status filter", flags: map[string]any{"customer-id": seamCustomerA, "limit": 10, "status": "failed"}, expectIDs: seamSorted(seamMessageFailed)},
		{name: "all filters together", flags: map[string]any{"customer-id": seamCustomerA, "limit": 10, "conversation-id": seamConversationMessage, "direction": "outgoing", "status": "done"}, expectIDs: seamSorted(seamMessageDone)},
		{name: "no row matches", flags: map[string]any{"customer-id": seamCustomerA, "limit": 10, "conversation-id": seamConversationLine, "status": "failed"}, expectIDs: []string{}},
		{name: "another customer", flags: map[string]any{"customer-id": seamCustomerB, "limit": 10}, expectIDs: seamSorted(seamMessageOther)},
		{name: "invalid conversation id", flags: map[string]any{"customer-id": seamCustomerA, "conversation-id": "x"}, expectErr: "invalid conversation-id format"},
		{name: "customer id is required", flags: map[string]any{}, expectErr: "failed to resolve customer ID: Customer ID is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			env := newSeamEnv(t, mc)
			seamSetFlags(tt.flags)

			out, err := env.run(t, false, func() error { return runMessageList(nil, nil) })
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if got := seamIDs(t, out); !reflect.DeepEqual(got, tt.expectIDs) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectIDs, got)
			}
		})
	}
}

func Test_runMessageDelete(t *testing.T) {
	tests := []struct {
		name string

		flags map[string]any

		expectErr     string
		expectDeleted int
	}{
		{name: "delete", flags: map[string]any{"id": seamMessageDone}, expectDeleted: 1},
		{name: "not found", flags: map[string]any{"id": seamMissingID}, expectErr: "failed to delete message: "},
		{name: "missing id", flags: map[string]any{}, expectErr: "failed to resolve message ID: Message ID is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			env := newSeamEnv(t, mc)
			seamSetFlags(tt.flags)

			deletedBefore := env.count(t, `select count(*) from conversation_messages where tm_delete is not null`)
			out, err := env.run(t, tt.expectErr == "", func() error { return runMessageDelete(nil, nil) })
			deleted := env.count(t, `select count(*) from conversation_messages where tm_delete is not null`) - deletedBefore
			if deleted != tt.expectDeleted {
				t.Errorf("Wrong match. expect: %d newly deleted rows, got: %d", tt.expectDeleted, deleted)
			}
			if tt.expectErr != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect: error prefix %q, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if item := seamObject(t, out); item["id"] != seamMessageDone || item["tm_delete"] == nil {
				t.Errorf("Wrong match. expect: the deleted message with a delete time, got: %v", item)
			}
			events, keys, _ := env.publishes()
			// the message event is published on the topic of its conversation
			if len(events) != 1 || !strings.HasSuffix(events[0], "message_deleted") || !strings.Contains(keys[0], seamConversationMessage) {
				t.Errorf("Wrong match. expect: one message deleted event for the conversation, got: %v %v", events, keys)
			}
		})
	}
}

func Test_initHandlers_cacheFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	env := newSeamEnv(t, mc)

	// the database connects, then the stopped cache fails before the broker is touched
	env.mr.Close()

	var err error
	_, err = env.run(t, false, func() error {
		_, _, _, errInit := initHandlers()
		return errInit
	})
	if err == nil || !strings.HasPrefix(err.Error(), "could not initialize the cache: could not connect to the cache") {
		t.Errorf("Wrong match. expect: the cache context, got: %v", err)
	}
	if n := env.factoryCalls.Load(); n != 0 {
		t.Errorf("Wrong match. expect: no broker handler created, got: %d", n)
	}
}

func Test_initCommand_execute(t *testing.T) {
	t.Run("a subcommand runs through the root command", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		env := newSeamEnv(t, mc)

		root := initCommand()
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		root.SetArgs([]string{"account", "get", "--id", seamAccountLine})

		out, err := env.run(t, false, func() error { return root.Execute() })
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if item := seamObject(t, out); item["id"] != seamAccountLine || item["name"] != "beta" {
			t.Errorf("Wrong match. expect: the line account, got: %v", item)
		}
	})

	t.Run("a failing subcommand returns its error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		env := newSeamEnv(t, mc)

		root := initCommand()
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		root.SetArgs([]string{"message", "get"})

		_, err := env.run(t, false, func() error { return root.Execute() })
		if err == nil || err.Error() != "failed to resolve message ID: Message ID is required" {
			t.Errorf("Wrong match. expect: the missing id error, got: %v", err)
		}
	})

	t.Run("migrate status opens only the database through the engine", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		env := newSeamEnv(t, mc)

		eng := &seamEngine{version: 3}
		t.Cleanup(migrationhandler.SetEngineFunc(func(dsn string, _ fs.FS, table string) (migrationhandler.Engine, uint, error) {
			eng.dsn, eng.table = dsn, table
			return eng, 3, nil
		}))

		root := initCommand()
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		root.SetArgs([]string{"migrate", "status"})

		out, err := env.run(t, false, func() error { return root.Execute() })
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if !strings.Contains(out, "3") {
			t.Errorf("Wrong match. expect: the current version in the output, got: %q", out)
		}
		if eng.dsn != "seam" || eng.table != "conversation_schema_migrations" || eng.closed != 1 {
			t.Errorf("Wrong match. expect: dsn seam, table conversation_schema_migrations, one close, got: %+v", eng)
		}
		if n := env.factoryCalls.Load(); n != 0 {
			t.Errorf("Wrong match. expect: no broker handler created, got: %d", n)
		}
	})
}

// seamEngine is a fake migration engine that records how it is opened and closed.
type seamEngine struct {
	version uint
	dsn     string
	table   string
	closed  int
}

func (e *seamEngine) Version() (uint, bool, error) { return e.version, false, nil }
func (e *seamEngine) Up() error                    { return nil }
func (e *seamEngine) Steps(int) error              { return nil }
func (e *seamEngine) Force(int) error              { return nil }
func (e *seamEngine) Close() (error, error)        { e.closed++; return nil, nil }
