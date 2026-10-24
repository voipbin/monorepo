package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// resetViper clears the process-wide viper state, which resolveUUID and the run functions read.
func resetViper(t *testing.T) {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)
}

func Test_resolveUUID(t *testing.T) {
	valid := "b25a1e10-0000-11ee-9c11-000000000001"

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
			name:      "empty value reports the label as required",
			value:     "",
			expectErr: "Test Label is required",
		},
		{
			name:      "malformed value reports the label and the value",
			value:     "not-a-uuid",
			expectErr: "invalid format for Test Label: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:      "the all zero uuid is rejected like a malformed one",
			value:     "00000000-0000-0000-0000-000000000000",
			expectErr: "invalid format for Test Label",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			viper.Set("test-flag", tt.value)

			res, err := resolveUUID("test-flag", "Test Label")
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
				}
				if res != uuid.Nil {
					t.Errorf("Wrong match. expect: nil uuid on failure, got: %s", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}

	t.Run("unset flag is reported as required", func(t *testing.T) {
		resetViper(t)

		_, err := resolveUUID("never-set", "Unset Label")
		if err == nil || !strings.Contains(err.Error(), "Unset Label is required") {
			t.Errorf("Wrong match. expect error containing: %q, got: %v", "Unset Label is required", err)
		}
	})
}

// captureStdout runs f and returns what it wrote to os.Stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe. err: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	f()

	_ = w.Close()
	out, errRead := io.ReadAll(r)
	if errRead != nil {
		t.Fatalf("could not read the captured output. err: %v", errRead)
	}

	return string(out)
}

