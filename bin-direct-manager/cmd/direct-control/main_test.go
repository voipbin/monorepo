package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-direct-manager/internal/config"
)

// resetState isolates a test from the process wide viper and config state. Tests in this
// package must not run in parallel.
func resetState(t *testing.T) {
	t.Helper()

	prev := *config.Get()
	viper.Reset()

	t.Cleanup(func() {
		viper.Reset()
		*config.Get() = prev
	})
}

// captureStdout returns what fn prints to the standard output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create a pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	fn()

	_ = w.Close()
	os.Stdout = orig
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("could not read the captured output: %v", err)
	}

	return string(out)
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name string

		value string

		expectRes uuid.UUID
		expectErr string
	}{
		{
			name:      "valid",
			value:     "a6b3cf48-2a4b-11ee-b574-2bad4f039ce5",
			expectRes: uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),
		},
		{
			name:      "empty",
			value:     "",
			expectRes: uuid.Nil,
			expectErr: "Direct ID is required",
		},
		{
			name:      "not a uuid",
			value:     "not-a-uuid",
			expectRes: uuid.Nil,
			expectErr: "invalid format for Direct ID: 'not-a-uuid' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetState(t)
			viper.Set("id", tt.value)

			res, err := resolveUUID("id", "Direct ID")
			if res != tt.expectRes {
				t.Errorf("resolveUUID() = %v, want %v", res, tt.expectRes)
			}
			if tt.expectErr == "" {
				if err != nil {
					t.Errorf("resolveUUID() error = %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.expectErr {
				t.Errorf("resolveUUID() error = %v, want %q", err, tt.expectErr)
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	t.Run("indents with two spaces", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(map[string]any{"hash": "direct.abcdef123456"})
		})
		if err != nil {
			t.Fatalf("printJSON() error = %v", err)
		}

		expect := "{\n  \"hash\": \"direct.abcdef123456\"\n}\n"
		if out != expect {
			t.Errorf("printJSON() wrote %q, want %q", out, expect)
		}
	})

	t.Run("a value that cannot be marshaled is an error and prints nothing", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(make(chan int))
		})
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("printJSON() error = %v, want the marshal context", err)
		}
		if out != "" {
			t.Errorf("printJSON() wrote %q, want nothing", out)
		}
	})
}

func Test_initCommand(t *testing.T) {
	resetState(t)

	root := initCommand()

	if root.Use != "direct-control" {
		t.Errorf("root Use = %q, want direct-control", root.Use)
	}

	// the connection settings are persistent flags of the root
	for _, name := range []string{"database_dsn", "redis_address", "rabbitmq_address", "redis_database"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("root persistent flag %q is not registered", name)
		}
	}

	subs := map[string]*cobra.Command{}
	for _, c := range root.Commands() {
		subs[c.Name()] = c
	}
	if subs["direct"] == nil || subs["migrate"] == nil {
		t.Fatalf("root commands = %v, want direct and migrate", subs)
	}

	ops := map[string][]string{
		"create":     {"customer-id", "resource-type", "resource-id"},
		"get":        {"id"},
		"list":       {"limit", "token", "customer-id"},
		"delete":     {"id"},
		"regenerate": {"id"},
	}
	found := map[string]*cobra.Command{}
	for _, c := range subs["direct"].Commands() {
		found[c.Name()] = c
	}
	if len(found) != len(ops) {
		t.Errorf("direct subcommands = %d, want %d", len(found), len(ops))
	}
	for op, flags := range ops {
		c := found[op]
		if c == nil {
			t.Errorf("direct %s is not registered", op)
			continue
		}
		if c.RunE == nil {
			t.Errorf("direct %s has no RunE", op)
		}
		for _, f := range flags {
			if c.Flags().Lookup(f) == nil {
				t.Errorf("direct %s flag %q is not registered", op, f)
			}
		}
	}

	if got := found["list"].Flags().Lookup("limit").DefValue; got != "100" {
		t.Errorf("list limit default = %q, want 100", got)
	}
}

func Test_runCreate_validation(t *testing.T) {
	validCustomer := "92883d56-7fe3-11ec-8931-37d08180a2b9"
	validResource := "c31676f0-4e69-11ec-afe3-77ba49fae527"

	// the checks run in this order and every failure returns before any connection is
	// opened, so these cases need neither a database nor a cache.
	tests := []struct {
		name string

		customerID   string
		resourceType string
		resourceID   string

		expectErr string
	}{
		{"customer id missing", "", "extension", validResource, "failed to resolve customer ID: Customer ID is required"},
		{"customer id invalid", "bad", "extension", validResource, "failed to resolve customer ID: invalid format for Customer ID"},
		{"resource type missing", validCustomer, "", validResource, "resource-type is required"},
		{"resource id missing", validCustomer, "extension", "", "failed to resolve resource ID: Resource ID is required"},
		{"resource id invalid", validCustomer, "extension", "bad", "failed to resolve resource ID: invalid format for Resource ID"},
		{"customer is checked before the other fields", "", "", "", "failed to resolve customer ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetState(t)
			viper.Set("customer-id", tt.customerID)
			viper.Set("resource-type", tt.resourceType)
			viper.Set("resource-id", tt.resourceID)

			err := runCreate(&cobra.Command{}, nil)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("runCreate() error = %v, want it to contain %q", err, tt.expectErr)
			}
		})
	}
}

