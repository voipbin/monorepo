package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-registrar-manager/internal/config"
	"monorepo/bin-registrar-manager/migrations"
	"monorepo/bin-registrar-manager/models/common"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

var errInjectedSeam = errors.New("injected seam failure")

func Test_initCache(t *testing.T) {
	env := newSeamEnv(t)

	res, err := initCache()
	if err != nil || res == nil {
		t.Fatalf("Wrong match. expect: a connected cache, got: %v, %v", res, err)
	}

	env.redis.Close()
	res, err = initCache()
	if err == nil || res != nil || !strings.Contains(err.Error(), "could not connect to cache") {
		t.Errorf("Wrong match. expect: nil cache and a connect error, got: %v, %v", res, err)
	}
}

func Test_initHandlers_success(t *testing.T) {
	tests := []struct {
		name string
		init func() error

		expectConnected []string
	}{
		{
			name: "extension handler",
			init: func() error {
				res, err := initExtensionHandler()
				if err == nil && res == nil {
					return errors.New("nil handler without an error")
				}
				return err
			},
			expectConnected: []string{caseDSNBinGood, caseDSNAsterisk},
		},
		{
			name: "trunk handler",
			init: func() error {
				res, err := initTrunkHandler()
				if err == nil && res == nil {
					return errors.New("nil handler without an error")
				}
				return err
			},
			expectConnected: []string{caseDSNBinGood},
		},
		{
			name: "domain migration deps",
			init: func() error {
				res, err := initDomainMigrationDeps()
				if err == nil && (res == nil || res.dbAst == nil || res.dbBin == nil || res.notifyHandler == nil || res.customerDomainHandler == nil) {
					return errors.New("incomplete deps without an error")
				}
				return err
			},
			expectConnected: []string{caseDSNBinGood, caseDSNAsterisk},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newSeamEnv(t)
			env.expectInit()

			if err := tt.init(); err != nil {
				t.Fatalf("Wrong match. expect: nil, got: %v", err)
			}

			if got := strings.Join(env.connected(), ","); got != strings.Join(tt.expectConnected, ",") {
				t.Errorf("Wrong match. expect connections: %v, got: %s", tt.expectConnected, got)
			}
			if uris := env.sockURIs(); len(uris) != 1 || uris[0] != caseRabbitURI {
				t.Errorf("Wrong match. expect one sock handler for the configured address, got: %v", uris)
			}
		})
	}
}

func Test_initHandlers_failures(t *testing.T) {
	tests := []struct {
		name  string
		setup func(env *seamEnv)
		init  func() error

		expectErr string
	}{
		{
			name:      "extension handler: the asterisk database cannot be reached",
			setup:     func(env *seamEnv) { env.connectFail[caseDSNAsterisk] = errInjectedSeam },
			init:      func() error { _, err := initExtensionHandler(); return err },
			expectErr: "could not connect to asterisk database",
		},
		{
			name:      "extension handler: the cache cannot be reached",
			setup:     func(env *seamEnv) { env.redis.Close() },
			init:      func() error { _, err := initExtensionHandler(); return err },
			expectErr: "could not initialize cache",
		},
		{
			name:      "trunk handler: the cache cannot be reached",
			setup:     func(env *seamEnv) { env.redis.Close() },
			init:      func() error { _, err := initTrunkHandler(); return err },
			expectErr: "could not initialize cache",
		},
		{
			name:      "domain migration deps: the asterisk database cannot be reached",
			setup:     func(env *seamEnv) { env.connectFail[caseDSNAsterisk] = errInjectedSeam },
			init:      func() error { _, err := initDomainMigrationDeps(); return err },
			expectErr: "could not connect to asterisk database",
		},
		{
			name:      "domain migration deps: the cache cannot be reached",
			setup:     func(env *seamEnv) { env.redis.Close() },
			init:      func() error { _, err := initDomainMigrationDeps(); return err },
			expectErr: "could not initialize cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newSeamEnv(t)
			tt.setup(env)
			// no sock expectation: the broker must not be touched after the failure

			err := tt.init()
			if err == nil || !strings.HasPrefix(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect an error starting with %q, got: %v", tt.expectErr, err)
			}
			if uris := env.sockURIs(); len(uris) != 0 {
				t.Errorf("Wrong match. expect no sock handler to be created, got: %v", uris)
			}
		})
	}
}

