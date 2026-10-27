package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-outdial-manager/internal/config"
	"monorepo/bin-outdial-manager/migrations"
	"monorepo/bin-outdial-manager/models/outdial"
)

const waitTimeout = 5 * time.Second

// fakeSock is a sock handler that records what the services publish and does nothing else. The
// embedded nil interface makes every method the commands are not expected to use fail loudly.
type fakeSock struct {
	sockhandler.SockHandler

	mu       sync.Mutex
	connects int
	events   []*sock.Event
	requests int
	registry *prometheus.Registry
}

func (f *fakeSock) Connect() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.connects++
}

func (f *fakeSock) TopicCreateWithKind(name string, kind string) error { return nil }

func (f *fakeSock) EventPublish(topic string, key string, evt *sock.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, evt)
	return nil
}

func (f *fakeSock) RequestPublish(ctx context.Context, queueName string, req *sock.Request) (*sock.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests++
	return &sock.Response{StatusCode: 200}, nil
}

func (f *fakeSock) snapshot() (connects int, events []*sock.Event, requests int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.connects, append([]*sock.Event(nil), f.events...), f.requests
}

// metricSum returns the sum of the samples of the metric families whose name ends with suffix.
func (f *fakeSock) metricSum(suffix string) float64 {
	f.mu.Lock()
	reg := f.registry
	f.mu.Unlock()
	if reg == nil {
		return 0
	}

	families, err := reg.Gather()
	if err != nil {
		return 0
	}

	var sum float64
	for _, fam := range families {
		if !strings.HasSuffix(fam.GetName(), suffix) {
			continue
		}
		for _, m := range fam.GetMetric() {
			switch {
			case m.GetHistogram() != nil:
				sum += float64(m.GetHistogram().GetSampleCount())
			case m.GetCounter() != nil:
				sum += m.GetCounter().GetValue()
			}
		}
	}
	return sum
}

// waitPublished waits until a command that changes an outdial has finished its fire-and-forget
// work: one event publish and one webhook request, including the request metric it writes last.
// The next command re-creates the request metrics, so nothing may still use them when it starts.
func (f *fakeSock) waitPublished(t *testing.T, expectEvents int) {
	t.Helper()

	deadline := time.Now().Add(waitTimeout)
	for time.Now().Before(deadline) {
		_, events, requests := f.snapshot()
		if len(events) >= expectEvents && requests >= expectEvents && f.metricSum("request_process_time") >= 1 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("the publish of the change did not finish in time")
}

type controlEnv struct {
	sock  *fakeSock
	db    *sql.DB
	redis *miniredis.Miniredis
}

// newSQLiteDB creates an in-memory database with the service tables. One connection only, so the
// in-memory database lives as long as the connection and is private to the test.
func newSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("could not open the sqlite database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	files, err := filepath.Glob(filepath.Join("..", "..", "scripts", "database_scripts", "*.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("could not find the table scripts. err: %v", err)
	}
	for _, f := range files {
		contents, errRead := os.ReadFile(f)
		if errRead != nil {
			t.Fatalf("could not read %s: %v", f, errRead)
		}
		if _, errExec := db.Exec(string(contents)); errExec != nil {
			t.Fatalf("could not execute %s: %v", f, errExec)
		}
	}

	return db
}

// setupControl points the process wide state at test doubles until the test ends: the global
// config, viper, the database connection, the sock factory and the metric registry.
func setupControl(t *testing.T) *controlEnv {
	t.Helper()

	resetViper(t)

	mr := miniredis.RunT(t)

	cfg := config.Get()
	orig := *cfg
	t.Cleanup(func() { *cfg = orig })
	cfg.DatabaseDSN = unreachableDSN() // never dialed, the connection function is replaced
	cfg.RedisAddress = mr.Addr()
	cfg.RedisPassword = ""
	cfg.RedisDatabase = 0
	cfg.RabbitMQAddress = "test-rabbit"

	db := newSQLiteDB(t)
	t.Cleanup(commondatabasehandler.SetConnectFunc(func(dsn string) (*sql.DB, error) { return db, nil }))

	fake := &fakeSock{}
	origRegisterer := prometheus.DefaultRegisterer
	t.Cleanup(func() { prometheus.DefaultRegisterer = origRegisterer })
	// the request handler registers its metrics without a guard, so every handler gets a new registry
	t.Cleanup(sockhandler.SetRabbitFactory(func(serverURI string) sockhandler.SockHandler {
		reg := prometheus.NewRegistry()
		fake.mu.Lock()
		fake.registry = reg
		fake.mu.Unlock()
		prometheus.DefaultRegisterer = reg
		return fake
	}))

	return &controlEnv{sock: fake, db: db, redis: mr}
}

