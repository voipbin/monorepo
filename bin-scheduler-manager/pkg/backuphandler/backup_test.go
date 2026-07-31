package backuphandler

import (
	"compress/gzip"
	"context"
	stderrors "errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fakeRunner is a hand-written commandRunner for tests: it captures the
// invoked name/args and optionally writes a payload to stdout or fails.
type fakeRunner struct {
	capturedName string
	capturedArgs []string

	payload []byte
	stderr  string
	runErr  error

	onRun func(name string, args []string)
}

func (r *fakeRunner) Run(_ context.Context, stdout io.Writer, name string, args ...string) (string, error) {
	r.capturedName = name
	r.capturedArgs = args

	if r.onRun != nil {
		r.onRun(name, args)
	}

	if r.runErr != nil {
		return r.stderr, r.runErr
	}

	if len(r.payload) > 0 {
		if _, err := stdout.Write(r.payload); err != nil {
			return r.stderr, err
		}
	}

	return r.stderr, nil
}

func Test_Backup(t *testing.T) {
	dir := t.TempDir()

	payload := []byte("-- fake mysqldump output\nCREATE TABLE t (id INT);\n")

	var defaultsPathSeen string
	var defaultsContentSeen []byte
	var defaultsModeSeen os.FileMode

	runner := &fakeRunner{
		payload: payload,
		onRun: func(_ string, args []string) {
			// the defaults file must exist DURING the run, 0600, with the password
			defaultsPathSeen = strings.TrimPrefix(args[0], "--defaults-extra-file=")
			info, err := os.Stat(defaultsPathSeen)
			if err != nil {
				t.Errorf("Wrong match. expect: defaults file present during run, got: %v", err)
				return
			}
			defaultsModeSeen = info.Mode().Perm()
			defaultsContentSeen, _ = os.ReadFile(defaultsPathSeen)
		},
	}

	h := &backupHandler{
		dsn:            "testuser:testpass@tcp(127.0.0.1:3306)/voipbin",
		backupDir:      dir,
		retentionCount: 7,
		runner:         runner,
	}

	ctx := context.Background()

	res, err := h.Backup(ctx)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
		return
	}

	// command name
	if runner.capturedName != "mysqldump" {
		t.Errorf("Wrong match. expect: mysqldump, got: %v", runner.capturedName)
	}

	// --defaults-extra-file MUST be the first argument
	if !strings.HasPrefix(runner.capturedArgs[0], "--defaults-extra-file=") {
		t.Errorf("Wrong match. expect: --defaults-extra-file first, got: %v", runner.capturedArgs[0])
	}

	// no password material anywhere in argv
	for _, arg := range runner.capturedArgs {
		if strings.Contains(arg, "testpass") {
			t.Errorf("Wrong match. expect: no password in argv, got: %v", arg)
		}
	}

	// exact flags after the defaults file
	expectArgs := []string{
		"-h", "127.0.0.1",
		"-P", "3306",
		"-u", "testuser",
		"--single-transaction",
		"--routines",
		"--triggers",
		"--set-gtid-purged=OFF",
		"voipbin",
	}
	if !reflect.DeepEqual(runner.capturedArgs[1:], expectArgs) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectArgs, runner.capturedArgs[1:])
	}

	// defaults file: 0600 with the password during the run, removed after
	if defaultsModeSeen != 0o600 {
		t.Errorf("Wrong match. expect: 0600, got: %v", defaultsModeSeen)
	}
	if !strings.Contains(string(defaultsContentSeen), "password=testpass") {
		t.Errorf("Wrong match. expect: password in defaults file, got: %s", defaultsContentSeen)
	}
	if _, errStat := os.Stat(defaultsPathSeen); !os.IsNotExist(errStat) {
		t.Errorf("Wrong match. expect: defaults file removed, got: %v", errStat)
	}

	// result correctness
	if filepath.Dir(res.Path) != dir {
		t.Errorf("Wrong match. expect: %v, got: %v", dir, filepath.Dir(res.Path))
	}
	base := filepath.Base(res.Path)
	if !strings.HasPrefix(base, backupFilePrefix) || !strings.HasSuffix(base, backupFileSuffix) {
		t.Errorf("Wrong match. expect: voipbin-<ts>.sql.gz, got: %v", base)
	}

	info, err := os.Stat(res.Path)
	if err != nil {
		t.Errorf("Wrong match. expect: backup file present, got: %v", err)
		return
	}
	if res.Bytes != info.Size() || res.Bytes <= 0 {
		t.Errorf("Wrong match. expect: %v, got: %v", info.Size(), res.Bytes)
	}

	// gunzip round-trip
	f, err := os.Open(res.Path)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
		return
	}
	defer func() {
		_ = f.Close()
	}()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
		return
	}
	content, err := io.ReadAll(gz)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
		return
	}
	if !reflect.DeepEqual(content, payload) {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", payload, content)
	}
}