// Test_runCommands_handlerFails verifies that every extension and trunk command reaches its
// handler after the initialisation and reports the handler failure with its own text.
func Test_runCommands_handlerFails(t *testing.T) {
	tests := []struct {
		name string
		cmd  func() *cobra.Command
		args []string

		expectErr string
	}{
		{"extension create", cmdExtensionCreate, []string{"--customer-id=" + caseValidID, "--username=case-user", "--password=case-pass-value", "--extension-number=2001"}, "failed to create extension"},
		{"extension get", cmdExtensionGet, []string{"--id=" + caseValidID}, "failed to retrieve extension"},
		{"extension list", cmdExtensionList, []string{"--customer-id=" + caseValidID, "--domain=case.example.test", "--username=case-user", "--extension-number=2001"}, "failed to retrieve extensions"},
		{"extension update", cmdExtensionUpdate, []string{"--id=" + caseValidID, "--username=case-user", "--extension-number=2002", "--domain=case.example.test"}, "failed to update extension"},
		{"extension delete", cmdExtensionDelete, []string{"--id=" + caseValidID}, "failed to delete extension"},
		{"trunk create", cmdTrunkCreate, []string{"--customer-id=" + caseValidID, "--domain=case.example.test", "--name=case-trunk"}, "failed to create trunk"},
		{"trunk get", cmdTrunkGet, []string{"--id=" + caseValidID}, "failed to retrieve trunk"},
		{"trunk list", cmdTrunkList, []string{"--customer-id=" + caseValidID, "--domain=case.example.test", "--username=case-user", "--name=case-trunk"}, "failed to retrieve trunks"},
		{"trunk update", cmdTrunkUpdate, []string{"--id=" + caseValidID, "--name=case-trunk"}, "failed to update trunk"},
		{"trunk delete", cmdTrunkDelete, []string{"--id=" + caseValidID}, "failed to delete trunk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newSeamEnv(t)
			env.expectInit()
			// the create commands first ask the billing service through the broker
			env.sock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errInjectedSeam).AnyTimes()

			err := runWithFlags(t, tt.cmd(), tt.args)
			if err == nil || !strings.HasPrefix(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect an error starting with %q, got: %v", tt.expectErr, err)
			}
			if strings.Contains(err.Error(), "failed to initialize handler") {
				t.Errorf("Wrong match. the handler must be initialised, got: %v", err)
			}
		})
	}
}