// closedAddress returns a loopback address that nothing listens on.
func closedAddress(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

func runAndCapture(t *testing.T, run func(*cobra.Command, []string) error) (string, error) {
	t.Helper()

	var err error
	out := captureStdout(t, func() { err = run(&cobra.Command{}, nil) })
	return out, err
}

func decodeOutdial(t *testing.T, out string) *outdial.Outdial {
	t.Helper()

	var res outdial.Outdial
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("could not decode the output %q: %v", out, err)
	}
	return &res
}

func Test_initCache(t *testing.T) {
	t.Run("connected", func(t *testing.T) {
		setupControl(t)

		res, err := initCache()
		if err != nil || res == nil {
			t.Fatalf("expected a cache handler, got: %v, %v", res, err)
		}
	})

	t.Run("cache unreachable", func(t *testing.T) {
		setupControl(t)
		config.Get().RedisAddress = closedAddress(t)

		res, err := initCache()
		if res != nil {
			t.Errorf("expected no handler, got: %v", res)
		}
		if err == nil || !strings.Contains(err.Error(), "could not connect to the cache") {
			t.Errorf("expected the cache connection error, got: %v", err)
		}
	})
}

func Test_initHandler(t *testing.T) {
	t.Run("success connects the sock handler once", func(t *testing.T) {
		env := setupControl(t)

		res, err := initHandler()
		if err != nil || res == nil {
			t.Fatalf("expected a handler, got: %v, %v", res, err)
		}
		if connects, _, _ := env.sock.snapshot(); connects != 1 {
			t.Errorf("expected one sock connect, got: %d", connects)
		}
	})

	t.Run("cache unreachable", func(t *testing.T) {
		env := setupControl(t)
		config.Get().RedisAddress = closedAddress(t)

		res, err := initHandler()
		if res != nil {
			t.Errorf("expected no handler, got: %v", res)
		}
		if err == nil || !strings.Contains(err.Error(), "could not initialize the cache") {
			t.Errorf("expected the cache error, got: %v", err)
		}
		if connects, _, _ := env.sock.snapshot(); connects != 0 {
			t.Errorf("expected no sock connect before the cache is ready, got: %d", connects)
		}
	})
}

