package migrationhandler

import (
	"errors"
	"strings"
	"testing"
)

func Test_NewMigrateCommand_structure(t *testing.T) {
	cmd := NewMigrateCommand("migrate-mysql", func() (*Migrator, error) { return nil, errors.New("not used") })

	if cmd.Use != "migrate-mysql" {
		t.Errorf("Wrong match. expect: migrate-mysql, got: %s", cmd.Use)
	}
	if cmd.Short != "Schema migration operations" {
		t.Errorf("Wrong match. expect: Schema migration operations, got: %s", cmd.Short)
	}

	names := []string{}
	for _, sub := range cmd.Commands() {
		names = append(names, sub.Name())
	}
	if got := strings.Join(names, ","); got != "down,force,status,up" {
		t.Errorf("Wrong match. expect: down,force,status,up, got: %s", got)
	}
}

func Test_NewMigrateCommand_run(t *testing.T) {
	errOpen := errors.New("dial failed")

	tests := []struct {
		name         string
		args         []string
		expectOpen   int
		expectSubstr string
	}{
		{"status wraps an open failure", []string{"status"}, 1, "could not initialize the migrator: dial failed"},
		{"up wraps an open failure", []string{"up"}, 1, "could not initialize the migrator: dial failed"},
		{"down wraps an open failure", []string{"down"}, 1, "could not initialize the migrator: dial failed"},
		{"force wraps an open failure", []string{"force", "1"}, 1, "could not initialize the migrator: dial failed"},
		{"force rejects a non numeric version before opening", []string{"force", "abc"}, 0, "invalid version: abc"},
		{"force requires one argument", []string{"force"}, 0, "accepts 1 arg(s)"},
		{"status takes no argument", []string{"status", "x"}, 0, "unknown command"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opened := 0
			cmd := NewMigrateCommand("migrate", func() (*Migrator, error) {
				opened++
				return nil, errOpen
			})
			cmd.SetArgs(tt.args)
			cmd.SilenceErrors = true

			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.expectSubstr) {
				t.Errorf("Wrong match. expect: %q, got: %v", tt.expectSubstr, err)
			}
			if opened != tt.expectOpen {
				t.Errorf("Wrong match. expect open calls: %d, got: %d", tt.expectOpen, opened)
			}
		})
	}
}
