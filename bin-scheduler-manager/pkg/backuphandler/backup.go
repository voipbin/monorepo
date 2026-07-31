package backuphandler

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	backupFilePrefix = "voipbin-"
	backupFileSuffix = ".sql.gz"
	backupTimeLayout = "20060102T150405Z"

	// stderrLimitBytes bounds the captured mysqldump stderr carried in error
	// messages.
	stderrLimitBytes = 4096
)

// Backup runs mysqldump against the DSN's database, gzips the dump into the
// backup directory, prunes old dumps to the retention count, and returns the
// produced file's path and size (design §7).
//
// Credential hygiene: the DB password goes through a 0600 --defaults-extra-file
// temp file (removed after the run) — never argv (world-readable via
// /proc/<pid>/cmdline) and never the environment. Error paths carry the
// redacted argv only (the defaults file path contains no secret) and never
// echo the DSN.
func (h *backupHandler) Backup(ctx context.Context) (*Result, error) {
	log := logrus.WithField("func", "Backup")

	if h.backupDir == "" {
		return nil, errors.New("scheduler_backup_dir not configured")
	}

	cfg, err := gomysql.ParseDSN(h.dsn)
	if err != nil {
		// never echo the DSN — it carries the password
		return nil, errors.New("could not parse the database DSN")
	}

	host, port := splitHostPort(cfg.Addr)

	defaultsPath, err := writeDefaultsFile(cfg.Passwd)
	if err != nil {
		return nil, errors.Wrap(err, "could not write the defaults file")
	}
	defer func() {
		if errRemove := os.Remove(defaultsPath); errRemove != nil {
			log.Errorf("Could not remove the defaults file. err: %v", errRemove)
		}
	}()

	// --defaults-extra-file MUST be the first argument (mysql client
	// requirement); host/port/user/dbname go as flags (design §7)
	args := []string{
		"--defaults-extra-file=" + defaultsPath,
		"-h", host,
		"-P", port,
		"-u", cfg.User,
		"--single-transaction",
		"--routines",
		"--triggers",
		"--set-gtid-purged=OFF",
		cfg.DBName,
	}

	now := time.Now().UTC()
	outPath := filepath.Join(h.backupDir, backupFilePrefix+now.Format(backupTimeLayout)+backupFileSuffix)

	size, err := h.runDump(ctx, outPath, args)
	if err != nil {
		// best effort: do not leave a partial dump behind
		_ = os.Remove(outPath)
		return nil, err
	}

	promBackupLastBytes.Set(float64(size))

	if errPrune := h.prune(); errPrune != nil {
		// the dump itself succeeded — a pruning failure is logged, not fatal
		log.Errorf("Could not prune old backups. err: %v", errPrune)
	}

	log.Debugf("Backup completed. path: %s, bytes: %d", outPath, size)

	return &Result{Path: outPath, Bytes: size}, nil
}

// runDump executes mysqldump with the given args, gzip-compressing its stdout
// into outPath. Returns the produced file's size in bytes.
func (h *backupHandler) runDump(ctx context.Context, outPath string, args []string) (int64, error) {
	f, err := os.Create(outPath)
	if err != nil {
		return 0, errors.Wrapf(err, "could not create the backup file. path: %s", outPath)
	}

	gz := gzip.NewWriter(f)

	stderr, errRun := h.runner.Run(ctx, gz, "mysqldump", args...)
	if errRun != nil {
		_ = gz.Close()
		_ = f.Close()
		// args carry no password material (defaults-extra-file), so the argv
		// is safe to include
		return 0, errors.Wrapf(errRun, "mysqldump failed. argv: mysqldump %s, stderr: %s", strings.Join(args, " "), truncateStderr(stderr))
	}

	if errClose := gz.Close(); errClose != nil {
		_ = f.Close()
		return 0, errors.Wrap(errClose, "could not finalize the gzip stream")
	}
	if errClose := f.Close(); errClose != nil {
		return 0, errors.Wrap(errClose, "could not close the backup file")
	}

	info, err := os.Stat(outPath)
	if err != nil {
		return 0, errors.Wrap(err, "could not stat the backup file")
	}

	return info.Size(), nil
}

// prune removes backup files beyond the retention count, keeping the newest
// (design §7). The timestamped filename format sorts chronologically.
func (h *backupHandler) prune() error {
	entries, err := os.ReadDir(h.backupDir)
	if err != nil {
		return errors.Wrapf(err, "could not read the backup directory. dir: %s", h.backupDir)
	}

	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, backupFilePrefix) && strings.HasSuffix(name, backupFileSuffix) {
			names = append(names, name)
		}
	}

	// newest first
	sort.Sort(sort.Reverse(sort.StringSlice(names)))

	retention := h.retentionCount
	if retention < 1 {
		retention = 1
	}

	for i := retention; i < len(names); i++ {
		path := filepath.Join(h.backupDir, names[i])
		if errRemove := os.Remove(path); errRemove != nil {
			return errors.Wrapf(errRemove, "could not remove the old backup. path: %s", path)
		}
	}

	return nil
}

// writeDefaultsFile writes a 0600 temp file carrying the client password for
// --defaults-extra-file (design §7 credential hygiene). The caller removes it
// after the run.
func writeDefaultsFile(password string) (string, error) {
	f, err := os.CreateTemp("", "scheduler-backup-*.cnf")
	if err != nil {
		return "", err
	}

	// os.CreateTemp already creates 0600; enforce it explicitly anyway
	if errChmod := f.Chmod(0o600); errChmod != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", errChmod
	}

	content := fmt.Sprintf("[client]\npassword=%s\n", password)
	if _, errWrite := f.WriteString(content); errWrite != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		// do not wrap errWrite — an I/O error string could echo the content
		return "", errors.New("could not write the defaults file content")
	}

	if errClose := f.Close(); errClose != nil {
		_ = os.Remove(f.Name())
		return "", errClose
	}

	return f.Name(), nil
}

// splitHostPort splits the DSN's Addr into host and port, defaulting the port
// to 3306 when absent.
func splitHostPort(addr string) (string, string) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, "3306"
	}

	return host, port
}

// truncateStderr bounds the captured stderr carried in error messages.
func truncateStderr(s string) string {
	if len(s) <= stderrLimitBytes {
		return s
	}

	return s[:stderrLimitBytes]
}

// execCommandRunner is the production commandRunner backed by
// exec.CommandContext.
type execCommandRunner struct{}

func (r *execCommandRunner) Run(ctx context.Context, stdout io.Writer, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = stdout

	stderr := &boundedBuffer{limit: stderrLimitBytes}
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return stderr.String(), err
	}

	return stderr.String(), nil
}

// boundedBuffer keeps at most limit bytes of what is written to it.
type boundedBuffer struct {
	limit int
	buf   []byte
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	remain := b.limit - len(b.buf)
	if remain > 0 {
		if len(p) < remain {
			remain = len(p)
		}
		b.buf = append(b.buf, p[:remain]...)
	}

	return len(p), nil
}

func (b *boundedBuffer) String() string {
	return string(b.buf)
}
