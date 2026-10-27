package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"monorepo/bin-email-manager/internal/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// closedLoopbackAddr returns a loopback address that nothing listens on.
func closedLoopbackAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	return addr
}

// useRefusedBackends points the global config at refused loopback addresses and restores it afterwards.
func useRefusedBackends(t *testing.T) {
	t.Helper()

	cfg := config.Get()
	saved := *cfg
	t.Cleanup(func() { *cfg = saved })

	cfg.DatabaseDSN = fmt.Sprintf("user@tcp(%s)/db", closedLoopbackAddr(t))
	cfg.RedisAddress = closedLoopbackAddr(t)
	cfg.RedisPassword = ""
	cfg.RedisDatabase = 0
}

// resetViper clears the viper state set by a test.
func resetViper(t *testing.T) {
	t.Helper()
	t.Cleanup(viper.Reset)
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name string

		flagName string
		label    string
		value    string

		expectID  uuid.UUID
		expectErr string
	}{
		{
			name:     "valid",
			flagName: "id",
			label:    "Email ID",
			value:    "5e8f0f0a-dddd-4ddd-8ddd-000000000001",
			expectID: uuid.FromStringOrNil("5e8f0f0a-dddd-4ddd-8ddd-000000000001"),
		},
		{
			name:     "valid_customer_flag",
			flagName: "customer-id",
			label:    "Customer ID",
			value:    "5e8f0f0a-dddd-4ddd-8ddd-000000000002",
			expectID: uuid.FromStringOrNil("5e8f0f0a-dddd-4ddd-8ddd-000000000002"),
		},
		{
			name:      "empty",
			flagName:  "id",
			label:     "Email ID",
			value:     "",
			expectErr: "Email ID is required",
		},
		{
			name:      "malformed",
			flagName:  "customer-id",
			label:     "Customer ID",
			value:     "not-a-uuid",
			expectErr: "invalid format for Customer ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:      "nil_uuid_is_rejected",
			flagName:  "id",
			label:     "Email ID",
			value:     "00000000-0000-0000-0000-000000000000",
			expectErr: "invalid format for Email ID: '00000000-0000-0000-0000-000000000000' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			viper.Set(tt.flagName, tt.value)

			res, err := resolveUUID(tt.flagName, tt.label)
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong match. expect error: %s, got: %v", tt.expectErr, err)
				}
				if res != uuid.Nil {
					t.Errorf("Expected a nil uuid. got: %s", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res != tt.expectID {
				t.Errorf("Wrong uuid. expect: %s, got: %s", tt.expectID, res)
			}
		})
	}
}

// captureStdout runs f and returns what it wrote to the standard output.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe. err: %v", err)
	}

	saved := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = saved })

	f()

	_ = w.Close()
	os.Stdout = saved

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("could not read the pipe. err: %v", err)
	}
	_ = r.Close()

	return string(out)
}

func Test_printJSON(t *testing.T) {
	var err error
	out := captureStdout(t, func() {
		err = printJSON(map[string]any{"id": "synthetic-id", "list": []int{1, 2}})
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := "{\n  \"id\": \"synthetic-id\",\n  \"list\": [\n    1,\n    2\n  ]\n}\n"
	if out != expect {
		t.Errorf("Wrong output. expect: %q, got: %q", expect, out)
	}
}

func Test_printJSON_marshalFails(t *testing.T) {
	var err error
	out := captureStdout(t, func() {
		err = printJSON(make(chan int))
	})

	if err == nil || !strings.HasPrefix(err.Error(), "failed to marshal JSON: ") {
		t.Errorf("Wrong match. expect a marshal error, got: %v", err)
	}
	if out != "" {
		t.Errorf("Expected no output. got: %q", out)
	}
}

func Test_initCommand(t *testing.T) {
	// config.Bootstrap sets the logrus formatter and level, so they are restored
	savedFormatter := logrus.StandardLogger().Formatter
	savedLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(savedFormatter)
		logrus.SetLevel(savedLevel)
	})
	resetViper(t)

	root := initCommand()
	if root.Use != "email-control" {
		t.Errorf("Wrong root command. got: %s", root.Use)
	}

	tests := []struct {
		name string

		path  []string
		flags map[string]string
	}{
		{name: "email_get", path: []string{"email", "get"}, flags: map[string]string{"id": ""}},
		{name: "email_list", path: []string{"email", "list"}, flags: map[string]string{"limit": "100", "token": "", "customer-id": ""}},
		{name: "email_delete", path: []string{"email", "delete"}, flags: map[string]string{"id": ""}},
		{name: "migrate_status", path: []string{"migrate", "status"}},
		{name: "migrate_up", path: []string{"migrate", "up"}},
		{name: "migrate_down", path: []string{"migrate", "down"}},
		{name: "migrate_force", path: []string{"migrate", "force"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, _, err := root.Find(tt.path)
			if err != nil || sub == nil || sub.Name() != tt.path[len(tt.path)-1] {
				t.Fatalf("Wrong match. expect the %v command, got: %v, err: %v", tt.path, sub, err)
			}

			for name, def := range tt.flags {
				f := sub.Flags().Lookup(name)
				if f == nil {
					t.Errorf("Missing flag: %s", name)
					continue
				}
				if f.DefValue != def {
					t.Errorf("Wrong default for %s. expect: %q, got: %q", name, def, f.DefValue)
				}
			}
		})
	}

	// the infrastructure flags are bound on the root command
	for _, name := range []string{"database_dsn", "redis_address", "rabbitmq_address"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("Missing persistent flag: %s", name)
		}
	}
}