func Test_initCache(t *testing.T) {
	t.Run("connects to a running cache", func(t *testing.T) {
		resetState(t)
		mr := miniredis.RunT(t)
		*config.Get() = config.Config{RedisAddress: mr.Addr()}

		res, err := initCache()
		if err != nil || res == nil {
			t.Errorf("initCache() = %v, %v, want a connected handler", res, err)
		}
	})

	t.Run("wrong password is reported with the context", func(t *testing.T) {
		resetState(t)
		mr := miniredis.RunT(t)
		mr.RequireAuth(strings.Repeat("r", 12))
		*config.Get() = config.Config{RedisAddress: mr.Addr()}
		config.Get().RedisPassword = strings.Repeat("w", 12)

		res, err := initCache()
		if res != nil {
			t.Errorf("initCache() = %v, want nil", res)
		}
		if err == nil || !strings.Contains(err.Error(), "could not connect to the cache") {
			t.Errorf("initCache() error = %v, want the connect context", err)
		}
	})

	t.Run("stopped cache is reported with the context", func(t *testing.T) {
		resetState(t)
		mr := miniredis.RunT(t)
		addr := mr.Addr()
		mr.Close()
		*config.Get() = config.Config{RedisAddress: addr}

		res, err := initCache()
		if res != nil || err == nil || !strings.Contains(err.Error(), "could not connect to the cache") {
			t.Errorf("initCache() = %v, %v, want nil and the connect context", res, err)
		}
	})
}

// invalidDSN is rejected by the database driver while the connection is opened, so the
// tests below fail at the connect step without any network access.
const invalidDSN = "not-a-dsn"

func Test_run_initHandlerFailure(t *testing.T) {
	validCustomer := "92883d56-7fe3-11ec-8931-37d08180a2b9"
	validResource := "c31676f0-4e69-11ec-afe3-77ba49fae527"

	tests := []struct {
		name string

		run func(cmd *cobra.Command, args []string) error
	}{
		{"create", runCreate},
		{"get", runGet},
		{"list", runList},
		{"delete", runDelete},
		{"regenerate", runRegenerate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetState(t)
			*config.Get() = config.Config{DatabaseDSN: invalidDSN}

			// runCreate validates these before it connects; the other commands connect first
			viper.Set("customer-id", validCustomer)
			viper.Set("resource-type", "extension")
			viper.Set("resource-id", validResource)

			var err error
			out := captureStdout(t, func() {
				err = tt.run(&cobra.Command{}, nil)
			})

			if err == nil {
				t.Fatalf("run() error = nil, want the connect failure")
			}
			for _, want := range []string{"failed to initialize handlers", "could not connect to the database", "database open error"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("run() error = %q, want it to contain %q", err.Error(), want)
				}
			}
			if out != "" {
				t.Errorf("run() wrote %q, want nothing", out)
			}
		})
	}
}

func Test_initCommand_execute(t *testing.T) {
	validCustomer := "92883d56-7fe3-11ec-8931-37d08180a2b9"
	validResource := "c31676f0-4e69-11ec-afe3-77ba49fae527"

	tests := []struct {
		name string

		args []string

		expectErr string
	}{
		{
			// runCreate reads its values from viper before it connects, so this only reaches the
			// connect step when the pre-run hook has bound the flags
			name: "direct create binds the flags before the run",
			args: []string{
				"direct", "create",
				"--customer-id", validCustomer,
				"--resource-type", "extension",
				"--resource-id", validResource,
				"--database_dsn", invalidDSN,
			},
			expectErr: "could not connect to the database",
		},
		{
			name:      "migrate reports the unparsable dsn from the migrator",
			args:      []string{"migrate", "status", "--database_dsn", invalidDSN},
			expectErr: "could not parse the database dsn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetState(t)
			// the config load of the pre-run hook happens once per process, so the invalid dsn is
			// set here as well as in the arguments; it is invalid in either state
			*config.Get() = config.Config{DatabaseDSN: invalidDSN}

			root := initCommand()
			root.SilenceUsage = true
			root.SilenceErrors = true
			buf := &bytes.Buffer{}
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			var err error
			out := captureStdout(t, func() {
				err = root.Execute()
			})

			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Execute() error = %v, want it to contain %q", err, tt.expectErr)
			}
			if out != "" || buf.Len() != 0 {
				t.Errorf("Execute() printed stdout %q and command output %q, want nothing", out, buf.String())
			}
		})
	}
}
