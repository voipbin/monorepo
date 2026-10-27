package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/internal/config"
	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/cachehandler"
)

const (
	testSeamBrokerAddress = "broker.test.invalid:5672"
	testSeamCallID        = "5b2d7a10-0000-4000-8000-000000000001"
	testSeamGroupcallID   = "5b2d7a10-0000-4000-8000-000000000002"
	testSeamTransferID    = "5b2d7a10-0000-4000-8000-000000000003"
	testSeamConfbridgeID  = "5b2d7a10-0000-4000-8000-000000000004"
	testSeamFlowID        = "5b2d7a10-0000-4000-8000-000000000005"
	testSeamCustomerID    = "5b2d7a10-0000-4000-8000-000000000006"
)

// isolateMetrics gives the test its own prometheus registerer. requesthandler.NewRequestHandler
// registers its collectors on every call and panics on the second registration in a process.
func isolateMetrics(t *testing.T) {
	t.Helper()

	saved := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = saved })
}

// seamEnv is what a test gets after seamSetup.
type seamEnv struct {
	mr       *miniredis.Miniredis
	mockSock *sockhandler.MockSockHandler
	cache    cachehandler.CacheHandler

	dsns       []string
	brokerURIs []string
}

// seamSetup installs the injection points of the common handler for the test and restores them
// when the test ends: the database connection is a lazy handle that never connects, the sock
// handler is a mock whose Connect is expected once, and the cache is a miniredis. The tests that use
// it must not run in parallel.
func seamSetup(t *testing.T) *seamEnv {
	t.Helper()

	isolateMetrics(t)

	env := &seamEnv{mr: miniredis.RunT(t)}

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	env.mockSock = sockhandler.NewMockSockHandler(mc)

	restoreDB := commondatabasehandler.SetConnectFunc(func(dsn string) (*sql.DB, error) {
		env.dsns = append(env.dsns, dsn)
		return sql.Open("mysql", dsn)
	})
	t.Cleanup(restoreDB)

	restoreSock := sockhandler.SetRabbitFactory(func(serverURI string) sockhandler.SockHandler {
		env.brokerURIs = append(env.brokerURIs, serverURI)
		return env.mockSock
	})
	t.Cleanup(restoreSock)

	setConfig(t, config.Config{
		DatabaseDSN:     testUnreachableDSN,
		RedisAddress:    env.mr.Addr(),
		RabbitMQAddress: testSeamBrokerAddress,
	})

	env.cache = cachehandler.NewHandler(env.mr.Addr(), "", 0)
	if err := env.cache.Connect(); err != nil {
		t.Fatalf("could not connect the test cache: %v", err)
	}

	return env
}

// seedTransfer stores a transfer in the test cache.
func (e *seamEnv) seedTransfer(t *testing.T) {
	t.Helper()

	tr := &transfer.Transfer{
		Type:             transfer.TypeBlind,
		TransfererCallID: uuid.FromStringOrNil(testSeamCallID),
		GroupcallID:      uuid.FromStringOrNil(testSeamGroupcallID),
	}
	tr.ID = uuid.FromStringOrNil(testSeamTransferID)

	if err := e.cache.TransferSet(context.Background(), tr); err != nil {
		t.Fatalf("could not seed the transfer: %v", err)
	}
}

func Test_initTransferHandler_seam(t *testing.T) {
	env := seamSetup(t)
	env.mockSock.EXPECT().Connect().Times(1)

	sqlDB, err := sql.Open("mysql", testUnreachableDSN)
	if err != nil {
		t.Fatalf("could not open the lazy handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	res, err := initTransferHandler(sqlDB, env.cache)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected a transfer handler, got nil")
	}
	if len(env.brokerURIs) != 1 || env.brokerURIs[0] != testSeamBrokerAddress {
		t.Errorf("expected the broker address of the config to be used, got: %v", env.brokerURIs)
	}
}

func Test_initTransferHandler_twice_seam(t *testing.T) {
	// every call registers its own metrics, so the isolation has to be per call
	for i := 0; i < 2; i++ {
		env := seamSetup(t)
		env.mockSock.EXPECT().Connect().Times(1)

		sqlDB, err := sql.Open("mysql", testUnreachableDSN)
		if err != nil {
			t.Fatalf("could not open the lazy handle: %v", err)
		}
		t.Cleanup(func() { _ = sqlDB.Close() })

		if _, err := initTransferHandler(sqlDB, env.cache); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
}

func Test_initHandler_success_seam(t *testing.T) {
	env := seamSetup(t)
	env.mockSock.EXPECT().Connect().Times(1)

	res, err := initHandler()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected a transfer handler, got nil")
	}
	if len(env.dsns) != 1 || env.dsns[0] != testUnreachableDSN {
		t.Errorf("expected the database to be opened with the DSN of the config, got: %v", env.dsns)
	}
	if len(env.brokerURIs) != 1 || env.brokerURIs[0] != testSeamBrokerAddress {
		t.Errorf("expected the broker address of the config to be used, got: %v", env.brokerURIs)
	}
}

