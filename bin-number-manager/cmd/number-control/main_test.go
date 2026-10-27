package main

import (
	"encoding/json"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resetViper clears the process wide viper state around a test.
func resetViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
}

// captureStdout runs f and returns what it wrote to os.Stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe. err: %v", err)
	}

	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	f()

	if errClose := w.Close(); errClose != nil {
		t.Fatalf("could not close the pipe. err: %v", errClose)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("could not read the pipe. err: %v", err)
	}
	return string(out)
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name string

		value string

		expectErr string
		expectRes string
	}{
		{
			name:      "normal",
			value:     "5b1e1b3e-8c4a-4f7e-9d0a-2f6a3b9c7d11",
			expectRes: "5b1e1b3e-8c4a-4f7e-9d0a-2f6a3b9c7d11",
		},
		{
			name:      "empty",
			value:     "",
			expectErr: "Number ID is required",
		},
		{
			name:      "malformed",
			value:     "not-a-uuid",
			expectErr: "invalid format for Number ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:      "nil uuid is rejected as malformed",
			value:     "00000000-0000-0000-0000-000000000000",
			expectErr: "is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			viper.Set("id", tt.value)

			res, err := resolveUUID("id", "Number ID")
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.String() != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res.String())
			}
		})
	}
}

func Test_resolveString(t *testing.T) {
	tests := []struct {
		name string

		value string

		expectErr string
	}{
		{
			name:  "normal",
			value: "US",
		},
		{
			name:      "empty",
			value:     "",
			expectErr: "Country code is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			viper.Set("country-code", tt.value)

			res, err := resolveString("country-code", "Country code")
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Fatalf("Wrong match. expect error: %s, got: %v", tt.expectErr, err)
				}
				if res != "" {
					t.Errorf("Wrong match. expect empty result, got: %s", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res != tt.value {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.value, res)
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	type sample struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	t.Run("normal", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(&sample{Name: "n1", Count: 3})
		})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		if !strings.Contains(out, "\n  \"name\"") {
			t.Errorf("Wrong match. expect indented output, got: %q", out)
		}

		var got sample
		if errUnmarshal := json.Unmarshal([]byte(out), &got); errUnmarshal != nil {
			t.Fatalf("could not parse the output. out: %q, err: %v", out, errUnmarshal)
		}
		if !reflect.DeepEqual(got, sample{Name: "n1", Count: 3}) {
			t.Errorf("Wrong match. expect: {n1 3}, got: %v", got)
		}
	})

	t.Run("unmarshalable value", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(make(chan int))
		})
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Fatalf("Wrong match. expect error containing: failed to marshal JSON, got: %v", err)
		}
		if out != "" {
			t.Errorf("Wrong match. expect no output, got: %q", out)
		}
	})
}

func Test_commandFlags(t *testing.T) {
	type flagSpec struct {
		name     string
		typeName string
		def      string
	}

	tests := []struct {
		name string

		build func() *cobra.Command

		expectUse   string
		expectFlags []flagSpec
	}{
		{
			name:      "create",
			build:     cmdCreate,
			expectUse: "create",
			expectFlags: []flagSpec{
				{"customer-id", "string", ""},
				{"number", "string", ""},
				{"call-flow-id", "string", ""},
				{"message-flow-id", "string", ""},
				{"name", "string", ""},
				{"detail", "string", ""},
				{"virtual-number", "bool", "false"},
			},
		},
		{
			name:        "get",
			build:       cmdGet,
			expectUse:   "get",
			expectFlags: []flagSpec{{"id", "string", ""}},
		},
		{
			name:      "list",
			build:     cmdList,
			expectUse: "list",
			expectFlags: []flagSpec{
				{"limit", "int", "100"},
				{"token", "string", ""},
				{"customer-id", "string", ""},
			},
		},
		{
			name:      "get-available",
			build:     cmdGetAvailable,
			expectUse: "get-available",
			expectFlags: []flagSpec{
				{"country-code", "string", ""},
				{"limit", "uint", "10"},
				{"virtual", "bool", "false"},
			},
		},
		{
			name:      "update",
			build:     cmdUpdate,
			expectUse: "update",
			expectFlags: []flagSpec{
				{"id", "string", ""},
				{"name", "string", ""},
				{"detail", "string", ""},
				{"call-flow-id", "string", ""},
				{"message-flow-id", "string", ""},
				{"status", "string", ""},
			},
		},
		{
			name:        "delete",
			build:       cmdDelete,
			expectUse:   "delete",
			expectFlags: []flagSpec{{"id", "string", ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.build()

			if cmd.Use != tt.expectUse {
				t.Errorf("Wrong match. expect use: %s, got: %s", tt.expectUse, cmd.Use)
			}
			if cmd.RunE == nil {
				t.Errorf("Wrong match. expect RunE to be set")
			}

			for _, want := range tt.expectFlags {
				f := cmd.Flags().Lookup(want.name)
				if f == nil {
					t.Errorf("Wrong match. flag %s is missing", want.name)
					continue
				}
				if f.Value.Type() != want.typeName {
					t.Errorf("Wrong match. flag %s expect type: %s, got: %s", want.name, want.typeName, f.Value.Type())
				}
				if f.DefValue != want.def {
					t.Errorf("Wrong match. flag %s expect default: %q, got: %q", want.name, want.def, f.DefValue)
				}
			}
		})
	}
}

func Test_initCommand(t *testing.T) {
	resetViper(t)
	origLevel := logrus.GetLevel()
	origFormatter := logrus.StandardLogger().Formatter
	t.Cleanup(func() {
		logrus.SetLevel(origLevel)
		logrus.SetFormatter(origFormatter)
	})

	cmd := initCommand()

	if cmd.Use != "number-control" {
		t.Errorf("Wrong match. expect: number-control, got: %s", cmd.Use)
	}

	names := []string{}
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, []string{"migrate", "number"}) {
		t.Errorf("Wrong match. expect: [migrate number], got: %v", names)
	}

	numberCmd, _, err := cmd.Find([]string{"number"})
	if err != nil {
		t.Fatalf("could not find the number command. err: %v", err)
	}
	subs := []string{}
	for _, c := range numberCmd.Commands() {
		subs = append(subs, c.Name())
	}
	sort.Strings(subs)
	expectSubs := []string{"create", "delete", "get", "get-available", "list", "update"}
	if !reflect.DeepEqual(subs, expectSubs) {
		t.Errorf("Wrong match. expect: %v, got: %v", expectSubs, subs)
	}

	for _, flagName := range []string{"database_dsn", "redis_address", "rabbitmq_address", "number_secret_encryption_keys"} {
		if cmd.PersistentFlags().Lookup(flagName) == nil {
			t.Errorf("Wrong match. persistent flag %s is missing", flagName)
		}
	}

	migrateCmd, _, err := cmd.Find([]string{"migrate", "status"})
	if err != nil || migrateCmd.Name() != "status" {
		t.Errorf("Wrong match. expect the migrate status command, got: %v, err: %v", migrateCmd, err)
	}
}

