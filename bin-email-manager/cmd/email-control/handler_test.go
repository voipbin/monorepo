package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-email-manager/internal/config"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/cachehandler"
	"monorepo/bin-email-manager/pkg/dbhandler"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"go.uber.org/mock/gomock"

	_ "github.com/mattn/go-sqlite3"
)

const (
	testCustomerA = "5e8f0f0a-eeee-4eee-8eee-00000000000a"
	testCustomerB = "5e8f0f0a-eeee-4eee-8eee-00000000000b"
	testEmailA1   = "5e8f0f0a-eeee-4eee-8eee-0000000000a1"
	testEmailA2   = "5e8f0f0a-eeee-4eee-8eee-0000000000a2"
	testEmailB1   = "5e8f0f0a-eeee-4eee-8eee-0000000000b1"
	testMissingID = "5e8f0f0a-eeee-4eee-8eee-0000000000ff"
)

// controlEnv holds the fake backends that the control commands run against.
type controlEnv struct {
	db     *sql.DB
	sock   *sockhandler.MockSockHandler
	mr     *miniredis.Miniredis
	dbDSN  string
	rabbit string

	// number of times the broker connection was started
	connects int

	// closed when the asynchronous notifications of a deletion were attempted
	published sync.WaitGroup
}

// setupControlEnv replaces the database connection and the RabbitMQ connection with an in-memory
// database and a mock, points the cache at an in-memory redis, and restores everything on cleanup.
// These seams are process-global, so the callers must not run in parallel.
func setupControlEnv(t *testing.T) *controlEnv {
	t.Helper()

	resetViper(t)

	saved := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = saved })

	env := &controlEnv{mr: miniredis.RunT(t)}

	cfg := config.Get()
	savedCfg := *cfg
	t.Cleanup(func() { *cfg = savedCfg })
	cfg.DatabaseDSN = "synthetic-dsn"
	cfg.RabbitMQAddress = "synthetic-broker"
	cfg.RedisAddress = env.mr.Addr()
	cfg.RedisPassword = ""
	cfg.RedisDatabase = 0

	// the database name is unique per test, so the shared in-memory database never leaks
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", name))
	if err != nil {
		t.Fatalf("could not open the database. err: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	schema, err := os.ReadFile("../../scripts/database_scripts/table_email_emails.sql")
	if err != nil {
		t.Fatalf("could not read the schema. err: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("could not create the schema. err: %v", err)
	}
	env.db = db

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	env.sock = sockhandler.NewMockSockHandler(mc)
	env.sock.EXPECT().Connect().Do(func() { env.connects++ }).AnyTimes()
	env.sock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	restoreConnect := commondatabasehandler.SetConnectFunc(func(dsn string) (*sql.DB, error) {
		env.dbDSN = dsn
		return db, nil
	})
	t.Cleanup(restoreConnect)

	restoreRabbit := sockhandler.SetRabbitFactory(func(uri string) sockhandler.SockHandler {
		env.rabbit = uri
		return env.sock
	})
	t.Cleanup(restoreRabbit)

	return env
}

// seed stores an email through the real database handler.
func (e *controlEnv) seed(t *testing.T, id, customerID, subject string) {
	t.Helper()

	cache := cachehandler.NewHandler(e.mr.Addr(), "", 0)
	if err := cache.Connect(); err != nil {
		t.Fatalf("could not connect the cache. err: %v", err)
	}

	err := dbhandler.NewHandler(e.db, cache).EmailCreate(context.Background(), &email.Email{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil(id),
			CustomerID: uuid.FromStringOrNil(customerID),
		},
		ProviderType: email.ProviderTypeSendgrid,
		Source:       &commonaddress.Address{Type: commonaddress.TypeEmail, Target: "source@example.com"},
		Destinations: []commonaddress.Address{{Type: commonaddress.TypeEmail, Target: "dest@example.com"}},
		Status:       email.StatusInitiated,
		Subject:      subject,
		Content:      "synthetic content",
	})
	if err != nil {
		t.Fatalf("could not seed the email. err: %v", err)
	}
}

// expectDeleteNotifications lets the asynchronous notifications of a deletion through and
// records their completion, so that nothing runs after the test has finished.
func (e *controlEnv) expectDeleteNotifications() {
	e.published.Add(2)

	e.sock.EXPECT().EventPublish(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(string, string, *sock.Event) error {
			e.published.Done()
			return nil
		})
	e.sock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, string, *sock.Request) (*sock.Response, error) {
			e.published.Done()
			return nil, fmt.Errorf("synthetic webhook failure")
		})
}

func (e *controlEnv) waitPublished(t *testing.T) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		e.published.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("The deletion notifications were not attempted")
	}
}

// runOutput runs f and returns what it wrote to the standard output.
func runOutput(t *testing.T, f func() error) (string, error) {
	t.Helper()

	var err error
	out := captureStdout(t, func() { err = f() })

	return out, err
}