func Test_Backup_error(t *testing.T) {
	tests := []struct {
		name string

		dsn       string
		backupDir string
		runner    *fakeRunner

		expectContains    string
		expectNotContains string
	}{
		{
			name: "backup dir not configured",

			dsn:       "testuser:testpass@tcp(127.0.0.1:3306)/voipbin",
			backupDir: "",
			runner:    &fakeRunner{},

			expectContains:    "scheduler_backup_dir not configured",
			expectNotContains: "testpass",
		},
		{
			name: "invalid dsn never echoes the dsn",

			dsn:       "testuser:secretpw@tcp-missing-slash",
			backupDir: "PLACEHOLDER_TMPDIR",
			runner:    &fakeRunner{},

			expectContains:    "could not parse the database DSN",
			expectNotContains: "secretpw",
		},
		{
			name: "mysqldump failure carries stderr but no password",

			dsn:       "testuser:testpass@tcp(127.0.0.1:3306)/voipbin",
			backupDir: "PLACEHOLDER_TMPDIR",
			runner: &fakeRunner{
				stderr: "mysqldump: Got error: 2003",
				runErr: stderrors.New("exit status 2"),
			},

			expectContains:    "mysqldump: Got error: 2003",
			expectNotContains: "testpass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backupDir := tt.backupDir
			if backupDir == "PLACEHOLDER_TMPDIR" {
				backupDir = t.TempDir()
			}

			h := &backupHandler{
				dsn:            tt.dsn,
				backupDir:      backupDir,
				retentionCount: 7,
				runner:         tt.runner,
			}

			ctx := context.Background()

			_, err := h.Backup(ctx)
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
				return
			}

			if !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("Wrong match. expect contains: %v, got: %v", tt.expectContains, err)
			}
			if tt.expectNotContains != "" && strings.Contains(err.Error(), tt.expectNotContains) {
				t.Errorf("Wrong match. expect no %v, got: %v", tt.expectNotContains, err)
			}

			// no partial dump left behind
			if backupDir != "" {
				entries, errRead := os.ReadDir(backupDir)
				if errRead != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", errRead)
					return
				}
				if len(entries) != 0 {
					t.Errorf("Wrong match. expect: no partial dump, got: %v entries", len(entries))
				}
			}
		})
	}
}

func Test_Backup_retention_prune(t *testing.T) {
	dir := t.TempDir()

	// pre-existing dumps, oldest to newest
	preExisting := []string{
		"voipbin-20260725T020000Z.sql.gz",
		"voipbin-20260726T020000Z.sql.gz",
		"voipbin-20260727T020000Z.sql.gz",
		"voipbin-20260728T020000Z.sql.gz",
	}
	for _, name := range preExisting {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("old dump"), 0o600); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
			return
		}
	}
	// an unrelated file must never be pruned
	if err := os.WriteFile(filepath.Join(dir, "unrelated.txt"), []byte("keep me"), 0o600); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
		return
	}

	h := &backupHandler{
		dsn:            "testuser:testpass@tcp(127.0.0.1:3306)/voipbin",
		backupDir:      dir,
		retentionCount: 2,
		runner:         &fakeRunner{payload: []byte("-- new dump")},
	}

	ctx := context.Background()

	res, err := h.Backup(ctx)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
		return
	}

	remaining := []string{}
	for _, e := range entries {
		remaining = append(remaining, e.Name())
	}

	// retention 2 keeps the new dump + the newest pre-existing one; the
	// unrelated file is untouched
	expectRemaining := []string{
		filepath.Base(res.Path),
		"voipbin-20260728T020000Z.sql.gz",
		"unrelated.txt",
	}

	if len(remaining) != len(expectRemaining) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRemaining, remaining)
		return
	}
	for _, want := range expectRemaining {
		found := false
		for _, got := range remaining {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Wrong match. expect: %v present, got: %v", want, remaining)
		}
	}
}

func Test_splitHostPort(t *testing.T) {
	tests := []struct {
		name string

		addr string

		expectHost string
		expectPort string
	}{
		{
			name: "host with port",

			addr: "10.0.0.5:3307",

			expectHost: "10.0.0.5",
			expectPort: "3307",
		},
		{
			name: "host without port defaults 3306",

			addr: "db.internal",

			expectHost: "db.internal",
			expectPort: "3306",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := splitHostPort(tt.addr)
			if host != tt.expectHost {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectHost, host)
			}
			if port != tt.expectPort {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectPort, port)
			}
		})
	}
}
