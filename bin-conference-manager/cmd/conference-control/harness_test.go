package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	gomock "go.uber.org/mock/gomock"

	sqlite3 "github.com/mattn/go-sqlite3"

	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-conference-manager/internal/config"
	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/cachehandler"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

// testRedis is the stand-in cache server, started once in TestMain before the config is loaded.
var testRedis *miniredis.Miniredis

var dbSeq atomic.Int64

// parkWebhooks holds back the answer to a webhook publication until the test binary has finished.
// The notify handler publishes webhooks from its own goroutine and that goroutine reads a metric
// variable of the request handler after the answer. Every construction of a request handler
// writes that variable, so a goroutine that outlives its test would race with the next test.
// A parked goroutine never reaches that read while tests run; TestMain releases it at the end.
var parkWebhooks = make(chan struct{})

const sqliteDriver = "sqlite3_json_array_append"

var registerDriver sync.Once

// registerSQLiteDriver registers a sqlite driver that knows json_array_append for the root path '$',
// which the database handler uses to append an id to a json list column and sqlite does not provide.
func registerSQLiteDriver() {
	registerDriver.Do(func() {
		sql.Register(sqliteDriver, &sqlite3.SQLiteDriver{
			ConnectHook: func(c *sqlite3.SQLiteConn) error {
				return c.RegisterFunc("json_array_append", func(doc, path, value string) (string, error) {
					if path != "$" {
						return "", fmt.Errorf("unsupported path: %s", path)
					}

					var list []string
					if err := json.Unmarshal([]byte(doc), &list); err != nil {
						return "", err
					}

					res, err := json.Marshal(append(list, value))
					return string(res), err
				}, true)
			},
		})
	})
}

// commandKeys are the viper keys that the commands read. They are cleared before each command test.
var commandKeys = []string{
	"id", "customer-id", "type", "name", "detail", "data", "timeout", "pre-flow-id", "post-flow-id",
	"confbridge-id", "limit", "token", "recording-id", "activeflow-id", "format", "duration",
	"on-end-flow-id", "lang",
}

// route is a synthetic answer of the message broker for the requests whose uri contains match.
type route struct {
	match  string
	status int
	body   any
	err    error
}

// testEnv is a fully wired command environment: sqlite database, stand-in cache and message broker.
type testEnv struct {
	t      *testing.T
	db     *sql.DB
	dbh    dbhandler.DBHandler
	routes []route

	mu    sync.Mutex
	calls []string // requests seen by the message broker; read it with requests()
}

