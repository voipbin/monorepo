//go:build linux

package backuphandler

import (
	"bytes"
	"context"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/sirupsen/logrus"
)

// limitFileSize lowers the soft file size limit of the process to bytes and
// returns the function that restores it. The limit is process-wide, so the
// tests that use it are not parallel and keep the window short. The Go
// runtime ignores SIGXFSZ, so a write over the limit fails with EFBIG.
func limitFileSize(t *testing.T, bytes uint64) func() {
	t.Helper()

	var old syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &old); err != nil {
		t.Skipf("could not read the file size limit: %v", err)
	}

	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: bytes, Max: old.Max}); err != nil {
		t.Skipf("could not lower the file size limit: %v", err)
	}

	restored := false
	restore := func() {
		if restored {
			return
		}
		restored = true
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &old); err != nil {
			t.Errorf("could not restore the file size limit. err: %v", err)
		}
	}
	t.Cleanup(restore)

	return restore
}

// quietLogger sends logrus output to a buffer while the file size limit is
// lowered, because a log line written to a file would fail as well.
func quietLogger(t *testing.T) {
	t.Helper()

	logger := logrus.StandardLogger()
	oldOut := logger.Out
	logger.SetOutput(&bytes.Buffer{})
	t.Cleanup(func() { logger.SetOutput(oldOut) })
}

func Test_Backup_defaults_file_write_fails(t *testing.T) {
	quietLogger(t)

	runner := &fakeRunner{payload: []byte("-- unused")}
	dir := t.TempDir()
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)

	h := &backupHandler{dsn: failureTestDSN, backupDir: dir, retentionCount: 7, runner: runner}

	restore := limitFileSize(t, 5)
	res, err := h.Backup(context.Background())
	restore()

	if err == nil || res != nil {
		t.Fatalf("Wrong match. expect: error and nil result, got: %v and %v", res, err)
	}
	if !strings.Contains(err.Error(), "could not write the defaults file") {
		t.Errorf("Wrong match. expect: defaults file error, got: %v", err)
	}
	if runner.capturedName != "" {
		t.Errorf("Wrong match. expect: the dump command is never run, got: %q", runner.capturedName)
	}
	if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
		t.Errorf("Wrong match. expect: no credentials file left, got: %d entries", len(entries))
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("Wrong match. expect: empty backup dir, got: %d entries", len(entries))
	}
}

func Test_Backup_gzip_finalize_fails(t *testing.T) {
	quietLogger(t)

	dir := t.TempDir()
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)

	var restore func()
	// the limit is lowered inside the run, after the credentials file exists;
	// the runner writes nothing, so the gzip header is written at finalize
	runner := &fakeRunner{onRun: func(_ string, _ []string) { restore = limitFileSize(t, 5) }}
	h := &backupHandler{dsn: failureTestDSN, backupDir: dir, retentionCount: 7, runner: runner}

	res, err := h.Backup(context.Background())
	if restore != nil {
		restore()
	}

	if err == nil || res != nil {
		t.Fatalf("Wrong match. expect: error and nil result, got: %v and %v", res, err)
	}
	if !strings.Contains(err.Error(), "could not finalize the gzip stream") {
		t.Errorf("Wrong match. expect: gzip finalize error, got: %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("Wrong match. expect: the partial dump is removed, got: %v", names)
	}
	if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
		t.Errorf("Wrong match. expect: no credentials file left, got: %d entries", len(entries))
	}
}
