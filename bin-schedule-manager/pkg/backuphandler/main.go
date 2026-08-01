package backuphandler

//go:generate mockgen -package backuphandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"io"
)

// Result describes a successful backup dump. It is recorded in the execution
// row's result column as {"path","bytes"} (design §7).
type Result struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// BackupHandler interface
type BackupHandler interface {
	Backup(ctx context.Context) (*Result, error)
}

// commandRunner abstracts subprocess execution so unit tests can assert the
// exact argv (incl. the no-password-in-argv guarantee) without running
// mysqldump.
type commandRunner interface {
	Run(ctx context.Context, stdout io.Writer, name string, args ...string) (stderr string, err error)
}

type backupHandler struct {
	dsn            string
	backupDir      string
	retentionCount int

	runner commandRunner
}

// NewBackupHandler returns BackupHandler interface
func NewBackupHandler(dsn string, backupDir string, retentionCount int) BackupHandler {
	return &backupHandler{
		dsn:            dsn,
		backupDir:      backupDir,
		retentionCount: retentionCount,

		runner: &execCommandRunner{},
	}
}
