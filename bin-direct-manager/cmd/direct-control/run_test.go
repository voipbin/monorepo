package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/internal/config"
	"monorepo/bin-direct-manager/models/direct"
	"monorepo/bin-direct-manager/pkg/cachehandler"
	"monorepo/bin-direct-manager/pkg/dbhandler"
)

var (
	testCustomerA = uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9")
	testCustomerB = uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54")
	testResource  = uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527")
	testDirectID  = uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a")
	testMissingID = uuid.FromStringOrNil("0f1e2d3c-4b5a-11ee-8c99-0242ac120002")
)

// farFuture is a pagination token later than any row the tests create; an empty token matches
// nothing.
const farFuture = "2999-01-01 00:00:00.000000"

var dbSeq atomic.Int64

// newTestDB returns a private in-memory database. With schema it holds the direct table.
func newTestDB(t *testing.T, schema bool) *sql.DB {
	t.Helper()

	// the name is a counter, never the test name, which can hold characters the driver rejects
	db, err := sql.Open("sqlite3", "file:direct_control_"+strconv.FormatInt(dbSeq.Add(1), 10)+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("could not open the test database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	if schema {
		contents, errRead := os.ReadFile(filepath.Join("..", "..", "scripts", "database_scripts_test", "directs.sql"))
		if errRead != nil {
			t.Fatalf("could not read the test schema: %v", errRead)
		}
		if _, errExec := db.Exec(string(contents)); errExec != nil {
			t.Fatalf("could not create the test schema: %v", errExec)
		}
	}

	return db
}

// fakeServices holds the replacements of the three connections the control binary opens.
type fakeServices struct {
	db       *sql.DB
	cache    *miniredis.Miniredis
	connects atomic.Int32
}

// useFakeServices points the control binary at an in-memory database, a miniredis server and
// a mock broker for the rest of the test. The seams are process wide, so the tests that use
// it must not run in parallel; every replacement is restored in reverse order by t.Cleanup.
func useFakeServices(t *testing.T, schema bool) *fakeServices {
	t.Helper()
	resetState(t)

	// the connection settings come from the config set below, never from the environment
	for _, name := range []string{
		"DATABASE_DSN", "PROMETHEUS_ENDPOINT", "PROMETHEUS_LISTEN_ADDRESS",
		"RABBITMQ_ADDRESS", "REDIS_ADDRESS", "REDIS_DATABASE", "REDIS_PASSWORD",
	} {
		t.Setenv(name, "")
	}

	res := &fakeServices{
		db:    newTestDB(t, schema),
		cache: miniredis.RunT(t),
	}
	*config.Get() = config.Config{
		DatabaseDSN:     "unused",
		RabbitMQAddress: "unused",
		RedisAddress:    res.cache.Addr(),
	}

	// the request handler and the circuit breaker register their collectors on every
	// construction, so each test gets its own registry
	prevRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = prevRegisterer })

	t.Cleanup(commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) {
		return res.db, nil
	}))

	mc := gomock.NewController(t)
	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().Connect().Do(func() { res.connects.Add(1) }).AnyTimes()
	// the notify handler declares the global topic exchange and a failure there ends the process
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil).AnyTimes()
	mockSock.EXPECT().EventPublish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	t.Cleanup(sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler { return mockSock }))

	return res
}

// seed stores a direct through the database handler and returns it.
func (f *fakeServices) seed(t *testing.T, id uuid.UUID, customerID uuid.UUID, resourceType string, resourceID uuid.UUID, hash string) *direct.Direct {
	t.Helper()

	d := &direct.Direct{ResourceType: resourceType, ResourceID: resourceID, Hash: hash}
	d.ID = id
	d.CustomerID = customerID
	if err := dbhandler.NewHandler(f.db).DirectCreate(context.Background(), d); err != nil {
		t.Fatalf("could not seed the direct: %v", err)
	}

	return d
}

