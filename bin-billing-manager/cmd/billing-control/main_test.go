package main

import (
	"bytes"
	"io"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-billing-manager/internal/config"
)

// unreachableDSN points at a unix socket that does not exist, so a connection attempt fails at once
// without any network access.
const unreachableDSN = "unix(/nonexistent/billing-test.sock)/billing_test?timeout=1s"

// useUnreachableDatabase installs a process config whose database cannot be reached and restores
// the previous config and the viper state when the test ends.
func useUnreachableDatabase(t *testing.T) {
	t.Helper()

	saved := *config.Get()
	t.Cleanup(func() {
		*config.Get() = saved
		viper.Reset()
	})

	*config.Get() = config.Config{DatabaseDSN: unreachableDSN}
}

// captureStdout runs fn and returns what it wrote to the standard output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	fn()

	os.Stdout = orig
	_ = w.Close()

	var buf bytes.Buffer
	if _, errCopy := io.Copy(&buf, r); errCopy != nil {
		t.Fatalf("could not read the output: %v", errCopy)
	}
	_ = r.Close()

	return buf.String()
}

func commandNames(cmds []*cobra.Command) []string {
	res := []string{}
	for _, c := range cmds {
		res = append(res, c.Name())
	}
	sort.Strings(res)
	return res
}

func Test_initCommand(t *testing.T) {
	origFormatter := logrus.StandardLogger().Formatter
	origLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(origFormatter)
		logrus.SetLevel(origLevel)
		viper.Reset()
	})

	root := initCommand()

	if root.Use != "billing-control" {
		t.Errorf("wrong root name. expect: billing-control, got: %s", root.Use)
	}
	if root.PersistentPreRunE == nil {
		t.Errorf("the root command must load the config before any subcommand runs")
	}

	for _, flagName := range []string{"database_dsn", "redis_address", "redis_password", "redis_database", "rabbitmq_address"} {
		if root.PersistentFlags().Lookup(flagName) == nil {
			t.Errorf("the root command must expose the persistent flag %s", flagName)
		}
	}

	tests := []struct {
		path           string
		expectChildren []string
	}{
		{"", []string{"account", "billing", "migrate", "topup"}},
		{"account", []string{"add-balance", "add-tokens", "create", "delete", "get", "list", "set-status", "subtract-balance", "subtract-tokens", "update", "update-payment-info", "update-plan-type"}},
		{"billing", []string{"get", "list"}},
		{"topup", []string{"run"}},
		{"migrate", []string{"down", "force", "status", "up"}},
	}

	for _, tt := range tests {
		t.Run("children of "+tt.path, func(t *testing.T) {
			cmd := root
			if tt.path != "" {
				found, _, err := root.Find([]string{tt.path})
				if err != nil || found == root {
					t.Fatalf("could not find the command %s: %v", tt.path, err)
				}
				cmd = found
			}

			got := commandNames(cmd.Commands())
			if strings.Join(got, ",") != strings.Join(tt.expectChildren, ",") {
				t.Errorf("wrong subcommands. expect: %v, got: %v", tt.expectChildren, got)
			}
		})
	}
}

