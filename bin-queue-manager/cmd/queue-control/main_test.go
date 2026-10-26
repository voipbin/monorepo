package main

import (
	"bytes"
	"io"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resetGlobalState isolates a test from the process-global state that the
// command construction and the viper lookups touch (viper keys and the logrus
// formatter and level), and restores it afterwards.
func resetGlobalState(t *testing.T) {
	t.Helper()

	prevFormatter := logrus.StandardLogger().Formatter
	prevLevel := logrus.GetLevel()

	viper.Reset()
	t.Cleanup(func() {
		viper.Reset()
		logrus.SetFormatter(prevFormatter)
		logrus.SetLevel(prevLevel)
	})

	// an empty value is treated as unset by viper's automatic env lookup, which
	// keeps the host environment from leaking into the expectations.
	for _, key := range []string{
		"DATABASE_DSN", "PROMETHEUS_ENDPOINT", "PROMETHEUS_LISTEN_ADDRESS",
		"RABBITMQ_ADDRESS", "REDIS_ADDRESS", "REDIS_PASSWORD", "REDIS_DATABASE",
	} {
		t.Setenv(key, "")
	}
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
	defer func() {
		os.Stdout = orig
	}()

	f()

	if errClose := w.Close(); errClose != nil {
		t.Fatalf("could not close the pipe writer. err: %v", errClose)
	}
	defer func() { _ = r.Close() }()

	var buf bytes.Buffer
	if _, errCopy := io.Copy(&buf, r); errCopy != nil {
		t.Fatalf("could not read the captured output. err: %v", errCopy)
	}

	return buf.String()
}

func commandNames(cmd *cobra.Command) []string {
	res := []string{}
	for _, c := range cmd.Commands() {
		res = append(res, c.Name())
	}
	sort.Strings(res)
	return res
}

func Test_initCommand(t *testing.T) {
	resetGlobalState(t)

	root := initCommand()
	if root.Use != "queue-control" {
		t.Errorf("Wrong match. expect: queue-control, got: %s", root.Use)
	}

	// the infrastructure flags are registered by config.Bootstrap
	for _, name := range []string{
		"database_dsn", "prometheus_endpoint", "prometheus_listen_address",
		"rabbitmq_address", "redis_address", "redis_password", "redis_database",
	} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("Wrong match. the persistent flag %s is not registered", name)
		}
	}

	expectRoot := []string{"migrate", "queue", "queuecall"}
	if got := commandNames(root); strings.Join(got, ",") != strings.Join(expectRoot, ",") {
		t.Errorf("Wrong match. expect: %v, got: %v", expectRoot, got)
	}

	tests := []struct {
		name   string
		path   []string
		expect []string
	}{
		{
			name:   "queue subcommands",
			path:   []string{"queue"},
			expect: []string{"create", "delete", "get", "list", "update", "update-routing-method", "update-tag-ids"},
		},
		{
			name:   "queuecall subcommands",
			path:   []string{"queuecall"},
			expect: []string{"delete", "get", "get-by-reference", "list"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, _, err := root.Find(tt.path)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if got := commandNames(sub); strings.Join(got, ",") != strings.Join(tt.expect, ",") {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

func Test_command_constructors(t *testing.T) {
	tests := []struct {
		name         string
		cmd          *cobra.Command
		expectUse    string
		expectFlags  map[string]string // flag name -> default value
		expectNoFlag []string
	}{
		{
			name:      "create",
			cmd:       cmdCreate(),
			expectUse: "create",
			expectFlags: map[string]string{
				"customer-id":     "",
				"name":            "",
				"detail":          "",
				"routing-method":  "random",
				"tag-ids":         "",
				"wait-flow-id":    "",
				"wait-timeout":    "300000",
				"service-timeout": "600000",
			},
		},
		{
			name:        "get",
			cmd:         cmdGet(),
			expectUse:   "get",
			expectFlags: map[string]string{"id": ""},
		},
		{
			name:      "list",
			cmd:       cmdList(),
			expectUse: "list",
			expectFlags: map[string]string{
				"limit":       "100",
				"token":       "",
				"customer-id": "",
			},
		},
		{
			name:      "update",
			cmd:       cmdUpdate(),
			expectUse: "update",
			expectFlags: map[string]string{
				"id":              "",
				"name":            "",
				"detail":          "",
				"routing-method":  "random",
				"tag-ids":         "",
				"wait-flow-id":    "",
				"wait-timeout":    "300000",
				"service-timeout": "600000",
			},
		},
		{
			name:      "update-tag-ids",
			cmd:       cmdUpdateTagIDs(),
			expectUse: "update-tag-ids",
			expectFlags: map[string]string{
				"id":      "",
				"tag-ids": "",
			},
			expectNoFlag: []string{"routing-method"},
		},
		{
			name:      "update-routing-method",
			cmd:       cmdUpdateRoutingMethod(),
			expectUse: "update-routing-method",
			expectFlags: map[string]string{
				"id":             "",
				"routing-method": "",
			},
			expectNoFlag: []string{"tag-ids"},
		},
		{
			name:        "delete",
			cmd:         cmdDelete(),
			expectUse:   "delete",
			expectFlags: map[string]string{"id": ""},
		},
		{
			name:        "queuecall get",
			cmd:         cmdQueuecallGet(),
			expectUse:   "get",
			expectFlags: map[string]string{"id": ""},
		},
		{
			name:        "queuecall get-by-reference",
			cmd:         cmdQueuecallGetByReferenceID(),
			expectUse:   "get-by-reference",
			expectFlags: map[string]string{"reference-id": ""},
		},
		{
			name:      "queuecall list",
			cmd:       cmdQueuecallList(),
			expectUse: "list",
			expectFlags: map[string]string{
				"limit":       "100",
				"token":       "",
				"customer-id": "",
				"queue-id":    "",
				"status":      "",
			},
		},
		{
			name:        "queuecall delete",
			cmd:         cmdQueuecallDelete(),
			expectUse:   "delete",
			expectFlags: map[string]string{"id": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.Use != tt.expectUse {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectUse, tt.cmd.Use)
			}
			if tt.cmd.RunE == nil {
				t.Errorf("Wrong match. the command has no run function")
			}

			flags := tt.cmd.Flags()
			for name, expectDefault := range tt.expectFlags {
				f := flags.Lookup(name)
				if f == nil {
					t.Errorf("Wrong match. the flag %s is not defined", name)
					continue
				}
				if f.DefValue != expectDefault {
					t.Errorf("Wrong match. flag: %s, expect: %q, got: %q", name, expectDefault, f.DefValue)
				}
			}

			for _, name := range tt.expectNoFlag {
				if flags.Lookup(name) != nil {
					t.Errorf("Wrong match. the flag %s must not be defined", name)
				}
			}
		})
	}
}

func Test_resolveUUID(t *testing.T) {
	validID := uuid.FromStringOrNil("6b4b2b1e-3d0c-4a6e-9b6a-0f1f2f3a4b5c")

	tests := []struct {
		name string

		value string

		expectRes   uuid.UUID
		expectErr   bool
		expectInErr []string
	}{
		{
			name:      "valid uuid",
			value:     validID.String(),
			expectRes: validID,
		},
		{
			name:        "empty value",
			value:       "",
			expectRes:   uuid.Nil,
			expectErr:   true,
			expectInErr: []string{"Customer ID is required"},
		},
		{
			name:        "invalid format",
			value:       "not-a-uuid",
			expectRes:   uuid.Nil,
			expectErr:   true,
			expectInErr: []string{"invalid format for Customer ID", "not-a-uuid"},
		},
		{
			name:        "nil uuid string is rejected as invalid",
			value:       uuid.Nil.String(),
			expectRes:   uuid.Nil,
			expectErr:   true,
			expectInErr: []string{"invalid format for Customer ID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalState(t)
			viper.Set("customer-id", tt.value)

			res, err := resolveUUID("customer-id", "Customer ID")
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
			for _, s := range tt.expectInErr {
				if !strings.Contains(err.Error(), s) {
					t.Errorf("Wrong match. expect the error to contain %q, got: %v", s, err)
				}
			}
		})
	}
}

func Test_resolveUUID_unsetFlag(t *testing.T) {
	resetGlobalState(t)

	res, err := resolveUUID("never-set-flag", "Queue ID")
	if err == nil || !strings.Contains(err.Error(), "Queue ID is required") {
		t.Errorf("Wrong match. expect: required error, got: %v", err)
	}
	if res != uuid.Nil {
		t.Errorf("Wrong match. expect: nil uuid, got: %s", res)
	}
}

func Test_parseUUIDs(t *testing.T) {
	id1 := uuid.FromStringOrNil("11111111-1111-4111-8111-111111111111")
	id2 := uuid.FromStringOrNil("22222222-2222-4222-8222-222222222222")

	tests := []struct {
		name string

		input string

		expectRes []uuid.UUID
		expectErr bool
	}{
		{
			name:      "empty input returns an empty non-nil slice",
			input:     "",
			expectRes: []uuid.UUID{},
		},
		{
			name:      "one id",
			input:     `["11111111-1111-4111-8111-111111111111"]`,
			expectRes: []uuid.UUID{id1},
		},
		{
			name:      "two ids keep their order",
			input:     `["22222222-2222-4222-8222-222222222222","11111111-1111-4111-8111-111111111111"]`,
			expectRes: []uuid.UUID{id2, id1},
		},
		{
			name:      "empty array",
			input:     `[]`,
			expectRes: []uuid.UUID{},
		},
		{
			name:      "not json",
			input:     `not-json`,
			expectErr: true,
		},
		{
			name:      "not an array",
			input:     `{"id":"11111111-1111-4111-8111-111111111111"}`,
			expectErr: true,
		},
		{
			name:      "an element is not a uuid",
			input:     `["11111111-1111-4111-8111-111111111111","zzz"]`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parseUUIDs(tt.input)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}

			if res == nil {
				t.Fatalf("Wrong match. expect a non-nil slice")
			}
			if len(res) != len(tt.expectRes) {
				t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
			for i := range res {
				if res[i] != tt.expectRes[i] {
					t.Errorf("Wrong match. index: %d, expect: %s, got: %s", i, tt.expectRes[i], res[i])
				}
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	type item struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	out := captureStdout(t, func() {
		if err := printJSON(&item{Name: "queue-a", Count: 2}); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	expect := "{\n  \"name\": \"queue-a\",\n  \"count\": 2\n}\n"
	if out != expect {
		t.Errorf("Wrong match.\nexpect: %q\ngot: %q", expect, out)
	}
}

func Test_printJSON_unsupportedValue(t *testing.T) {
	out := captureStdout(t, func() {
		err := printJSON(make(chan int))
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("Wrong match. expect: marshal error, got: %v", err)
		}
	})

	if out != "" {
		t.Errorf("Wrong match. nothing must be printed on failure, got: %q", out)
	}
}

func Test_runCreate_validation(t *testing.T) {
	validCustomerID := "6b4b2b1e-3d0c-4a6e-9b6a-0f1f2f3a4b5c"

	tests := []struct {
		name string

		values map[string]string

		expectInErr []string
	}{
		{
			name:        "customer id is missing",
			values:      map[string]string{"name": "queue-a"},
			expectInErr: []string{"failed to resolve customer ID", "Customer ID is required"},
		},
		{
			name:        "customer id is malformed",
			values:      map[string]string{"customer-id": "bad", "name": "queue-a"},
			expectInErr: []string{"failed to resolve customer ID", "invalid format for Customer ID"},
		},
		{
			name:        "name is missing",
			values:      map[string]string{"customer-id": validCustomerID},
			expectInErr: []string{"name is required"},
		},
		{
			name: "tag ids are not a json array",
			values: map[string]string{
				"customer-id": validCustomerID,
				"name":        "queue-a",
				"tag-ids":     "not-json",
			},
			expectInErr: []string{"failed to parse tag IDs"},
		},
		{
			name: "a tag id is not a uuid",
			values: map[string]string{
				"customer-id": validCustomerID,
				"name":        "queue-a",
				"tag-ids":     `["zzz"]`,
			},
			expectInErr: []string{"failed to parse tag IDs"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalState(t)
			for k, v := range tt.values {
				viper.Set(k, v)
			}

			err := runCreate(&cobra.Command{}, nil)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			for _, s := range tt.expectInErr {
				if !strings.Contains(err.Error(), s) {
					t.Errorf("Wrong match. expect the error to contain %q, got: %v", s, err)
				}
			}
		})
	}
}
