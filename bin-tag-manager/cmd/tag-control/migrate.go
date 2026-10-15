package main

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"monorepo/bin-tag-manager/internal/config"
	"monorepo/bin-tag-manager/internal/migrator"
	"monorepo/bin-tag-manager/migrations"
)

// initMigrator opens only the database. The migrate commands are recovery tools, so they must
// not depend on Redis or RabbitMQ being reachable.
func initMigrator() (*migrator.Migrator, error) {
	res, err := migrator.New(config.Get().DatabaseDSN, migrations.FS, migrations.VersionTable)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize the migrator")
	}
	return res, nil
}

func cmdMigrate() *cobra.Command {
	cmd := &cobra.Command{Use: "migrate", Short: "Schema migration operations"}
	cmd.AddCommand(cmdMigrateStatus())
	cmd.AddCommand(cmdMigrateUp())
	cmd.AddCommand(cmdMigrateDown())
	cmd.AddCommand(cmdMigrateForce())
	return cmd
}

func cmdMigrateStatus() *cobra.Command {
	return &cobra.Command{
		Use:          "status",
		Short:        "Show the applied migration version (creates the version table when it is missing)",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := initMigrator()
			if err != nil {
				return err
			}
			defer func() { _ = m.Close() }()

			version, dirty, err := m.Status()
			if err != nil {
				return err
			}

			fmt.Printf("Current version: %d\n", version)
			fmt.Printf("Image version: %d\n", m.MaxVersion())
			fmt.Printf("Dirty: %v\n", dirty)
			return nil
		},
	}
}

func cmdMigrateUp() *cobra.Command {
	return &cobra.Command{
		Use:          "up",
		Short:        "Apply all pending migrations",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := initMigrator()
			if err != nil {
				return err
			}
			defer func() { _ = m.Close() }()

			if err := m.Up(); err != nil {
				return err
			}

			fmt.Println("Migrations applied successfully")
			return nil
		},
	}
}

func cmdMigrateDown() *cobra.Command {
	return &cobra.Command{
		Use:          "down",
		Short:        "Revert the latest applied migration (development and emergency use only)",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := initMigrator()
			if err != nil {
				return err
			}
			defer func() { _ = m.Close() }()

			if err := m.Down(); err != nil {
				return err
			}

			fmt.Println("Rollback finished (nothing is reverted when no version is applied)")
			return nil
		},
	}
}

func cmdMigrateForce() *cobra.Command {
	return &cobra.Command{
		Use:          "force <version>",
		Short:        "Record a version as applied and clear the dirty flag without running migrations (-1 resets). Pass -1 after --",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			version, err := strconv.Atoi(args[0])
			if err != nil {
				return errors.Wrapf(err, "invalid version: %s", args[0])
			}

			m, err := initMigrator()
			if err != nil {
				return err
			}
			defer func() { _ = m.Close() }()

			if err := m.Force(version); err != nil {
				return err
			}

			fmt.Printf("Forced version: %d\n", version)
			return nil
		},
	}
}
