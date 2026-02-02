package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-timeline-manager/internal/config"
	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
)

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "timeline-control",
		Short: "Voipbin Timeline Management CLI",
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

	// Event commands
	cmdEvent := &cobra.Command{Use: "event", Short: "Event operations"}
	cmdEvent.AddCommand(cmdEventList())
	cmdRoot.AddCommand(cmdEvent)

	// Migrate commands
	cmdMigrate := &cobra.Command{Use: "migrate", Short: "Migration operations"}
	cmdMigrate.AddCommand(cmdMigrateUp())
	cmdMigrate.AddCommand(cmdMigrateDown())
	cmdMigrate.AddCommand(cmdMigrateStatus())
	cmdRoot.AddCommand(cmdMigrate)

	return cmdRoot
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSON")
	}
	fmt.Println(string(data))
	return nil
}

// Event commands

func cmdEventList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List events",
		RunE:  runEventList,
	}

	flags := cmd.Flags()
	flags.String("publisher", "", "Publisher service name (required)")
	flags.String("id", "", "Resource ID (required)")
	flags.String("events", "", "Event patterns comma-separated (required, e.g., 'activeflow_*,flow_created')")
	flags.Int("page-size", 100, "Page size")
	flags.String("page-token", "", "Page token for pagination")

	return cmd
}

func runEventList(cmd *cobra.Command, args []string) error {
	publisher := viper.GetString("publisher")
	if publisher == "" {
		return fmt.Errorf("publisher is required")
	}

	idStr := viper.GetString("id")
	if idStr == "" {
		return fmt.Errorf("id is required")
	}
	id := uuid.FromStringOrNil(idStr)
	if id == uuid.Nil {
		return fmt.Errorf("invalid id format")
	}

	eventsStr := viper.GetString("events")
	if eventsStr == "" {
		return fmt.Errorf("events is required")
	}
	events := strings.Split(eventsStr, ",")

	db := dbhandler.NewHandler(config.Get().ClickHouseAddress, config.Get().ClickHouseDatabase)
	handler := eventhandler.NewEventHandler(db)

	req := &event.EventListRequest{
		Publisher: commonoutline.ServiceName(publisher),
		ID:        id,
		Events:    events,
		PageSize:  viper.GetInt("page-size"),
		PageToken: viper.GetString("page-token"),
	}

	result, err := handler.List(context.Background(), req)
	if err != nil {
		return errors.Wrap(err, "failed to list events")
	}

	return printJSON(result)
}

// Migration commands

func getMigrate() (*migrate.Migrate, error) {
	addr := config.Get().ClickHouseAddress
	db := config.Get().ClickHouseDatabase
	if addr == "" {
		return nil, fmt.Errorf("CLICKHOUSE_ADDRESS is required")
	}

	dsn := fmt.Sprintf("clickhouse://%s/%s?x-multi-statement=true", addr, db)
	return migrate.New("file://migrations", dsn)
}

func cmdMigrateUp() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := getMigrate()
			if err != nil {
				return errors.Wrap(err, "failed to initialize migrate")
			}
			defer func() { _, _ = m.Close() }()

			if err := m.Up(); err != nil && err != migrate.ErrNoChange {
				return errors.Wrap(err, "migration failed")
			}

			fmt.Println("Migrations applied successfully")
			return nil
		},
	}
}

func cmdMigrateDown() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Rollback last migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := getMigrate()
			if err != nil {
				return errors.Wrap(err, "failed to initialize migrate")
			}
			defer func() { _, _ = m.Close() }()

			if err := m.Steps(-1); err != nil {
				return errors.Wrap(err, "rollback failed")
			}

			fmt.Println("Rollback completed successfully")
			return nil
		},
	}
}

func cmdMigrateStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := getMigrate()
			if err != nil {
				return errors.Wrap(err, "failed to initialize migrate")
			}
			defer func() { _, _ = m.Close() }()

			version, dirty, err := m.Version()
			if err != nil && err != migrate.ErrNilVersion {
				return errors.Wrap(err, "failed to get version")
			}

			fmt.Printf("Current version: %d\n", version)
			fmt.Printf("Dirty: %v\n", dirty)
			return nil
		},
	}
}