func Test_commandBuilders(t *testing.T) {
	type flagSpec struct {
		name         string
		flagType     string
		defaultValue string
	}

	tests := []struct {
		name         string
		build        func() *cobra.Command
		expectUse    string
		expectShort  string
		expectsRunE  bool
		expectsFlags []flagSpec
	}{
		{
			"account create", cmdAccountCreate, "create", "Create a new account", true,
			[]flagSpec{{"customer-id", "string", ""}, {"name", "string", ""}, {"detail", "string", ""}, {"payment-type", "string", "prepaid"}, {"payment-method", "string", ""}},
		},
		{"account get", cmdAccountGet, "get", "Get an account by ID", true, []flagSpec{{"id", "string", ""}}},
		{
			"account list", cmdAccountList, "list", "Get account list", true,
			[]flagSpec{{"limit", "int", "100"}, {"token", "string", ""}, {"customer-id", "string", ""}},
		},
		{
			"account update", cmdAccountUpdate, "update", "Update an account basic info", true,
			[]flagSpec{{"id", "string", ""}, {"name", "string", ""}, {"detail", "string", ""}},
		},
		{
			"account update-payment-info", cmdAccountUpdatePaymentInfo, "update-payment-info", "Update account payment info", true,
			[]flagSpec{{"id", "string", ""}, {"payment-type", "string", ""}, {"payment-method", "string", ""}},
		},
		{
			"account update-plan-type", cmdAccountUpdatePlanType, "update-plan-type", "Update account plan type", true,
			[]flagSpec{{"id", "string", ""}, {"plan-type", "string", ""}},
		},
		{"account delete", cmdAccountDelete, "delete", "Delete an account", true, []flagSpec{{"id", "string", ""}}},
		{
			"account add-balance", cmdAccountAddBalance, "add-balance", "Add balance (credit micros) to an account", true,
			[]flagSpec{{"id", "string", ""}, {"amount", "int64", "0"}},
		},
		{
			"account subtract-balance", cmdAccountSubtractBalance, "subtract-balance", "Subtract balance (credit micros) from an account", true,
			[]flagSpec{{"id", "string", ""}, {"amount", "int64", "0"}},
		},
		{
			"account add-tokens", cmdAccountAddTokens, "add-tokens", "Add tokens to an account", true,
			[]flagSpec{{"id", "string", ""}, {"amount", "int64", "0"}},
		},
		{
			"account subtract-tokens", cmdAccountSubtractTokens, "subtract-tokens", "Subtract tokens from an account", true,
			[]flagSpec{{"id", "string", ""}, {"amount", "int64", "0"}},
		},
		{
			"account set-status", cmdAccountSetStatus, "set-status", "Set billing account status (active, frozen, deleted)", true,
			[]flagSpec{{"id", "string", ""}, {"status", "string", ""}},
		},
		{"billing get", cmdBillingGet, "get", "Get a billing record by ID", true, []flagSpec{{"id", "string", ""}}},
		{
			"billing list", cmdBillingList, "list", "Get billing records list", true,
			[]flagSpec{{"limit", "int", "100"}, {"token", "string", ""}, {"customer-id", "string", ""}, {"account-id", "string", ""}},
		},
		{"topup run", cmdTopUpRun, "run", "Run immediate monthly top-up processing for all eligible accounts", true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.build()

			if cmd.Use != tt.expectUse {
				t.Errorf("wrong Use. expect: %s, got: %s", tt.expectUse, cmd.Use)
			}
			if cmd.Short != tt.expectShort {
				t.Errorf("wrong Short. expect: %s, got: %s", tt.expectShort, cmd.Short)
			}
			if (cmd.RunE != nil) != tt.expectsRunE {
				t.Errorf("wrong RunE presence. expect: %v", tt.expectsRunE)
			}

			gotFlags := 0
			for _, line := range strings.Split(cmd.Flags().FlagUsagesWrapped(0), "\n") {
				if strings.TrimSpace(line) != "" {
					gotFlags++
				}
			}
			if gotFlags != len(tt.expectsFlags) {
				t.Errorf("wrong flag count. expect: %d, got: %d", len(tt.expectsFlags), gotFlags)
			}

			for _, spec := range tt.expectsFlags {
				f := cmd.Flags().Lookup(spec.name)
				if f == nil {
					t.Errorf("missing flag %s", spec.name)
					continue
				}
				if f.Value.Type() != spec.flagType {
					t.Errorf("flag %s: wrong type. expect: %s, got: %s", spec.name, spec.flagType, f.Value.Type())
				}
				if f.DefValue != spec.defaultValue {
					t.Errorf("flag %s: wrong default. expect: %q, got: %q", spec.name, spec.defaultValue, f.DefValue)
				}
			}
		})
	}
}

// Test_runCommands_databaseUnavailable verifies that every command stops with the initialization
// error when the database cannot be reached, before it reads any argument.
func Test_runCommands_databaseUnavailable(t *testing.T) {
	tests := []struct {
		name         string
		run          func(cmd *cobra.Command, args []string) error
		expectPrefix string
	}{
		{"account create", runAccountCreate, "failed to initialize handlers"},
		{"account get", runAccountGet, "failed to initialize handlers"},
		{"account list", runAccountList, "failed to initialize handlers"},
		{"account update", runAccountUpdate, "failed to initialize handlers"},
		{"account update-payment-info", runAccountUpdatePaymentInfo, "failed to initialize handlers"},
		{"account update-plan-type", runAccountUpdatePlanType, "failed to initialize handlers"},
		{"account delete", runAccountDelete, "failed to initialize handlers"},
		{"account add-balance", runAccountAddBalance, "failed to initialize handlers"},
		{"account subtract-balance", runAccountSubtractBalance, "failed to initialize handlers"},
		{"account add-tokens", runAccountAddTokens, "failed to initialize handlers"},
		{"account subtract-tokens", runAccountSubtractTokens, "failed to initialize handlers"},
		{"account set-status", runAccountSetStatus, "failed to initialize handlers"},
		{"billing get", runBillingGet, "failed to initialize handlers"},
		{"billing list", runBillingList, "failed to initialize handlers"},
		{"topup run", runTopUpRun, "failed to initialize database"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useUnreachableDatabase(t)

			err := tt.run(nil, nil)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !strings.HasPrefix(err.Error(), tt.expectPrefix) {
				t.Errorf("wrong error prefix. expect: %q, got: %v", tt.expectPrefix, err)
			}
			if !strings.Contains(err.Error(), "could not connect to the database") {
				t.Errorf("the connection failure must be reported. got: %v", err)
			}
		})
	}
}

