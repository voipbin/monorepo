package main

// NOTE: the tests in this package must not use t.Parallel. They read and write
// process-wide state (viper, the config singleton, logrus and os.Stdout).

import (
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-flow-manager/internal/config"
	"monorepo/bin-flow-manager/models/action"
)

// resetViper clears the global viper state now and again when the test ends.
func resetViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
}

// useMalformedDSN makes the database connection fail while the DSN is parsed,
// before any network access is attempted.
func useMalformedDSN(t *testing.T) {
	t.Helper()
	cfg := config.Get()
	prev := cfg.DatabaseDSN
	cfg.DatabaseDSN = "malformed-dsn-without-separator"
	t.Cleanup(func() { cfg.DatabaseDSN = prev })
}

func Test_resolveUUID(t *testing.T) {
	validID := "5f0a1b2c-3d4e-4f60-8a7b-9c0d1e2f3a4b"

	tests := []struct {
		name string

		value string

		expectRes     uuid.UUID
		expectErrPart string
	}{
		{
			name:      "valid uuid",
			value:     validID,
			expectRes: uuid.FromStringOrNil(validID),
		},
		{
			name:          "empty value",
			value:         "",
			expectRes:     uuid.Nil,
			expectErrPart: "Flow ID is required",
		},
		{
			name:          "not a uuid",
			value:         "not-a-uuid",
			expectRes:     uuid.Nil,
			expectErrPart: "invalid format for Flow ID: 'not-a-uuid' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			viper.Set("id", tt.value)

			res, err := resolveUUID("id", "Flow ID")
			if tt.expectErrPart == "" && err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if tt.expectErrPart != "" && (err == nil || err.Error() != tt.expectErrPart) {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectErrPart, err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_parseActions(t *testing.T) {
	actionID := "6a0a1b2c-3d4e-4f60-8a7b-9c0d1e2f3a4b"

	tests := []struct {
		name string

		input string

		expectRes []action.Action
		expectErr bool
	}{
		{
			name:      "empty string returns empty list",
			input:     "",
			expectRes: []action.Action{},
		},
		{
			name:  "valid json array",
			input: `[{"id":"` + actionID + `","type":"answer"},{"type":"talk","option":{"text":"hello"}}]`,
			expectRes: []action.Action{
				{ID: uuid.FromStringOrNil(actionID), Type: action.TypeAnswer},
				{Type: action.TypeTalk, Option: map[string]any{"text": "hello"}},
			},
		},
		{
			name:      "json object instead of array",
			input:     `{"type":"answer"}`,
			expectErr: true,
		},
		{
			name:      "broken json",
			input:     `[{"type":`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parseActions(tt.input)
			if tt.expectErr {
				if err == nil || !strings.Contains(err.Error(), "failed to parse actions JSON") {
					t.Fatalf("Wrong match. expect: parse error, got: %v", err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil result, got: %v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// captureStdout runs fn while os.Stdout is redirected and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create pipe: %v", err)
	}

	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()
	_ = w.Close()
	return <-done
}

func Test_printJSON(t *testing.T) {
	t.Run("indented output", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(map[string]any{"name": "test", "count": 2})
		})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		expect := "{\n  \"count\": 2,\n  \"name\": \"test\"\n}\n"
		if out != expect {
			t.Errorf("Wrong match.\nexpect: %q\ngot: %q", expect, out)
		}
	})

	t.Run("value that cannot be marshaled", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(make(chan int))
		})
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("Wrong match. expect: marshal error, got: %v", err)
		}
		if out != "" {
			t.Errorf("Wrong match. expect: no output, got: %q", out)
		}
	})
}

func Test_commandConstructors(t *testing.T) {
	tests := []struct {
		name string

		build func() *cobra.Command

		expectUse   string
		expectFlags map[string]string // flag name -> default value
	}{
		{
			name:      "create",
			build:     cmdCreate,
			expectUse: "create",
			expectFlags: map[string]string{
				"customer-id":         "",
				"type":                "flow",
				"name":                "",
				"detail":              "",
				"persist":             "false",
				"actions":             "",
				"on-complete-flow-id": "",
			},
		},
		{
			name:        "get",
			build:       cmdGet,
			expectUse:   "get",
			expectFlags: map[string]string{"id": ""},
		},
		{
			name:      "list",
			build:     cmdList,
			expectUse: "list",
			expectFlags: map[string]string{
				"limit":       "100",
				"token":       "",
				"customer-id": "",
				"type":        "",
			},
		},
		{
			name:      "update",
			build:     cmdUpdate,
			expectUse: "update",
			expectFlags: map[string]string{
				"id":                  "",
				"name":                "",
				"detail":              "",
				"actions":             "",
				"on-complete-flow-id": "",
			},
		},
		{
			name:        "update-actions",
			build:       cmdUpdateActions,
			expectUse:   "update-actions",
			expectFlags: map[string]string{"id": "", "actions": ""},
		},
		{
			name:        "delete",
			build:       cmdDelete,
			expectUse:   "delete",
			expectFlags: map[string]string{"id": ""},
		},
		{
			name:        "action-get",
			build:       cmdActionGet,
			expectUse:   "action-get",
			expectFlags: map[string]string{"flow-id": "", "action-id": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.build()
			if cmd.Use != tt.expectUse {
				t.Errorf("Wrong use. expect: %s, got: %s", tt.expectUse, cmd.Use)
			}
			if cmd.RunE == nil {
				t.Errorf("Wrong match. expect: RunE set, got: nil")
			}

			for name, def := range tt.expectFlags {
				f := cmd.Flags().Lookup(name)
				if f == nil {
					t.Errorf("Wrong match. expect: flag %s, got: none", name)
					continue
				}
				if f.DefValue != def {
					t.Errorf("Wrong default for flag %s. expect: %q, got: %q", name, def, f.DefValue)
				}
			}

			// every registered flag is listed in the usage, one per line.
			usage := strings.TrimSpace(cmd.Flags().FlagUsages())
			if got := len(strings.Split(usage, "\n")); got != len(tt.expectFlags) {
				t.Errorf("Wrong flag count. expect: %d, got: %d\n%s", len(tt.expectFlags), got, usage)
			}
		})
	}
}

func Test_initCommand(t *testing.T) {
	resetViper(t)

	// initCommand switches the global logrus formatter and level.
	prevFormatter := logrus.StandardLogger().Formatter
	prevLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(prevFormatter)
		logrus.SetLevel(prevLevel)
	})

	root := initCommand()
	if root.Use != "flow-control" {
		t.Errorf("Wrong root use. expect: flow-control, got: %s", root.Use)
	}

	for _, name := range []string{"rabbitmq_address", "database_dsn", "redis_address", "redis_database"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("Wrong match. expect: persistent flag %s, got: none", name)
		}
	}

	gotRoot := []string{}
	for _, c := range root.Commands() {
		gotRoot = append(gotRoot, c.Name())
	}
	if !reflect.DeepEqual(sortedCopy(gotRoot), []string{"flow", "migrate"}) {
		t.Errorf("Wrong root subcommands. got: %v", gotRoot)
	}

	flowCmd, _, err := root.Find([]string{"flow"})
	if err != nil || flowCmd.Name() != "flow" {
		t.Fatalf("Wrong match. expect: flow command, got: %v, %v", flowCmd, err)
	}
	gotFlow := []string{}
	for _, c := range flowCmd.Commands() {
		gotFlow = append(gotFlow, c.Name())
	}
	expectFlow := []string{"action-get", "create", "delete", "get", "list", "update", "update-actions"}
	if !reflect.DeepEqual(sortedCopy(gotFlow), expectFlow) {
		t.Errorf("Wrong flow subcommands.\nexpect: %v\ngot: %v", expectFlow, gotFlow)
	}
}

