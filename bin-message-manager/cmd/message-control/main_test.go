package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-message-manager/internal/config"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Test_newRequestExternal_tokenOrder guards the argument order of the provider tokens.
// It replaces http.DefaultTransport, so it must not run in parallel.
func Test_newRequestExternal_tokenOrder(t *testing.T) {
	orig := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = orig
	})

	var gotAuth string
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "api.telnyx.com" {
			t.Errorf("Unexpected host: %s", req.URL.Host)
			return nil, errors.New("unexpected host")
		}

		gotAuth = req.Header.Get("Authorization")
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200",
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(`{"data":{}}`)),
		}, nil
	})

	cfg := &config.Config{
		AuthtokenMessagebird: "fake-mb",
		AuthtokenTelnyx:      "fake-tn",
	}

	h := newRequestExternal(cfg)
	if _, err := h.TelnyxSendMessage(context.Background(), "+15550000001", "+15550000002", "hello"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if gotAuth != "Bearer fake-tn" {
		t.Errorf("Wrong authorization header. expect: Bearer fake-tn, got: %s", gotAuth)
	}
}

// invalidDSN is rejected by the mysql driver while the DSN is parsed, so no connection is attempted.
const invalidDSN = "not-a-dsn"

// resetControlState isolates the viper and config globals of one test.
func resetControlState(t *testing.T) {
	t.Helper()

	origCfg := *config.Get()
	origLevel := logrus.GetLevel()
	origFormatter := logrus.StandardLogger().Formatter
	viper.Reset()
	config.Get().DatabaseDSN = invalidDSN

	t.Cleanup(func() {
		viper.Reset()
		*config.Get() = origCfg
		logrus.SetLevel(origLevel)
		logrus.SetFormatter(origFormatter)
	})
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name string

		value string

		expectRes uuid.UUID
		expectErr string
	}{
		{
			name:      "valid uuid",
			value:     "7c5d2f0e-b0b1-11ee-8d3a-6b1f4c2a9e01",
			expectRes: uuid.FromStringOrNil("7c5d2f0e-b0b1-11ee-8d3a-6b1f4c2a9e01"),
		},
		{
			name:      "empty value",
			value:     "",
			expectRes: uuid.Nil,
			expectErr: "Customer ID is required",
		},
		{
			name:      "malformed value",
			value:     "not-a-uuid",
			expectRes: uuid.Nil,
			expectErr: "invalid format for Customer ID: 'not-a-uuid' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetControlState(t)
			viper.Set("flag-x", tt.value)

			res, err := resolveUUID("flag-x", "Customer ID")
			if tt.expectErr == "" && err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if tt.expectErr != "" && (err == nil || err.Error() != tt.expectErr) {
				t.Fatalf("Wrong match. expect: %s, got: %v", tt.expectErr, err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

// captureStdout runs fn and returns what it wrote to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Could not create the pipe. err: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = orig
		_ = r.Close()
	})

	fn()
	_ = w.Close()
	os.Stdout = orig

	out, errRead := io.ReadAll(r)
	if errRead != nil {
		t.Fatalf("Could not read the pipe. err: %v", errRead)
	}
	return string(out)
}

func Test_printJSON(t *testing.T) {
	t.Run("indents the value", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(map[string]int{"a": 1})
		})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if out != "{\n  \"a\": 1\n}\n" {
			t.Errorf("Wrong match. expect: indented json, got: %q", out)
		}
	})

	t.Run("value that cannot be marshaled", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(make(chan int))
		})
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Fatalf("Wrong match. expect: failed to marshal JSON, got: %v", err)
		}
		if out != "" {
			t.Errorf("Wrong match. expect: no output, got: %q", out)
		}
	})
}

func Test_commandFlags(t *testing.T) {
	tests := []struct {
		name string

		cmd *cobra.Command

		expectUse   string
		expectFlags map[string]string // flag name -> default value
	}{
		{
			name:      "send",
			cmd:       cmdSend(),
			expectUse: "send",
			expectFlags: map[string]string{
				"customer-id":  "",
				"source":       "",
				"destinations": "[]",
				"text":         "",
			},
		},
		{
			name:        "get",
			cmd:         cmdGet(),
			expectUse:   "get",
			expectFlags: map[string]string{"id": ""},
		},
		{
			name:      "list",
			cmd:       cmdList(),
			expectUse: "list",
			expectFlags: map[string]string{
				"limit":       "100",
				"token":       "",
				"customer-id": "",
			},
		},
		{
			name:        "delete",
			cmd:         cmdDelete(),
			expectUse:   "delete",
			expectFlags: map[string]string{"id": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.Use != tt.expectUse {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectUse, tt.cmd.Use)
			}
			if tt.cmd.RunE == nil {
				t.Fatalf("Wrong match. expect: RunE set, got: nil")
			}
			for name, def := range tt.expectFlags {
				f := tt.cmd.Flags().Lookup(name)
				if f == nil {
					t.Errorf("Wrong match. expect: flag %s, got: none", name)
					continue
				}
				if f.DefValue != def {
					t.Errorf("Wrong default for %s. expect: %s, got: %s", name, def, f.DefValue)
				}
			}
		})
	}
}