func Test_initHandlers_databaseUnavailable(t *testing.T) {
	useUnreachableDatabase(t)

	accHandler, billHandler, err := initHandlers()
	if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("wrong error. expect the connection failure, got: %v", err)
	}
	if accHandler != nil || billHandler != nil {
		t.Errorf("no handler may be returned on failure. got: %v, %v", accHandler, billHandler)
	}
}

func Test_initDB_databaseUnavailable(t *testing.T) {
	useUnreachableDatabase(t)

	db, err := initDB()
	if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("wrong error. expect the connection failure, got: %v", err)
	}
	if db != nil {
		t.Errorf("no handler may be returned on failure. got: %v", db)
	}
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expectRes uuid.UUID
		expectErr string
	}{
		{"valid", "5c8d4a1e-3b2f-4e6a-9d0c-7a1b2c3d4e5f", uuid.FromStringOrNil("5c8d4a1e-3b2f-4e6a-9d0c-7a1b2c3d4e5f"), ""},
		{"empty", "", uuid.Nil, "Account ID is required"},
		{"not a uuid", "not-a-uuid", uuid.Nil, "invalid format for Account ID: 'not-a-uuid' is not a valid UUID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(viper.Reset)
			viper.Reset()
			viper.Set("id", tt.value)

			res, err := resolveUUID("id", "Account ID")
			if tt.expectErr == "" {
				if err != nil {
					t.Fatalf("expected ok, got: %v", err)
				}
			} else if err == nil || err.Error() != tt.expectErr {
				t.Fatalf("wrong error. expect: %q, got: %v", tt.expectErr, err)
			}

			if res != tt.expectRes {
				t.Errorf("wrong result. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	t.Run("indented output", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(map[string]any{"name": "test", "count": 2})
		})
		if err != nil {
			t.Fatalf("expected ok, got: %v", err)
		}

		expect := "{\n  \"count\": 2,\n  \"name\": \"test\"\n}\n"
		if out != expect {
			t.Errorf("wrong output. expect: %q, got: %q", expect, out)
		}
	})

	t.Run("value that cannot be encoded", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(make(chan int))
		})
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("wrong error. expect the marshal failure, got: %v", err)
		}
		if out != "" {
			t.Errorf("nothing may be printed on failure. got: %q", out)
		}
	})
}

// Test_initCommand_execute runs the assembled command tree the way the binary does and verifies
// that the flags of the root command reach the configuration and the subcommand runs.
func Test_initCommand_execute(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectErrSub string
	}{
		{"top-up run stops at the unreachable database", []string{"topup", "run", "--database_dsn", unreachableDSN}, "failed to initialize database: could not connect to the database: database ping error: dial unix /nonexistent/billing-test.sock"},
		{"migrate status stops at the unreachable database", []string{"migrate", "status", "--database_dsn", unreachableDSN}, "dial unix /nonexistent/billing-test.sock"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origFormatter := logrus.StandardLogger().Formatter
			origLevel := logrus.GetLevel()
			t.Cleanup(func() {
				logrus.SetFormatter(origFormatter)
				logrus.SetLevel(origLevel)
			})
			useUnreachableDatabase(t)

			root := initCommand()
			root.SetArgs(tt.args)
			root.SilenceUsage = true
			root.SilenceErrors = true
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)

			err := root.Execute()
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if tt.expectErrSub != "" && !strings.Contains(err.Error(), tt.expectErrSub) {
				t.Errorf("wrong error. expect to contain: %q, got: %v", tt.expectErrSub, err)
			}
			if got := viper.GetString("database_dsn"); got != unreachableDSN {
				t.Errorf("the flag value must reach the configuration source. expect: %s, got: %s", unreachableDSN, got)
			}
		})
	}
}

func Test_initCache(t *testing.T) {
	t.Run("server reachable", func(t *testing.T) {
		saved := *config.Get()
		t.Cleanup(func() { *config.Get() = saved })

		s := miniredis.RunT(t)
		*config.Get() = config.Config{RedisAddress: s.Addr()}

		res, err := initCache()
		if err != nil || res == nil {
			t.Errorf("expected a connected cache, got: %v, %v", res, err)
		}
	})

	t.Run("server down", func(t *testing.T) {
		saved := *config.Get()
		t.Cleanup(func() { *config.Get() = saved })

		s := miniredis.RunT(t)
		addr := s.Addr()
		s.Close()
		*config.Get() = config.Config{RedisAddress: addr}

		res, err := initCache()
		if err == nil || res != nil {
			t.Errorf("expected the connection failure without a handler, got: %v, %v", res, err)
		}
	})
}
