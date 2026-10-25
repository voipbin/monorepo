package main

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-conference-manager/internal/config"
)

// invalidDSN is rejected by the mysql driver while the DSN is parsed in sql.Open,
// so no network connection is ever attempted.
const invalidDSN = "not-a-valid-dsn"

func TestMain(m *testing.M) {
	viper.Set("database_dsn", invalidDSN)
	config.LoadGlobalConfig()

	os.Exit(m.Run())
}

// setViper sets the given viper keys for the test and restores the previous values afterwards.
func setViper(t *testing.T, kv map[string]any) {
	t.Helper()

	for k, v := range kv {
		old := viper.Get(k)
		viper.Set(k, v)
		key := k
		t.Cleanup(func() { viper.Set(key, old) })
	}
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name string

		value string

		expectErr    string
		expectResult string
	}{
		{
			name:      "empty value",
			value:     "",
			expectErr: "Conference ID is required",
		},
		{
			name:      "malformed uuid",
			value:     "not-a-uuid",
			expectErr: "invalid format for Conference ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:         "valid uuid",
			value:        "7b0d1c52-3f4e-4a7b-9c1d-2e8f6a5b4c3d",
			expectResult: "7b0d1c52-3f4e-4a7b-9c1d-2e8f6a5b4c3d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setViper(t, map[string]any{"id": tt.value})

			res, err := resolveUUID("id", "Conference ID")
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong match. expect: %s, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.String() != tt.expectResult {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectResult, res.String())
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	tests := []struct {
		name string

		value any

		expectErr bool
	}{
		{
			name:  "serializable value",
			value: map[string]string{"id": "7b0d1c52-3f4e-4a7b-9c1d-2e8f6a5b4c3d"},
		},
		{
			name:      "value that cannot be serialized",
			value:     make(chan int),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := printJSON(tt.value)
			if tt.expectErr {
				if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
					t.Errorf("Wrong match. expect: failed to marshal JSON, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_command_constructors(t *testing.T) {
	tests := []struct {
		name string

		build func() *cobra.Command

		expectUse   string
		expectFlags map[string]string // flag name -> default value
	}{
		{"create", cmdCreate, "create", map[string]string{"id": "", "customer-id": "", "type": "conference", "name": "", "detail": "", "timeout": "0", "pre-flow-id": "", "post-flow-id": ""}},
		{"get", cmdGet, "get", map[string]string{"id": ""}},
		{"get-by-confbridge", cmdGetByConfbridge, "get-by-confbridge", map[string]string{"confbridge-id": ""}},
		{"list", cmdList, "list", map[string]string{"limit": "100", "token": "", "customer-id": ""}},
		{"update", cmdUpdate, "update", map[string]string{"id": "", "name": "", "detail": "", "data": "", "timeout": "0", "pre-flow-id": "", "post-flow-id": ""}},
		{"delete", cmdDelete, "delete", map[string]string{"id": ""}},
		{"update-recording-id", cmdUpdateRecordingID, "update-recording-id", map[string]string{"id": "", "recording-id": ""}},
		{"recording-start", cmdRecordingStart, "recording-start", map[string]string{"id": "", "activeflow-id": "", "format": "wav", "duration": "0", "on-end-flow-id": ""}},
		{"recording-stop", cmdRecordingStop, "recording-stop", map[string]string{"id": ""}},
		{"terminating", cmdTerminating, "terminating", map[string]string{"id": ""}},
		{"transcribe-start", cmdTranscribeStart, "transcribe-start", map[string]string{"id": "", "lang": "en"}},
		{"transcribe-stop", cmdTranscribeStop, "transcribe-stop", map[string]string{"id": ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.build()

			if cmd.Use != tt.expectUse {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectUse, cmd.Use)
			}
			if cmd.RunE == nil {
				t.Errorf("Wrong match. expect: RunE is set, got: nil")
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

func Test_initCommand(t *testing.T) {
	root := initCommand()

	if root.Use != "conference-control" {
		t.Errorf("Wrong match. expect: conference-control, got: %s", root.Use)
	}

	for _, name := range []string{"rabbitmq_address", "prometheus_endpoint", "prometheus_listen_address", "database_dsn", "redis_address", "redis_password", "redis_database"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("Wrong match. expect persistent flag: %s, got: nil", name)
		}
	}

	expectSub := []string{
		"create", "get", "get-by-confbridge", "list", "update", "delete", "update-recording-id",
		"recording-start", "recording-stop", "terminating", "transcribe-start", "transcribe-stop",
	}
	sub, _, err := root.Find([]string{"conference"})
	if err != nil || sub == nil || sub.Use != "conference" {
		t.Fatalf("Wrong match. expect: conference command, got: %v, err: %v", sub, err)
	}
	if len(sub.Commands()) != len(expectSub) {
		t.Errorf("Wrong match. expect subcommand count: %d, got: %d", len(expectSub), len(sub.Commands()))
	}
	for _, name := range expectSub {
		found, _, errFind := root.Find([]string{"conference", name})
		if errFind != nil || found == nil || found.Use != name {
			t.Errorf("Wrong match. expect subcommand: %s, got: %v, err: %v", name, found, errFind)
		}
	}

	migrate, _, err := root.Find([]string{"migrate"})
	if err != nil || migrate == nil || migrate.Use != "migrate" {
		t.Errorf("Wrong match. expect: migrate command, got: %v, err: %v", migrate, err)
	}

	if root.PersistentPreRunE == nil {
		t.Fatalf("Wrong match. expect: PersistentPreRunE is set, got: nil")
	}
	if errPre := root.PersistentPreRunE(root, nil); errPre != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errPre)
	}
	if config.Get().DatabaseDSN != invalidDSN {
		t.Errorf("Wrong match. expect the fixed test DSN to be kept, got: %s", config.Get().DatabaseDSN)
	}
}

func Test_runCreate_validation(t *testing.T) {
	const validID = "7b0d1c52-3f4e-4a7b-9c1d-2e8f6a5b4c3d"

	tests := []struct {
		name string

		values map[string]any

		expectErr string
	}{
		{
			name:      "customer id is missing",
			values:    map[string]any{"customer-id": "", "id": "", "type": "conference", "pre-flow-id": "", "post-flow-id": ""},
			expectErr: "failed to resolve customer ID: Customer ID is required",
		},
		{
			name:      "customer id is malformed",
			values:    map[string]any{"customer-id": "bad", "id": "", "type": "conference", "pre-flow-id": "", "post-flow-id": ""},
			expectErr: "failed to resolve customer ID: invalid format for Customer ID: 'bad' is not a valid UUID",
		},
		{
			name:      "conference id is malformed",
			values:    map[string]any{"customer-id": validID, "id": "bad", "type": "conference", "pre-flow-id": "", "post-flow-id": ""},
			expectErr: "invalid format for Conference ID: 'bad' is not a valid UUID",
		},
		{
			name:      "unknown conference type",
			values:    map[string]any{"customer-id": validID, "id": "", "type": "unknown", "pre-flow-id": "", "post-flow-id": ""},
			expectErr: "invalid conference type: must be 'conference', 'connect', or 'queue'",
		},
		{
			name:      "pre flow id is malformed",
			values:    map[string]any{"customer-id": validID, "id": "", "type": "connect", "pre-flow-id": "bad", "post-flow-id": ""},
			expectErr: "invalid format for Pre-flow ID: 'bad' is not a valid UUID",
		},
		{
			name:      "post flow id is malformed",
			values:    map[string]any{"customer-id": validID, "id": "", "type": "queue", "pre-flow-id": "", "post-flow-id": "bad"},
			expectErr: "invalid format for Post-flow ID: 'bad' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setViper(t, tt.values)

			err := runCreate(&cobra.Command{}, nil)
			if err == nil || err.Error() != tt.expectErr {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectErr, err)
			}
		})
	}
}

// Every run function initializes the handlers first (runCreate after its input checks),
// so with a DSN that cannot be parsed they all stop with the same initialization error.
func Test_run_initHandlerFailure(t *testing.T) {
	tests := []struct {
		name string

		run func(*cobra.Command, []string) error
	}{
		{"create", runCreate},
		{"get", runGet},
		{"get-by-confbridge", runGetByConfbridge},
		{"list", runList},
		{"update", runUpdate},
		{"delete", runDelete},
		{"update-recording-id", runUpdateRecordingID},
		{"recording-start", runRecordingStart},
		{"recording-stop", runRecordingStop},
		{"terminating", runTerminating},
		{"transcribe-start", runTranscribeStart},
		{"transcribe-stop", runTranscribeStop},
	}

	setViper(t, map[string]any{
		"customer-id":  "7b0d1c52-3f4e-4a7b-9c1d-2e8f6a5b4c3d",
		"id":           "",
		"type":         "conference",
		"pre-flow-id":  "",
		"post-flow-id": "",
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(&cobra.Command{}, nil)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}

			for _, expect := range []string{"failed to initialize handlers", "could not connect to the database", "invalid DSN"} {
				if !strings.Contains(err.Error(), expect) {
					t.Errorf("Wrong match. expect error to contain: %q, got: %v", expect, err)
				}
			}
		})
	}
}
