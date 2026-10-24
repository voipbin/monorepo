package main

import (
	"bytes"
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

// resetViper clears the global viper state before and after a test, so the flag
// values and the environment bindings of one test do not leak into the next.
func resetViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
}

// captureStdout runs fn and returns what it wrote to os.Stdout. The original
// os.Stdout is restored right after the swap is registered.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe: %v", err)
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

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	restore()
	out := <-done
	_ = r.Close()
	return out
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name string

		value string

		expectErr string
		expectRes string
	}{
		{
			name:      "empty value is required",
			value:     "",
			expectErr: "Agent ID is required",
		},
		{
			name:      "not a uuid",
			value:     "not-a-uuid",
			expectErr: "invalid format for Agent ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:      "valid uuid",
			value:     "464a277e-2d8d-11ef-8bc6-d7b95604d6f6",
			expectRes: "464a277e-2d8d-11ef-8bc6-d7b95604d6f6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			viper.Set("id", tt.value)

			res, err := resolveUUID("id", "Agent ID")
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong match. expect error: %q, got: %v", tt.expectErr, err)
				}
				if res.String() != "00000000-0000-0000-0000-000000000000" {
					t.Errorf("Wrong match. expect the nil uuid on error, got: %s", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect ok, got: %v", err)
			}
			if res.String() != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_resolveString(t *testing.T) {
	tests := []struct {
		name string

		value string

		expectErr string
		expectRes string
	}{
		{
			name:      "empty value is required",
			value:     "",
			expectErr: "Username is required",
		},
		{
			name:      "value is returned as it is",
			value:     " spaced value ",
			expectRes: " spaced value ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			viper.Set("username", tt.value)

			res, err := resolveString("username", "Username")
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong match. expect error: %q, got: %v", tt.expectErr, err)
				}
				if res != "" {
					t.Errorf("Wrong match. expect empty string on error, got: %q", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect ok, got: %v", err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectRes, res)
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	t.Run("indents the value with two spaces", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(map[string]any{"id": "a", "n": 1})
		})
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}

		expect := "{\n  \"id\": \"a\",\n  \"n\": 1\n}\n"
		if out != expect {
			t.Errorf("Wrong match.\nexpect: %q\ngot: %q", expect, out)
		}
	})

	t.Run("unserializable value returns an error and prints nothing", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(make(chan int))
		})
		if err == nil || !strings.HasPrefix(err.Error(), "failed to marshal JSON") {
			t.Errorf("Wrong match. expect a marshal error, got: %v", err)
		}
		if out != "" {
			t.Errorf("Wrong match. expect no output, got: %q", out)
		}
	})
}

