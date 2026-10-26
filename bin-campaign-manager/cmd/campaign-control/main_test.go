package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-campaign-manager/internal/config"
)

// hostEnvKeys are the environment keys that viper reads, directly or through the automatic
// environment binding, so the tests must not see the values of the machine they run on.
var hostEnvKeys = []string{
	"DATABASE_DSN", "PROMETHEUS_ENDPOINT", "PROMETHEUS_LISTEN_ADDRESS", "RABBITMQ_ADDRESS",
	"REDIS_ADDRESS", "REDIS_DATABASE", "REDIS_PASSWORD",
	"ID", "NAME", "DETAIL", "TYPE", "TOKEN", "LIMIT", "STATUS", "CUSTOMER_ID", "OUTPLAN_ID", "OUTDIAL_ID",
	"QUEUE_ID", "NEXT_CAMPAIGN_ID", "SERVICE_LEVEL", "END_HANDLE",
}

// isolateGlobals resets viper, removes host environment keys and restores the shared config.
func isolateGlobals(t *testing.T) {
	t.Helper()

	savedConfig := *config.Get()
	type savedEnv struct {
		key   string
		value string
		found bool
	}
	var saved []savedEnv
	for _, k := range hostEnvKeys {
		v, ok := os.LookupEnv(k)
		saved = append(saved, savedEnv{key: k, value: v, found: ok})
		_ = os.Unsetenv(k)
	}
	viper.Reset()

	t.Cleanup(func() {
		viper.Reset()
		*config.Get() = savedConfig
		for _, s := range saved {
			if s.found {
				_ = os.Setenv(s.key, s.value)
			} else {
				_ = os.Unsetenv(s.key)
			}
		}
	})
}

// captureStdout runs f and returns what it wrote to os.Stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	f()

	_ = w.Close()
	os.Stdout = orig
	return <-done
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name string

		value string

		expectID  string
		expectErr string
	}{
		{
			name:      "empty value",
			value:     "",
			expectErr: "Campaign ID is required",
		},
		{
			name:      "not a uuid",
			value:     "not-a-uuid",
			expectErr: "invalid format for Campaign ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:     "valid uuid",
			value:    "11111111-2222-3333-4444-555555555555",
			expectID: "11111111-2222-3333-4444-555555555555",
		},
		{
			name:      "nil uuid is rejected",
			value:     "00000000-0000-0000-0000-000000000000",
			expectErr: "invalid format for Campaign ID: '00000000-0000-0000-0000-000000000000' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateGlobals(t)
			viper.Set("id", tt.value)

			res, err := resolveUUID("id", "Campaign ID")
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong error. expect: %s, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.String() != tt.expectID {
				t.Errorf("Wrong id. expect: %s, got: %s", tt.expectID, res)
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	t.Run("indents with two spaces", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(map[string]any{"name": "campaign-a", "count": 2})
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expect := "{\n  \"count\": 2,\n  \"name\": \"campaign-a\"\n}\n"
		if out != expect {
			t.Errorf("Wrong output. expect: %q, got: %q", expect, out)
		}
	})

	t.Run("unmarshalable value", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(make(chan int))
		})
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("Wrong error. got: %v", err)
		}
		if out != "" {
			t.Errorf("nothing must be printed on error, got: %q", out)
		}
	})
}