func Test_initHandler_success(t *testing.T) {
	env := setupControlEnv(t)

	handler, err := initHandler()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if handler == nil {
		t.Fatalf("Expected an email handler. got nil")
	}

	if env.dbDSN != "synthetic-dsn" {
		t.Errorf("Wrong database dsn. expect: synthetic-dsn, got: %q", env.dbDSN)
	}
	if env.rabbit != "synthetic-broker" {
		t.Errorf("Wrong broker address. expect: synthetic-broker, got: %q", env.rabbit)
	}
	if env.connects != 1 {
		t.Errorf("Wrong broker connect count. expect: 1, got: %d", env.connects)
	}
}

func Test_initHandler_connectFails(t *testing.T) {
	env := setupControlEnv(t)

	restore := commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) {
		return nil, fmt.Errorf("synthetic connect failure")
	})
	defer restore()

	handler, err := initHandler()
	if err == nil || err.Error() != "could not connect to the database: synthetic connect failure" {
		t.Errorf("Wrong match. got: %v", err)
	}
	if handler != nil {
		t.Errorf("Expected a nil handler. got: %v", handler)
	}
	if env.rabbit != "" {
		t.Errorf("Expected no broker connection after a database failure. got: %q", env.rabbit)
	}
}

func Test_initHandler_cacheFails(t *testing.T) {
	env := setupControlEnv(t)
	env.mr.Close()

	handler, err := initHandler()
	if err == nil || !strings.HasPrefix(err.Error(), "could not initialize the cache: could not connect to the cache: ") {
		t.Errorf("Wrong match. expect a cache error, got: %v", err)
	}
	if handler != nil {
		t.Errorf("Expected a nil handler. got: %v", handler)
	}
	if env.rabbit != "" {
		t.Errorf("Expected no broker connection after a cache failure. got: %q", env.rabbit)
	}
}

func Test_runGet(t *testing.T) {
	tests := []struct {
		name string

		id string

		expectErr     string
		expectSubject string
	}{
		{
			name:          "found",
			id:            testEmailA1,
			expectSubject: "subject a1",
		},
		{
			name:      "missing_id",
			id:        "",
			expectErr: "failed to resolve email ID: Email ID is required",
		},
		{
			name:      "malformed_id",
			id:        "not-a-uuid",
			expectErr: "failed to resolve email ID: invalid format for Email ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:      "not_found",
			id:        testMissingID,
			expectErr: "failed to retrieve email: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupControlEnv(t)
			env.seed(t, testEmailA1, testCustomerA, "subject a1")
			env.seed(t, testEmailB1, testCustomerB, "subject b1")

			viper.Set("id", tt.id)

			out, err := runOutput(t, func() error { return runGet(cmdGet(), nil) })

			if tt.expectErr != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect error: %q, got: %v", tt.expectErr, err)
				}
				if out != "" {
					t.Errorf("Expected no output on failure. got: %q", out)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			var res email.Email
			if errUnmarshal := json.Unmarshal([]byte(out), &res); errUnmarshal != nil {
				t.Fatalf("Output is not a json email. out: %q, err: %v", out, errUnmarshal)
			}
			if res.ID.String() != tt.id || res.Subject != tt.expectSubject || res.CustomerID.String() != testCustomerA {
				t.Errorf("Wrong email. id: %s, subject: %q, customer: %s", res.ID, res.Subject, res.CustomerID)
			}
		})
	}
}

