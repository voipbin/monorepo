package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"monorepo/bin-webchat-manager/internal/config"
)

// resetGlobals gives a test a clean viper and puts the logrus settings and the
// config value back afterwards. The sync.Once inside the config package is
// private and stays consumed once LoadGlobalConfig ran, so only the value is
// restored and tests must not assert on the config singleton.
func resetGlobals(t *testing.T) {
	t.Helper()

	formatter := logrus.StandardLogger().Formatter
	level := logrus.GetLevel()
	saved := *config.Get()

	viper.Reset()
	t.Cleanup(func() {
		viper.Reset()
		logrus.SetFormatter(formatter)
		logrus.SetLevel(level)
		*config.Get() = saved
	})
}

func Test_initCommand_structure(t *testing.T) {
	expectDefaults := map[string]string{
		"prometheus_endpoint":       "/metrics",
		"prometheus_listen_address": ":2112",
		"redis_address":             "127.0.0.1:6379",
		"redis_password":            "",
		"redis_database":            "1",
	}
	// flags whose defaults look like connection strings are only checked for existence
	expectExisting := []string{"database_dsn", "rabbitmq_address"}

	// the same assertions run twice to catch state leaking between two builds
	for i := 0; i < 2; i++ {
		resetGlobals(t)

		cmd := initCommand()
		if cmd.Use != "webchat-control" {
			t.Errorf("Wrong command name. expect: webchat-control, got: %s", cmd.Use)
		}

		for name, expect := range expectDefaults {
			f := cmd.PersistentFlags().Lookup(name)
			if f == nil {
				t.Errorf("Missing flag: %s", name)
				continue
			}
			if f.DefValue != expect {
				t.Errorf("Wrong default. flag: %s, expect: %q, got: %q", name, expect, f.DefValue)
			}
		}
		for _, name := range expectExisting {
			if cmd.PersistentFlags().Lookup(name) == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}

		found := false
		for _, sub := range cmd.Commands() {
			if sub.Name() == "migrate" {
				found = true
			}
		}
		if !found {
			t.Errorf("The migrate subcommand is not registered.")
		}
	}
}

func Test_initCommand_persistentPreRun(t *testing.T) {
	resetGlobals(t)
	t.Setenv("DATABASE_DSN", "fake-dsn-value")

	cmd := initCommand()
	if err := cmd.PersistentPreRunE(cmd, nil); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the environment binding built by initCommand
	if got := viper.GetString("database_dsn"); got != "fake-dsn-value" {
		t.Errorf("Wrong value. expect: fake-dsn-value, got: %s", got)
	}
}

// captureStdout returns a function that restores os.Stdout and gives back what
// was written. The pipe is read in a goroutine so a large output can not block.
func captureStdout(t *testing.T) func() string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe. err: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	restored := false
	restore := func() string {
		if restored {
			return ""
		}
		restored = true

		os.Stdout = orig
		_ = w.Close()
		return <-done
	}
	t.Cleanup(func() { _ = restore() })

	return restore
}

func Test_main_help(t *testing.T) {
	resetGlobals(t)

	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{"webchat-control", "--help"}

	restore := captureStdout(t)
	main() // the help output returns without exiting the process
	out := restore()

	for _, expect := range []string{"webchat-control", "migrate", "--database_dsn"} {
		if !strings.Contains(out, expect) {
			t.Errorf("Wrong output. expect: containing %q, got: %s", expect, out)
		}
	}
}

// The migrate command opens only the database. The connection string points to
// a unix socket that does not exist, so the failure is local and immediate, and
// the message proves that the environment value was used.
func Test_initCommand_migrateStatus_unreachableDatabase(t *testing.T) {
	resetGlobals(t)

	socketPath := filepath.Join(t.TempDir(), "missing.sock")
	t.Setenv("DATABASE_DSN", "unix("+socketPath+")/synthetic")

	cmd := initCommand()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"migrate", "status"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !strings.Contains(err.Error(), socketPath) {
		t.Errorf("Wrong error. expect: containing the socket path, got: %v", err)
	}
}