func Test_runSend_validation(t *testing.T) {
	const customerID = "7c5d2f0e-b0b1-11ee-8d3a-6b1f4c2a9e01"

	tests := []struct {
		name string

		values map[string]any

		expectErr string
	}{
		{
			name:      "customer id missing",
			values:    map[string]any{},
			expectErr: "failed to resolve customer ID: Customer ID is required",
		},
		{
			name:      "customer id invalid",
			values:    map[string]any{"customer-id": "not-a-uuid"},
			expectErr: "failed to resolve customer ID: invalid format for Customer ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:      "source empty",
			values:    map[string]any{"customer-id": customerID},
			expectErr: "source is required",
		},
		{
			name:      "destinations empty",
			values:    map[string]any{"customer-id": customerID, "source": "+15550000001"},
			expectErr: "destinations is required",
		},
		{
			name: "text empty",
			values: map[string]any{
				"customer-id":  customerID,
				"source":       "+15550000001",
				"destinations": []string{"+15550000002"},
			},
			expectErr: "text is required",
		},
		{
			// all inputs are valid: the command moves on to the initialization, which fails offline on the invalid DSN.
			name: "valid input reaches the initialization",
			values: map[string]any{
				"customer-id":  customerID,
				"source":       "+15550000001",
				"destinations": []string{"+15550000002", "+15550000003"},
				"text":         "hello",
			},
			expectErr: "failed to initialize handlers: could not connect to the database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetControlState(t)
			for k, v := range tt.values {
				viper.Set(k, v)
			}

			err := runSend(nil, nil)
			if err == nil || !strings.HasPrefix(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect prefix: %s, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_initHandler_invalidDSN(t *testing.T) {
	resetControlState(t)

	res, err := initHandler()
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if err == nil || !strings.HasPrefix(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. expect: could not connect to the database, got: %v", err)
	}
}

func Test_runGetListDelete_initFailure(t *testing.T) {
	tests := []struct {
		name string
		run  func(*cobra.Command, []string) error
	}{
		{name: "get", run: runGet},
		{name: "list", run: runList},
		{name: "delete", run: runDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetControlState(t)
			viper.Set("id", "7c5d2f0e-b0b1-11ee-8d3a-6b1f4c2a9e01")
			viper.Set("customer-id", "7c5d2f0e-b0b1-11ee-8d3a-6b1f4c2a9e01")

			err := tt.run(nil, nil)
			if err == nil || !strings.HasPrefix(err.Error(), "failed to initialize handlers: could not connect to the database") {
				t.Errorf("Wrong match. expect: failed to initialize handlers, got: %v", err)
			}
		})
	}
}

func Test_initCommand_tree(t *testing.T) {
	resetControlState(t)

	root := initCommand()
	if root.Use != "message-control" {
		t.Errorf("Wrong match. expect: message-control, got: %s", root.Use)
	}

	for _, path := range [][]string{
		{"message", "send"},
		{"message", "get"},
		{"message", "list"},
		{"message", "delete"},
		{"migrate", "status"},
		{"migrate", "up"},
		{"migrate", "down"},
		{"migrate", "force"},
	} {
		found, _, err := root.Find(path)
		if err != nil || found == nil || found.Name() != path[len(path)-1] {
			t.Errorf("Wrong match. expect: command %v, got: %v (err %v)", path, found, err)
		}
	}

	if root.PersistentFlags().Lookup("database_dsn") == nil {
		t.Errorf("Wrong match. expect: infrastructure flag database_dsn, got: none")
	}
}

func Test_initCommand_migrateInvalidDSN(t *testing.T) {
	resetControlState(t)
	// Execute loads the global config once from the environment, so the invalid DSN is given there too.
	t.Setenv("DATABASE_DSN", invalidDSN)

	root := initCommand()
	root.SetArgs([]string{"migrate", "status"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "could not initialize the migrator") {
		t.Errorf("Wrong match. expect: could not initialize the migrator, got: %v", err)
	}
}
