package main

import (
	"log"

	"monorepo/bin-common-handler/pkg/migrationhandler"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-push-manager/internal/config"
	"monorepo/bin-push-manager/migrations"
)

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "push-control",
		Short: "Voipbin Push Management CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if errBind := viper.BindPFlags(cmd.Flags()); errBind != nil {
				return errors.Wrap(errBind, "failed to bind flags")
			}

			config.LoadGlobalConfig()
			return nil
		},
	}

	if err := config.Bootstrap(cmdRoot); err != nil {
		cobra.CheckErr(errors.Wrap(err, "failed to bind infrastructure config"))
	}

	// the migrate commands open only the database, so they do not depend on RabbitMQ
	cmdRoot.AddCommand(migrationhandler.NewMigrateCommand("migrate", func() (*migrationhandler.Migrator, error) {
		return migrationhandler.New(config.Get().DatabaseDSN, migrations.FS, migrations.VersionTable)
	}))

	return cmdRoot
}
