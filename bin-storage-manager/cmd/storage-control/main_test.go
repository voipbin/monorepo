package main

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-storage-manager/internal/config"
)

const (
	testUUID1 = "11111111-1111-4111-8111-111111111111"
	testUUID2 = "22222222-2222-4222-8222-222222222222"
	testUUID3 = "33333333-3333-4333-8333-333333333333"

	// invalidDSN has no slash, so the driver rejects it while parsing and no connection is attempted.
	invalidDSN = "invalid"
)

// setupTest resets the global viper and points the database config at an unparsable DSN.
// The previous values are restored when the test ends.
func setupTest(t *testing.T) {
	t.Helper()

	viper.Reset()
	old := *config.Get()
	config.Get().DatabaseDSN = invalidDSN

	t.Cleanup(func() {
		viper.Reset()
		*config.Get() = old
	})
}

// captureStdout runs f with os.Stdout redirected to a pipe and returns what was written.
// Only small outputs may be used, because the pipe is read after f returns.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe. err: %v", err)
	}

	old := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	f()

	if errClose := w.Close(); errClose != nil {
		t.Fatalf("could not close the pipe. err: %v", errClose)
	}
	os.Stdout = old

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("could not read the pipe. err: %v", err)
	}
	return string(out)
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expectID string
		expectEr string
	}{
		{
			name:     "valid",
			value:    testUUID1,
			expectID: testUUID1,
		},
		{
			name:     "empty",
			value:    "",
			expectEr: "Target ID is required",
		},
		{
			name:     "malformed",
			value:    "not-a-uuid",
			expectEr: "invalid format for Target ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:     "nil uuid is rejected",
			value:    "00000000-0000-0000-0000-000000000000",
			expectEr: "invalid format for Target ID: '00000000-0000-0000-0000-000000000000' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest(t)
			viper.Set("target-id", tt.value)

			res, err := resolveUUID("target-id", "Target ID")
			if tt.expectEr != "" {
				if err == nil || err.Error() != tt.expectEr {
					t.Fatalf("Wrong match. expect error: %q, got: %v", tt.expectEr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.String() != tt.expectID {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectID, res)
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	setupTest(t)

	type sample struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	out := captureStdout(t, func() {
		if err := printJSON(sample{Name: "abc", Count: 3}); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	expect := "{\n  \"name\": \"abc\",\n  \"count\": 3\n}\n"
	if out != expect {
		t.Errorf("Wrong match.\nexpect: %q\ngot: %q", expect, out)
	}
}

func Test_printJSON_marshalError(t *testing.T) {
	setupTest(t)

	var err error
	out := captureStdout(t, func() {
		err = printJSON(make(chan int))
	})

	if err == nil || !strings.HasPrefix(err.Error(), "failed to marshal JSON") {
		t.Errorf("Wrong match. expect error prefix: failed to marshal JSON, got: %v", err)
	}
	if out != "" {
		t.Errorf("Wrong match. expect no output, got: %q", out)
	}
}

type flagSpec struct {
	name string
	kind string
	def  string
}

func Test_commandConstructors(t *testing.T) {
	tests := []struct {
		name   string
		build  func() *cobra.Command
		use    string
		expect []flagSpec
	}{
		{
			name:  "file create",
			build: cmdFileCreate,
			use:   "create",
			expect: []flagSpec{
				{"customer-id", "string", ""},
				{"owner-id", "string", ""},
				{"reference-type", "string", ""},
				{"reference-id", "string", ""},
				{"name", "string", ""},
				{"detail", "string", ""},
				{"filename", "string", ""},
				{"bucket-name", "string", ""},
				{"filepath", "string", ""},
			},
		},
		{
			name:   "file get",
			build:  cmdFileGet,
			use:    "get",
			expect: []flagSpec{{"id", "string", ""}},
		},
		{
			name:  "file list",
			build: cmdFileList,
			use:   "list",
			expect: []flagSpec{
				{"limit", "uint64", "100"},
				{"token", "string", ""},
				{"customer-id", "string", ""},
				{"reference-id", "string", ""},
				{"reference-type", "string", ""},
			},
		},
		{
			name:   "file delete",
			build:  cmdFileDelete,
			use:    "delete",
			expect: []flagSpec{{"id", "string", ""}},
		},
		{
			name:   "account create",
			build:  cmdAccountCreate,
			use:    "create",
			expect: []flagSpec{{"customer-id", "string", ""}},
		},
		{
			name:   "account get",
			build:  cmdAccountGet,
			use:    "get",
			expect: []flagSpec{{"id", "string", ""}},
		},
		{
			name:  "account list",
			build: cmdAccountList,
			use:   "list",
			expect: []flagSpec{
				{"limit", "uint64", "100"},
				{"token", "string", ""},
				{"customer-id", "string", ""},
			},
		},
		{
			name:   "account delete",
			build:  cmdAccountDelete,
			use:    "delete",
			expect: []flagSpec{{"id", "string", ""}},
		},
		{
			name:   "recording get",
			build:  cmdRecordingGet,
			use:    "get",
			expect: []flagSpec{{"reference-id", "string", ""}},
		},
		{
			name:   "recording delete",
			build:  cmdRecordingDelete,
			use:    "delete",
			expect: []flagSpec{{"reference-id", "string", ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.build()

			if cmd.Use != tt.use {
				t.Errorf("Wrong match. expect use: %s, got: %s", tt.use, cmd.Use)
			}
			if cmd.RunE == nil {
				t.Errorf("Wrong match. expect RunE to be set")
			}

			var got []flagSpec
			for _, want := range tt.expect {
				f := cmd.Flags().Lookup(want.name)
				if f == nil {
					t.Errorf("Wrong match. expect flag: %s", want.name)
					continue
				}
				got = append(got, flagSpec{f.Name, f.Value.Type(), f.DefValue})
			}
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expect, got)
			}

			// every usage line is one defined flag, so this detects flags missing from the expectation
			if lines := strings.Count(cmd.Flags().FlagUsages(), "\n"); lines != len(tt.expect) {
				t.Errorf("Wrong match. expect flag count: %d, got: %d", len(tt.expect), lines)
			}
		})
	}
}

func Test_initCommand_tree(t *testing.T) {
	setupTest(t)

	root := initCommand()
	if root.Use != "storage-control" {
		t.Errorf("Wrong match. expect: storage-control, got: %s", root.Use)
	}
	if root.PersistentPreRunE == nil {
		t.Errorf("Wrong match. expect PersistentPreRunE to be set")
	}

	names := func(c *cobra.Command) []string {
		var res []string
		for _, sub := range c.Commands() {
			// completion and help are added by cobra on execution
			if sub.Name() == "completion" || sub.Name() == "help" {
				continue
			}
			res = append(res, sub.Name())
		}
		sort.Strings(res)
		return res
	}

	if got, expect := names(root), []string{"account", "file", "migrate", "recording"}; !reflect.DeepEqual(got, expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, got)
	}

	groups := map[string][]string{
		"file":      {"create", "delete", "get", "list"},
		"account":   {"create", "delete", "get", "list"},
		"recording": {"delete", "get"},
		"migrate":   {"down", "force", "status", "up"},
	}
	for _, sub := range root.Commands() {
		expect, ok := groups[sub.Name()]
		if !ok {
			continue
		}
		if got := names(sub); !reflect.DeepEqual(got, expect) {
			t.Errorf("Wrong match. group: %s, expect: %v, got: %v", sub.Name(), expect, got)
		}
	}

	for _, flagName := range []string{"database_dsn", "redis_address", "rabbitmq_address", "gcp_bucket_name_media"} {
		if root.PersistentFlags().Lookup(flagName) == nil {
			t.Errorf("Wrong match. expect persistent flag: %s", flagName)
		}
	}
}

func Test_initCommand_migrateStatusRejectsDSN(t *testing.T) {
	setupTest(t)

	root := initCommand()
	root.SetArgs([]string{"migrate", "status", "--database_dsn=" + invalidDSN})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})

	err := root.Execute()
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !strings.Contains(err.Error(), "could not parse the database dsn") {
		t.Errorf("Wrong match. expect error containing: could not parse the database dsn, got: %v", err)
	}
}

func Test_runFileCreate_validation(t *testing.T) {
	valid := map[string]string{
		"customer-id":    testUUID1,
		"owner-id":       testUUID2,
		"reference-id":   testUUID3,
		"reference-type": "normal",
		"name":           "name",
		"filename":       "a.wav",
		"bucket-name":    "bucket",
		"filepath":       "path/a.wav",
	}

	tests := []struct {
		name   string
		key    string
		value  string
		expect string
	}{
		{"customer id empty", "customer-id", "", "failed to resolve customer ID: Customer ID is required"},
		{"customer id malformed", "customer-id", "x", "failed to resolve customer ID: invalid format for Customer ID: 'x' is not a valid UUID"},
		{"owner id empty", "owner-id", "", "failed to resolve owner ID: Owner ID is required"},
		{"owner id malformed", "owner-id", "x", "failed to resolve owner ID: invalid format for Owner ID: 'x' is not a valid UUID"},
		{"reference id empty", "reference-id", "", "failed to resolve reference ID: Reference ID is required"},
		{"reference id malformed", "reference-id", "x", "failed to resolve reference ID: invalid format for Reference ID: 'x' is not a valid UUID"},
		{"reference type empty", "reference-type", "", "reference-type is required"},
		{"name empty", "name", "", "name is required"},
		{"filename empty", "filename", "", "filename is required"},
		{"bucket name empty", "bucket-name", "", "bucket-name is required"},
		{"filepath empty", "filepath", "", "filepath is required"},
		{"all valid reaches handler init", "detail", "", "failed to initialize handlers: could not connect to the database"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest(t)
			for k, v := range valid {
				viper.Set(k, v)
			}
			viper.Set(tt.key, tt.value)

			err := runFileCreate(nil, nil)
			if err == nil || !strings.HasPrefix(err.Error(), tt.expect) {
				t.Errorf("Wrong match.\nexpect prefix: %s\ngot: %v", tt.expect, err)
			}
		})
	}
}

func Test_runAccountCreate_validation(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		expect string
	}{
		{"customer id empty", "", "failed to resolve customer ID: Customer ID is required"},
		{"customer id malformed", "x", "failed to resolve customer ID: invalid format for Customer ID: 'x' is not a valid UUID"},
		{"valid reaches handler init", testUUID1, "failed to initialize handlers: could not connect to the database"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest(t)
			viper.Set("customer-id", tt.value)

			err := runAccountCreate(nil, nil)
			if err == nil || !strings.HasPrefix(err.Error(), tt.expect) {
				t.Errorf("Wrong match.\nexpect prefix: %s\ngot: %v", tt.expect, err)
			}
		})
	}
}