// seedCache stores a direct under its hash in the cache.
func (f *fakeServices) seedCache(t *testing.T, d *direct.Direct) {
	t.Helper()

	c := cachehandler.NewHandler(f.cache.Addr(), "", 0)
	if err := c.DirectSetByHash(context.Background(), d.Hash, d); err != nil {
		t.Fatalf("could not seed the cache: %v", err)
	}
}

type rawRow struct {
	customerID   uuid.UUID
	resourceType string
	resourceID   uuid.UUID
	hash         string
}

// readRow reads a row with a plain statement on the test's own handle, so that the
// assertions do not depend on the code under test. It returns false when there is no row.
func (f *fakeServices) readRow(t *testing.T, id uuid.UUID) (rawRow, bool) {
	t.Helper()

	var customer, resource []byte
	var res rawRow
	err := f.db.QueryRow("select customer_id, resource_type, resource_id, hash from direct_directs where id = ?", id.Bytes()).
		Scan(&customer, &res.resourceType, &resource, &res.hash)
	if err == sql.ErrNoRows {
		return rawRow{}, false
	}
	if err != nil {
		t.Fatalf("could not read the row: %v", err)
	}
	res.customerID = uuid.FromBytesOrNil(customer)
	res.resourceID = uuid.FromBytesOrNil(resource)

	return res, true
}

func (f *fakeServices) countRows(t *testing.T) int {
	t.Helper()

	var n int
	if err := f.db.QueryRow("select count(*) from direct_directs").Scan(&n); err != nil {
		t.Fatalf("could not count the rows: %v", err)
	}

	return n
}

// runCommand runs a control command function and returns its error and what it printed.
func runCommand(t *testing.T, fn func(cmd *cobra.Command, args []string) error) (string, error) {
	t.Helper()

	var err error
	out := captureStdout(t, func() {
		err = fn(&cobra.Command{}, nil)
	})

	return out, err
}

func Test_initHandler(t *testing.T) {
	t.Run("a cache failure is wrapped and the broker is not opened", func(t *testing.T) {
		f := useFakeServices(t, true)
		f.cache.Close()

		h, err := initHandler()
		if h != nil {
			t.Errorf("initHandler() = %v, want nil", h)
		}
		if err == nil || !strings.Contains(err.Error(), "could not initialize the cache") {
			t.Errorf("initHandler() error = %v, want the cache context", err)
		}
		if got := f.connects.Load(); got != 0 {
			t.Errorf("the broker was connected %d times, want 0", got)
		}
	})

	t.Run("success returns a handler on the opened connections", func(t *testing.T) {
		f := useFakeServices(t, true)
		f.seed(t, testDirectID, testCustomerA, direct.ResourceTypeExtension, testResource, "direct.aaaaaaaaaaaa")

		h, err := initHandler()
		if err != nil || h == nil {
			t.Fatalf("initHandler() = %v, %v, want a handler", h, err)
		}
		if got := f.connects.Load(); got != 1 {
			t.Errorf("the broker was connected %d times, want 1", got)
		}

		res, err := h.Get(context.Background(), testDirectID)
		if err != nil || res.Hash != "direct.aaaaaaaaaaaa" {
			t.Errorf("Get() = %v, %v, want the seeded direct", res, err)
		}
	})
}

func Test_initDirectHandler(t *testing.T) {
	f := useFakeServices(t, true)
	cache := cachehandler.NewHandler(f.cache.Addr(), "", 0)

	h, err := initDirectHandler(f.db, cache)
	if err != nil || h == nil {
		t.Fatalf("initDirectHandler() = %v, %v, want a handler", h, err)
	}
	if got := f.connects.Load(); got != 1 {
		t.Errorf("the broker was connected %d times, want 1", got)
	}
}