func Test_outdialLifecycle(t *testing.T) {
	env := setupControl(t)
	otherCampaignID := "a1b2c3d4-0000-4000-8000-000000000003"

	// create
	resetViper(t)
	viper.Set("customer-id", testCustomerID)
	viper.Set("campaign-id", testCampaignID)
	viper.Set("name", "outdial one")
	viper.Set("detail", "first detail")
	viper.Set("data", `{"key":"value"}`)
	out, err := runAndCapture(t, runCreate)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	created := decodeOutdial(t, out)
	if created.ID == uuid.Nil || created.CustomerID.String() != testCustomerID || created.CampaignID.String() != testCampaignID ||
		created.Name != "outdial one" || created.Detail != "first detail" || created.Data != `{"key":"value"}` {
		t.Fatalf("unexpected created outdial: %+v", created)
	}
	env.sock.waitPublished(t, 1)
	id := created.ID.String()

	// get
	resetViper(t)
	viper.Set("id", id)
	out, err = runAndCapture(t, runGet)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got := decodeOutdial(t, out); got.ID != created.ID || got.Name != "outdial one" {
		t.Errorf("unexpected outdial: %+v", got)
	}

	// list: only the outdials of the customer
	resetViper(t)
	viper.Set("customer-id", testCustomerID)
	viper.Set("limit", 10)
	out, err = runAndCapture(t, runList)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	var list []*outdial.Outdial
	if errJSON := json.Unmarshal([]byte(out), &list); errJSON != nil {
		t.Fatalf("could not decode the list %q: %v", out, errJSON)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Errorf("expected only the created outdial, got: %+v", list)
	}

	resetViper(t)
	viper.Set("customer-id", otherCampaignID)
	viper.Set("limit", 10)
	out, err = runAndCapture(t, runList)
	if err != nil {
		t.Fatalf("list of another customer failed: %v", err)
	}
	list = nil
	if errJSON := json.Unmarshal([]byte(out), &list); errJSON != nil || len(list) != 0 {
		t.Errorf("expected an empty list for another customer, got: %q, %v", out, errJSON)
	}

	// update basic info
	resetViper(t)
	viper.Set("id", id)
	viper.Set("name", "renamed")
	viper.Set("detail", "second detail")
	out, err = runAndCapture(t, runUpdateBasicInfo)
	if err != nil {
		t.Fatalf("update-basic-info failed: %v", err)
	}
	if got := decodeOutdial(t, out); got.Name != "renamed" || got.Detail != "second detail" || got.Data != `{"key":"value"}` {
		t.Errorf("unexpected outdial after the basic info update: %+v", got)
	}
	env.sock.waitPublished(t, 2)

	// update campaign id
	resetViper(t)
	viper.Set("id", id)
	viper.Set("campaign-id", otherCampaignID)
	out, err = runAndCapture(t, runUpdateCampaignID)
	if err != nil {
		t.Fatalf("update-campaign-id failed: %v", err)
	}
	if got := decodeOutdial(t, out); got.CampaignID.String() != otherCampaignID || got.Name != "renamed" {
		t.Errorf("unexpected outdial after the campaign update: %+v", got)
	}
	env.sock.waitPublished(t, 3)

	// update data
	resetViper(t)
	viper.Set("id", id)
	viper.Set("data", `{"key":"changed"}`)
	out, err = runAndCapture(t, runUpdateData)
	if err != nil {
		t.Fatalf("update-data failed: %v", err)
	}
	if got := decodeOutdial(t, out); got.Data != `{"key":"changed"}` || got.CampaignID.String() != otherCampaignID {
		t.Errorf("unexpected outdial after the data update: %+v", got)
	}
	env.sock.waitPublished(t, 4)

	// delete
	resetViper(t)
	viper.Set("id", id)
	out, err = runAndCapture(t, runDelete)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if got := decodeOutdial(t, out); got.ID != created.ID || got.TMDelete == nil {
		t.Errorf("expected the deleted outdial with a delete time, got: %+v", got)
	}
	env.sock.waitPublished(t, 5)

	// the published events follow the changes
	_, events, _ := env.sock.snapshot()
	expectTypes := []string{
		outdial.EventTypeOutdialCreated, outdial.EventTypeOutdialUpdated, outdial.EventTypeOutdialUpdated,
		outdial.EventTypeOutdialUpdated, outdial.EventTypeOutdialDeleted,
	}
	if len(events) != len(expectTypes) {
		t.Fatalf("expected %d events, got: %d", len(expectTypes), len(events))
	}
	for i, evt := range events {
		if evt.Type != expectTypes[i] {
			t.Errorf("event %d: expected %s, got %s", i, expectTypes[i], evt.Type)
		}
	}

	// a deleted outdial is not listed any more
	resetViper(t)
	viper.Set("customer-id", testCustomerID)
	viper.Set("limit", 10)
	out, err = runAndCapture(t, runList)
	if err != nil {
		t.Fatalf("list after the delete failed: %v", err)
	}
	list = nil
	if errJSON := json.Unmarshal([]byte(out), &list); errJSON != nil || len(list) != 0 {
		t.Errorf("expected an empty list after the delete, got: %q, %v", out, errJSON)
	}
}

func Test_run_failures(t *testing.T) {
	missing := "a1b2c3d4-0000-4000-8000-0000000000ff"

	tests := []struct {
		name         string
		run          func(*cobra.Command, []string) error
		flags        map[string]any
		breakTable   string
		expectErrors []string
	}{
		{"get unknown id", runGet, map[string]any{"id": missing}, "", []string{"failed to retrieve outdial"}},
		{"get invalid id", runGet, map[string]any{"id": "bad"}, "", []string{"failed to resolve outdial ID", "invalid format for Outdial ID: 'bad'"}},
		{"delete unknown id", runDelete, map[string]any{"id": missing}, "", []string{"failed to delete outdial"}},
		{"delete missing id", runDelete, nil, "", []string{"failed to resolve outdial ID", "Outdial ID is required"}},
		{"update-basic-info unknown id", runUpdateBasicInfo, map[string]any{"id": missing, "name": "n"}, "", []string{"failed to update outdial basic info"}},
		{"update-basic-info missing id", runUpdateBasicInfo, map[string]any{"name": "n"}, "", []string{"failed to resolve outdial ID"}},
		{"update-basic-info missing name", runUpdateBasicInfo, map[string]any{"id": missing}, "", []string{"name is required"}},
		{"update-campaign-id unknown id", runUpdateCampaignID, map[string]any{"id": missing, "campaign-id": testCampaignID}, "", []string{"failed to update outdial campaign ID"}},
		{"update-campaign-id missing id", runUpdateCampaignID, map[string]any{"campaign-id": testCampaignID}, "", []string{"failed to resolve outdial ID"}},
		{"update-campaign-id missing campaign id", runUpdateCampaignID, map[string]any{"id": missing}, "", []string{"failed to resolve campaign ID", "Campaign ID is required"}},
		{"update-data unknown id", runUpdateData, map[string]any{"id": missing, "data": "{}"}, "", []string{"failed to update outdial data"}},
		{"update-data missing id", runUpdateData, map[string]any{"data": "{}"}, "", []string{"failed to resolve outdial ID"}},
		{"update-data missing data", runUpdateData, map[string]any{"id": missing}, "", []string{"data is required"}},
		{"list missing customer id", runList, nil, "", []string{"failed to resolve customer ID", "Customer ID is required"}},
		{"list storage failure", runList, map[string]any{"customer-id": testCustomerID, "limit": 10}, "outdial_outdials", []string{"failed to retrieve outdials"}},
		{"create storage failure", runCreate, map[string]any{"customer-id": testCustomerID, "campaign-id": testCampaignID, "name": "n"}, "outdial_outdials", []string{"failed to create outdial"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupControl(t)
			resetViper(t)
			for k, v := range tt.flags {
				viper.Set(k, v)
			}
			if tt.breakTable != "" {
				if _, err := env.db.Exec("DROP TABLE " + tt.breakTable); err != nil {
					t.Fatalf("could not drop the table: %v", err)
				}
			}

			out, err := runAndCapture(t, tt.run)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			for _, part := range tt.expectErrors {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("expected the error to contain %q, got: %v", part, err)
				}
			}
			if out != "" {
				t.Errorf("expected no output on failure, got %q", out)
			}
			if _, events, requests := env.sock.snapshot(); len(events) != 0 || requests != 0 {
				t.Errorf("expected nothing to be published on failure, got %d events and %d requests", len(events), requests)
			}
		})
	}
}

