package main

import (
	"encoding/json"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestMain fixes the process-wide configuration before the first command runs:
// config.LoadGlobalConfig reads the environment once per process, and the database
// value is a string that fails in DSN parsing, so no connection is ever attempted.
func TestMain(m *testing.M) {
	for _, key := range []string{"ID", "TOKEN", "LIMIT", "EMAIL", "NOTES", "SOURCE", "COMPANY"} {
		if err := os.Unsetenv(key); err != nil {
			panic(err)
		}
	}

	fixed := map[string]string{
		"DATABASE_DSN":     "not-a-dsn",
		"REDIS_ADDRESS":    "redis.invalid:6379",
		"RABBITMQ_ADDRESS": "amqp://rabbit.invalid:5672",
	}
	for key, val := range fixed {
		if err := os.Setenv(key, val); err != nil {
			panic(err)
		}
	}

	os.Exit(m.Run())
}

// execute builds a fresh command tree and runs it with the given arguments.
func execute(args ...string) error {
	root := initCommand()
	root.SetArgs(args)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceUsage = true
	root.SilenceErrors = true

	return root.Execute()
}

func subcommandNames(cmd *cobra.Command) []string {
	res := []string{}
	for _, c := range cmd.Commands() {
		res = append(res, c.Name())
	}
	sort.Strings(res)

	return res
}

func Test_initCommand_tree(t *testing.T) {
	root := initCommand()

	if root.Use != "contact-control" {
		t.Errorf("Wrong match. expect: contact-control, got: %s", root.Use)
	}

	if got, expect := subcommandNames(root), []string{"contact", "migrate"}; !reflect.DeepEqual(got, expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, got)
	}

	cmdContact, _, err := root.Find([]string{"contact"})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	expectContact := []string{"create", "delete", "get", "list", "lookup", "update"}
	if got := subcommandNames(cmdContact); !reflect.DeepEqual(got, expectContact) {
		t.Errorf("Wrong match. expect: %v, got: %v", expectContact, got)
	}

	cmdMigrate, _, err := root.Find([]string{"migrate"})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	expectMigrate := []string{"down", "force", "status", "up"}
	if got := subcommandNames(cmdMigrate); !reflect.DeepEqual(got, expectMigrate) {
		t.Errorf("Wrong match. expect: %v, got: %v", expectMigrate, got)
	}

	for _, name := range []string{"database_dsn", "redis_address", "redis_database", "redis_password", "rabbitmq_address"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("Wrong match. expect: persistent flag %s, got: none", name)
		}
	}
}

func Test_command_constructors(t *testing.T) {
	tests := []struct {
		name string

		build func() *cobra.Command

		expectUse   string
		expectFlags map[string]string
	}{
		{
			name:      "create",
			build:     cmdCreate,
			expectUse: "create",
			expectFlags: map[string]string{
				"customer-id": "", "first-name": "", "last-name": "", "display-name": "", "company": "",
				"job-title": "", "source": "", "external-id": "", "notes": "",
			},
		},
		{
			name:        "get",
			build:       cmdGet,
			expectUse:   "get",
			expectFlags: map[string]string{"id": ""},
		},
		{
			name:        "list",
			build:       cmdList,
			expectUse:   "list",
			expectFlags: map[string]string{"limit": "100", "token": "", "customer-id": ""},
		},
		{
			name:      "update",
			build:     cmdUpdate,
			expectUse: "update",
			expectFlags: map[string]string{
				"id": "", "first-name": "", "last-name": "", "display-name": "", "company": "",
				"job-title": "", "external-id": "", "notes": "",
			},
		},
		{
			name:        "delete",
			build:       cmdDelete,
			expectUse:   "delete",
			expectFlags: map[string]string{"id": ""},
		},
		{
			name:        "lookup",
			build:       cmdLookup,
			expectUse:   "lookup",
			expectFlags: map[string]string{"customer-id": "", "phone-e164": "", "email": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.build()

			if cmd.Use != tt.expectUse {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectUse, cmd.Use)
			}
			if cmd.RunE == nil {
				t.Errorf("Wrong match. expect: RunE set, got: nil")
			}

			for name, expectDefault := range tt.expectFlags {
				f := cmd.Flags().Lookup(name)
				if f == nil {
					t.Errorf("Wrong match. expect: flag %s, got: none", name)
					continue
				}
				if f.DefValue != expectDefault {
					t.Errorf("Wrong match. flag: %s, expect default: %q, got: %q", name, expectDefault, f.DefValue)
				}
			}

			// every flag renders as one usage line, so the line count detects flags the table does not know
			if got := strings.Count(cmd.Flags().FlagUsages(), "\n"); got != len(tt.expectFlags) {
				t.Errorf("Wrong match. expect: %d flags, got: %d usage lines", len(tt.expectFlags), got)
			}
		})
	}
}