func Test_runCreate(t *testing.T) {
	t.Run("stores the direct and prints it", func(t *testing.T) {
		f := useFakeServices(t, true)
		viper.Set("customer-id", testCustomerA.String())
		viper.Set("resource-type", direct.ResourceTypeExtension)
		viper.Set("resource-id", testResource.String())

		out, err := runCommand(t, runCreate)
		if err != nil {
			t.Fatalf("runCreate() error = %v", err)
		}

		var printed direct.Direct
		if errJSON := json.Unmarshal([]byte(out), &printed); errJSON != nil {
			t.Fatalf("the output is not a direct: %v. output: %q", errJSON, out)
		}

		row, ok := f.readRow(t, printed.ID)
		if !ok {
			t.Fatalf("the printed direct %s is not stored", printed.ID)
		}
		if row.customerID != testCustomerA || row.resourceType != direct.ResourceTypeExtension || row.resourceID != testResource {
			t.Errorf("stored row = %+v, want the requested customer, type and resource", row)
		}
		if row.hash != printed.Hash || !strings.HasPrefix(row.hash, direct.DirectPrefix) {
			t.Errorf("stored hash = %q, printed hash = %q, want the same value with the prefix", row.hash, printed.Hash)
		}
	})

	t.Run("a resource that already has a direct is an error", func(t *testing.T) {
		f := useFakeServices(t, true)
		f.seed(t, testDirectID, testCustomerA, direct.ResourceTypeExtension, testResource, "direct.aaaaaaaaaaaa")
		viper.Set("customer-id", testCustomerA.String())
		viper.Set("resource-type", direct.ResourceTypeExtension)
		viper.Set("resource-id", testResource.String())

		out, err := runCommand(t, runCreate)
		if err == nil || !strings.Contains(err.Error(), "failed to create direct") {
			t.Errorf("runCreate() error = %v, want the create context", err)
		}
		if out != "" {
			t.Errorf("runCreate() wrote %q, want nothing", out)
		}
		if got := f.countRows(t); got != 1 {
			t.Errorf("rows = %d, want the seeded row only", got)
		}
	})
}

func Test_runGet(t *testing.T) {
	t.Run("prints the stored direct", func(t *testing.T) {
		f := useFakeServices(t, true)
		f.seed(t, testDirectID, testCustomerA, direct.ResourceTypeConference, testResource, "direct.bbbbbbbbbbbb")
		viper.Set("id", testDirectID.String())

		out, err := runCommand(t, runGet)
		if err != nil {
			t.Fatalf("runGet() error = %v", err)
		}

		var printed direct.Direct
		if errJSON := json.Unmarshal([]byte(out), &printed); errJSON != nil {
			t.Fatalf("the output is not a direct: %v", errJSON)
		}
		row, _ := f.readRow(t, testDirectID)
		if printed.ID != testDirectID || printed.Hash != row.hash || printed.ResourceType != row.resourceType || printed.CustomerID != row.customerID {
			t.Errorf("printed = %+v, want the stored row %+v", printed, row)
		}
	})

	t.Run("an unknown id is reported with the context", func(t *testing.T) {
		useFakeServices(t, true)
		viper.Set("id", testMissingID.String())

		out, err := runCommand(t, runGet)
		if err == nil || !strings.Contains(err.Error(), "failed to retrieve direct") {
			t.Errorf("runGet() error = %v, want the retrieve context", err)
		}
		if out != "" {
			t.Errorf("runGet() wrote %q, want nothing", out)
		}
	})

	t.Run("an invalid id is reported after the handlers are ready", func(t *testing.T) {
		f := useFakeServices(t, true)
		viper.Set("id", "bad")

		_, err := runCommand(t, runGet)
		if err == nil || !strings.Contains(err.Error(), "failed to resolve direct ID") {
			t.Errorf("runGet() error = %v, want the resolve context", err)
		}
		if got := f.connects.Load(); got != 1 {
			t.Errorf("the broker was connected %d times, want 1 (the id is checked after the init)", got)
		}
	})
}