func Test_runList(t *testing.T) {
	tests := []struct {
		name string

		customerID string
		limit      int
		deleteID   string

		expectErr string
		expectIDs []string
	}{
		{
			name:       "all_of_the_customer_only",
			customerID: testCustomerA,
			limit:      10,
			expectIDs:  []string{testEmailA2, testEmailA1},
		},
		{
			name:       "limit_applies",
			customerID: testCustomerA,
			limit:      1,
			expectIDs:  []string{testEmailA2},
		},
		{
			name:       "deleted_are_excluded",
			customerID: testCustomerA,
			limit:      10,
			deleteID:   testEmailA2,
			expectIDs:  []string{testEmailA1},
		},
		{
			name:       "other_customer",
			customerID: testCustomerB,
			limit:      10,
			expectIDs:  []string{testEmailB1},
		},
		{
			name:       "customer_without_emails",
			customerID: testMissingID,
			limit:      10,
			expectIDs:  []string{},
		},
		{
			name:       "missing_customer",
			customerID: "",
			limit:      10,
			expectErr:  "failed to resolve customer ID: Customer ID is required",
		},
		{
			name:       "malformed_customer",
			customerID: "not-a-uuid",
			limit:      10,
			expectErr:  "failed to resolve customer ID: invalid format for Customer ID: 'not-a-uuid' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupControlEnv(t)

			env.seed(t, testEmailA1, testCustomerA, "subject a1")
			env.seed(t, testEmailA2, testCustomerA, "subject a2")
			env.seed(t, testEmailB1, testCustomerB, "subject b1")

			// fixed creation times, so the order does not depend on the clock: the newest comes first
			for id, created := range map[string]string{
				testEmailA1: "2030-01-01 00:00:01.000000",
				testEmailA2: "2030-01-01 00:00:02.000000",
				testEmailB1: "2030-01-01 00:00:03.000000",
			} {
				if _, err := env.db.Exec("update email_emails set tm_create = ? where id = ?", created, uuid.FromStringOrNil(id).Bytes()); err != nil {
					t.Fatalf("could not set the creation time. err: %v", err)
				}
			}

			if tt.deleteID != "" {
				if _, err := env.db.Exec("update email_emails set tm_delete = '2030-01-01 00:00:00.000000' where id = ?", uuid.FromStringOrNil(tt.deleteID).Bytes()); err != nil {
					t.Fatalf("could not mark the email deleted. err: %v", err)
				}
			}

			viper.Set("customer-id", tt.customerID)
			viper.Set("limit", tt.limit)
			viper.Set("token", "2031-01-01 00:00:00.000000")

			out, err := runOutput(t, func() error { return runList(cmdList(), nil) })

			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong match. expect error: %q, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			var res []*email.Email
			if errUnmarshal := json.Unmarshal([]byte(out), &res); errUnmarshal != nil {
				t.Fatalf("Output is not a json list. out: %q, err: %v", out, errUnmarshal)
			}

			ids := make([]string, 0, len(res))
			for _, r := range res {
				ids = append(ids, r.ID.String())
			}
			if strings.Join(ids, ",") != strings.Join(tt.expectIDs, ",") {
				t.Errorf("Wrong emails. expect: %v, got: %v", tt.expectIDs, ids)
			}
		})
	}
}

func Test_runList_databaseFails(t *testing.T) {
	env := setupControlEnv(t)
	if _, err := env.db.Exec("drop table email_emails"); err != nil {
		t.Fatalf("could not drop the table. err: %v", err)
	}

	viper.Set("customer-id", testCustomerA)
	viper.Set("limit", 10)

	out, err := runOutput(t, func() error { return runList(cmdList(), nil) })
	if err == nil || !strings.HasPrefix(err.Error(), "failed to retrieve emails: ") {
		t.Errorf("Wrong match. expect a retrieve error, got: %v", err)
	}
	if out != "" {
		t.Errorf("Expected no output on failure. got: %q", out)
	}
}

func Test_runDelete(t *testing.T) {
	env := setupControlEnv(t)
	env.seed(t, testEmailA1, testCustomerA, "subject a1")
	env.seed(t, testEmailA2, testCustomerA, "subject a2")
	env.expectDeleteNotifications()

	viper.Set("id", testEmailA1)

	out, err := runOutput(t, func() error { return runDelete(cmdDelete(), nil) })
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	env.waitPublished(t)

	var res email.Email
	if errUnmarshal := json.Unmarshal([]byte(out), &res); errUnmarshal != nil {
		t.Fatalf("Output is not a json email. out: %q, err: %v", out, errUnmarshal)
	}
	if res.ID.String() != testEmailA1 || res.TMDelete == nil {
		t.Errorf("Expected the deleted email. id: %s, tm_delete: %v", res.ID, res.TMDelete)
	}

	// only the requested email is marked as deleted
	var deleted int
	if errQuery := env.db.QueryRow("select count(*) from email_emails where tm_delete is not null").Scan(&deleted); errQuery != nil {
		t.Fatalf("could not count. err: %v", errQuery)
	}
	if deleted != 1 {
		t.Errorf("Wrong deleted count. expect: 1, got: %d", deleted)
	}
}

func Test_runDelete_errors(t *testing.T) {
	tests := []struct {
		name string

		id string

		expectErr string
	}{
		{name: "missing_id", id: "", expectErr: "failed to resolve email ID: Email ID is required"},
		{name: "malformed_id", id: "not-a-uuid", expectErr: "failed to resolve email ID: invalid format for Email ID: 'not-a-uuid' is not a valid UUID"},
		{name: "not_found", id: testMissingID, expectErr: "failed to delete email: could not get deleted email: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupControlEnv(t)
			env.seed(t, testEmailA1, testCustomerA, "subject a1")

			viper.Set("id", tt.id)

			out, err := runOutput(t, func() error { return runDelete(cmdDelete(), nil) })
			if err == nil || !strings.HasPrefix(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error: %q, got: %v", tt.expectErr, err)
			}
			if out != "" {
				t.Errorf("Expected no output on failure. got: %q", out)
			}

			var deleted int
			if errQuery := env.db.QueryRow("select count(*) from email_emails where tm_delete is not null").Scan(&deleted); errQuery != nil {
				t.Fatalf("could not count. err: %v", errQuery)
			}
			if deleted != 0 {
				t.Errorf("Expected no deleted email after a failure. got: %d", deleted)
			}
		})
	}
}
