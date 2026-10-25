package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	caseValidID = "00000000-0000-4000-8000-0000000000c1"
)

// Test_initCommand_tree verifies the command tree of registrar-control: every operator
// facing command is reachable by its path and the leaf commands are runnable. Building
// the tree must not open any connection (the hooks only run on Execute).
func Test_initCommand_tree(t *testing.T) {
	root := initCommand()

	leaves := [][]string{
		{"extension", "create"},
		{"extension", "get"},
		{"extension", "list"},
		{"extension", "update"},
		{"extension", "delete"},
		{"trunk", "create"},
		{"trunk", "get"},
		{"trunk", "list"},
		{"trunk", "update"},
		{"trunk", "delete"},
		{"domain-migrate"},
		{"domain-migrate-rollback"},
		{"migrate", "status"},
		{"migrate", "up"},
		{"migrate", "down"},
		{"migrate", "force"},
	}
	for _, path := range leaves {
		name := strings.Join(path, " ")
		cmd, rest, err := root.Find(path)
		if err != nil {
			t.Errorf("Wrong match. %q: could not find the command: %v", name, err)
			continue
		}
		if cmd.Name() != path[len(path)-1] || len(rest) != 0 {
			t.Errorf("Wrong match. %q: resolved to %q with leftover args %v", name, cmd.CommandPath(), rest)
			continue
		}
		if cmd.RunE == nil {
			t.Errorf("Wrong match. %q: expect a runnable command", name)
		}
	}

	groups := []string{"extension", "trunk", "migrate"}
	for _, name := range groups {
		cmd, _, err := root.Find([]string{name})
		if err != nil || cmd.Name() != name {
			t.Errorf("Wrong match. group %q is missing: %v", name, err)
		}
	}
}

// Test_runCommands_validation verifies, for every extension and trunk command, that
// a missing or malformed required flag is rejected with a labelled error before any
// handler is initialised. The flags are parsed on the real command and bound to viper
// the way the root hook binds them, so a flag whose name differs from the key the
// run function reads changes the outcome of these cases. Every case fails exactly one
// required field, so a validation that passed would try to connect and fail the
// assertion on the error text.
func Test_runCommands_validation(t *testing.T) {
	tests := []struct {
		name string
		cmd  func() *cobra.Command
		args []string

		expectErr string
	}{
		{"extension create without customer id", cmdExtensionCreate, nil, "invalid customer ID: Customer ID is required"},
		{"extension create with a malformed customer id", cmdExtensionCreate, []string{"--customer-id=not-a-uuid"}, "invalid format for Customer ID: 'not-a-uuid' is not a valid UUID"},
		{"extension create without username", cmdExtensionCreate, []string{"--customer-id=" + caseValidID}, "invalid username: Username is required"},
		{"extension create without password", cmdExtensionCreate, []string{"--customer-id=" + caseValidID, "--username=case-user"}, "invalid password: Password is required"},

		{"extension get without id", cmdExtensionGet, nil, "invalid extension ID: Extension ID is required"},
		{"extension get with a malformed id", cmdExtensionGet, []string{"--id=bad"}, "invalid format for Extension ID"},
		{"extension list without customer id", cmdExtensionList, nil, "invalid customer ID: Customer ID is required"},
		{"extension update without id", cmdExtensionUpdate, nil, "invalid extension ID: Extension ID is required"},
		{"extension update without any field", cmdExtensionUpdate, []string{"--id=" + caseValidID}, "at least one field must be provided for update"},
		{"extension delete without id", cmdExtensionDelete, nil, "invalid extension ID: Extension ID is required"},

		{"trunk create without customer id", cmdTrunkCreate, nil, "invalid customer ID: Customer ID is required"},
		{"trunk create without domain", cmdTrunkCreate, []string{"--customer-id=" + caseValidID}, "invalid domain: Domain is required"},
		{"trunk get without id", cmdTrunkGet, nil, "invalid trunk ID: Trunk ID is required"},
		{"trunk get with a malformed id", cmdTrunkGet, []string{"--id=bad"}, "invalid format for Trunk ID"},
		{"trunk list without customer id", cmdTrunkList, nil, "invalid customer ID: Customer ID is required"},
		{"trunk update without id", cmdTrunkUpdate, nil, "invalid trunk ID: Trunk ID is required"},
		{"trunk update without any field", cmdTrunkUpdate, []string{"--id=" + caseValidID}, "at least one field must be provided for update"},
		{"trunk delete without id", cmdTrunkDelete, nil, "invalid trunk ID: Trunk ID is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)

			cmd := tt.cmd()
			if errParse := cmd.Flags().Parse(tt.args); errParse != nil {
				t.Fatalf("could not parse the arguments: %v", errParse)
			}
			if errBind := viper.BindPFlags(cmd.Flags()); errBind != nil {
				t.Fatalf("could not bind the flags: %v", errBind)
			}

			err := cmd.RunE(cmd, nil)
			if err == nil {
				t.Fatalf("Wrong match. expect: an error containing %q, got: nil", tt.expectErr)
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect an error containing %q, got: %v", tt.expectErr, err)
			}
			if strings.Contains(err.Error(), "failed to initialize handler") {
				t.Errorf("Wrong match. the validation must fail before any handler is initialised, got: %v", err)
			}
		})
	}
}