func Test_runList(t *testing.T) {
	seedAll := func(t *testing.T) *fakeServices {
		f := useFakeServices(t, true)
		for i, hash := range []string{"direct.a00000000001", "direct.a00000000002", "direct.a00000000003"} {
			id := uuid.FromStringOrNil("1b2c3d4e-5f60-11ee-8a00-00000000000" + strconv.Itoa(i+1))
			resource := uuid.FromStringOrNil("2b2c3d4e-5f60-11ee-8a00-00000000000" + strconv.Itoa(i+1))
			f.seed(t, id, testCustomerA, direct.ResourceTypeAI, resource, hash)
		}
		f.seed(t, uuid.FromStringOrNil("3b2c3d4e-5f60-11ee-8a00-000000000001"), testCustomerB, direct.ResourceTypeAI,
			uuid.FromStringOrNil("4b2c3d4e-5f60-11ee-8a00-000000000001"), "direct.b00000000001")

		return f
	}

	tests := []struct {
		name string

		customerID uuid.UUID
		limit      int
		token      string

		expectCount int
	}{
		{"all rows of the customer", testCustomerA, 100, farFuture, 3},
		{"the limit caps the result", testCustomerA, 2, farFuture, 2},
		{"another customer sees only its own row", testCustomerB, 100, farFuture, 1},
		{"a token earlier than every row returns nothing", testCustomerA, 100, "2000-01-01 00:00:00.000000", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedAll(t)
			viper.Set("customer-id", tt.customerID.String())
			viper.Set("limit", tt.limit)
			viper.Set("token", tt.token)

			out, err := runCommand(t, runList)
			if err != nil {
				t.Fatalf("runList() error = %v", err)
			}

			var printed []*direct.Direct
			if errJSON := json.Unmarshal([]byte(out), &printed); errJSON != nil {
				t.Fatalf("the output is not a list: %v. output: %q", errJSON, out)
			}
			if len(printed) != tt.expectCount {
				t.Fatalf("runList() printed %d directs, want %d", len(printed), tt.expectCount)
			}
			for _, d := range printed {
				if d.CustomerID != tt.customerID {
					t.Errorf("runList() printed a direct of customer %s, want only %s", d.CustomerID, tt.customerID)
				}
			}
		})
	}

	t.Run("a query failure is reported with the context", func(t *testing.T) {
		useFakeServices(t, false) // no table
		viper.Set("customer-id", testCustomerA.String())
		viper.Set("limit", 10)
		viper.Set("token", farFuture)

		out, err := runCommand(t, runList)
		if err == nil || !strings.Contains(err.Error(), "failed to retrieve directs") {
			t.Errorf("runList() error = %v, want the retrieve context", err)
		}
		if out != "" {
			t.Errorf("runList() wrote %q, want nothing", out)
		}
	})

	t.Run("an invalid customer id is reported", func(t *testing.T) {
		useFakeServices(t, true)
		viper.Set("customer-id", "bad")

		_, err := runCommand(t, runList)
		if err == nil || !strings.Contains(err.Error(), "failed to resolve customer ID") {
			t.Errorf("runList() error = %v, want the resolve context", err)
		}
	})
}