func Test_run_handlerInitFailure(t *testing.T) {
	tests := []struct {
		name string
		run  func(*cobra.Command, []string) error
	}{
		{"file get", runFileGet},
		{"file list", runFileList},
		{"file delete", runFileDelete},
		{"account get", runAccountGet},
		{"account list", runAccountList},
		{"account delete", runAccountDelete},
		{"recording get", runRecordingGet},
		{"recording delete", runRecordingDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest(t)
			// valid ids prove the failure comes from the handler init that precedes the id resolution
			for _, k := range []string{"id", "customer-id", "reference-id"} {
				viper.Set(k, testUUID1)
			}

			err := tt.run(nil, nil)
			expect := "failed to initialize handlers: could not connect to the database"
			if err == nil || !strings.HasPrefix(err.Error(), expect) {
				t.Errorf("Wrong match.\nexpect prefix: %s\ngot: %v", expect, err)
			}
		})
	}
}

func Test_initHandlers_connectFailure(t *testing.T) {
	setupTest(t)

	if _, err := initStorageHandler(); err == nil || !strings.HasPrefix(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. initStorageHandler expect prefix: could not connect to the database, got: %v", err)
	}
	if _, err := initAccountHandler(); err == nil || !strings.HasPrefix(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. initAccountHandler expect prefix: could not connect to the database, got: %v", err)
	}
}