func Test_initHandler_cacheFailure_seam(t *testing.T) {
	env := seamSetup(t)

	addr := env.mr.Addr()
	env.mr.Close()
	config.Get().RedisAddress = addr

	res, err := initHandler()
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if res != nil {
		t.Errorf("expected no handler on error, got: %v", res)
	}
	if !strings.HasPrefix(err.Error(), "could not initialize the cache") {
		t.Errorf("unexpected error: %v", err)
	}
	if len(env.dsns) != 1 {
		t.Errorf("expected the database to be opened once before the cache failure, got: %v", env.dsns)
	}
}

func Test_initHandler_databaseError_seam(t *testing.T) {
	env := seamSetup(t)

	errConn := errors.New("injected connection failure")
	restore := commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) { return nil, errConn })
	t.Cleanup(restore)

	res, err := initHandler()
	if res != nil {
		t.Errorf("expected no handler on error, got: %v", res)
	}
	if err == nil || !strings.HasPrefix(err.Error(), "could not connect to the database") || !errors.Is(err, errConn) {
		t.Errorf("unexpected error: %v", err)
	}
	if len(env.brokerURIs) != 0 {
		t.Errorf("the broker must not be used after the database failure, got: %v", env.brokerURIs)
	}
}

// Test_runGetBy_seam covers the lookup commands after a successful initialization.
func Test_runGetBy_seam(t *testing.T) {
	tests := []struct {
		name string

		run      func(cmd *cobra.Command, args []string) error
		flagName string
		flagVal  string
		seed     bool

		expectErrPrefix string
		expectErrPart   string
		expectID        string
	}{
		{
			name:     "get-by-call found",
			run:      runGetByCall,
			flagName: "call-id",
			flagVal:  testSeamCallID,
			seed:     true,
			expectID: testSeamTransferID,
		},
		{
			name:     "get-by-groupcall found",
			run:      runGetByGroupcall,
			flagName: "groupcall-id",
			flagVal:  testSeamGroupcallID,
			seed:     true,
			expectID: testSeamTransferID,
		},
		{
			name:            "get-by-call not found",
			run:             runGetByCall,
			flagName:        "call-id",
			flagVal:         testSeamCallID,
			expectErrPrefix: "failed to retrieve transfer",
			expectErrPart:   "could not get transfer",
		},
		{
			name:            "get-by-groupcall not found",
			run:             runGetByGroupcall,
			flagName:        "groupcall-id",
			flagVal:         testSeamGroupcallID,
			expectErrPrefix: "failed to retrieve transfer",
			expectErrPart:   "could not get transfer",
		},
		{
			name:            "get-by-call invalid id",
			run:             runGetByCall,
			flagName:        "call-id",
			flagVal:         "not-a-uuid",
			expectErrPrefix: "failed to resolve call ID",
			expectErrPart:   "not a valid UUID",
		},
		{
			name:            "get-by-groupcall missing id",
			run:             runGetByGroupcall,
			flagName:        "groupcall-id",
			flagVal:         "",
			expectErrPrefix: "failed to resolve groupcall ID",
			expectErrPart:   "Groupcall ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := seamSetup(t)
			env.mockSock.EXPECT().Connect().Times(1)
			resetViper(t)
			viper.Set(tt.flagName, tt.flagVal)
			if tt.seed {
				env.seedTransfer(t)
			}

			var err error
			out := captureStdout(t, func() { err = tt.run(&cobra.Command{}, nil) })

			if tt.expectErrPrefix != "" {
				if err == nil {
					t.Fatalf("expected an error, got nil. output: %s", out)
				}
				if !strings.HasPrefix(err.Error(), tt.expectErrPrefix) || !strings.Contains(err.Error(), tt.expectErrPart) {
					t.Errorf("unexpected error: %v", err)
				}
				if out != "" {
					t.Errorf("expected no output on error, got: %s", out)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var got transfer.Transfer
			if errUnmarshal := json.Unmarshal([]byte(out), &got); errUnmarshal != nil {
				t.Fatalf("the output is not a transfer json: %v. output: %s", errUnmarshal, out)
			}
			if got.ID.String() != tt.expectID {
				t.Errorf("expected transfer id: %s, got: %s", tt.expectID, got.ID)
			}
			if got.Type != transfer.TypeBlind {
				t.Errorf("expected type: %s, got: %s", transfer.TypeBlind, got.Type)
			}
		})
	}
}

// seamCallRoutes answers the call-manager and flow-manager requests of the service-start path.
func seamCallRoutes(t *testing.T, env *seamEnv, callBody string, flowStatus int) *[]string {
	t.Helper()

	seen := &[]string{}
	env.mockSock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, queueName string, req *sock.Request) (*sock.Response, error) {
			*seen = append(*seen, string(req.Method)+" "+req.URI)

			switch {
			case req.Method == sock.RequestMethodGet && req.URI == "/v1/calls/"+testSeamCallID:
				return &sock.Response{StatusCode: 200, DataType: "application/json", Data: json.RawMessage(callBody)}, nil

			case req.Method == sock.RequestMethodPost && req.URI == "/v1/flows":
				return &sock.Response{StatusCode: flowStatus, DataType: "application/json", Data: json.RawMessage(`{"id":"` + testSeamFlowID + `"}`)}, nil
			}

			t.Errorf("unexpected request: %s %s", req.Method, req.URI)
			return nil, errors.New("unexpected request")
		}).AnyTimes()

	return seen
}