func Test_runCreate_validation(t *testing.T) {
	validCustomer := "7b0a1b2c-3d4e-4f60-8a7b-9c0d1e2f3a4b"

	tests := []struct {
		name string

		values map[string]string

		expectErrParts []string
	}{
		{
			name:           "customer id missing",
			values:         map[string]string{},
			expectErrParts: []string{"failed to resolve customer ID", "Customer ID is required"},
		},
		{
			name:           "customer id malformed",
			values:         map[string]string{"customer-id": "xyz"},
			expectErrParts: []string{"failed to resolve customer ID", "'xyz' is not a valid UUID"},
		},
		{
			name:           "name missing",
			values:         map[string]string{"customer-id": validCustomer},
			expectErrParts: []string{"name is required"},
		},
		{
			name:           "actions malformed",
			values:         map[string]string{"customer-id": validCustomer, "name": "test", "actions": "[{"},
			expectErrParts: []string{"failed to parse actions", "failed to parse actions JSON"},
		},
		{
			name:           "valid input stops at handler initialization",
			values:         map[string]string{"customer-id": validCustomer, "name": "test", "actions": "[]"},
			expectErrParts: []string{"failed to initialize handlers", "could not connect to the database"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			useMalformedDSN(t)
			for k, v := range tt.values {
				viper.Set(k, v)
			}

			err := runCreate(nil, nil)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			for _, part := range tt.expectErrParts {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("Wrong match. expect error containing %q, got: %v", part, err)
				}
			}
		})
	}
}

// Test_run_initializationFailure verifies that every command that needs the
// handlers stops with the initialization error when the database can not be
// reached, and reports the underlying cause.
func Test_run_initializationFailure(t *testing.T) {
	tests := []struct {
		name string
		run  func(*cobra.Command, []string) error
	}{
		{"get", runGet},
		{"list", runList},
		{"update", runUpdate},
		{"update-actions", runUpdateActions},
		{"delete", runDelete},
		{"action-get", runActionGet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			useMalformedDSN(t)

			err := tt.run(nil, nil)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			for _, part := range []string{"failed to initialize handlers", "could not connect to the database", "invalid DSN"} {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("Wrong match. expect error containing %q, got: %v", part, err)
				}
			}
		})
	}
}

func Test_initHandler_databaseError(t *testing.T) {
	useMalformedDSN(t)

	res, err := initHandler()
	if res != nil {
		t.Errorf("Wrong match. expect: nil handler, got: %v", res)
	}
	if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. expect: database error, got: %v", err)
	}
}

func sortedCopy(in []string) []string {
	out := append([]string{}, in...)
	sort.Strings(out)
	return out
}
