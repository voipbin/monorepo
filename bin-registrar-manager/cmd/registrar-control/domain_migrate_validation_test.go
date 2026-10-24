package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/spf13/viper"

	"monorepo/bin-registrar-manager/models/extension"
)

// failingWriter fails every write.
type failingWriter struct{}

var errInjectedWrite = errors.New("injected write failure")

func (failingWriter) Write(p []byte) (int, error) { return 0, errInjectedWrite }

// syncFailingWriter accepts writes but fails to sync them to disk.
type syncFailingWriter struct {
	written strings.Builder
}

var errInjectedSync = errors.New("injected sync failure")

func (w *syncFailingWriter) Write(p []byte) (int, error) { return w.written.Write(p) }
func (w *syncFailingWriter) Sync() error                 { return errInjectedSync }

func Test_writeMigrationLogLine_failures(t *testing.T) {
	record := &migrationRecord{CustomerID: uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000a1")}

	t.Run("write failure is returned", func(t *testing.T) {
		err := writeMigrationLogLine(failingWriter{}, record)
		if !errors.Is(err, errInjectedWrite) {
			t.Errorf("expected the write failure, got: %v", err)
		}
	})

	t.Run("sync failure is returned after the line was written", func(t *testing.T) {
		w := &syncFailingWriter{}
		err := writeMigrationLogLine(w, record)
		if !errors.Is(err, errInjectedSync) {
			t.Errorf("expected the sync failure, got: %v", err)
		}
		if !strings.HasSuffix(w.written.String(), "\n") || !strings.Contains(w.written.String(), "00000000-0000-4000-8000-0000000000a1") {
			t.Errorf("the line must be written before the sync, got: %q", w.written.String())
		}
	})
}

func Test_buildExtensionMigrationRecord_emptyRealm(t *testing.T) {
	ext := &extension.Extension{Extension: "1001", EndpointID: "1001@legacy.reg.example.test"}

	res := buildExtensionMigrationRecord(ext, "ab12.reg.example.test")

	if res.OldRealm != "legacy.reg.example.test" {
		t.Errorf("the old realm must be recovered from the endpoint id, got: %q", res.OldRealm)
	}
	if res.NewEndpointID != "1001@ab12.reg.example.test" || res.NewRealm != "ab12.reg.example.test" {
		t.Errorf("unexpected target state: %+v", res)
	}
	if res.OldEndpointID != "1001@legacy.reg.example.test" {
		t.Errorf("the old endpoint id must be kept as stored, got: %q", res.OldEndpointID)
	}
}

func Test_runDomainMigrate_validation(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "absent", "migration.log")

	tests := []struct {
		name  string
		setup func(t *testing.T)
		want  string
	}{
		{
			name: "dry-run and execute together",
			setup: func(t *testing.T) {
				viper.Set("execute", true)
				viper.Set("dry-run", true)
			},
			want: "mutually exclusive",
		},
		{
			name: "invalid customer id",
			setup: func(t *testing.T) {
				viper.Set("customer-id", "not-a-uuid")
			},
			want: "invalid format for customer-id",
		},
		{
			name: "execute without a log path",
			setup: func(t *testing.T) {
				viper.Set("execute", true)
			},
			want: "--log is required",
		},
		{
			name: "execute with a log path that cannot be opened",
			setup: func(t *testing.T) {
				viper.Set("execute", true)
				viper.Set("log", missingDir)
			},
			want: "could not open the log file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)

			cmd := cmdDomainMigrate()
			if tt.name == "dry-run and execute together" {
				// the conflict is only detected when --dry-run was passed explicitly.
				if err := cmd.Flags().Set("dry-run", "true"); err != nil {
					t.Fatalf("could not set the flag: %v", err)
				}
			}
			tt.setup(t)

			err := runDomainMigrate(cmd, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected an error containing %q, got: %v", tt.want, err)
			}
		})
	}

	// the failed attempts must not leave a log file behind.
	if _, err := os.Stat(missingDir); err == nil {
		t.Errorf("no log file may be created for a rejected request")
	}
}

func Test_runDomainMigrateRollback_validation(t *testing.T) {
	dir := t.TempDir()

	badLog := filepath.Join(dir, "bad.log")
	if err := os.WriteFile(badLog, []byte("{not json}\n"), 0o600); err != nil {
		t.Fatalf("could not write the fixture: %v", err)
	}

	noCustomerLog := filepath.Join(dir, "nocustomer.log")
	if err := os.WriteFile(noCustomerLog, []byte(`{"old_label":"ab12"}`+"\n"), 0o600); err != nil {
		t.Fatalf("could not write the fixture: %v", err)
	}

	tests := []struct {
		name string
		log  string
		want string
	}{
		{"log path missing", "", "--log is required"},
		{"log file does not exist", filepath.Join(dir, "absent.log"), "could not open the log file"},
		{"log file has an invalid line", badLog, "could not parse the migration log"},
		{"log record without a customer id", noCustomerLog, "missing customer_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("log", tt.log)

			err := runDomainMigrateRollback(cmdDomainMigrateRollback(), nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected an error containing %q, got: %v", tt.want, err)
			}
		})
	}
}

func Test_domainMigrateCommands_flags(t *testing.T) {
	migrate := cmdDomainMigrate()
	if f := migrate.Flags().Lookup("dry-run"); f == nil || f.DefValue != "true" {
		t.Errorf("dry-run must exist and default to true, got: %+v", f)
	}
	if f := migrate.Flags().Lookup("execute"); f == nil || f.DefValue != "false" {
		t.Errorf("execute must exist and default to false, got: %+v", f)
	}
	for _, name := range []string{"customer-id", "log"} {
		if f := migrate.Flags().Lookup(name); f == nil || f.DefValue != "" {
			t.Errorf("%s must exist with an empty default, got: %+v", name, f)
		}
	}

	rollback := cmdDomainMigrateRollback()
	if f := rollback.Flags().Lookup("execute"); f == nil || f.DefValue != "false" {
		t.Errorf("rollback execute must default to false, got: %+v", f)
	}
	if f := rollback.Flags().Lookup("log"); f == nil || f.DefValue != "" {
		t.Errorf("rollback log must exist with an empty default, got: %+v", f)
	}
}
