package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-registrar-manager/internal/config"
)

const (
	caseDSNBin      = "case-dsn-bin"
	caseDSNAsterisk = "case-dsn-asterisk"
	caseRabbitURI   = "case-rabbit-uri"
)

var caseEndpointSeq atomic.Int64

// fakeEngine is a migration engine that records its calls and returns the configured errors.
type fakeEngine struct {
	errUp error

	mu     sync.Mutex
	upN    int
	closeN int
}

func (e *fakeEngine) Version() (uint, bool, error) { return 1, false, nil }
func (e *fakeEngine) Steps(n int) error            { return nil }
func (e *fakeEngine) Force(version int) error      { return nil }

func (e *fakeEngine) Up() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.upN++
	return e.errUp
}

func (e *fakeEngine) ups() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.upN
}

func (e *fakeEngine) closes() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.closeN
}

func (e *fakeEngine) Close() (error, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closeN++
	return nil, nil
}

// failHook records the messages of the log entries it sees.
type failHook struct {
	mu   sync.Mutex
	msgs []string
}

func (h *failHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *failHook) Fire(e *logrus.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.msgs = append(h.msgs, e.Message)
	return nil
}

func (h *failHook) has(substr string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, m := range h.msgs {
		if strings.Contains(m, substr) {
			return true
		}
	}
	return false
}

// seamEnv holds the fakes behind the process-wide start seams for one test.
type seamEnv struct {
	mc   *gomock.Controller
	sock *sockhandler.MockSockHandler

	dbBin     *sql.DB
	mockBin   sqlmock.Sqlmock
	dbAst     *sql.DB
	mockAst   sqlmock.Sqlmock
	redis     *miniredis.Miniredis
	engine    *fakeEngine
	engineErr error

	mu          sync.Mutex
	connectLog  []string
	connectFail map[string]error
	engineArgs  []string
	rabbitURIs  []string
}

func (e *seamEnv) connected() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.connectLog...)
}

func (e *seamEnv) engineCalls() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.engineArgs...)
}

func (e *seamEnv) sockURIs() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.rabbitURIs...)
}

func drainDone() {
	for {
		select {
		case <-chDone:
		default:
			return
		}
	}
}

// newSeamEnv installs the fakes and puts everything back when the test ends. The seams and the
// config are process-wide, so a test that uses it must not run in parallel.
func newSeamEnv(t *testing.T) *seamEnv {
	t.Helper()

	env := &seamEnv{connectFail: map[string]error{}}
	env.mc = gomock.NewController(t)
	env.sock = sockhandler.NewMockSockHandler(env.mc)
	env.engine = &fakeEngine{}

	var err error
	env.dbBin, env.mockBin, err = sqlmock.New()
	if err != nil {
		t.Fatalf("could not create the bin database mock: %v", err)
	}
	env.dbAst, env.mockAst, err = sqlmock.New()
	if err != nil {
		t.Fatalf("could not create the asterisk database mock: %v", err)
	}

	env.redis, err = miniredis.Run()
	if err != nil {
		t.Fatalf("could not start the cache server: %v", err)
	}

	cfg := config.Get()
	saved := *cfg
	cfg.DatabaseDSNBin = caseDSNBin
	cfg.DatabaseDSNAsterisk = caseDSNAsterisk
	cfg.RabbitMQAddress = caseRabbitURI
	cfg.RedisAddress = env.redis.Addr()
	cfg.RedisDatabase = 0
	cfg.PrometheusEndpoint = fmt.Sprintf("/metrics-case-%d", caseEndpointSeq.Add(1))
	cfg.PrometheusListenAddress = "127.0.0.1:0"

	savedRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	restoreConnect := commondatabasehandler.SetConnectFunc(func(dsn string) (*sql.DB, error) {
		env.mu.Lock()
		defer env.mu.Unlock()
		env.connectLog = append(env.connectLog, dsn)
		if errFail := env.connectFail[dsn]; errFail != nil {
			return nil, errFail
		}
		switch dsn {
		case caseDSNBin:
			return env.dbBin, nil
		case caseDSNAsterisk:
			return env.dbAst, nil
		}
		return nil, fmt.Errorf("unexpected dsn: %s", dsn)
	})
	restoreSock := sockhandler.SetRabbitFactory(func(serverURI string) sockhandler.SockHandler {
		env.mu.Lock()
		defer env.mu.Unlock()
		env.rabbitURIs = append(env.rabbitURIs, serverURI)
		return env.sock
	})
	restoreEngine := migrationhandler.SetEngineFunc(func(dsn string, fsys fs.FS, versionTable string) (migrationhandler.Engine, uint, error) {
		env.mu.Lock()
		defer env.mu.Unlock()
		env.engineArgs = append(env.engineArgs, dsn+"|"+versionTable)
		if fsys == nil {
			return nil, 0, fmt.Errorf("no migration files")
		}
		if env.engineErr != nil {
			return nil, 0, env.engineErr
		}
		return env.engine, 1, nil
	})

	drainDone()

	t.Cleanup(func() {
		restoreEngine()
		restoreSock()
		restoreConnect()
		prometheus.DefaultRegisterer = savedRegisterer
		*cfg = saved
		env.redis.Close()
		drainDone()
	})

	return env
}

// expectStartCalls sets the broker calls of a complete start. The returned channels are closed
// when the consume goroutines of the code under test call in.
func (e *seamEnv) expectStartCalls() (rpcDone, eventDone chan struct{}) {
	rpcDone = make(chan struct{})
	eventDone = make(chan struct{})

	e.sock.EXPECT().Connect().Times(1)
	// the notify handler declares the event exchange once and the subscriber once more
	e.sock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil).Times(2)
	e.sock.EXPECT().QueueCreate(gomock.Any(), "normal").Return(nil).Times(2)
	e.sock.EXPECT().QueueBind(gomock.Any(), gomock.Any(), gomock.Any(), false, nil).Return(nil).MinTimes(1)
	e.sock.EXPECT().ConsumeRPC(gomock.Any(), gomock.Any(), gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cb sock.CbMsgRPC) error {
			close(rpcDone)
			return nil
		})
	e.sock.EXPECT().ConsumeMessage(gomock.Any(), gomock.Any(), gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cb sock.CbMsgConsume) error {
			close(eventDone)
			return nil
		})

	return rpcDone, eventDone
}

func waitBoth(t *testing.T, rpcDone, eventDone <-chan struct{}) {
	t.Helper()

	waitClosed(t, rpcDone, "ConsumeRPC")
	waitClosed(t, eventDone, "ConsumeMessage")
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(caseWaitTimeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Wrong match. %s did not happen in time", what)
}