func Test_create_argumentValidation(t *testing.T) {
	customerID := "0a1b2c3d-1659-4000-8000-000000000001"

	tests := []struct {
		name string
		args []string

		expectTexts    []string
		expectNotTexts []string
	}{
		{
			name:        "customer id missing",
			args:        []string{"contact", "create"},
			expectTexts: []string{"failed to resolve customer ID", "Customer ID is required"},
		},
		{
			name:        "customer id malformed",
			args:        []string{"contact", "create", "--customer-id", "not-a-uuid"},
			expectTexts: []string{"failed to resolve customer ID", "invalid format for Customer ID", "'not-a-uuid' is not a valid UUID"},
		},
		{
			name:        "nil uuid is rejected",
			args:        []string{"contact", "create", "--customer-id", "00000000-0000-0000-0000-000000000000"},
			expectTexts: []string{"invalid format for Customer ID"},
		},
		{
			name:           "valid customer id reaches the handler setup and fails in dsn parsing",
			args:           []string{"contact", "create", "--customer-id", customerID, "--first-name", "Alice"},
			expectTexts:    []string{"failed to initialize handlers", "could not connect to the database", "database open error"},
			expectNotTexts: []string{"database ping error", "failed to resolve customer ID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := execute(tt.args...)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}

			for _, text := range tt.expectTexts {
				if !strings.Contains(err.Error(), text) {
					t.Errorf("Wrong match. expect: %q in error, got: %v", text, err)
				}
			}
			for _, text := range tt.expectNotTexts {
				if strings.Contains(err.Error(), text) {
					t.Errorf("Wrong match. expect: no %q in error, got: %v", text, err)
				}
			}
		})
	}
}

// The commands other than create initialize their handlers before validating the arguments,
// so without a reachable database they all stop at the wrapped initialization failure.
func Test_run_initHandlerFailure(t *testing.T) {
	id := "0a1b2c3d-1659-4000-8000-000000000002"
	customerID := "0a1b2c3d-1659-4000-8000-000000000001"

	tests := []struct {
		name string
		args []string
	}{
		{name: "get", args: []string{"contact", "get", "--id", id}},
		{name: "list", args: []string{"contact", "list", "--customer-id", customerID, "--limit", "5"}},
		{name: "update", args: []string{"contact", "update", "--id", id, "--notes", "synthetic"}},
		{name: "delete", args: []string{"contact", "delete", "--id", id}},
		{name: "lookup", args: []string{"contact", "lookup", "--customer-id", customerID, "--email", "alice@example.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := execute(tt.args...)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}

			for _, text := range []string{"failed to initialize handlers", "could not connect to the database", "database open error"} {
				if !strings.Contains(err.Error(), text) {
					t.Errorf("Wrong match. expect: %q in error, got: %v", text, err)
				}
			}
			if strings.Contains(err.Error(), "database ping error") {
				t.Errorf("Wrong match. expect: failure before any ping, got: %v", err)
			}
		})
	}
}

func Test_migrate_statusDSNFailure(t *testing.T) {
	err := execute("migrate", "status")
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !strings.Contains(err.Error(), "could not initialize the migrator") {
		t.Errorf("Wrong match. expect: migrator initialization failure, got: %v", err)
	}
}

// captureStdout runs fn with os.Stdout replaced by a pipe that is drained concurrently.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Wrong match. expect: pipe, got: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(r)
		done <- string(data)
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("Wrong match. expect: close, got: %v", err)
	}
	os.Stdout = orig

	return <-done
}

func Test_printJSON(t *testing.T) {
	type sample struct {
		Name string `json:"name"`
		Tags []int  `json:"tags"`
	}

	tests := []struct {
		name string

		value any

		expectOut string
	}{
		{name: "struct is indented", value: sample{Name: "alice", Tags: []int{1, 2}}, expectOut: "{\n  \"name\": \"alice\",\n  \"tags\": [\n    1,\n    2\n  ]\n}\n"},
		{name: "nil is null", value: nil, expectOut: "null\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			out := captureStdout(t, func() {
				err = printJSON(tt.value)
			})

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if out != tt.expectOut {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectOut, out)
			}
			if tt.value != nil && !json.Valid([]byte(out)) {
				t.Errorf("Wrong match. expect: valid json, got: %q", out)
			}
		})
	}
}

func Test_printJSON_marshalError(t *testing.T) {
	var err error
	out := captureStdout(t, func() {
		err = printJSON(make(chan int))
	})

	if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
		t.Errorf("Wrong match. expect: marshal failure, got: %v", err)
	}
	if out != "" {
		t.Errorf("Wrong match. expect: no output, got: %q", out)
	}
}