// Test_initCommand_execute runs the commands through cobra, so the persistent pre run binds the
// flags and loads the global config the way the binary does.
func Test_initCommand_execute(t *testing.T) {
	env := setupControl(t)

	origFormatter := logrus.StandardLogger().Formatter
	origLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(origFormatter)
		logrus.SetLevel(origLevel)
	})

	root := initCommand()
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetArgs([]string{
		"outdial", "create",
		"--customer-id", testCustomerID, "--campaign-id", testCampaignID, "--name", "via cobra",
		"--database_dsn", config.Get().DatabaseDSN, "--redis_address", env.redis.Addr(), "--rabbitmq_address", "test-rabbit",
	})

	var err error
	out := captureStdout(t, func() { err = root.Execute() })
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if got := decodeOutdial(t, out); got.Name != "via cobra" || got.CustomerID.String() != testCustomerID {
		t.Errorf("unexpected outdial: %+v", got)
	}
	env.sock.waitPublished(t, 1)

	// a failing command returns its error through cobra
	resetViper(t)
	root = initCommand()
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetArgs([]string{"outdial", "get", "--id", "bad"})
	if err = root.Execute(); err == nil || !strings.Contains(err.Error(), "failed to resolve outdial ID") {
		t.Errorf("expected the id error, got: %v", err)
	}
}

// fakeEngine is a migration engine that reports a fixed state.
type fakeEngine struct {
	version uint
	dirty   bool
}

func (f *fakeEngine) Version() (uint, bool, error) { return f.version, f.dirty, nil }
func (f *fakeEngine) Up() error                    { return nil }
func (f *fakeEngine) Steps(n int) error            { return nil }
func (f *fakeEngine) Force(version int) error      { return nil }
func (f *fakeEngine) Close() (error, error)        { return nil, nil }

// Test_migrateStatus runs the migrate command through cobra with a fake engine, so the factory of
// the command takes the database settings from the config and no database is opened.
func Test_migrateStatus(t *testing.T) {
	resetViper(t)

	cfg := config.Get()
	orig := *cfg
	t.Cleanup(func() { *cfg = orig })
	// the persistent pre run loads the global config once per process, from the same flag value
	cfg.DatabaseDSN = unreachableDSN()

	origFormatter := logrus.StandardLogger().Formatter
	origLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(origFormatter)
		logrus.SetLevel(origLevel)
	})

	var gotDSN, gotTable string
	t.Cleanup(migrationhandler.SetEngineFunc(func(dsn string, fsys fs.FS, versionTable string) (migrationhandler.Engine, uint, error) {
		gotDSN, gotTable = dsn, versionTable
		return &fakeEngine{version: 3, dirty: true}, 5, nil
	}))

	root := initCommand()
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetArgs([]string{"migrate", "status", "--database_dsn", unreachableDSN()})

	var err error
	out := captureStdout(t, func() { err = root.Execute() })
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	expect := "Current version: 3\nImage version: 5\nDirty: true\n"
	if out != expect {
		t.Errorf("expected %q, got %q", expect, out)
	}
	if gotDSN != unreachableDSN() || gotTable != migrations.VersionTable {
		t.Errorf("expected the migrator to use the configured dsn and the service version table, got: %q, %q", gotDSN, gotTable)
	}
}