func Test_runExtensionGet_prints(t *testing.T) {
	env := newSeamEnv(t)
	env.expectInit()

	id := uuid.FromStringOrNil(caseValidID)
	seed := cachehandler.NewHandler(env.redis.Addr(), "", 0)
	if err := seed.ExtensionSet(context.Background(), &extension.Extension{
		Identity:  commonidentity.Identity{ID: id, CustomerID: uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000c2")},
		Extension: "2001",
		Name:      "case extension",
	}); err != nil {
		t.Fatalf("could not seed the cache: %v", err)
	}

	var err error
	out := captureStdout(t, func() {
		err = runWithFlags(t, cmdExtensionGet(), []string{"--id=" + caseValidID})
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}
	assertContains(t, out, `"id": "`+caseValidID+`"`)
	assertContains(t, out, `"extension": "2001"`)
	assertContains(t, out, `"name": "case extension"`)
}

func Test_runTrunkGet_prints(t *testing.T) {
	env := newSeamEnv(t)
	env.expectInit()

	id := uuid.FromStringOrNil(caseValidID)
	seed := cachehandler.NewHandler(env.redis.Addr(), "", 0)
	if err := seed.TrunkSet(context.Background(), &trunk.Trunk{
		Identity:   commonidentity.Identity{ID: id},
		Name:       "case trunk",
		DomainName: "case.example.test",
	}); err != nil {
		t.Fatalf("could not seed the cache: %v", err)
	}

	var err error
	out := captureStdout(t, func() {
		err = runWithFlags(t, cmdTrunkGet(), []string{"--id=" + caseValidID})
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}
	assertContains(t, out, `"id": "`+caseValidID+`"`)
	assertContains(t, out, `"domain_name": "case.example.test"`)
}

func Test_runList_printsEmpty(t *testing.T) {
	tests := []struct {
		name string
		cmd  func() *cobra.Command
	}{
		{"extension list", cmdExtensionList},
		{"trunk list", cmdTrunkList},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newSeamEnv(t)
			env.expectInit()
			env.mockBin.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))

			var err error
			out := captureStdout(t, func() {
				err = runWithFlags(t, tt.cmd(), []string{"--customer-id=" + caseValidID})
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: nil, got: %v", err)
			}
			if strings.TrimSpace(out) != "[]" {
				t.Errorf("Wrong match. expect: [], got: %q", out)
			}
			if errMet := env.mockBin.ExpectationsWereMet(); errMet != nil {
				t.Errorf("Wrong match. the query was not run: %v", errMet)
			}
		})
	}
}

func Test_runDomainMigrate_emptyData(t *testing.T) {
	t.Run("dry-run", func(t *testing.T) {
		env := newSeamEnv(t)
		env.expectInit()
		env.mockBin.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}))

		var err error
		out := captureStdout(t, func() {
			err = runWithFlags(t, cmdDomainMigrate(), nil)
		})
		if err != nil {
			t.Fatalf("Wrong match. expect: nil, got: %v", err)
		}
		assertContains(t, out, "domain-migrate [DRY-RUN] done. migrated: 0, skipped(already short): 0")
	})

	t.Run("execute writes nothing to the log without customers", func(t *testing.T) {
		env := newSeamEnv(t)
		env.expectInit()
		env.mockBin.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}))
		logPath := filepath.Join(t.TempDir(), "migrate.log")

		var err error
		out := captureStdout(t, func() {
			err = runWithFlags(t, cmdDomainMigrate(), []string{"--execute", "--log=" + logPath})
		})
		if err != nil {
			t.Fatalf("Wrong match. expect: nil, got: %v", err)
		}
		assertContains(t, out, "domain-migrate [EXECUTE] done. migrated: 0, skipped(already short): 0")
		if info, errStat := os.Stat(logPath); errStat != nil || info.Size() != 0 {
			t.Errorf("Wrong match. expect an empty log file, got: %v, %v", info, errStat)
		}
	})

	t.Run("the listing failure is reported", func(t *testing.T) {
		env := newSeamEnv(t)
		env.expectInit()
		env.mockBin.ExpectQuery("SELECT").WillReturnError(errInjectedSeam)

		err := runWithFlags(t, cmdDomainMigrate(), nil)
		if err == nil || !strings.HasPrefix(err.Error(), "domain migration failed") || !strings.Contains(err.Error(), "could not list customer domains") {
			t.Errorf("Wrong match. expect a domain migration failure, got: %v", err)
		}
	})
}

func Test_runDomainMigrateRollback_emptyLog(t *testing.T) {
	for _, tt := range []struct {
		name   string
		args   []string
		expect string
	}{
		{"dry-run", nil, "domain-migrate-rollback [DRY-RUN] done. customers: 0"},
		{"execute", []string{"--execute"}, "domain-migrate-rollback [EXECUTE] done. customers: 0"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			env := newSeamEnv(t)
			env.expectInit()
			logPath := filepath.Join(t.TempDir(), "rollback.log")
			if errWrite := os.WriteFile(logPath, nil, 0o600); errWrite != nil {
				t.Fatalf("could not write the fixture: %v", errWrite)
			}

			var err error
			out := captureStdout(t, func() {
				err = runWithFlags(t, cmdDomainMigrateRollback(), append([]string{"--log=" + logPath}, tt.args...))
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: nil, got: %v", err)
			}
			assertContains(t, out, tt.expect)
		})
	}
}

