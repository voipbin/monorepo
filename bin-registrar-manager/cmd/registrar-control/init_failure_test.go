package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-registrar-manager/internal/config"
)

// caseBrokenDSN has no slash before the database name, so the MySQL driver rejects it
// while parsing the DSN inside sql.Open. The connection helper therefore fails without
// any network access and before any other dependency is wired.
const caseBrokenDSN = "nonsense-dsn"

const (
	caseOuterHandler  = "failed to initialize handler"
	caseOuterHandlers = "failed to initialize handlers"
	caseInnerConnect  = "could not connect to bin database"
	caseInnerOpen     = "database open error"
)

// setBrokenBinDSN points the bin database setting to the broken DSN and puts the
// previous value back when the test ends. The config struct is changed in place.
func setBrokenBinDSN(t *testing.T) {
	t.Helper()

	cfg := config.Get()
	saved := cfg.DatabaseDSNBin
	cfg.DatabaseDSNBin = caseBrokenDSN
	t.Cleanup(func() { cfg.DatabaseDSNBin = saved })
}

// runWithFlags parses the arguments on the real command, binds them to viper the way
// the root hook does and runs the command.
func runWithFlags(t *testing.T, cmd *cobra.Command, args []string) error {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)

	if errParse := cmd.Flags().Parse(args); errParse != nil {
		t.Fatalf("could not parse the arguments: %v", errParse)
	}
	if errBind := viper.BindPFlags(cmd.Flags()); errBind != nil {
		t.Fatalf("could not bind the flags: %v", errBind)
	}

	return cmd.RunE(cmd, nil)
}

func assertInitFailure(t *testing.T, err error, outer string) {
	t.Helper()

	if err == nil {
		t.Fatalf("Wrong match. expect: an error containing %q, got: nil", outer)
	}

	msg := err.Error()
	if !strings.HasPrefix(msg, outer) {
		t.Errorf("Wrong match. expect an error starting with %q, got: %v", outer, err)
	}
	// "failed to initialize handlers" also starts with the singular text, so the plural
	// form is told apart explicitly.
	if outer == caseOuterHandler && strings.HasPrefix(msg, caseOuterHandlers) {
		t.Errorf("Wrong match. expect the singular text, got: %v", err)
	}
	for _, inner := range []string{caseInnerConnect, caseInnerOpen} {
		if !strings.Contains(msg, inner) {
			t.Errorf("Wrong match. expect an error containing %q, got: %v", inner, err)
		}
	}
}

// Test_runCommands_initFailure verifies that every extension and trunk command accepts a
// complete flag set and then reports the handler initialisation failure with the expected
// outer and inner error texts. Only the propagation of that failure is asserted: the option
// maps built from the flags before the init call are not observable from here, so the case
// names describe the flag sets that are passed, not an assembled result.
func Test_runCommands_initFailure(t *testing.T) {
	setBrokenBinDSN(t)

	passwordValue := "case-password"

	tests := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
	}{
		{"extension create", cmdExtensionCreate, []string{"--customer-id=" + caseValidID, "--username=case-user", "--password=" + passwordValue, "--extension-number=2001", "--domain=case.example.test"}},
		{"extension get", cmdExtensionGet, []string{"--id=" + caseValidID}},
		{"extension list", cmdExtensionList, []string{"--customer-id=" + caseValidID, "--domain=case.example.test", "--username=case-user", "--extension-number=2001"}},
		{"extension update with all update flags passed", cmdExtensionUpdate, []string{"--id=" + caseValidID, "--password=" + passwordValue, "--username=case-user", "--extension-number=2002", "--domain=case.example.test"}},
		{"extension update with one update flag passed", cmdExtensionUpdate, []string{"--id=" + caseValidID, "--domain=case.example.test"}},
		{"extension delete", cmdExtensionDelete, []string{"--id=" + caseValidID}},
		{"trunk create with credential and allowed ip flags passed", cmdTrunkCreate, []string{"--customer-id=" + caseValidID, "--domain=case.example.test", "--name=case-trunk", "--username=case-user", "--password=" + passwordValue, "--allowed-ips=192.0.2.1, 192.0.2.2,,"}},
		{"trunk create with the required flags only", cmdTrunkCreate, []string{"--customer-id=" + caseValidID, "--domain=case.example.test"}},
		{"trunk get", cmdTrunkGet, []string{"--id=" + caseValidID}},
		{"trunk list", cmdTrunkList, []string{"--customer-id=" + caseValidID, "--domain=case.example.test", "--username=case-user", "--name=case-trunk"}},
		{"trunk update with all update flags passed", cmdTrunkUpdate, []string{"--id=" + caseValidID, "--password=" + passwordValue, "--username=case-user", "--name=case-trunk", "--domain=case.example.test", "--allowed-ips=192.0.2.1,192.0.2.2"}},
		{"trunk update with one update flag passed", cmdTrunkUpdate, []string{"--id=" + caseValidID, "--name=case-trunk"}},
		{"trunk delete", cmdTrunkDelete, []string{"--id=" + caseValidID}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runWithFlags(t, tt.cmd(), tt.args)
			assertInitFailure(t, err, caseOuterHandler)
		})
	}
}