// Test_run_validation checks the input validation of every run function. Only
// inputs that fail validation are used, because a passing input continues to
// initHandler, which connects to the real database, cache and message broker.
func Test_run_validation(t *testing.T) {
	const validID = "464a277e-2d8d-11ef-8bc6-d7b95604d6f6"

	tests := []struct {
		name string

		run      func(*cobra.Command, []string) error
		settings map[string]string

		expectErr string
	}{
		{"create: customer id missing", runCreate, map[string]string{}, "invalid customer ID: Customer ID is required"},
		{"create: customer id checked before username", runCreate, map[string]string{"customer-id": "not-a-uuid"}, "invalid customer ID: invalid format for Customer ID: 'not-a-uuid' is not a valid UUID"},
		{"create: username missing", runCreate, map[string]string{"customer-id": validID}, "invalid username: Username is required"},
		{"create: password missing", runCreate, map[string]string{"customer-id": validID, "username": "user-a"}, "invalid password: Password is required"},

		{"get: id missing", runGet, map[string]string{}, "invalid agent ID: Agent ID is required"},
		{"get: id invalid", runGet, map[string]string{"id": "not-a-uuid"}, "invalid agent ID: invalid format for Agent ID: 'not-a-uuid' is not a valid UUID"},

		{"list: customer id missing", runList, map[string]string{}, "invalid customer ID: Customer ID is required"},
		{"list: customer id invalid", runList, map[string]string{"customer-id": "not-a-uuid"}, "invalid customer ID: invalid format for Customer ID: 'not-a-uuid' is not a valid UUID"},

		{"update-permission: id missing", runUpdatePermission, map[string]string{}, "invalid agent ID: Agent ID is required"},

		{"update-password: id missing", runUpdatePassword, map[string]string{"password": "secret-a"}, "invalid agent ID: Agent ID is required"},
		{"update-password: password missing", runUpdatePassword, map[string]string{"id": validID}, "invalid password: Password is required"},

		{"delete: id missing", runDelete, map[string]string{}, "invalid agent ID: Agent ID is required"},

		{"login: username missing", runLogin, map[string]string{"password": "secret-a"}, "invalid username: Username is required"},
		{"login: password missing", runLogin, map[string]string{"username": "user-a"}, "invalid password: Password is required"},

		{"update-basic-info: id missing", runUpdateBasicInfo, map[string]string{}, "invalid agent ID: Agent ID is required"},

		{"update-addresses: id missing", runUpdateAddresses, map[string]string{"addresses": "[]"}, "invalid agent ID: Agent ID is required"},
		{"update-addresses: addresses missing", runUpdateAddresses, map[string]string{"id": validID}, "invalid addresses: Addresses is required"},
		{"update-addresses: addresses not json", runUpdateAddresses, map[string]string{"id": validID, "addresses": "{not json"}, "failed to parse addresses JSON"},

		{"update-status: id missing", runUpdateStatus, map[string]string{"status": "available"}, "invalid agent ID: Agent ID is required"},
		{"update-status: status missing", runUpdateStatus, map[string]string{"id": validID}, "invalid status: Status is required"},

		{"update-tag-ids: id missing", runUpdateTagIDs, map[string]string{"tag-ids": "[]"}, "invalid agent ID: Agent ID is required"},
		{"update-tag-ids: tag ids missing", runUpdateTagIDs, map[string]string{"id": validID}, "invalid tag IDs: Tag IDs is required"},
		{"update-tag-ids: tag ids not json", runUpdateTagIDs, map[string]string{"id": validID, "tag-ids": "[1,"}, "failed to parse tag IDs JSON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			for k, v := range tt.settings {
				viper.Set(k, v)
			}

			err := tt.run(&cobra.Command{}, nil)
			if err == nil {
				t.Fatalf("Wrong match. expect error: %q, got nil", tt.expectErr)
			}
			if strings.Contains(err.Error(), "failed to initialize handlers") {
				t.Fatalf("Wrong match. the validation must fail before the handler initialization, got: %v", err)
			}
			if !strings.HasPrefix(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match.\nexpect prefix: %q\ngot: %q", tt.expectErr, err.Error())
			}
		})
	}
}

// Test_cmd_flagWiring checks that the flag names declared by the command
// constructors are the viper keys the run functions read. A mismatch makes the
// run function see an empty value, so only invalid but non-empty values are
// used: an empty value fails the same way with a correct and a wrong wiring.
func Test_cmd_flagWiring(t *testing.T) {
	const validID = "464a277e-2d8d-11ef-8bc6-d7b95604d6f6"

	tests := []struct {
		name string

		newCmd func() *cobra.Command
		flags  map[string]string

		expectContains string
	}{
		{"create customer-id", cmdCreate, map[string]string{"customer-id": "not-a-uuid"}, "is not a valid UUID"},
		{"get id", cmdGet, map[string]string{"id": "not-a-uuid"}, "is not a valid UUID"},
		{"list customer-id", cmdList, map[string]string{"customer-id": "not-a-uuid"}, "is not a valid UUID"},
		{"update-permission id", cmdUpdatePermission, map[string]string{"id": "not-a-uuid"}, "is not a valid UUID"},
		{"update-password id", cmdUpdatePassword, map[string]string{"id": "not-a-uuid"}, "is not a valid UUID"},
		{"delete id", cmdDelete, map[string]string{"id": "not-a-uuid"}, "is not a valid UUID"},
		{"update-basic-info id", cmdUpdateBasicInfo, map[string]string{"id": "not-a-uuid"}, "is not a valid UUID"},
		{"update-addresses id", cmdUpdateAddresses, map[string]string{"id": "not-a-uuid"}, "is not a valid UUID"},
		{"update-status id", cmdUpdateStatus, map[string]string{"id": "not-a-uuid"}, "is not a valid UUID"},
		{"update-tag-ids id", cmdUpdateTagIDs, map[string]string{"id": "not-a-uuid"}, "is not a valid UUID"},
		{"update-addresses addresses", cmdUpdateAddresses, map[string]string{"id": validID, "addresses": "{not json"}, "failed to parse addresses JSON"},
		{"update-tag-ids tag-ids", cmdUpdateTagIDs, map[string]string{"id": validID, "tag-ids": "[1,"}, "failed to parse tag IDs JSON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)

			cmd := tt.newCmd()
			if cmd.RunE == nil {
				t.Fatalf("Wrong match. expect RunE to be set")
			}
			if errBind := viper.BindPFlags(cmd.Flags()); errBind != nil {
				t.Fatalf("could not bind the flags: %v", errBind)
			}
			for k, v := range tt.flags {
				if errSet := cmd.Flags().Set(k, v); errSet != nil {
					t.Fatalf("could not set the flag %q: %v", k, errSet)
				}
			}

			err := cmd.RunE(cmd, nil)
			if err == nil || !strings.Contains(err.Error(), tt.expectContains) {
				t.Fatalf("Wrong match. expect an error containing %q, got: %v", tt.expectContains, err)
			}
			if strings.Contains(err.Error(), "is required") {
				t.Errorf("Wrong match. the flag value was not seen by the run function: %v", err)
			}
		})
	}
}

