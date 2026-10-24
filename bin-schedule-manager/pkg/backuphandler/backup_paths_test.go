package backuphandler

import (
	"bytes"
	"context"
	stderrors "errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func Test_execCommandRunner_Run(t *testing.T) {
	r := &execCommandRunner{}

	t.Run("success writes stdout and returns stderr text", func(t *testing.T) {
		var out bytes.Buffer

		stderr, err := r.Run(context.Background(), &out, "sh", "-c", "printf hello; printf warn >&2")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if out.String() != "hello" {
			t.Errorf("Wrong match. expect: hello, got: %q", out.String())
		}
		if stderr != "warn" {
			t.Errorf("Wrong match. expect: warn, got: %q", stderr)
		}
	})

	t.Run("non-zero exit returns the exit error and stderr", func(t *testing.T) {
		var out bytes.Buffer

		stderr, err := r.Run(context.Background(), &out, "sh", "-c", "printf boom >&2; exit 3")

		var exitErr *exec.ExitError
		if !stderrors.As(err, &exitErr) || exitErr.ExitCode() != 3 {
			t.Errorf("Wrong match. expect: exit code 3, got: %v", err)
		}
		if stderr != "boom" {
			t.Errorf("Wrong match. expect: boom, got: %q", stderr)
		}
	})

	t.Run("stderr is bounded", func(t *testing.T) {
		var out bytes.Buffer

		stderr, err := r.Run(context.Background(), &out, "sh", "-c", "head -c 10000 /dev/zero | tr '\\0' x >&2")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if len(stderr) != stderrLimitBytes {
			t.Errorf("Wrong match. expect: %d, got: %d", stderrLimitBytes, len(stderr))
		}
	})

	t.Run("missing binary returns an error with empty stderr", func(t *testing.T) {
		var out bytes.Buffer

		stderr, err := r.Run(context.Background(), &out, "schedule-test-binary-that-does-not-exist")
		if !stderrors.Is(err, exec.ErrNotFound) {
			t.Errorf("Wrong match. expect: exec.ErrNotFound, got: %v", err)
		}
		if stderr != "" {
			t.Errorf("Wrong match. expect: empty, got: %q", stderr)
		}
	})

	t.Run("canceled context kills the command", func(t *testing.T) {
		var out bytes.Buffer

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			// exec replaces the shell, so the kill reaches the process holding the pipes
			_, err := r.Run(ctx, &out, "sh", "-c", "exec sleep 30")
			done <- err
		}()

		select {
		case err := <-done:
			if err == nil {
				t.Errorf("Wrong match. expect: error after cancel, got: nil")
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("Wrong match. expect: command killed, got: timeout")
		}
	})
}

func Test_boundedBuffer_Write(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		writes []string

		expectRes string
	}{
		{name: "below the limit", limit: 5, writes: []string{"abc"}, expectRes: "abc"},
		{name: "exactly the limit", limit: 5, writes: []string{"abcde"}, expectRes: "abcde"},
		{name: "one byte over the limit", limit: 5, writes: []string{"abcdef"}, expectRes: "abcde"},
		{name: "several writes crossing the limit", limit: 5, writes: []string{"abc", "def", "ghi"}, expectRes: "abcde"},
		{name: "writes after full are ignored", limit: 3, writes: []string{"abc", "d"}, expectRes: "abc"},
		{name: "empty write", limit: 3, writes: []string{""}, expectRes: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &boundedBuffer{limit: tt.limit}

			for _, w := range tt.writes {
				// the full length is always reported, otherwise exec reports a short write
				n, err := b.Write([]byte(w))
				if err != nil || n != len(w) {
					t.Errorf("Wrong match. expect: %d, nil, got: %d, %v", len(w), n, err)
				}
			}

			if b.String() != tt.expectRes {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectRes, b.String())
			}
		})
	}
}

func Test_truncateStderr(t *testing.T) {
	short := strings.Repeat("a", stderrLimitBytes)
	long := strings.Repeat("a", stderrLimitBytes) + "tail"

	if got := truncateStderr(short); got != short {
		t.Errorf("Wrong match. expect: unchanged at the limit, got length: %d", len(got))
	}

	got := truncateStderr(long)
	if len(got) != stderrLimitBytes || strings.Contains(got, "tail") {
		t.Errorf("Wrong match. expect: cut to %d bytes, got length: %d", stderrLimitBytes, len(got))
	}
}