func Test_runServiceStart_seam(t *testing.T) {
	validAddresses := `[{"type":"tel","target":"+15550100"}]`

	tests := []struct {
		name string

		transferType string
		callID       string
		addresses    string
		callBody     string
		flowStatus   int

		expectErrPrefix string
		expectErrPart   string
		expectRequests  []string
	}{
		{
			name:            "transfer type missing",
			callID:          testSeamCallID,
			addresses:       validAddresses,
			expectErrPrefix: "failed to resolve transfer type",
			expectErrPart:   "Transfer type is required",
		},
		{
			name:            "call id missing",
			transferType:    "blind",
			addresses:       validAddresses,
			expectErrPrefix: "failed to resolve transferer call ID",
			expectErrPart:   "Transferer call ID is required",
		},
		{
			name:            "call id invalid",
			transferType:    "blind",
			callID:          "not-a-uuid",
			addresses:       validAddresses,
			expectErrPrefix: "failed to resolve transferer call ID",
			expectErrPart:   "not a valid UUID",
		},
		{
			name:            "addresses missing",
			transferType:    "blind",
			callID:          testSeamCallID,
			expectErrPrefix: "failed to resolve transferee addresses",
			expectErrPart:   "Transferee addresses is required",
		},
		{
			name:            "addresses not json",
			transferType:    "blind",
			callID:          testSeamCallID,
			addresses:       "{not json",
			expectErrPrefix: "failed to parse transferee addresses JSON",
		},
		{
			name:            "transferer call without confbridge",
			transferType:    "blind",
			callID:          testSeamCallID,
			addresses:       validAddresses,
			callBody:        "",
			expectErrPrefix: "failed to start transfer service",
			expectErrPart:   "no confbridge id found",
			expectRequests:  []string{"GET /v1/calls/" + testSeamCallID},
		},
		{
			name:            "unknown transfer type",
			transferType:    "unknown",
			callID:          testSeamCallID,
			addresses:       validAddresses,
			callBody:        `{"id":"` + testSeamCallID + `","customer_id":"` + testSeamCustomerID + `","confbridge_id":"` + testSeamConfbridgeID + `"}`,
			flowStatus:      200,
			expectErrPrefix: "failed to start transfer service",
			expectErrPart:   "transfer type not found. transfer_type: unknown",
			expectRequests:  []string{"GET /v1/calls/" + testSeamCallID, "POST /v1/flows"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := seamSetup(t)
			env.mockSock.EXPECT().Connect().Times(1)
			resetViper(t)
			viper.Set("transfer-type", tt.transferType)
			viper.Set("transferer-call-id", tt.callID)
			viper.Set("transferee-addresses", tt.addresses)

			var seen *[]string
			if tt.expectRequests != nil {
				body := tt.callBody
				if body == "" {
					// a call without a confbridge makes the initialization fail after the lookup
					body = `{"id":"` + testSeamCallID + `"}`
				}
				seen = seamCallRoutes(t, env, body, tt.flowStatus)
			}

			var err error
			out := captureStdout(t, func() { err = runServiceStart(&cobra.Command{}, nil) })

			if err == nil {
				t.Fatalf("expected an error, got nil. output: %s", out)
			}
			if !strings.HasPrefix(err.Error(), tt.expectErrPrefix) || !strings.Contains(err.Error(), tt.expectErrPart) {
				t.Errorf("unexpected error: %v", err)
			}
			if out != "" {
				t.Errorf("expected no output on error, got: %s", out)
			}

			if seen != nil {
				if strings.Join(*seen, ",") != strings.Join(tt.expectRequests, ",") {
					t.Errorf("expected requests: %v, got: %v", tt.expectRequests, *seen)
				}
			}
		})
	}
}