func Test_printJSON(t *testing.T) {
	t.Run("indents with two spaces and ends with a newline", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(map[string]any{"id": "x1", "count": 2})
		})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		expect := "{\n  \"count\": 2,\n  \"id\": \"x1\"\n}\n"
		if out != expect {
			t.Errorf("Wrong match.\nexpect: %q\ngot: %q", expect, out)
		}
	})

	t.Run("a value that cannot be marshaled is an error and prints nothing", func(t *testing.T) {
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

func Test_commandDefinitions(t *testing.T) {
	tests := []struct {
		name string

		cmd *cobra.Command

		expectUse   string
		expectShort string
		// flag name -> default value
		expectFlags map[string]string
	}{
		{
			name:        "start",
			cmd:         cmdStart(),
			expectUse:   "start",
			expectShort: "Start a new transcription session",
			expectFlags: map[string]string{
				"customer-id": "", "activeflow-id": "", "on-end-flow-id": "", "reference-type": "", "reference-id": "",
				"language": "en-US", "direction": "both", "provider": "",
			},
		},
		{
			name:        "stop",
			cmd:         cmdStop(),
			expectUse:   "stop",
			expectShort: "Stop a transcription session",
			expectFlags: map[string]string{"id": ""},
		},
		{
			name:        "get",
			cmd:         cmdGet(),
			expectUse:   "get",
			expectShort: "Get a transcription session by ID",
			expectFlags: map[string]string{"id": ""},
		},
		{
			name:        "get by reference",
			cmd:         cmdGetByReferenceIDAndLanguage(),
			expectUse:   "get-by-reference",
			expectShort: "Get a transcription session by reference ID and language",
			expectFlags: map[string]string{"reference-id": "", "language": ""},
		},
		{
			name:        "list",
			cmd:         cmdList(),
			expectUse:   "list",
			expectShort: "Get transcription session list",
			expectFlags: map[string]string{
				"limit": "100", "token": "", "customer-id": "", "reference-type": "", "reference-id": "", "status": "",
			},
		},
		{
			name:        "delete",
			cmd:         cmdDelete(),
			expectUse:   "delete",
			expectShort: "Delete a transcription session",
			expectFlags: map[string]string{"id": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.Use != tt.expectUse || tt.cmd.Short != tt.expectShort {
				t.Errorf("Wrong match. expect: %q/%q, got: %q/%q", tt.expectUse, tt.expectShort, tt.cmd.Use, tt.cmd.Short)
			}
			if tt.cmd.RunE == nil {
				t.Errorf("Wrong match. expect a RunE function")
			}

			count := 0
			tt.cmd.Flags().VisitAll(func(*pflag.Flag) { count++ })
			if count != len(tt.expectFlags) {
				t.Errorf("Wrong match. expect %d flags, got: %d", len(tt.expectFlags), count)
			}

			for name, expectDefault := range tt.expectFlags {
				f := tt.cmd.Flags().Lookup(name)
				if f == nil {
					t.Errorf("Wrong match. expect flag --%s, got: none", name)
					continue
				}
				if f.DefValue != expectDefault {
					t.Errorf("Wrong match. flag --%s expect default: %q, got: %q", name, expectDefault, f.DefValue)
				}
			}
		})
	}
}

func Test_initCommand_tree(t *testing.T) {
	resetViper(t)

	root := initCommand()

	if root.Use != "transcribe-control" {
		t.Errorf("Wrong match. expect root use: transcribe-control, got: %s", root.Use)
	}

	sub, _, err := root.Find([]string{"transcribe"})
	if err != nil || sub == nil || sub.Use != "transcribe" {
		t.Fatalf("Wrong match. expect the transcribe command, got: %v (err: %v)", sub, err)
	}

	names := map[string]bool{}
	for _, c := range sub.Commands() {
		names[c.Use] = true
	}
	for _, expect := range []string{"start", "stop", "get", "get-by-reference", "list", "delete"} {
		if !names[expect] {
			t.Errorf("Wrong match. expect the transcribe subcommand %q, got: %v", expect, names)
		}
	}
	if len(names) != 6 {
		t.Errorf("Wrong match. expect exactly 6 transcribe subcommands, got: %v", names)
	}

	migrate, _, err := root.Find([]string{"migrate"})
	if err != nil || migrate == nil || migrate.Use != "migrate" {
		t.Errorf("Wrong match. expect the migrate command, got: %v (err: %v)", migrate, err)
	}

	// the infrastructure flags are registered on the root as persistent flags
	for _, name := range []string{"database_dsn", "redis_address", "rabbitmq_address"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("Wrong match. expect the persistent flag --%s on the root command", name)
		}
	}
}

func Test_runStart_validation(t *testing.T) {
	const (
		customer   = "b25a1e10-0000-11ee-9c11-000000000011"
		activeflow = "b25a1e10-0000-11ee-9c11-000000000012"
		onEnd      = "b25a1e10-0000-11ee-9c11-000000000013"
		reference  = "b25a1e10-0000-11ee-9c11-000000000014"
	)

	tests := []struct {
		name string

		values map[string]string

		expectErr string
	}{
		{
			name:      "nothing set: the customer id is checked first",
			values:    map[string]string{},
			expectErr: "failed to resolve customer ID: Customer ID is required",
		},
		{
			name:      "malformed customer id",
			values:    map[string]string{"customer-id": "bad"},
			expectErr: "failed to resolve customer ID: invalid format for Customer ID",
		},
		{
			name:      "missing activeflow id",
			values:    map[string]string{"customer-id": customer},
			expectErr: "failed to resolve active flow ID: Active Flow ID is required",
		},
		{
			name:      "missing on end flow id",
			values:    map[string]string{"customer-id": customer, "activeflow-id": activeflow},
			expectErr: "failed to resolve on end flow ID: On End Flow ID is required",
		},
		{
			name:      "missing reference id",
			values:    map[string]string{"customer-id": customer, "activeflow-id": activeflow, "on-end-flow-id": onEnd},
			expectErr: "failed to resolve reference ID: Reference ID is required",
		},
		{
			name:      "missing reference type",
			values:    map[string]string{"customer-id": customer, "activeflow-id": activeflow, "on-end-flow-id": onEnd, "reference-id": reference},
			expectErr: "reference-type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			for k, v := range tt.values {
				viper.Set(k, v)
			}

			err := runStart(&cobra.Command{}, nil)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}