func Test_runDump_errors(t *testing.T) {
	t.Run("output directory does not exist", func(t *testing.T) {
		h := &backupHandler{runner: &fakeRunner{}}

		outPath := filepath.Join(t.TempDir(), "missing-dir", "dump.sql.gz")
		size, err := h.runDump(context.Background(), outPath, nil)
		if err == nil || !strings.Contains(err.Error(), "could not create the backup file") {
			t.Errorf("Wrong match. expect: create error, got: %v", err)
		}
		if size != 0 {
			t.Errorf("Wrong match. expect: 0, got: %d", size)
		}
	})

	t.Run("runner failure carries stderr and argv", func(t *testing.T) {
		h := &backupHandler{runner: &fakeRunner{stderr: "dump failed", runErr: stderrors.New("exit status 2")}}

		outPath := filepath.Join(t.TempDir(), "dump.sql.gz")
		_, err := h.runDump(context.Background(), outPath, []string{"--single-transaction", "voipbin"})
		if err == nil {
			t.Fatalf("Wrong match. expect: error, got: nil")
		}
		for _, want := range []string{"dump failed", "mariadb-dump --single-transaction voipbin"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("Wrong match. expect: %q in the error, got: %v", want, err)
			}
		}
	})

	t.Run("success file is owner-only", func(t *testing.T) {
		h := &backupHandler{runner: &fakeRunner{payload: []byte("data")}}

		outPath := filepath.Join(t.TempDir(), "dump.sql.gz")
		size, err := h.runDump(context.Background(), outPath, nil)
		if err != nil || size <= 0 {
			t.Fatalf("Wrong match. expect: ok and size, got: %d, %v", size, err)
		}

		info, errStat := os.Stat(outPath)
		if errStat != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", errStat)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("Wrong match. expect: 0600, got: %v", info.Mode().Perm())
		}
	})
}

func Test_Backup_failure_leaves_no_files(t *testing.T) {
	dir := t.TempDir()
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)

	h := &backupHandler{
		dsn:            "testuser@tcp(127.0.0.1:3306)/voipbin",
		backupDir:      dir,
		retentionCount: 3,
		runner: &fakeRunner{
			// the partial dump is written before the failure
			payload: []byte("partial"),
			runErr:  stderrors.New("exit status 2"),
			stderr:  "dump failed",
		},
	}

	if _, err := h.Backup(context.Background()); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}

	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("Wrong match. expect: no backup file left, got: %d", len(entries))
	}
	// the credentials file is removed even when the dump fails
	if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
		t.Errorf("Wrong match. expect: no temp file left, got: %d", len(entries))
	}
}

func Test_prune_paths(t *testing.T) {
	t.Run("missing directory", func(t *testing.T) {
		h := &backupHandler{backupDir: filepath.Join(t.TempDir(), "missing"), retentionCount: 1}

		if err := h.prune(); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	for _, retention := range []int{0, -3} {
		t.Run("retention below one keeps the newest file only", func(t *testing.T) {
			dir := t.TempDir()
			h := &backupHandler{backupDir: dir, retentionCount: retention}

			names := []string{
				"voipbin-20260801T000000Z.sql.gz",
				"voipbin-20260802T000000Z.sql.gz",
				"voipbin-20260803T000000Z.sql.gz",
			}
			for _, name := range names {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
			}

			if err := h.prune(); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			entries, _ := os.ReadDir(dir)
			if len(entries) != 1 || entries[0].Name() != names[2] {
				t.Errorf("Wrong match. expect: only %s, got: %v", names[2], entries)
			}
		})
	}

	t.Run("unrelated entries are never removed", func(t *testing.T) {
		dir := t.TempDir()
		h := &backupHandler{backupDir: dir, retentionCount: 1}

		backups := []string{"voipbin-20260801T000000Z.sql.gz", "voipbin-20260802T000000Z.sql.gz"}
		unrelated := []string{"notes.txt", "voipbin-20260801T000000Z.sql", "other-20260801T000000Z.sql.gz"}
		for _, name := range append(append([]string{}, backups...), unrelated...) {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
		}
		// a directory that matches the backup name pattern must be skipped too
		if err := os.Mkdir(filepath.Join(dir, "voipbin-20260701T000000Z.sql.gz"), 0o700); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		if err := h.prune(); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		if _, err := os.Stat(filepath.Join(dir, backups[0])); !os.IsNotExist(err) {
			t.Errorf("Wrong match. expect: oldest backup removed, got: %v", err)
		}
		for _, name := range append([]string{backups[1], "voipbin-20260701T000000Z.sql.gz"}, unrelated...) {
			if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
				t.Errorf("Wrong match. expect: %s kept, got: %v", name, err)
			}
		}
	})
}

func Test_writeDefaultsFile_create_error(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))

	path, err := writeDefaultsFile("anything")
	if err == nil || path != "" {
		t.Errorf("Wrong match. expect: error and empty path, got: %q, %v", path, err)
	}
}