func Test_runDomainMigrate_initFailure(t *testing.T) {
	setBrokenBinDSN(t)

	t.Run("dry-run", func(t *testing.T) {
		err := runWithFlags(t, cmdDomainMigrate(), nil)
		assertInitFailure(t, err, caseOuterHandlers)
	})

	t.Run("dry-run for one customer", func(t *testing.T) {
		err := runWithFlags(t, cmdDomainMigrate(), []string{"--customer-id=" + caseValidID})
		assertInitFailure(t, err, caseOuterHandlers)
	})

	t.Run("execute creates the log before the handlers are initialised", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "migrate.log")

		err := runWithFlags(t, cmdDomainMigrate(), []string{"--execute", "--log=" + logPath})
		assertInitFailure(t, err, caseOuterHandlers)

		info, errStat := os.Stat(logPath)
		if errStat != nil {
			t.Fatalf("Wrong match. expect the log file to exist, got: %v", errStat)
		}
		if info.Size() != 0 {
			t.Errorf("Wrong match. expect an empty log file, got size: %d", info.Size())
		}
	})
}

func Test_runDomainMigrateRollback_initFailure(t *testing.T) {
	setBrokenBinDSN(t)

	logPath := filepath.Join(t.TempDir(), "rollback.log")
	if errWrite := os.WriteFile(logPath, nil, 0o600); errWrite != nil {
		t.Fatalf("could not write the fixture: %v", errWrite)
	}

	for _, args := range [][]string{
		{"--log=" + logPath},
		{"--log=" + logPath, "--execute"},
	} {
		err := runWithFlags(t, cmdDomainMigrateRollback(), args)
		assertInitFailure(t, err, caseOuterHandlers)
	}
}

func Test_initHandlers_brokenBinDSN(t *testing.T) {
	setBrokenBinDSN(t)

	if res, err := initExtensionHandler(); err == nil || res != nil {
		t.Errorf("Wrong match. initExtensionHandler expect: nil handler and an error, got: %v, %v", res, err)
	} else if !strings.HasPrefix(err.Error(), caseInnerConnect) || !strings.Contains(err.Error(), caseInnerOpen) {
		t.Errorf("Wrong match. initExtensionHandler got: %v", err)
	}

	if res, err := initTrunkHandler(); err == nil || res != nil {
		t.Errorf("Wrong match. initTrunkHandler expect: nil handler and an error, got: %v, %v", res, err)
	} else if !strings.HasPrefix(err.Error(), caseInnerConnect) || !strings.Contains(err.Error(), caseInnerOpen) {
		t.Errorf("Wrong match. initTrunkHandler got: %v", err)
	}

	if res, err := initDomainMigrationDeps(); err == nil || res != nil {
		t.Errorf("Wrong match. initDomainMigrationDeps expect: nil deps and an error, got: %v, %v", res, err)
	} else if !strings.HasPrefix(err.Error(), caseInnerConnect) || !strings.Contains(err.Error(), caseInnerOpen) {
		t.Errorf("Wrong match. initDomainMigrationDeps got: %v", err)
	}
}