func Test_initCommand_structure(t *testing.T) {
	isolateGlobals(t)

	root := initCommand()

	if root.Use != "campaign-control" {
		t.Errorf("Wrong root use. got: %s", root.Use)
	}

	var campaignCmd, migrateCmd *cobra.Command
	var rootNames []string
	for _, c := range root.Commands() {
		rootNames = append(rootNames, c.Name())
		switch c.Name() {
		case "campaign":
			campaignCmd = c
		case "migrate":
			migrateCmd = c
		}
	}
	if campaignCmd == nil || migrateCmd == nil {
		t.Fatalf("root commands campaign and migrate are required. got: %v", rootNames)
	}

	var campaignNames []string
	for _, c := range campaignCmd.Commands() {
		campaignNames = append(campaignNames, c.Name())
	}
	for _, expect := range []string{"create", "get", "list", "update-basic-info", "update-status", "delete"} {
		if !contains(campaignNames, expect) {
			t.Errorf("campaign sub-command %s is missing. got: %v", expect, campaignNames)
		}
	}

	var migrateNames []string
	for _, c := range migrateCmd.Commands() {
		migrateNames = append(migrateNames, c.Name())
	}
	for _, expect := range []string{"status", "up", "down", "force"} {
		if !contains(migrateNames, expect) {
			t.Errorf("migrate sub-command %s is missing. got: %v", expect, migrateNames)
		}
	}

	flags := map[string]map[string]string{
		"create": {
			"customer-id": "", "type": "call", "name": "", "detail": "", "service-level": "100",
			"end-handle": "stop", "outplan-id": "", "outdial-id": "", "queue-id": "", "next-campaign-id": "",
		},
		"get":               {"id": ""},
		"list":              {"limit": "100", "token": "", "customer-id": ""},
		"update-basic-info": {"id": "", "name": "", "detail": "", "type": "", "service-level": "0", "end-handle": ""},
		"update-status":     {"id": "", "status": "stop"},
		"delete":            {"id": ""},
	}
	for _, c := range campaignCmd.Commands() {
		expects, ok := flags[c.Name()]
		if !ok {
			t.Errorf("unexpected campaign sub-command %s", c.Name())
			continue
		}
		for flagName, def := range expects {
			f := c.Flags().Lookup(flagName)
			if f == nil {
				t.Errorf("%s: flag %s is missing", c.Name(), flagName)
				continue
			}
			if f.DefValue != def {
				t.Errorf("%s: wrong default for %s. expect: %q, got: %q", c.Name(), flagName, def, f.DefValue)
			}
		}
		if c.RunE == nil {
			t.Errorf("%s: RunE is not set", c.Name())
		}
	}

	// the infrastructure flags come from config.Bootstrap
	for _, name := range []string{"database_dsn", "redis_address", "rabbitmq_address"} {
		if root.Flags().Lookup(name) == nil {
			t.Errorf("flag %s is missing", name)
		}
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Test_runCreate_validation checks the input checks of the create command. They run before
// any connection is opened, so every case ends with the validation error.
func Test_runCreate_validation(t *testing.T) {
	const validID = "11111111-2222-3333-4444-555555555555"

	tests := []struct {
		name string

		args []string

		expectErr string
	}{
		{
			name:      "customer id is missing",
			args:      []string{"campaign", "create", "--name=campaign-a"},
			expectErr: "failed to resolve customer ID: Customer ID is required",
		},
		{
			name:      "customer id is not a uuid",
			args:      []string{"campaign", "create", "--customer-id=abc", "--name=campaign-a"},
			expectErr: "failed to resolve customer ID: invalid format for Customer ID: 'abc' is not a valid UUID",
		},
		{
			name:      "name is missing",
			args:      []string{"campaign", "create", "--customer-id=" + validID},
			expectErr: "name is required",
		},
		{
			name:      "outplan id is missing",
			args:      []string{"campaign", "create", "--customer-id=" + validID, "--name=campaign-a"},
			expectErr: "failed to resolve outplan ID: Outplan ID is required",
		},
		{
			name:      "outdial id is not a uuid",
			args:      []string{"campaign", "create", "--customer-id=" + validID, "--name=campaign-a", "--outplan-id=" + validID, "--outdial-id=abc"},
			expectErr: "failed to resolve outdial ID: invalid format for Outdial ID: 'abc' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateGlobals(t)

			root := initCommand()
			root.SilenceUsage = true
			root.SilenceErrors = true
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs(tt.args)

			err := root.Execute()
			if err == nil || err.Error() != tt.expectErr {
				t.Errorf("Wrong error. expect: %s, got: %v", tt.expectErr, err)
			}
		})
	}
}
