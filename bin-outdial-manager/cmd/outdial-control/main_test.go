package main

import (
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

const (
	testCustomerID = "a1b2c3d4-0000-4000-8000-000000000001"
	testCampaignID = "a1b2c3d4-0000-4000-8000-000000000002"
)

// resetViper clears the global viper state now and when the test ends, because the
// keys set by one case would otherwise decide the branch of the next one.
func resetViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
}

// captureStdout runs fn with os.Stdout replaced by a pipe and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	_ = w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("could not read the output: %v", err)
	}
	return string(out)
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expectID  string
		expectErr string
	}{
		{"empty value names the label", "", "", "Outdial ID is required"},
		{"invalid value is quoted", "not-a-uuid", "", "invalid format for Outdial ID: 'not-a-uuid' is not a valid UUID"},
		{"valid value", testCustomerID, testCustomerID, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			viper.Set("id", tt.value)

			res, err := resolveUUID("id", "Outdial ID")
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("expected error %q, got: %v", tt.expectErr, err)
				}
				if res.String() != "00000000-0000-0000-0000-000000000000" {
					t.Errorf("expected the nil uuid with an error, got: %s", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if res.String() != tt.expectID {
				t.Errorf("expected %s, got %s", tt.expectID, res)
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	t.Run("indented output", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(map[string]any{"name": "x", "count": 2})
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		expect := "{\n  \"count\": 2,\n  \"name\": \"x\"\n}\n"
		if out != expect {
			t.Errorf("expected %q, got %q", expect, out)
		}
	})

	t.Run("value that cannot be marshaled", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(make(chan int))
		})
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("expected a marshal error, got: %v", err)
		}
		if out != "" {
			t.Errorf("expected no output on failure, got %q", out)
		}
	})
}

func Test_commandDefinitions(t *testing.T) {
	tests := []struct {
		name         string
		build        func() *cobra.Command
		expectUse    string
		expectFlags  []string
		expectString map[string]string
	}{
		{"create", cmdCreate, "create", []string{"customer-id", "campaign-id", "name", "detail", "data"}, nil},
		{"get", cmdGet, "get", []string{"id"}, nil},
		{"list", cmdList, "list", []string{"limit", "token", "customer-id"}, map[string]string{"limit": "100"}},
		{"update-basic-info", cmdUpdateBasicInfo, "update-basic-info", []string{"id", "name", "detail"}, nil},
		{"update-campaign-id", cmdUpdateCampaignID, "update-campaign-id", []string{"id", "campaign-id"}, nil},
		{"update-data", cmdUpdateData, "update-data", []string{"id", "data"}, nil},
		{"delete", cmdDelete, "delete", []string{"id"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.build()
			if cmd.Use != tt.expectUse {
				t.Errorf("expected Use %q, got %q", tt.expectUse, cmd.Use)
			}
			if cmd.RunE == nil {
				t.Errorf("expected a RunE function")
			}

			for _, name := range tt.expectFlags {
				f := cmd.Flags().Lookup(name)
				if f == nil {
					t.Errorf("flag %q is not registered", name)
					continue
				}
				if expect, ok := tt.expectString[name]; ok && f.DefValue != expect {
					t.Errorf("flag %q: expected default %q, got %q", name, expect, f.DefValue)
				}
			}
		})
	}
}

func Test_initCommand(t *testing.T) {
	resetViper(t)

	// config.Bootstrap changes the global log settings
	origFormatter := logrus.StandardLogger().Formatter
	origLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(origFormatter)
		logrus.SetLevel(origLevel)
	})

	root := initCommand()
	if root.Use != "outdial-control" {
		t.Errorf("expected the root command outdial-control, got %q", root.Use)
	}

	subs := map[string]*cobra.Command{}
	for _, c := range root.Commands() {
		subs[c.Name()] = c
	}
	outdialCmd, ok := subs["outdial"]
	if !ok {
		t.Fatalf("the outdial command is not registered. got: %v", subs)
	}
	if _, ok := subs["migrate"]; !ok {
		t.Errorf("the migrate command is not registered. got: %v", subs)
	}

	var children []string
	for _, c := range outdialCmd.Commands() {
		children = append(children, c.Name())
	}
	sort.Strings(children)
	expect := []string{"create", "delete", "get", "list", "update-basic-info", "update-campaign-id", "update-data"}
	if !reflect.DeepEqual(children, expect) {
		t.Errorf("expected outdial sub commands %v, got %v", expect, children)
	}

	for _, name := range []string{"database_dsn", "prometheus_endpoint", "prometheus_listen_address", "rabbitmq_address", "redis_address", "redis_password", "redis_database"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("the persistent flag %q is not registered", name)
		}
	}
}

func Test_runCreate_validation(t *testing.T) {
	tests := []struct {
		name         string
		flags        map[string]string
		expectErrors []string
	}{
		{"customer id missing", map[string]string{"campaign-id": testCampaignID, "name": "n"},
			[]string{"failed to resolve customer ID", "Customer ID is required"}},
		{"customer id invalid", map[string]string{"customer-id": "bad", "campaign-id": testCampaignID, "name": "n"},
			[]string{"failed to resolve customer ID", "invalid format for Customer ID: 'bad'"}},
		{"campaign id missing", map[string]string{"customer-id": testCustomerID, "name": "n"},
			[]string{"failed to resolve campaign ID", "Campaign ID is required"}},
		{"campaign id invalid", map[string]string{"customer-id": testCustomerID, "campaign-id": "bad", "name": "n"},
			[]string{"failed to resolve campaign ID", "invalid format for Campaign ID: 'bad'"}},
		{"name missing", map[string]string{"customer-id": testCustomerID, "campaign-id": testCampaignID},
			[]string{"name is required"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)

			// the values reach viper through the flags of the command, as PersistentPreRunE binds them
			cmd := cmdCreate()
			for k, v := range tt.flags {
				if err := cmd.Flags().Set(k, v); err != nil {
					t.Fatalf("could not set the flag %q: %v", k, err)
				}
			}
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				t.Fatalf("could not bind the flags: %v", err)
			}

			err := runCreate(cmd, nil)
			if err == nil {
				t.Fatalf("expected a validation error, got nil")
			}
			for _, part := range tt.expectErrors {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("expected the error to contain %q, got: %v", part, err)
				}
			}
		})
	}
}
