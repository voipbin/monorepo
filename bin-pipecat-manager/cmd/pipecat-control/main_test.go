package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"monorepo/bin-pipecat-manager/internal/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// testRedis is the in-memory redis server whose address is loaded into the global config.
var testRedis *miniredis.Miniredis

// TestMain loads the global config exactly once. The config singleton can not be reset from this
// package, so every test of this package shares these values. The database dsn is malformed on
// purpose: the driver rejects it while opening, so no test dials a database.
func TestMain(m *testing.M) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	testRedis = mr

	settings := map[string]string{
		"DATABASE_DSN":              "invalid-dsn",
		"REDIS_ADDRESS":             mr.Addr(),
		"REDIS_PASSWORD":            "",
		"REDIS_DATABASE":            "0",
		"RABBITMQ_ADDRESS":          "",
		"PROMETHEUS_ENDPOINT":       "/metrics",
		"PROMETHEUS_LISTEN_ADDRESS": "127.0.0.1:0",
		"LOG_LEVEL":                 "info",
		"LOCAL_LLM_MODELS":          "",
	}
	for k, v := range settings {
		if errSet := os.Setenv(k, v); errSet != nil {
			panic(errSet)
		}
	}

	if errBootstrap := config.Bootstrap(&cobra.Command{}); errBootstrap != nil {
		panic(errBootstrap)
	}
	config.LoadGlobalConfig()

	code := m.Run()
	mr.Close()
	os.Exit(code)
}

// captureStdout runs f and returns what it printed to stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe. err: %v", err)
	}

	orig := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	f()

	os.Stdout = orig
	_ = w.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("could not read the output. err: %v", err)
	}

	return string(out)
}

func Test_resolveUUID(t *testing.T) {
	const flagName = "resolve-test-id"
	valid := "6f1d2c3b-4a5e-4f60-8a7b-9c0d1e2f3a4b"

	tests := []struct {
		name string

		value string

		expectRes uuid.UUID
		expectErr string
	}{
		{
			name:      "valid uuid",
			value:     valid,
			expectRes: uuid.FromStringOrNil(valid),
		},
		{
			name:      "empty value is required",
			value:     "",
			expectErr: "test label is required",
		},
		{
			name:      "not a uuid",
			value:     "not-a-uuid",
			expectErr: "invalid format for test label: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:      "nil uuid string is rejected as invalid",
			value:     "00000000-0000-0000-0000-000000000000",
			expectErr: "is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set(flagName, tt.value)

			res, err := resolveUUID(flagName, "test label")
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong error. expect: %q, got: %v", tt.expectErr, err)
				}
				if res != uuid.Nil {
					t.Errorf("Wrong result. expect: nil uuid, got: %s", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong error. expect: nil, got: %v", err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong result. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	t.Run("indented json", func(t *testing.T) {
		in := map[string]any{"id": "abc", "count": 2}

		var err error
		out := captureStdout(t, func() { err = printJSON(in) })
		if err != nil {
			t.Fatalf("Wrong error. expect: nil, got: %v", err)
		}

		expect, _ := json.MarshalIndent(in, "", "  ")
		if out != string(expect)+"\n" {
			t.Errorf("Wrong output.\nexpect: %q\ngot: %q", string(expect)+"\n", out)
		}
		if !strings.Contains(out, "\n  \"count\": 2") {
			t.Errorf("Output is not indented with two spaces: %q", out)
		}
	})

	t.Run("value that can not be marshaled", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() { err = printJSON(make(chan int)) })
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("Wrong error. expect: failed to marshal JSON, got: %v", err)
		}
		if out != "" {
			t.Errorf("Wrong output. expect: nothing, got: %q", out)
		}
	})
}

func Test_commandFlags(t *testing.T) {
	tests := []struct {
		name string

		build func() *cobra.Command

		expectUse   string
		expectFlags map[string]string // flag name -> default value
	}{
		{
			name:      "get",
			build:     cmdGet,
			expectUse: "get",
			expectFlags: map[string]string{
				"id": "",
			},
		},
		{
			name:      "start",
			build:     cmdStart,
			expectUse: "start",
			expectFlags: map[string]string{
				"id":             "",
				"customer-id":    "",
				"activeflow-id":  "",
				"reference-type": "",
				"reference-id":   "",
				"llm-type":       "",
				"stt-type":       "",
				"stt-language":   "",
				"tts-type":       "",
				"tts-language":   "",
				"tts-voice-id":   "",
			},
		},
		{
			name:      "terminate",
			build:     cmdTerminate,
			expectUse: "terminate",
			expectFlags: map[string]string{
				"id": "",
			},
		},
		{
			name:      "send-message",
			build:     cmdSendMessage,
			expectUse: "send-message",
			expectFlags: map[string]string{
				"id":              "",
				"message-id":      "",
				"message-text":    "",
				"run-immediately": "false",
				"audio-response":  "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.build()

			if cmd.Use != tt.expectUse {
				t.Errorf("Wrong use. expect: %s, got: %s", tt.expectUse, cmd.Use)
			}
			if cmd.RunE == nil {
				t.Errorf("Wrong command. expect: RunE to be set")
			}

			count := 0
			cmd.Flags().VisitAll(func(f *pflag.Flag) { count++ })
			if count != len(tt.expectFlags) {
				t.Errorf("Wrong flag count. expect: %d, got: %d", len(tt.expectFlags), count)
			}

			for name, def := range tt.expectFlags {
				f := cmd.Flags().Lookup(name)
				if f == nil {
					t.Errorf("Missing flag: %s", name)
					continue
				}
				if f.DefValue != def {
					t.Errorf("Wrong default for %s. expect: %q, got: %q", name, def, f.DefValue)
				}
			}
		})
	}
}