// Test_initCommand_execute runs a real subcommand without arguments. The pre-run hook binds the
// flags and the run function stops at its input validation, before any connection is opened.
func Test_initCommand_execute(t *testing.T) {
	tests := []struct {
		name string

		args []string

		expectErr string
	}{
		{
			name:      "get without id",
			args:      []string{"number", "get"},
			expectErr: "Number ID is required",
		},
		{
			name:      "get with malformed id",
			args:      []string{"number", "get", "--id", "not-a-uuid"},
			expectErr: "is not a valid UUID",
		},
		{
			name:      "list without customer id",
			args:      []string{"number", "list"},
			expectErr: "Customer ID is required",
		},
		{
			name:      "delete without id",
			args:      []string{"number", "delete"},
			expectErr: "Number ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			// AutomaticEnv maps these keys to the environment, so keep a developer shell out of the test
			for _, key := range []string{"ID", "TOKEN", "LIMIT", "CUSTOMER-ID", "CUSTOMER_ID", "DATABASE_DSN", "REDIS_ADDRESS", "RABBITMQ_ADDRESS"} {
				t.Setenv(key, "")
			}

			origLevel := logrus.GetLevel()
			origFormatter := logrus.StandardLogger().Formatter
			t.Cleanup(func() {
				logrus.SetLevel(origLevel)
				logrus.SetFormatter(origFormatter)
			})

			cmd := initCommand()
			cmd.SetArgs(tt.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_run_validation(t *testing.T) {
	const validID = "5b1e1b3e-8c4a-4f7e-9d0a-2f6a3b9c7d11"

	tests := []struct {
		name string

		run  func(cmd *cobra.Command, args []string) error
		keys map[string]string

		expectErr string
	}{
		{
			name:      "create without customer id",
			run:       runCreate,
			keys:      map[string]string{"number": "+15550100001"},
			expectErr: "invalid customer ID: Customer ID is required",
		},
		{
			name:      "create with malformed customer id",
			run:       runCreate,
			keys:      map[string]string{"customer-id": "x", "number": "+15550100001"},
			expectErr: "invalid customer ID: invalid format for Customer ID",
		},
		{
			name:      "create without number",
			run:       runCreate,
			keys:      map[string]string{"customer-id": validID},
			expectErr: "invalid phone number: Phone number is required",
		},
		{
			name:      "get without id",
			run:       runGet,
			keys:      map[string]string{},
			expectErr: "invalid number ID: Number ID is required",
		},
		{
			name:      "get with malformed id",
			run:       runGet,
			keys:      map[string]string{"id": "x"},
			expectErr: "invalid number ID: invalid format for Number ID",
		},
		{
			name:      "list without customer id",
			run:       runList,
			keys:      map[string]string{},
			expectErr: "invalid customer ID: Customer ID is required",
		},
		{
			name:      "list with malformed customer id",
			run:       runList,
			keys:      map[string]string{"customer-id": "x"},
			expectErr: "invalid customer ID: invalid format for Customer ID",
		},
		{
			name:      "update without id",
			run:       runUpdate,
			keys:      map[string]string{"name": "n1"},
			expectErr: "invalid number ID: Number ID is required",
		},
		{
			name:      "update with malformed id",
			run:       runUpdate,
			keys:      map[string]string{"id": "x"},
			expectErr: "invalid number ID: invalid format for Number ID",
		},
		{
			name:      "delete without id",
			run:       runDelete,
			keys:      map[string]string{},
			expectErr: "invalid number ID: Number ID is required",
		},
		{
			name:      "delete with malformed id",
			run:       runDelete,
			keys:      map[string]string{"id": "x"},
			expectErr: "invalid number ID: invalid format for Number ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			for k, v := range tt.keys {
				viper.Set(k, v)
			}

			err := tt.run(&cobra.Command{}, nil)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
			}
		})
	}
}
