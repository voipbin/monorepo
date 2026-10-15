package migrationhandler

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

// NewMigrateCommand returns the migration recovery command (use is the command name, normally
// "migrate") with the subcommands status, up, down and force. open creates the Migrator for
// every invocation; it must open the database only, so that the commands keep working when the
// cache or the message broker are unreachable.
func NewMigrateCommand(use string, open func() (*Migrator, error)) *cobra.Command {
	cmd := &cobra.Command{Use: use, Short: "Schema migration operations"}
	cmd.AddCommand(newStatusCommand(open))
	cmd.AddCommand(newUpCommand(open))
	cmd.AddCommand(newDownCommand(open))
	cmd.AddCommand(newForceCommand(open))
	return cmd
}

func openMigrator(open func() (*Migrator, error)) (*Migrator, error) {
	m, err := open()
	if err != nil {
		return nil, fmt.Errorf("could not initialize the migrator: %w", err)
	}
	return m, nil
}

func newStatusCommand(open func() (*Migrator, error)) *cobra.Command {
	return &cobra.Command{
		Use:          "status",
		Short:        "Show the applied migration version (creates the version table when it is missing)",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := openMigrator(open)
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

func newUpCommand(open func() (*Migrator, error)) *cobra.Command {
	return &cobra.Command{
		Use:          "up",
		Short:        "Apply all pending migrations",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := openMigrator(open)
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

func newDownCommand(open func() (*Migrator, error)) *cobra.Command {
	return &cobra.Command{
		Use:          "down",
		Short:        "Revert the latest applied migration (development and emergency use only)",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := openMigrator(open)
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

func newForceCommand(open func() (*Migrator, error)) *cobra.Command {
	return &cobra.Command{
		Use:          "force <version>",
		Short:        "Record a version as applied and clear the dirty flag without running migrations (-1 resets). Pass -1 after --",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			version, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid version: %s: %w", args[0], err)
			}

			m, err := openMigrator(open)
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
