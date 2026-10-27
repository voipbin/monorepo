package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resetState isolates a test from the process wide viper and logrus state.
// Tests in this package must not run in parallel.
func resetState(t *testing.T) {
	t.Helper()

	prevFormatter := logrus.StandardLogger().Formatter
	prevLevel := logrus.GetLevel()
	viper.Reset()

	t.Cleanup(func() {
		viper.Reset()
		logrus.SetFormatter(prevFormatter)
		logrus.SetLevel(prevLevel)
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
			expectErr: "Call ID is required",
		},
		{
			name:      "not a uuid",
			value:     "not-a-uuid",
			expectRes: uuid.Nil,
			expectErr: "invalid format for Call ID: 'not-a-uuid' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetState(t)
			viper.Set("id", tt.value)

			res, err := resolveUUID("id", "Call ID")
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong match. expect error: %q, got: %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	t.Run("indented output", func(t *testing.T) {
		out := captureStdout(t, func() {
			if err := printJSON(map[string]any{"id": "abc", "count": 2}); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})

		expect := "{\n  \"count\": 2,\n  \"id\": \"abc\"\n}\n"
		if out != expect {
			t.Errorf("Wrong match.\nexpect: %q\ngot: %q", expect, out)
		}
	})

	t.Run("unmarshalable value", func(t *testing.T) {
		out := captureStdout(t, func() {
			err := printJSON(make(chan int))
			if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
				t.Errorf("Wrong match. expect: marshal error, got: %v", err)
			}
		})

		if out != "" {
			t.Errorf("Wrong match. expect: no output, got: %q", out)
		}
	})
}

func Test_subcommandConstructors(t *testing.T) {
	tests := []struct {
		name string

		build func() *cobra.Command

		expectUse   string
		expectFlags map[string]string // flag name -> default value
	}{
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
			name:        "delete",
			build:       cmdDelete,
			expectUse:   "delete",
			expectFlags: map[string]string{"id": ""},
		},
		{
			name:        "update-status",
			build:       cmdUpdateStatus,
			expectUse:   "update-status",
			expectFlags: map[string]string{"id": "", "status": ""},
		},
		{
			name:        "hangup",
			build:       cmdHangup,
			expectUse:   "hangup",
			expectFlags: map[string]string{"id": "", "reason": "normal"},
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

			for name, def := range tt.expectFlags {
				f := cmd.Flags().Lookup(name)
				if f == nil {
					t.Errorf("Wrong match. expect flag %q to exist", name)
					continue
				}
				if f.DefValue != def {
					t.Errorf("Wrong match. flag %q expect default: %q, got: %q", name, def, f.DefValue)
				}
			}
		})
	}
}

func Test_initCommand(t *testing.T) {
	resetState(t)

	root := initCommand()

	if root.Use != "call-control" {
		t.Errorf("Wrong match. expect: call-control, got: %s", root.Use)
	}
	if root.PersistentPreRunE == nil {
		t.Errorf("Wrong match. expect: PersistentPreRunE set, got: nil")
	}

	for _, name := range []string{"rabbitmq_address", "database_dsn", "redis_address", "redis_database", "asterisk_ws_port"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("Wrong match. expect persistent flag %q to exist", name)
		}
	}

	sub, _, err := root.Find([]string{"call"})
	if err != nil || sub == nil || sub.Use != "call" {
		t.Fatalf("Wrong match. expect: call command, got: %v, err: %v", sub, err)
	}

	got := map[string]bool{}
	for _, c := range sub.Commands() {
		got[c.Use] = true
	}
	for _, name := range []string{"get", "list", "delete", "update-status", "hangup"} {
		if !got[name] {
			t.Errorf("Wrong match. expect call subcommand %q to exist", name)
		}
	}
	if len(got) != 5 {
		t.Errorf("Wrong match. expect 5 call subcommands, got: %d", len(got))
	}

	migrate, _, err := root.Find([]string{"migrate"})
	if err != nil || migrate == nil || migrate.Name() != "migrate" {
		t.Errorf("Wrong match. expect: migrate command, got: %v, err: %v", migrate, err)
	}
}