func Test_runServiceStart_requestFailure_seam(t *testing.T) {
	env := seamSetup(t)
	env.mockSock.EXPECT().Connect().Times(1)
	resetViper(t)
	viper.Set("transfer-type", "attended")
	viper.Set("transferer-call-id", testSeamCallID)
	viper.Set("transferee-addresses", `[{"type":"tel","target":"+15550100"}]`)

	errSend := errors.New("injected send failure")
	env.mockSock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errSend).AnyTimes()

	err := runServiceStart(&cobra.Command{}, nil)
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !strings.HasPrefix(err.Error(), "failed to start transfer service") || !errors.Is(err, errSend) {
		t.Errorf("unexpected error: %v", err)
	}
}

// seamExecute runs the root command the way the binary does, with the given arguments.
func seamExecute(t *testing.T, env *seamEnv, args ...string) (string, error) {
	t.Helper()

	// the flags and the environment must give the same values the test config has, so the result
	// does not depend on whether an earlier test already consumed the one-time config load
	for _, key := range []string{
		"RABBITMQ_ADDRESS", "PROMETHEUS_ENDPOINT", "PROMETHEUS_LISTEN_ADDRESS",
		"DATABASE_DSN", "REDIS_ADDRESS", "REDIS_PASSWORD", "REDIS_DATABASE",
	} {
		t.Setenv(key, "")
	}

	savedFormatter := logrus.StandardLogger().Formatter
	savedLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(savedFormatter)
		logrus.SetLevel(savedLevel)
	})
	resetViper(t)

	root := initCommand()
	root.SetArgs(append(args,
		"--rabbitmq_address", testSeamBrokerAddress,
		"--database_dsn", testUnreachableDSN,
		"--redis_address", env.mr.Addr(),
	))

	var err error
	out := captureStdout(t, func() { err = root.Execute() })
	return out, err
}

func Test_Execute_getByCall_seam(t *testing.T) {
	env := seamSetup(t)
	env.mockSock.EXPECT().Connect().Times(1)
	env.seedTransfer(t)

	out, err := seamExecute(t, env, "transfer", "get-by-call", "--call-id", testSeamCallID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got transfer.Transfer
	if errUnmarshal := json.Unmarshal([]byte(out), &got); errUnmarshal != nil {
		t.Fatalf("the output is not a transfer json: %v. output: %s", errUnmarshal, out)
	}
	if got.ID.String() != testSeamTransferID {
		t.Errorf("expected transfer id: %s, got: %s", testSeamTransferID, got.ID)
	}

	// the local flag of the sub command is bound into viper by the pre-run hook of the root
	if v := viper.GetString("call-id"); v != testSeamCallID {
		t.Errorf("expected the local flag to be bound, got: %q", v)
	}
}

func Test_Execute_notFound_seam(t *testing.T) {
	env := seamSetup(t)
	env.mockSock.EXPECT().Connect().Times(1)

	out, err := seamExecute(t, env, "transfer", "get-by-groupcall", "--groupcall-id", testSeamGroupcallID)
	if err == nil {
		t.Fatalf("expected an error, got nil. output: %s", out)
	}
	if !strings.Contains(err.Error(), "failed to retrieve transfer") {
		t.Errorf("unexpected error: %v", err)
	}
}

func Test_main_help_seam(t *testing.T) {
	savedArgs := os.Args
	t.Cleanup(func() { os.Args = savedArgs })
	os.Args = []string{"transfer-control", "--help"}

	savedFormatter := logrus.StandardLogger().Formatter
	savedLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(savedFormatter)
		logrus.SetLevel(savedLevel)
	})
	resetViper(t)

	out := captureStdout(t, main)
	if !strings.Contains(out, "Voipbin Transfer Management CLI") {
		t.Errorf("expected the help of the command in the output, got: %s", out)
	}
	if !strings.Contains(out, "Available Commands:") || !strings.Contains(out, "  transfer ") {
		t.Errorf("expected the transfer sub command in the help, got: %s", out)
	}
}
