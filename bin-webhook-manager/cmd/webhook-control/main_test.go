package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/spf13/viper"
)

// resetViper clears the process-wide viper state before and after a test.
func resetViper(t *testing.T) {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name string

		value string

		expectRes         uuid.UUID
		expectErrContains string
	}{
		{
			name:              "missing value",
			value:             "",
			expectRes:         uuid.Nil,
			expectErrContains: "Customer ID is required",
		},
		{
			name:              "not a uuid",
			value:             "not-a-uuid",
			expectRes:         uuid.Nil,
			expectErrContains: "invalid format for Customer ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:              "nil uuid text is rejected like an invalid one",
			value:             "00000000-0000-0000-0000-000000000000",
			expectRes:         uuid.Nil,
			expectErrContains: "is not a valid UUID",
		},
		{
			name:      "valid uuid",
			value:     "5a0b3a3c-9d0b-11ee-8f4f-0b0e2b1c2d3e",
			expectRes: uuid.FromStringOrNil("5a0b3a3c-9d0b-11ee-8f4f-0b0e2b1c2d3e"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			viper.Set("customer-id", tt.value)

			res, err := resolveUUID("customer-id", "Customer ID")
			if tt.expectErrContains == "" {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.expectErrContains) {
				t.Fatalf("Wrong match. expect error containing: %q, got: %v", tt.expectErrContains, err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

// captureStdout runs f with os.Stdout redirected and returns what f printed.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Could not create a pipe. err: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	f()

	os.Stdout = orig
	if errClose := w.Close(); errClose != nil {
		t.Fatalf("Could not close the pipe. err: %v", errClose)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Could not read the pipe. err: %v", err)
	}

	return string(out)
}

func Test_printJSON(t *testing.T) {
	t.Run("indented output", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(map[string]any{"status": "success"})
		})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		expect := "{\n  \"status\": \"success\"\n}\n"
		if out != expect {
			t.Errorf("Wrong match.\nexpect: %q\ngot: %q", expect, out)
		}
	})

	t.Run("value that cannot be encoded", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(map[string]any{"bad": make(chan int)})
		})
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("Wrong match. expect error containing: failed to marshal JSON, got: %v", err)
		}

		if out != "" {
			t.Errorf("Wrong match. expect: nothing printed, got: %q", out)
		}
	})
}

// Test_runSend_validation checks the input errors of the send commands. All of them are returned
// before any handler (database, cache, message broker) is initialized.
func Test_runSend_validation(t *testing.T) {
	const validID = "5a0b3a3c-9d0b-11ee-8f4f-0b0e2b1c2d3e"

	tests := []struct {
		name string

		args []string

		expectErrContains string
	}{
		{
			name:              "customer: missing customer id",
			args:              []string{"webhook", "send-to-customer", "--data", "{}"},
			expectErrContains: "failed to resolve customer ID: Customer ID is required",
		},
		{
			name:              "customer: invalid customer id",
			args:              []string{"webhook", "send-to-customer", "--customer-id", "abc", "--data", "{}"},
			expectErrContains: "invalid format for Customer ID",
		},
		{
			name:              "customer: missing data",
			args:              []string{"webhook", "send-to-customer", "--customer-id", validID},
			expectErrContains: "data is required",
		},
		{
			name:              "customer: data is not json",
			args:              []string{"webhook", "send-to-customer", "--customer-id", validID, "--data", "{broken"},
			expectErrContains: "invalid JSON data",
		},
		{
			name:              "uri: missing customer id",
			args:              []string{"webhook", "send-to-uri", "--uri", "https://example.com/hook", "--data", "{}"},
			expectErrContains: "failed to resolve customer ID: Customer ID is required",
		},
		{
			name:              "uri: missing uri",
			args:              []string{"webhook", "send-to-uri", "--customer-id", validID, "--data", "{}"},
			expectErrContains: "uri is required",
		},
		{
			name:              "uri: missing data",
			args:              []string{"webhook", "send-to-uri", "--customer-id", validID, "--uri", "https://example.com/hook"},
			expectErrContains: "data is required",
		},
		{
			name:              "uri: data is not json",
			args:              []string{"webhook", "send-to-uri", "--customer-id", validID, "--uri", "https://example.com/hook", "--data", "[1,"},
			expectErrContains: "invalid JSON data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)

			cmd := initCommand()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.expectErrContains) {
				t.Errorf("Wrong match. expect error containing: %q, got: %v", tt.expectErrContains, err)
			}
		})
	}
}

func Test_initCommand_structure(t *testing.T) {
	resetViper(t)

	root := initCommand()
	if root.Use != "webhook-control" {
		t.Errorf("Wrong match. expect: webhook-control, got: %s", root.Use)
	}

	tests := []struct {
		name string

		path []string

		expectFlags map[string]string
	}{
		{
			name: "send-to-customer",
			path: []string{"webhook", "send-to-customer"},
			expectFlags: map[string]string{
				"customer-id": "",
				"data-type":   "application/json",
				"data":        "",
			},
		},
		{
			name: "send-to-uri",
			path: []string{"webhook", "send-to-uri"},
			expectFlags: map[string]string{
				"customer-id": "",
				"uri":         "",
				"method":      "POST",
				"data-type":   "application/json",
				"data":        "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _, err := root.Find(tt.path)
			if err != nil || cmd.Name() != tt.name {
				t.Fatalf("Wrong match. expect command: %s, got: %v, err: %v", tt.name, cmd, err)
			}

			for name, def := range tt.expectFlags {
				f := cmd.Flags().Lookup(name)
				if f == nil {
					t.Errorf("Wrong match. expect flag: %s, got: nil", name)
					continue
				}

				if f.DefValue != def {
					t.Errorf("Wrong match. flag: %s, expect default: %q, got: %q", name, def, f.DefValue)
				}
			}
		})
	}
}