func Test_initCommand(t *testing.T) {
	root := initCommand()

	if root.Use != "pipecat-control" {
		t.Errorf("Wrong root use. got: %s", root.Use)
	}
	if root.PersistentPreRunE == nil {
		t.Errorf("Wrong root. expect: PersistentPreRunE to be set")
	}
	if root.PersistentFlags().Lookup("database_dsn") == nil {
		t.Errorf("Missing the bootstrapped persistent flag database_dsn")
	}

	byName := map[string]*cobra.Command{}
	for _, c := range root.Commands() {
		byName[c.Name()] = c
	}
	if byName["pipecatcall"] == nil || byName["migrate"] == nil {
		t.Fatalf("Wrong subcommands. got: %v", byName)
	}

	subs := map[string]bool{}
	for _, c := range byName["pipecatcall"].Commands() {
		subs[c.Name()] = true
	}
	for _, name := range []string{"get", "start", "terminate", "send-message"} {
		if !subs[name] {
			t.Errorf("Missing pipecatcall subcommand: %s", name)
		}
	}
	if len(subs) != 4 {
		t.Errorf("Wrong pipecatcall subcommands. expect: 4, got: %v", subs)
	}
}

// Test_commandsFailBeforeAnyBackendCall runs every command through cobra with the malformed dsn:
// each one must stop at the database step and wrap the failure for the user.
func Test_commandsFailBeforeAnyBackendCall(t *testing.T) {
	id := "6f1d2c3b-4a5e-4f60-8a7b-9c0d1e2f3a4b"

	tests := []struct {
		name string

		args []string

		expectErr string
	}{
		{
			name:      "get",
			args:      []string{"pipecatcall", "get", "--id", id},
			expectErr: "failed to initialize handlers: could not connect to the database",
		},
		{
			name:      "start",
			args:      []string{"pipecatcall", "start", "--id", id},
			expectErr: "failed to initialize handlers: could not connect to the database",
		},
		{
			name:      "terminate",
			args:      []string{"pipecatcall", "terminate", "--id", id},
			expectErr: "failed to initialize handlers: could not connect to the database",
		},
		{
			name:      "send-message",
			args:      []string{"pipecatcall", "send-message", "--id", id},
			expectErr: "failed to initialize handlers: could not connect to the database",
		},
		{
			name:      "migrate status",
			args:      []string{"migrate", "status"},
			expectErr: "invalid DSN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := initCommand()
			root.SetArgs(tt.args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SilenceUsage = true
			root.SilenceErrors = true

			err := root.Execute()
			if err == nil {
				t.Fatalf("Wrong result. expect: error, got: nil")
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong error.\nexpect: %q\ngot: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_initHandler_databaseFailure(t *testing.T) {
	res, err := initHandler()
	if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong error. expect: could not connect to the database, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong result. expect: nil handler")
	}
}

// Test_initCache keeps the server-down case last: it stops and restarts the shared server.
func Test_initCache(t *testing.T) {
	t.Run("connected", func(t *testing.T) {
		res, err := initCache()
		if err != nil {
			t.Fatalf("Wrong error. expect: nil, got: %v", err)
		}
		if res == nil {
			t.Errorf("Wrong result. expect: a cache handler")
		}
	})

	t.Run("server is down", func(t *testing.T) {
		testRedis.Close()
		t.Cleanup(func() {
			if err := testRedis.Restart(); err != nil {
				t.Fatalf("could not restart the redis server. err: %v", err)
			}
		})

		res, err := initCache()
		if err == nil || !strings.Contains(err.Error(), "could not connect to the cache") {
			t.Errorf("Wrong error. expect: could not connect to the cache, got: %v", err)
		}
		if res != nil {
			t.Errorf("Wrong result. expect: nil handler")
		}
	})
}