func Test_runDelete(t *testing.T) {
	t.Run("removes the row and the cached hash", func(t *testing.T) {
		f := useFakeServices(t, true)
		d := f.seed(t, testDirectID, testCustomerA, direct.ResourceTypeAgent, testResource, "direct.cccccccccccc")
		f.seedCache(t, d)
		if got := len(f.cache.Keys()); got != 1 {
			t.Fatalf("seeded cache keys = %d, want 1", got)
		}
		viper.Set("id", testDirectID.String())

		out, err := runCommand(t, runDelete)
		if err != nil {
			t.Fatalf("runDelete() error = %v", err)
		}

		var printed direct.Direct
		if errJSON := json.Unmarshal([]byte(out), &printed); errJSON != nil {
			t.Fatalf("the output is not a direct: %v", errJSON)
		}
		if printed.ID != testDirectID || printed.Hash != "direct.cccccccccccc" {
			t.Errorf("printed = %+v, want the deleted direct", printed)
		}
		if _, ok := f.readRow(t, testDirectID); ok {
			t.Errorf("the row is still stored")
		}
		if got := len(f.cache.Keys()); got != 0 {
			t.Errorf("cache keys = %d, want the cached hash removed", got)
		}
	})

	t.Run("an unknown id is reported with the context", func(t *testing.T) {
		useFakeServices(t, true)
		viper.Set("id", testMissingID.String())

		out, err := runCommand(t, runDelete)
		if err == nil || !strings.Contains(err.Error(), "failed to delete direct") {
			t.Errorf("runDelete() error = %v, want the delete context", err)
		}
		if out != "" {
			t.Errorf("runDelete() wrote %q, want nothing", out)
		}
	})

	t.Run("an invalid id is reported and nothing is deleted", func(t *testing.T) {
		f := useFakeServices(t, true)
		f.seed(t, testDirectID, testCustomerA, direct.ResourceTypeAgent, testResource, "direct.cccccccccccc")
		viper.Set("id", "bad")

		_, err := runCommand(t, runDelete)
		if err == nil || !strings.Contains(err.Error(), "failed to resolve direct ID") {
			t.Errorf("runDelete() error = %v, want the resolve context", err)
		}
		if got := f.countRows(t); got != 1 {
			t.Errorf("rows = %d, want the seeded row kept", got)
		}
	})
}

func Test_runRegenerate(t *testing.T) {
	t.Run("replaces the hash and drops the old cache entry", func(t *testing.T) {
		f := useFakeServices(t, true)
		d := f.seed(t, testDirectID, testCustomerA, direct.ResourceTypeQueue, testResource, "direct.dddddddddddd")
		f.seedCache(t, d)
		viper.Set("id", testDirectID.String())

		out, err := runCommand(t, runRegenerate)
		if err != nil {
			t.Fatalf("runRegenerate() error = %v", err)
		}

		var printed direct.Direct
		if errJSON := json.Unmarshal([]byte(out), &printed); errJSON != nil {
			t.Fatalf("the output is not a direct: %v", errJSON)
		}
		row, ok := f.readRow(t, testDirectID)
		if !ok {
			t.Fatalf("the row disappeared")
		}
		if row.hash == "direct.dddddddddddd" || !strings.HasPrefix(row.hash, direct.DirectPrefix) {
			t.Errorf("stored hash = %q, want a new hash with the prefix", row.hash)
		}
		if printed.Hash != row.hash {
			t.Errorf("printed hash = %q, stored hash = %q, want the same value", printed.Hash, row.hash)
		}
		if row.resourceID != testResource || row.customerID != testCustomerA {
			t.Errorf("stored row = %+v, want the same customer and resource", row)
		}
		if got := len(f.cache.Keys()); got != 0 {
			t.Errorf("cache keys = %d, want the old entry removed", got)
		}
	})

	t.Run("an unknown id is reported with the context", func(t *testing.T) {
		useFakeServices(t, true)
		viper.Set("id", testMissingID.String())

		out, err := runCommand(t, runRegenerate)
		if err == nil || !strings.Contains(err.Error(), "failed to regenerate direct") {
			t.Errorf("runRegenerate() error = %v, want the regenerate context", err)
		}
		if out != "" {
			t.Errorf("runRegenerate() wrote %q, want nothing", out)
		}
	})

	t.Run("an invalid id is reported", func(t *testing.T) {
		useFakeServices(t, true)
		viper.Set("id", "bad")

		_, err := runCommand(t, runRegenerate)
		if err == nil || !strings.Contains(err.Error(), "failed to resolve direct ID") {
			t.Errorf("runRegenerate() error = %v, want the resolve context", err)
		}
	})
}
