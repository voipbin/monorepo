package main

import (
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	gomock "go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-registrar-manager/internal/config"
)

const (
	caseDSNAsterisk = "case-dsn-asterisk"
	caseDSNBinGood  = "case-dsn-bin"
	caseRabbitURI   = "case-rabbit-uri"
)

// fakeEngine is a migration engine that returns fixed values.
type fakeEngine struct {
	version uint
}

func (e *fakeEngine) Version() (uint, bool, error) { return e.version, false, nil }
func (e *fakeEngine) Up() error                    { return nil }
func (e *fakeEngine) Steps(n int) error            { return nil }
func (e *fakeEngine) Force(version int) error      { return nil }
func (e *fakeEngine) Close() (error, error)        { return nil, nil }

// seamEnv holds the fakes behind the process-wide start seams for one test.
type seamEnv struct {
	mc   *gomock.Controller
	sock *sockhandler.MockSockHandler

	dbBin   *sql.DB
	mockBin sqlmock.Sqlmock
	dbAst   *sql.DB
	mockAst sqlmock.Sqlmock
	redis   *miniredis.Miniredis

	mu          sync.Mutex
	connectLog  []string
	connectFail map[string]error
	rabbitURIs  []string
	engineDSNs  []string
}

func (e *seamEnv) sockURIs() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.rabbitURIs...)
}

func (e *seamEnv) engines() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.engineDSNs...)
}

func (e *seamEnv) connected() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.connectLog...)
}

// newSeamEnv installs the fakes and puts everything back when the test ends. The seams, the
// config and the metrics registry are process-wide, so a test that uses it must not run in
// parallel.
func newSeamEnv(t *testing.T) *seamEnv {
	t.Helper()

	env := &seamEnv{connectFail: map[string]error{}}
	env.mc = gomock.NewController(t)
	env.sock = sockhandler.NewMockSockHandler(env.mc)

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
	cfg.DatabaseDSNBin = caseDSNBinGood
	cfg.DatabaseDSNAsterisk = caseDSNAsterisk
	cfg.RabbitMQAddress = caseRabbitURI
	cfg.RedisAddress = env.redis.Addr()
	cfg.RedisDatabase = 0

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
		case caseDSNBinGood:
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
		env.engineDSNs = append(env.engineDSNs, dsn+"|"+versionTable)
		return &fakeEngine{version: 3}, 3, nil
	})

	t.Cleanup(func() {
		restoreEngine()
		restoreSock()
		restoreConnect()
		prometheus.DefaultRegisterer = savedRegisterer
		*cfg = saved
		env.redis.Close()
	})

	return env
}

// expectInit sets the broker calls of one handler initialisation.
func (e *seamEnv) expectInit() {
	e.sock.EXPECT().Connect().Times(1)
	// the notify handler declares the global event exchange
	e.sock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil).Times(1)
}

// captureStdout runs fn and returns what it printed to the standard output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create a pipe: %v", err)
	}
	os.Stdout = w
	restored := false
	restore := func() {
		if !restored {
			os.Stdout = orig
			restored = true
		}
	}
	t.Cleanup(restore)

	fn()
	_ = w.Close()
	restore()

	out, errRead := io.ReadAll(r)
	if errRead != nil {
		t.Fatalf("could not read the output: %v", errRead)
	}
	return string(out)
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()

	if !strings.Contains(got, want) {
		t.Errorf("Wrong match. expect to contain %q, got: %q", want, got)
	}
}