// newEnv installs the injection points of the shared library and returns the environment.
// The injection points are process wide, so the tests that use it never run in parallel.
func newEnv(t *testing.T) *testEnv {
	t.Helper()

	e := &testEnv{t: t}

	// every construction of the request handler registers metrics, so each test gets its own registry.
	prevReg, prevGather := prometheus.DefaultRegisterer, prometheus.DefaultGatherer
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer, prometheus.DefaultGatherer = reg, reg
	t.Cleanup(func() { prometheus.DefaultRegisterer, prometheus.DefaultGatherer = prevReg, prevGather })

	testRedis.FlushAll()

	t.Cleanup(func() {
		for _, k := range commandKeys {
			viper.Set(k, nil)
		}
	})

	// reset the keys of the commands: no override and a fresh unchanged flag, because viper keeps
	// the flags that an earlier command bound to the same key.
	for _, k := range commandKeys {
		viper.Set(k, nil)

		c := &cobra.Command{}
		c.Flags().String(k, "", "")
		if err := viper.BindPFlag(k, c.Flags().Lookup(k)); err != nil {
			t.Fatalf("could not reset the key %s: %v", k, err)
		}
	}

	// database
	name := fmt.Sprintf("file:ccontrol%d?mode=memory&cache=shared", dbSeq.Add(1))
	registerSQLiteDriver()
	db, err := sql.Open(sqliteDriver, name)
	if err != nil {
		t.Fatalf("could not open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	files, err := filepath.Glob("../../scripts/database_scripts_test/*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("could not find the test schema. err: %v", err)
	}
	for _, f := range files {
		contents, errRead := os.ReadFile(f)
		if errRead != nil {
			t.Fatalf("could not read the test schema: %v", errRead)
		}
		if _, errExec := db.Exec(string(contents)); errExec != nil {
			t.Fatalf("could not apply the test schema: %v", errExec)
		}
	}
	e.db = db

	restoreDB := commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) { return db, nil })
	t.Cleanup(restoreDB)

	e.dbh = dbhandler.NewHandler(db, cachehandler.NewHandler(testRedis.Addr(), "", config.Get().RedisDatabase))

	// message broker
	mc := gomock.NewController(t)
	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().Connect().AnyTimes()
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockSock.EXPECT().EventPublish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockSock.EXPECT().EventPublishWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockSock.EXPECT().RequestPublishWithDelay(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockSock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(e.answer).AnyTimes()

	restoreSock := sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler { return mockSock })
	t.Cleanup(restoreSock)

	// the notify handler publishes webhooks from a goroutine, so the answer is always available.
	e.on("/v1/webhooks", 200, nil)

	return e
}

// answer routes a request to the first matching synthetic route.
func (e *testEnv) answer(_ context.Context, _ string, req *sock.Request) (*sock.Response, error) {
	e.mu.Lock()
	e.calls = append(e.calls, string(req.Method)+" "+req.URI)
	e.mu.Unlock()

	if strings.HasSuffix(req.URI, "/v1/webhooks") {
		<-parkWebhooks
	}

	for _, r := range e.routes {
		if !strings.Contains(req.URI, r.match) {
			continue
		}
		if r.err != nil {
			return nil, r.err
		}

		data, err := json.Marshal(r.body)
		if err != nil {
			return nil, err
		}
		return &sock.Response{StatusCode: r.status, DataType: "application/json", Data: data}, nil
	}

	return nil, fmt.Errorf("no route for %s %s", req.Method, req.URI)
}

// requests returns a copy of the requests seen by the message broker, webhook publications excluded
// because the notify handler sends them from its own goroutine.
func (e *testEnv) requests() []string {
	e.mu.Lock()
	defer e.mu.Unlock()

	res := []string{}
	for _, c := range e.calls {
		if !strings.HasSuffix(c, "/v1/webhooks") {
			res = append(res, c)
		}
	}
	return res
}

// on registers a synthetic answer.
func (e *testEnv) on(match string, status int, body any) {
	e.routes = append(e.routes, route{match: match, status: status, body: body})
}

// seed stores a conference through the real database handler and returns it as stored.
func (e *testEnv) seed(cf *conference.Conference) *conference.Conference {
	e.t.Helper()

	if cf.ID == uuid.Nil {
		cf.ID = uuid.Must(uuid.NewV4())
	}
	if cf.ConferencecallIDs == nil {
		cf.ConferencecallIDs = []uuid.UUID{}
	}
	if cf.RecordingIDs == nil {
		cf.RecordingIDs = []uuid.UUID{}
	}
	if cf.TranscribeIDs == nil {
		cf.TranscribeIDs = []uuid.UUID{}
	}

	if err := e.dbh.ConferenceCreate(context.Background(), cf); err != nil {
		e.t.Fatalf("could not seed the conference: %v", err)
	}
	res, err := e.dbh.ConferenceGet(context.Background(), cf.ID)
	if err != nil {
		e.t.Fatalf("could not read the seeded conference: %v", err)
	}
	return res
}

// capture runs fn and returns what it wrote to stdout.
func capture(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	errRun := fn()

	_ = w.Close()
	os.Stdout = orig

	select {
	case out := <-done:
		return out, errRun
	case <-time.After(5 * time.Second):
		t.Fatalf("could not read the captured output")
		return "", errRun
	}
}

// decodeConference decodes the printed JSON.
func decodeConference(t *testing.T, out string) *conference.Conference {
	t.Helper()

	var res conference.Conference
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("could not decode the output: %v, out: %s", err, out)
	}
	return &res
}

// exec runs the root command with the given arguments and returns what it printed.
func (e *testEnv) exec(args ...string) (string, error) {
	e.t.Helper()

	root := initCommand()
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs(args)

	return capture(e.t, root.Execute)
}
