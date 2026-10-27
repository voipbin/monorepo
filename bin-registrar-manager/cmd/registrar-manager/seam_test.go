package main

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	gomock "go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-registrar-manager/migrations"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

func Test_run_success(t *testing.T) {
	env := newSeamEnv(t)
	rpcDone, eventDone := env.expectStartCalls()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)

	if err := run(env.dbAst, env.dbBin, mockCache); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}
	waitBoth(t, rpcDone, eventDone)

	if uris := env.sockURIs(); len(uris) != 1 || uris[0] != caseRabbitURI {
		t.Errorf("Wrong match. expect the sock handler to be created once with the configured address, got: %v", uris)
	}
}

func Test_run_subscribeFails(t *testing.T) {
	env := newSeamEnv(t)

	rpcDone := make(chan struct{})
	env.sock.EXPECT().Connect().Times(1)
	env.sock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	env.sock.EXPECT().QueueCreate(string(commonoutline.QueueNameRegistrarRequest), "normal").Return(nil)
	env.sock.EXPECT().ConsumeRPC(gomock.Any(), gomock.Any(), gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cb sock.CbMsgRPC) error {
			close(rpcDone)
			return nil
		})
	env.sock.EXPECT().QueueCreate(string(commonoutline.QueueNameRegistrarSubscribe), "normal").Return(errInjectedSock)

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)

	err := run(env.dbAst, env.dbBin, mockCache)
	if !errors.Is(err, errInjectedSock) {
		t.Errorf("Wrong match. expect: %v, got: %v", errInjectedSock, err)
	}
	waitClosed(t, rpcDone, "ConsumeRPC")
}

func Test_initCache(t *testing.T) {
	env := newSeamEnv(t)

	res, err := initCache()
	if err != nil || res == nil {
		t.Fatalf("Wrong match. expect: a connected cache, got: %v, %v", res, err)
	}

	env.redis.Close()
	res, err = initCache()
	if err == nil || res != nil || !strings.Contains(err.Error(), "cache connect error") {
		t.Errorf("Wrong match. expect: nil cache and a connect error, got: %v, %v", res, err)
	}
}

func Test_initSignal(t *testing.T) {
	newSeamEnv(t)
	t.Cleanup(func() {
		signal.Stop(chSigs)
		drainDone()
	})

	initSignal()
	chSigs <- syscall.SIGTERM

	select {
	case <-chDone:
	case <-time.After(caseWaitTimeout):
		t.Fatalf("Wrong match. the stop signal was not turned into a done token in time")
	}
}

func Test_initProm_listenFails(t *testing.T) {
	hook := &failHook{}
	prev := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	logrus.AddHook(hook)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(prev) })

	// an address without a port cannot be listened on, so the server goroutine logs the failure
	initProm(fmt.Sprintf("/metrics-case-listen-fails-%d", caseEndpointSeq.Add(1)), "case-invalid-address")

	waitFor(t, "the metrics server failure log", func() bool { return hook.has("Prometheus server error") })
}

