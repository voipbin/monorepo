package main

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-route-manager/internal/config"
)

// unreachableDSN points at a local port that refuses the connection at once.
const unreachableDSN = "tcp(127.0.0.1:1)/test"

func TestMain(m *testing.M) {
	// the config is a singleton that is loaded only once, so the environment is prepared before the first load.
	if err := os.Setenv("DATABASE_DSN", unreachableDSN); err != nil {
		panic(err)
	}

	boot := &cobra.Command{Use: "bootstrap"}
	if err := config.Bootstrap(boot); err != nil {
		panic(err)
	}
	config.LoadGlobalConfig()

	os.Exit(m.Run())
}

// executeCommand runs the cli with the given arguments and returns the execution error.
func executeCommand(t *testing.T, args ...string) error {
	t.Helper()
	t.Cleanup(viper.Reset)

	cmd := initCommand()
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	return cmd.Execute()
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name string

		value string

		expectRes   uuid.UUID
		expectError string
	}{
		{
			name:        "empty value is required",
			value:       "",
			expectRes:   uuid.Nil,
			expectError: "Customer ID is required",
		},
		{
			name:        "invalid format",
			value:       "not-a-uuid",
			expectRes:   uuid.Nil,
			expectError: "invalid format for Customer ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:      "valid uuid",
			value:     "b9d2e1a4-5c3f-4a8e-9f10-2d7c6e5b4a39",
			expectRes: uuid.FromStringOrNil("b9d2e1a4-5c3f-4a8e-9f10-2d7c6e5b4a39"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(viper.Reset)
			viper.Set("customer-id", tt.value)

			res, err := resolveUUID("customer-id", "Customer ID")
			if tt.expectError != "" {
				if err == nil || err.Error() != tt.expectError {
					t.Fatalf("Wrong match. expect: %q, got: %v", tt.expectError, err)
				}
			} else if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

// captureStdout runs f and returns what it wrote to the standard output.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe. err: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	f()
	_ = w.Close()

	return <-done
}

func Test_printJSON(t *testing.T) {
	t.Run("prints the indented json", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(map[string]any{"name": "route a"})
		})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		expect := "{\n  \"name\": \"route a\"\n}\n"
		if out != expect {
			t.Errorf("Wrong match.\nexpect: %q\ngot: %q", expect, out)
		}
	})

	t.Run("a value that cannot be marshaled is an error", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(make(chan int))
		})
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("Wrong match. expect: failed to marshal JSON, got: %v", err)
		}
		if out != "" {
			t.Errorf("Wrong match. expect: no output, got: %q", out)
		}
	})
}

func Test_initCommand(t *testing.T) {
	root := initCommand()

	if root.Use != "route-control" {
		t.Errorf("Wrong match. expect: route-control, got: %s", root.Use)
	}

	// the cli contract: sub commands and the flags of each one
	tests := []struct {
		path []string

		expectRunE     func(cmd *cobra.Command, args []string) error
		expectFlags    []string
		expectDefaults map[string]string
	}{
		{
			path:           []string{"route", "create"},
			expectRunE:     runRouteCreate,
			expectFlags:    []string{"customer-id", "name", "detail", "provider-id", "priority", "target"},
			expectDefaults: map[string]string{"priority": "0", "name": ""},
		},
		{
			path:        []string{"route", "get"},
			expectRunE:  runRouteGet,
			expectFlags: []string{"id"},
		},
		{
			path:           []string{"route", "list"},
			expectRunE:     runRouteList,
			expectFlags:    []string{"limit", "token", "customer-id"},
			expectDefaults: map[string]string{"limit": "100", "token": ""},
		},
		{
			path:        []string{"route", "list-by-target"},
			expectRunE:  runRouteListByTarget,
			expectFlags: []string{"customer-id", "target"},
		},
		{
			path:        []string{"route", "update"},
			expectRunE:  runRouteUpdate,
			expectFlags: []string{"id", "name", "detail", "provider-id", "priority", "target"},
		},
		{
			path:        []string{"route", "delete"},
			expectRunE:  runRouteDelete,
			expectFlags: []string{"id"},
		},
		{
			path:        []string{"route", "dialroute-list"},
			expectRunE:  runDialrouteList,
			expectFlags: []string{"customer-id", "target"},
		},
		{
			path: []string{"migrate", "status"},
		},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.path, " "), func(t *testing.T) {
			cmd, _, err := root.Find(tt.path)
			if err != nil || cmd == nil || cmd.Name() != tt.path[len(tt.path)-1] {
				t.Fatalf("Wrong match. expect: command %v, got: %v, err: %v", tt.path, cmd, err)
			}

			// the command must run its own function: the commands fail in the same way without a database
			if tt.expectRunE != nil {
				if got, expect := reflect.ValueOf(cmd.RunE).Pointer(), reflect.ValueOf(tt.expectRunE).Pointer(); got != expect {
					t.Errorf("Wrong match. command: %v, expect run function: %s, got: %s", tt.path,
						runtime.FuncForPC(expect).Name(), runtime.FuncForPC(got).Name())
				}
			}

			for _, name := range tt.expectFlags {
				if cmd.Flags().Lookup(name) == nil {
					t.Errorf("Wrong match. expect: flag %s, got: none", name)
				}
			}
			for name, value := range tt.expectDefaults {
				f := cmd.Flags().Lookup(name)
				if f == nil || f.DefValue != value {
					t.Errorf("Wrong match. flag: %s, expect default: %q, got: %v", name, value, f)
				}
			}
		})
	}
}

func Test_commands_databaseUnreachable(t *testing.T) {
	// every command opens the database first, so an unreachable database stops it before anything else is used.
	tests := []struct {
		name string
		args []string
	}{
		{"create", []string{"route", "create"}},
		{"get", []string{"route", "get"}},
		{"list", []string{"route", "list"}},
		{"list-by-target", []string{"route", "list-by-target"}},
		{"update", []string{"route", "update"}},
		{"delete", []string{"route", "delete"}},
		{"dialroute-list", []string{"route", "dialroute-list"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executeCommand(t, tt.args...)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}

			for _, expect := range []string{"failed to initialize handlers", "could not connect to the database"} {
				if !strings.Contains(err.Error(), expect) {
					t.Errorf("Wrong match. expect: message containing %q, got: %v", expect, err)
				}
			}
		})
	}
}

func Test_migrateStatus_databaseUnreachable(t *testing.T) {
	err := executeCommand(t, "migrate", "status")
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
}

func Test_initHandler_databaseUnreachable(t *testing.T) {
	res, err := initHandler()
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. expect: could not connect to the database, got: %v", err)
	}
}

func Test_initCache(t *testing.T) {
	t.Run("connects to a running cache", func(t *testing.T) {
		mr := miniredis.RunT(t)

		cfg := config.Get()
		orig := cfg.RedisAddress
		t.Cleanup(func() { cfg.RedisAddress = orig })
		cfg.RedisAddress = mr.Addr()

		res, err := initCache()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res == nil {
			t.Errorf("Wrong match. expect: handler, got: nil")
		}
	})

	t.Run("a closed cache is an error", func(t *testing.T) {
		mr := miniredis.RunT(t)
		addr := mr.Addr()
		mr.Close()

		cfg := config.Get()
		orig := cfg.RedisAddress
		t.Cleanup(func() { cfg.RedisAddress = orig })
		cfg.RedisAddress = addr

		res, err := initCache()
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
		if err == nil || !strings.Contains(err.Error(), "could not connect to the cache") {
			t.Errorf("Wrong match. expect: could not connect to the cache, got: %v", err)
		}
	})
}
