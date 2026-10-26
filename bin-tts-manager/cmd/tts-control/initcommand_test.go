package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// Test_initCommand_executeRunsValidationThroughCLI exercises initCommand's
// wiring (PersistentPreRunE binding flags and loading config, cmdSub/cmdRoot
// composition) end to end via Execute, using only input-validation error
// paths of "tts create" so initHandler (which would dial RabbitMQ) is never
// reached.
func Test_initCommand_executeRunsValidationThroughCLI(t *testing.T) {
	tests := []struct {
		name string
		args []string

		expectErr string
	}{
		{
			name:      "missing call-id surfaces the required-field message",
			args:      []string{"tts", "create", "--text", "hello"},
			expectErr: "Call ID is required",
		},
		{
			name:      "malformed call-id surfaces the invalid-format message",
			args:      []string{"tts", "create", "--call-id", "xyz", "--text", "hello"},
			expectErr: "is not a valid UUID",
		},
		{
			name:      "missing text surfaces the required-field message",
			args:      []string{"tts", "create", "--call-id", "a1b2c3d4-0000-4000-8000-000000000001"},
			expectErr: "text is required",
		},
		{
			name: "unknown provider surfaces the invalid-provider message",
			args: []string{
				"tts", "create",
				"--call-id", "a1b2c3d4-0000-4000-8000-000000000001",
				"--text", "hello",
				"--provider", "other",
			},
			expectErr: "invalid provider: other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)

			cmd := initCommand()
			cmd.SetArgs(tt.args)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)

			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong error. expected to contain %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

// Test_initCommand_bindsFlagsBeforeValidation proves PersistentPreRunE's
// viper.BindPFlags call is what makes "--call-id" reach resolveUUID via
// viper. If it were missing, Execute would return the "required" message
// (an empty string) instead of the "invalid format" message for a malformed
// but non-empty value.
func Test_initCommand_bindsFlagsBeforeValidation(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	cmd := initCommand()
	cmd.SetArgs([]string{"tts", "create", "--call-id", "not-a-uuid", "--text", "hello"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "'not-a-uuid' is not a valid UUID") {
		t.Fatalf("Wrong error. expected the flag value to reach validation, got: %v", err)
	}
}