func Test_cmd_flagDefaults(t *testing.T) {
	tests := []struct {
		name string

		newCmd func() *cobra.Command
		flag   string

		expectDefault string
		expectType    string
	}{
		{"list limit", cmdList, "limit", "100", "int"},
		{"create permission", cmdCreate, "permission", "0", "uint64"},
		{"update-permission permission", cmdUpdatePermission, "permission", "0", "uint64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.newCmd().Flags().Lookup(tt.flag)
			if f == nil {
				t.Fatalf("Wrong match. expect the flag %q to exist", tt.flag)
			}
			if f.DefValue != tt.expectDefault {
				t.Errorf("Wrong match. expect default: %q, got: %q", tt.expectDefault, f.DefValue)
			}
			if f.Value.Type() != tt.expectType {
				t.Errorf("Wrong match. expect type: %q, got: %q", tt.expectType, f.Value.Type())
			}
		})
	}

	// the normalize backfill must preview by default
	f := cmdNormalizeAddresses().Flags().Lookup("dry-run")
	if f == nil || f.DefValue != "true" {
		t.Errorf("Wrong match. expect dry-run to default to true, got: %v", f)
	}
}

func Test_initCommand(t *testing.T) {
	resetViper(t)

	// config.Bootstrap is not undone by viper.Reset: restore the global logrus state too.
	origFormatter := logrus.StandardLogger().Formatter
	origLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(origFormatter)
		logrus.SetLevel(origLevel)
	})

	root := initCommand()

	rootNames := []string{}
	var agentCmd *cobra.Command
	for _, c := range root.Commands() {
		rootNames = append(rootNames, c.Name())
		if c.Name() == "agent" {
			agentCmd = c
		}
	}
	sort.Strings(rootNames)
	if !reflect.DeepEqual(rootNames, []string{"agent", "migrate"}) {
		t.Errorf("Wrong match. expect root commands: [agent migrate], got: %v", rootNames)
	}
	if agentCmd == nil {
		t.Fatalf("Wrong match. expect the agent command")
	}

	agentNames := []string{}
	for _, c := range agentCmd.Commands() {
		agentNames = append(agentNames, c.Name())
	}
	sort.Strings(agentNames)
	expectAgentNames := []string{
		"create", "delete", "get", "list", "login", "normalize-addresses",
		"update-addresses", "update-basic-info", "update-password", "update-permission",
		"update-status", "update-tag-ids",
	}
	if !reflect.DeepEqual(agentNames, expectAgentNames) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectAgentNames, agentNames)
	}

	if root.PersistentPreRunE == nil {
		t.Errorf("Wrong match. expect the persistent pre-run to be set")
	}
}