func Test_parseMigrationLog_readError(t *testing.T) {
	res, err := parseMigrationLog(iotest.ErrReader(errInjectedSeam))
	if err == nil || res != nil || !strings.Contains(err.Error(), "could not read the log") || !errors.Is(err, errInjectedSeam) {
		t.Errorf("Wrong match. expect a wrapped read error and no records, got: %v, %v", res, err)
	}
}

func Test_initCommand_rootHook(t *testing.T) {
	// the root hook loads the configuration once per process and sets the base domain names once,
	// so the state is reset before and after and the flags are bound on a fresh command tree
	// the hook also replaces the logrus formatter and level, which are put back afterwards
	std := logrus.StandardLogger()
	savedFormatter, savedLevel, savedOut := std.Formatter, std.GetLevel(), std.Out
	viper.Reset()
	common.ResetBaseDomainNamesForTest()
	t.Cleanup(func() {
		viper.Reset()
		common.ResetBaseDomainNamesForTest()
		std.SetFormatter(savedFormatter)
		std.SetLevel(savedLevel)
		std.SetOutput(savedOut)
	})

	t.Run("a valid domain configuration is accepted", func(t *testing.T) {
		t.Setenv("DOMAIN_NAME_EXTENSION", "ext.case.test")
		t.Setenv("DOMAIN_NAME_TRUNK", "trunk.case.test")
		cfg := config.Get()
		saved := *cfg
		t.Cleanup(func() { *cfg = saved })
		cfg.DomainNameExtension = "ext.case.test"
		cfg.DomainNameTrunk = "trunk.case.test"

		root := initCommand()
		statusCmd := findSubcommand(t, root, "migrate", "status")
		if err := root.PersistentPreRunE(statusCmd, nil); err != nil {
			t.Fatalf("Wrong match. expect: nil, got: %v", err)
		}

		// the hook may load the configuration over the fields, so the test sets its own values
		// after it and runs the migrate command through the engine seam
		env := newSeamEnv(t)
		engineDSN := "case-migrate-dsn"
		config.Get().DatabaseDSNBin = engineDSN

		var err error
		out := captureStdout(t, func() { err = statusCmd.RunE(statusCmd, nil) })
		if err != nil {
			t.Fatalf("Wrong match. expect: nil, got: %v", err)
		}
		assertContains(t, out, "Current version: 3")
		assertContains(t, out, "Image version: 3")
		assertContains(t, out, "Dirty: false")

		if got := env.engines(); len(got) != 1 || got[0] != engineDSN+"|"+migrations.VersionTable {
			t.Errorf("Wrong match. expect the engine on the bin database with the version table, got: %v", got)
		}
	})

	t.Run("an invalid domain configuration is rejected", func(t *testing.T) {
		viper.Reset()
		common.ResetBaseDomainNamesForTest()
		t.Setenv("DOMAIN_NAME_EXTENSION", "")
		t.Setenv("DOMAIN_NAME_TRUNK", "")
		cfg := config.Get()
		saved := *cfg
		t.Cleanup(func() { *cfg = saved })
		cfg.DomainNameExtension = ""
		cfg.DomainNameTrunk = ""

		root := initCommand()
		statusCmd := findSubcommand(t, root, "migrate", "status")
		err := root.PersistentPreRunE(statusCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "could not set base domain names") {
			t.Errorf("Wrong match. expect a base domain error, got: %v", err)
		}
	})
}

func findSubcommand(t *testing.T, root *cobra.Command, names ...string) *cobra.Command {
	t.Helper()

	cur := root
	for _, name := range names {
		var next *cobra.Command
		for _, c := range cur.Commands() {
			if c.Name() == name {
				next = c
				break
			}
		}
		if next == nil {
			t.Fatalf("subcommand %q not found", name)
		}
		cur = next
	}
	return cur
}
