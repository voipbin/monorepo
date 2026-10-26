package main

import (
	"bytes"
	"context"
	"errors"
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
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-schedule-manager/internal/config"
	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/dbhandler"
)

// useTestConfig replaces the process-wide config with synthetic values for the
// test and restores everything it touches afterwards. The config singleton is
// loaded once before any command runs, so the command's own load never
// overwrites the test values and no environment value can reach a real server.
func useTestConfig(t *testing.T, cfg config.Config) {
	t.Helper()

	viper.Reset()
	config.LoadGlobalConfig()

	oldConfig := *config.Get()
	oldFormatter := logrus.StandardLogger().Formatter
	oldLevel := logrus.GetLevel()
	*config.Get() = cfg

	t.Cleanup(func() {
		*config.Get() = oldConfig
		viper.Reset()
		logrus.SetFormatter(oldFormatter)
		logrus.SetLevel(oldLevel)
	})
}

// invalidDSNConfig has a DSN that fails to parse (no network is used) and an
// unused loopback cache address.
func invalidDSNConfig() config.Config {
	return config.Config{
		DatabaseDSN:  "not-a-dsn",
		RedisAddress: "127.0.0.1:1",
	}
}

func runCLI(t *testing.T, args ...string) error {
	t.Helper()

	cmd := initCommand()
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	return cmd.Execute()
}

// captureStdout runs fn and returns what it wrote to os.Stdout. The pipe is
// drained in a goroutine so a large output cannot block the writer.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	old := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = old

	return <-done
}

func childNames(cmd *cobra.Command) []string {
	res := []string{}
	for _, c := range cmd.Commands() {
		res = append(res, c.Name())
	}
	sort.Strings(res)

	return res
}

func Test_initCommand_tree(t *testing.T) {
	useTestConfig(t, invalidDSNConfig())

	root := initCommand()

	tests := []struct {
		path []string

		expectChildren []string
		expectFlags    []string
	}{
		{path: nil, expectChildren: []string{"execution", "migrate", "schedule"}},
		{path: []string{"schedule"}, expectChildren: []string{"disable", "enable", "get", "list"}},
		{path: []string{"execution"}, expectChildren: []string{"list"}},
		{path: []string{"migrate"}, expectChildren: []string{"down", "force", "status", "up"}},
		{path: []string{"schedule", "list"}, expectFlags: []string{"customer-id", "deleted"}},
		{path: []string{"execution", "list"}, expectFlags: []string{"schedule-id"}},
	}

	for _, tt := range tests {
		name := strings.Join(append([]string{"root"}, tt.path...), "/")
		t.Run(name, func(t *testing.T) {
			cmd, _, err := root.Find(tt.path)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if tt.expectChildren != nil {
				got := childNames(cmd)
				if strings.Join(got, ",") != strings.Join(tt.expectChildren, ",") {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChildren, got)
				}
			}
			for _, flag := range tt.expectFlags {
				if cmd.Flags().Lookup(flag) == nil {
					t.Errorf("Wrong match. expect: flag %s, got: none", flag)
				}
			}
		})
	}

	// the infrastructure flags are persistent so every subcommand accepts them
	if root.PersistentFlags().Lookup("database_dsn") == nil {
		t.Errorf("Wrong match. expect: persistent flag database_dsn, got: none")
	}
}

func Test_commands_fail_at_handler_init(t *testing.T) {
	useTestConfig(t, invalidDSNConfig())

	tests := []struct {
		name string
		args []string
	}{
		{"schedule list", []string{"schedule", "list"}},
		{"schedule get", []string{"schedule", "get", "number-renew"}},
		{"schedule enable", []string{"schedule", "enable", "number-renew"}},
		{"schedule disable", []string{"schedule", "disable", "number-renew"}},
		{"execution list", []string{"execution", "list"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runCLI(t, tt.args...)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			for _, want := range []string{"failed to initialize handlers", "could not connect to the database"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("Wrong match. expect %q in the error, got: %v", want, err)
				}
			}
		})
	}
}