func Test_runDaemon_failures(t *testing.T) {
	tests := []struct {
		name  string
		setup func(env *seamEnv)

		expectErr       string
		expectConnected []string
		expectEngineNew int // attempts to create the migration engine
		expectUp        int
	}{
		{
			name: "the bin database cannot be reached",
			setup: func(env *seamEnv) {
				env.connectFail[caseDSNBin] = errInjectedSock
			},
			expectErr:       "could not connect to the voipbin database",
			expectConnected: []string{caseDSNBin},
		},
		{
			name: "the migration engine cannot be created",
			setup: func(env *seamEnv) {
				env.engineErr = errInjectedSock
				env.mockBin.ExpectClose()
			},
			expectErr:       "could not apply the database migrations",
			expectConnected: []string{caseDSNBin},
			expectEngineNew: 1,
		},
		{
			name: "the migrations fail to apply",
			setup: func(env *seamEnv) {
				env.engine.errUp = errInjectedSock
				env.mockBin.ExpectClose()
			},
			expectErr:       "could not apply the database migrations",
			expectConnected: []string{caseDSNBin},
			expectEngineNew: 1,
			expectUp:        1,
		},
		{
			name: "the asterisk database cannot be reached",
			setup: func(env *seamEnv) {
				env.connectFail[caseDSNAsterisk] = errInjectedSock
				env.mockBin.ExpectClose()
			},
			expectErr:       "could not connect to the asterisk database",
			expectConnected: []string{caseDSNBin, caseDSNAsterisk},
			expectEngineNew: 1,
			expectUp:        1,
		},
		{
			name: "the cache cannot be reached",
			setup: func(env *seamEnv) {
				env.redis.Close()
				env.mockBin.ExpectClose()
				env.mockAst.ExpectClose()
			},
			expectErr:       "could not initialize the cache",
			expectConnected: []string{caseDSNBin, caseDSNAsterisk},
			expectEngineNew: 1,
			expectUp:        1,
		},
		{
			name: "the services cannot be started",
			setup: func(env *seamEnv) {
				env.sock.EXPECT().Connect().Times(1)
				env.sock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				env.sock.EXPECT().QueueCreate(string(commonoutline.QueueNameRegistrarRequest), "normal").Return(errInjectedSock)
				env.sock.EXPECT().QueueCreate(string(commonoutline.QueueNameRegistrarSubscribe), "normal").Return(errInjectedSock)
				env.mockBin.ExpectClose()
				env.mockAst.ExpectClose()
			},
			expectErr:       "could not start services",
			expectConnected: []string{caseDSNBin, caseDSNAsterisk},
			expectEngineNew: 1,
			expectUp:        1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newSeamEnv(t)
			t.Cleanup(func() { signal.Stop(chSigs) })
			tt.setup(env)

			// run in a goroutine: a regression that lets the daemon start would block on the stop
			// token, and the test must fail quickly instead of waiting
			result := make(chan error, 1)
			go func() { result <- runDaemon() }()

			var err error
			select {
			case err = <-result:
			case <-time.After(caseWaitTimeout):
				t.Fatalf("Wrong match. expect the start to fail with %q, the daemon kept running", tt.expectErr)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect an error containing %q, got: %v", tt.expectErr, err)
			}

			if got := env.connected(); strings.Join(got, ",") != strings.Join(tt.expectConnected, ",") {
				t.Errorf("Wrong match. expect connections: %v, got: %v", tt.expectConnected, got)
			}
			if got := len(env.engineCalls()); got != tt.expectEngineNew {
				t.Errorf("Wrong match. expect migration engine attempts: %d, got: %d", tt.expectEngineNew, got)
			}
			if got := env.engine.ups(); got != tt.expectUp {
				t.Errorf("Wrong match. expect migrations applied: %d, got: %d", tt.expectUp, got)
			}
			if err := env.mockBin.ExpectationsWereMet(); err != nil {
				t.Errorf("Wrong match. the bin database was not closed as expected: %v", err)
			}
			if err := env.mockAst.ExpectationsWereMet(); err != nil {
				t.Errorf("Wrong match. the asterisk database was not closed as expected: %v", err)
			}
		})
	}
}

func Test_runDaemon_success(t *testing.T) {
	env := newSeamEnv(t)
	t.Cleanup(func() { signal.Stop(chSigs) })

	rpcDone, eventDone := env.expectStartCalls()
	env.mockBin.ExpectClose()
	env.mockAst.ExpectClose()

	// the stop token is already waiting, so runDaemon returns right after the start
	chDone <- true

	if err := runDaemon(); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}
	waitBoth(t, rpcDone, eventDone)

	if got := strings.Join(env.connected(), ","); got != caseDSNBin+","+caseDSNAsterisk {
		t.Errorf("Wrong match. expect the bin database first and then the asterisk database, got: %s", got)
	}
	if got := env.engineCalls(); len(got) != 1 || got[0] != caseDSNBin+"|"+migrations.VersionTable {
		t.Errorf("Wrong match. expect the migrations on the bin database with the version table, got: %v", got)
	}
	if up, closed := env.engine.ups(), env.engine.closes(); up != 1 || closed != 1 {
		t.Errorf("Wrong match. expect one migration run and one release, got up: %d, close: %d", up, closed)
	}
	if err := env.mockBin.ExpectationsWereMet(); err != nil {
		t.Errorf("Wrong match. the bin database was not closed: %v", err)
	}
	if err := env.mockAst.ExpectationsWereMet(); err != nil {
		t.Errorf("Wrong match. the asterisk database was not closed: %v", err)
	}
}

// Test_runDaemon_waitsForStop verifies that a started daemon keeps running until the stop token
// arrives. The short wait can only make the test pass wrongly, never fail it wrongly.
func Test_runDaemon_waitsForStop(t *testing.T) {
	env := newSeamEnv(t)
	t.Cleanup(func() { signal.Stop(chSigs) })

	rpcDone, eventDone := env.expectStartCalls()
	env.mockBin.ExpectClose()
	env.mockAst.ExpectClose()

	result := make(chan error, 1)
	go func() { result <- runDaemon() }()
	waitBoth(t, rpcDone, eventDone)

	select {
	case err := <-result:
		t.Fatalf("Wrong match. expect the daemon to keep running before the stop token, returned: %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	chDone <- true

	select {
	case err := <-result:
		if err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	case <-time.After(caseWaitTimeout):
		t.Fatalf("Wrong match. the daemon did not return after the stop token")
	}
}