func Test_initHandler_databaseConnectFails(t *testing.T) {
	useRefusedBackends(t)

	handler, err := initHandler()
	if err == nil || !strings.HasPrefix(err.Error(), "could not connect to the database: database ping error: ") {
		t.Errorf("Wrong match. expect a database error, got: %v", err)
	}
	if handler != nil {
		t.Errorf("Expected a nil handler. got: %v", handler)
	}
}

func Test_initCache_connectFails(t *testing.T) {
	useRefusedBackends(t)

	cache, err := initCache()
	if err == nil || !strings.HasPrefix(err.Error(), "could not connect to the cache: ") {
		t.Errorf("Wrong match. expect a cache error, got: %v", err)
	}
	if cache != nil {
		t.Errorf("Expected a nil cache. got: %v", cache)
	}
}

func Test_run_initializationFails(t *testing.T) {
	tests := []struct {
		name string

		run func() error
	}{
		{name: "get", run: func() error { return runGet(cmdGet(), nil) }},
		{name: "list", run: func() error { return runList(cmdList(), nil) }},
		{name: "delete", run: func() error { return runDelete(cmdDelete(), nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useRefusedBackends(t)

			err := tt.run()
			if err == nil || !strings.HasPrefix(err.Error(), "failed to initialize handlers: could not connect to the database: ") {
				t.Errorf("Wrong match. expect an initialization error, got: %v", err)
			}
		})
	}
}

func Test_initCache_connected(t *testing.T) {
	mr := miniredis.RunT(t)

	cfg := config.Get()
	saved := *cfg
	t.Cleanup(func() { *cfg = saved })

	cfg.RedisAddress = mr.Addr()
	cfg.RedisPassword = ""
	cfg.RedisDatabase = 0

	cache, err := initCache()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if cache == nil {
		t.Errorf("Expected a cache handler. got nil")
	}
}

// Test_initCommand_execute runs the subcommands through the root command, so the
// pre-run hook and the migrator constructor run against a refused database address.
func Test_initCommand_execute(t *testing.T) {
	savedFormatter := logrus.StandardLogger().Formatter
	savedLevel := logrus.GetLevel()
	cfg := config.Get()
	savedCfg := *cfg
	t.Cleanup(func() {
		logrus.SetFormatter(savedFormatter)
		logrus.SetLevel(savedLevel)
		*cfg = savedCfg
	})
	resetViper(t)

	dsn := fmt.Sprintf("user@tcp(%s)/db", closedLoopbackAddr(t))
	redisAddr := closedLoopbackAddr(t)

	tests := []struct {
		name string

		args      []string
		expectErr string
	}{
		{
			name:      "migrate_status_database_unreachable",
			args:      []string{"migrate", "status", "--database_dsn", dsn},
			expectErr: "",
		},
		{
			name:      "email_get_database_unreachable",
			args:      []string{"email", "get", "--database_dsn", dsn, "--redis_address", redisAddr},
			expectErr: "failed to initialize handlers: could not connect to the database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := initCommand()
			root.SilenceUsage = true
			root.SilenceErrors = true
			root.SetArgs(tt.args)

			err := root.Execute()
			if err == nil {
				t.Fatalf("Wrong match. expect an error, got nil")
			}
			if tt.expectErr != "" && !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}