func Test_commands_reject_wrong_argument_count(t *testing.T) {
	useTestConfig(t, invalidDSNConfig())

	for _, sub := range []string{"get", "enable", "disable"} {
		t.Run(sub, func(t *testing.T) {
			err := runCLI(t, "schedule", sub)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
				t.Errorf("Wrong match. expect: the argument count error, got: %v", err)
			}
			if strings.Contains(err.Error(), "failed to initialize handlers") {
				t.Errorf("Wrong match. expect: rejection before any connection, got: %v", err)
			}
		})
	}
}

func Test_migrate_status_fails_at_migrator_open(t *testing.T) {
	useTestConfig(t, invalidDSNConfig())

	err := runCLI(t, "migrate", "status")
	if err == nil || !strings.Contains(err.Error(), "could not initialize the migrator") {
		t.Errorf("Wrong match. expect: migrator init error, got: %v", err)
	}
}

func Test_initCache(t *testing.T) {
	mr := miniredis.RunT(t)
	useTestConfig(t, config.Config{RedisAddress: mr.Addr()})

	cache, err := initCache()
	if err != nil || cache == nil {
		t.Errorf("Wrong match. expect: a connected cache, got: %v, %v", cache, err)
	}
}

func Test_printJSON(t *testing.T) {
	t.Run("indented output", func(t *testing.T) {
		out := captureStdout(t, func() {
			if err := printJSON(map[string]int{"a": 1}); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})

		expect := "{\n  \"a\": 1\n}\n"
		if out != expect {
			t.Errorf("Wrong match.\nexpect: %q\ngot: %q", expect, out)
		}
	})

	t.Run("unmarshalable value", func(t *testing.T) {
		err := printJSON(make(chan int))
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("Wrong match. expect: marshal error, got: %v", err)
		}
	})
}

func Test_resolveSchedule(t *testing.T) {
	id := uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8")
	errBoom := errors.New("synthetic failure")
	found := &schedule.Schedule{Identity: commonidentity.Identity{ID: id}, Name: "number-renew"}

	tests := []struct {
		name string

		nameOrID string

		responseSchedule *schedule.Schedule
		responseErr      error

		expectErrContains string
	}{
		{
			name:             "a uuid resolves by id",
			nameOrID:         id.String(),
			responseSchedule: found,
		},
		{
			name:              "a uuid lookup failure names the id",
			nameOrID:          id.String(),
			responseErr:       errBoom,
			expectErrContains: "could not get the schedule by id. id: " + id.String(),
		},
		{
			name:             "anything else resolves by name in the platform namespace",
			nameOrID:         "number-renew",
			responseSchedule: found,
		},
		{
			name:              "a name lookup failure names the name",
			nameOrID:          "number-renew",
			responseErr:       errBoom,
			expectErrContains: "could not get the schedule by name. name: number-renew",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			ctx := context.Background()

			// the strict mock allows only the lookup that matches the argument kind
			if tt.nameOrID == id.String() {
				mockDB.EXPECT().ScheduleGet(ctx, id).Return(tt.responseSchedule, tt.responseErr)
			} else {
				mockDB.EXPECT().ScheduleGetByCustomerIDName(ctx, uuid.Nil, tt.nameOrID).Return(tt.responseSchedule, tt.responseErr)
			}

			res, err := resolveSchedule(ctx, mockDB, tt.nameOrID)
			if tt.expectErrContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErrContains) {
					t.Errorf("Wrong match. expect error containing %q, got: %v", tt.expectErrContains, err)
				}
				if !errors.Is(err, errBoom) {
					t.Errorf("Wrong match. expect: the original error reachable, got: %v", err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil result, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res != found {
				t.Errorf("Wrong match. expect: the stored schedule, got: %v", res)
			}
		})
	}
}
