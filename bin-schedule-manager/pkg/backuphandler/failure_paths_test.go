package backuphandler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

const failureTestDSN = "testuser@tcp(127.0.0.1:3306)/voipbin"

// newFailureHandler returns a handler that writes into a fresh backup
// directory and keeps its credentials file in a private temp directory.
func newFailureHandler(t *testing.T, retention int, onRun func(args []string)) (*backupHandler, string, string) {
	t.Helper()

	dir := t.TempDir()
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)

	runner := &fakeRunner{
		payload: []byte("-- fake dump\n"),
		onRun:   func(_ string, args []string) { onRun(args) },
	}

	return &backupHandler{dsn: failureTestDSN, backupDir: dir, retentionCount: retention, runner: runner}, dir, tmp
}

func defaultsPathOf(args []string) string {
	return strings.TrimPrefix(args[0], "--defaults-extra-file=")
}

// hasLog reports whether the hook captured an entry whose message contains text.
func hasLog(hook *logtest.Hook, text string) bool {
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, text) {
			return true
		}
	}
	return false
}

// skipIfPermissionsNotEnforced skips the test when directory permission bits
// do not stop the current user (for example under root): a restricted
// directory can then still be listed or its files removed.
func skipIfPermissionsNotEnforced(t *testing.T) {
	t.Helper()

	probe := t.TempDir()
	victim := filepath.Join(probe, "victim")
	if err := os.WriteFile(victim, nil, 0o600); err != nil {
		t.Fatalf("could not create the probe file. err: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(probe, 0o700) })

	if err := os.Chmod(probe, 0o300); err != nil {
		t.Fatalf("could not restrict the probe directory. err: %v", err)
	}
	if _, err := os.ReadDir(probe); err == nil {
		t.Skip("directory permissions are not enforced for this user")
	}

	if err := os.Chmod(probe, 0o500); err != nil {
		t.Fatalf("could not restrict the probe directory. err: %v", err)
	}
	if err := os.Remove(victim); err == nil {
		t.Skip("directory permissions are not enforced for this user")
	}
}

func Test_Backup_defaults_file_removed_during_run(t *testing.T) {
	oldHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(oldHooks) })
	hook := logtest.NewGlobal()

	h, _, _ := newFailureHandler(t, 7, func(args []string) {
		_ = os.Remove(defaultsPathOf(args))
	})

	res, err := h.Backup(context.Background())
	if err != nil || res == nil {
		t.Fatalf("Wrong match. expect: the backup still succeeds, got: %v and %v", res, err)
	}
	if _, errStat := os.Stat(res.Path); errStat != nil || res.Bytes <= 0 {
		t.Errorf("Wrong match. expect: the dump file exists with a size, got: %v, bytes: %d", errStat, res.Bytes)
	}
	if !hasLog(hook, "Could not remove the defaults file") {
		t.Errorf("Wrong match. expect: the failed credentials file removal is logged")
	}
}

func Test_Backup_dump_file_removed_during_run(t *testing.T) {
	var dir string
	removed := 0
	h, d, tmp := newFailureHandler(t, 7, func(_ []string) {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if os.Remove(filepath.Join(dir, e.Name())) == nil {
				removed++
			}
		}
	})
	dir = d

	res, err := h.Backup(context.Background())
	if err == nil || res != nil {
		t.Fatalf("Wrong match. expect: error and nil result, got: %v and %v", res, err)
	}
	if removed != 1 || !strings.Contains(err.Error(), "could not stat the backup file") {
		t.Errorf("Wrong match. expect: the one dump file removed and a stat error, got: removed: %d, err: %v", removed, err)
	}
	if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
		t.Errorf("Wrong match. expect: no credentials file left, got: %d entries", len(entries))
	}
}

func Test_Backup_prune_failures_are_logged_not_fatal(t *testing.T) {
	tests := []struct {
		name string
		mode os.FileMode // applied to the backup directory while the dump runs

		oldFiles int
	}{
		{"directory cannot be listed", 0o300, 0},
		{"old dumps cannot be removed", 0o500, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skipIfPermissionsNotEnforced(t)

			oldHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
			t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(oldHooks) })
			hook := logtest.NewGlobal()

			var dir string
			h, d, _ := newFailureHandler(t, 1, func(_ []string) {
				if err := os.Chmod(dir, tt.mode); err != nil {
					t.Errorf("could not restrict the backup directory. err: %v", err)
				}
			})
			dir = d
			// registered after TempDir, so it runs first and the directory can be removed
			t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

			for i := 0; i < tt.oldFiles; i++ {
				name := backupFilePrefix + "2000010" + string(rune('1'+i)) + "T000000Z" + backupFileSuffix
				if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
					t.Fatalf("could not create the old dump. err: %v", err)
				}
			}

			res, err := h.Backup(context.Background())
			if err != nil || res == nil {
				t.Fatalf("Wrong match. expect: the backup succeeds although pruning fails, got: %v and %v", res, err)
			}
			if !hasLog(hook, "Could not prune old backups") {
				t.Errorf("Wrong match. expect: the pruning failure is logged")
			}
			if _, errStat := os.Stat(res.Path); errStat != nil {
				t.Errorf("Wrong match. expect: the new dump is kept, got: %v", errStat)
			}
		})
	}
}
