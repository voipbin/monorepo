package backuphandler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test_Backup_defaults_file_error checks that a Backup whose credentials file
// cannot be created fails before the dump command runs and leaves the backup
// directory empty.
func Test_Backup_defaults_file_error(t *testing.T) {
	// the backup directory is made first: TempDir needs the real TMPDIR
	dir := t.TempDir()
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))

	runner := &fakeRunner{payload: []byte("-- unused")}
	h := &backupHandler{
		dsn:            "testuser@tcp(127.0.0.1:3306)/voipbin",
		backupDir:      dir,
		retentionCount: 7,
		runner:         runner,
	}

	res, err := h.Backup(context.Background())
	if err == nil || res != nil {
		t.Fatalf("Wrong match. expect: error and nil result, got: %v and %v", err, res)
	}
	if !strings.Contains(err.Error(), "could not write the defaults file") {
		t.Errorf("Wrong match. expect: defaults file error, got: %v", err)
	}
	if runner.capturedName != "" {
		t.Errorf("Wrong match. expect: the dump command is never run, got: %q", runner.capturedName)
	}

	entries, errRead := os.ReadDir(dir)
	if errRead != nil || len(entries) != 0 {
		t.Errorf("Wrong match. expect: empty backup dir, got: %v entries, err: %v", len(entries), errRead)
	}
}
