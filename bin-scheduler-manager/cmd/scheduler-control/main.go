// scheduler-control is the direct-DB admin CLI for scheduler-manager
// (design §12): it works without RabbitMQ so `schedule disable` can stop a
// misbehaving dispatch loop even when the broker path is unhealthy.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-scheduler-manager/internal/config"
	"monorepo/bin-scheduler-manager/models/execution"
	"monorepo/bin-scheduler-manager/models/schedule"
	"monorepo/bin-scheduler-manager/pkg/cachehandler"
	"monorepo/bin-scheduler-manager/pkg/dbhandler"
)

// listLimit bounds the rows returned by the list commands.
const listLimit = 100

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

// initHandler connects to the database and cache directly — no RabbitMQ.
func initHandler() (dbhandler.DBHandler, error) {
	sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return dbhandler.NewHandler(sqlDB, cache), nil
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "scheduler-control",
		Short: "Voipbin Scheduler Management CLI (direct DB, no RabbitMQ)",
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

	cmdSchedule := &cobra.Command{Use: "schedule", Short: "Schedule operations"}
	cmdSchedule.AddCommand(cmdScheduleList())
	cmdSchedule.AddCommand(cmdScheduleGet())
	cmdSchedule.AddCommand(cmdScheduleEnable())
	cmdSchedule.AddCommand(cmdScheduleDisable())

	cmdExecution := &cobra.Command{Use: "execution", Short: "Execution operations"}
	cmdExecution.AddCommand(cmdExecutionList())

	cmdRoot.AddCommand(cmdSchedule)
	cmdRoot.AddCommand(cmdExecution)
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

// resolveSchedule resolves the given name-or-id to a schedule. A valid UUID
// resolves by id; anything else resolves by name in the nil-customer
// (platform) namespace.
func resolveSchedule(ctx context.Context, db dbhandler.DBHandler, nameOrID string) (*schedule.Schedule, error) {
	if id := uuid.FromStringOrNil(nameOrID); id != uuid.Nil {
		res, err := db.ScheduleGet(ctx, id)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get the schedule by id. id: %s", id)
		}
		return res, nil
	}

	res, err := db.ScheduleGetByCustomerIDName(ctx, uuid.Nil, nameOrID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the schedule by name. name: %s", nameOrID)
	}
	return res, nil
}

func cmdScheduleList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List schedules",
		RunE:  runScheduleList,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Filter by customer ID (optional)")
	flags.Bool("deleted", false, "Include soft-deleted schedules")

	return cmd
}

func runScheduleList(cmd *cobra.Command, args []string) error {
	db, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	filters := map[schedule.Field]any{
		schedule.FieldDeleted: viper.GetBool("deleted"),
	}
	if customerID := viper.GetString("customer-id"); customerID != "" {
		id := uuid.FromStringOrNil(customerID)
		if id == uuid.Nil {
			return fmt.Errorf("invalid format for customer ID: '%s' is not a valid UUID", customerID)
		}
		filters[schedule.FieldCustomerID] = id
	}

	token := utilhandler.NewUtilHandler().TimeGetCurTime()
	res, err := db.ScheduleList(context.Background(), listLimit, token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to list schedules")
	}

	return printJSON(res)
}

func cmdScheduleGet() *cobra.Command {
	return &cobra.Command{
		Use:   "get <name-or-id>",
		Short: "Get a schedule by name or ID",
		Args:  cobra.ExactArgs(1),
		RunE:  runScheduleGet,
	}
}

func runScheduleGet(cmd *cobra.Command, args []string) error {
	db, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := resolveSchedule(context.Background(), db, args[0])
	if err != nil {
		return errors.Wrap(err, "failed to get schedule")
	}

	return printJSON(res)
}

func cmdScheduleEnable() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <name-or-id>",
		Short: "Enable a schedule (by platform schedule name or UUID)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScheduleSetEnabled(args[0], true)
		},
	}
}

func cmdScheduleDisable() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <name-or-id>",
		Short: "Disable a schedule (by platform schedule name or UUID)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScheduleSetEnabled(args[0], false)
		},
	}
}

func runScheduleSetEnabled(nameOrID string, enabled bool) error {
	db, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	ctx := context.Background()

	s, err := resolveSchedule(ctx, db, nameOrID)
	if err != nil {
		return err
	}

	// an enabled change resets tm_next_run so the dispatcher re-initializes
	// the next slot (design §5.2 — same behavior as the RPC update path)
	fields := map[schedule.Field]any{
		schedule.FieldEnabled:   enabled,
		schedule.FieldTMNextRun: nil,
	}
	if errUpdate := db.ScheduleUpdate(ctx, s.ID, fields); errUpdate != nil {
		return errors.Wrap(errUpdate, "failed to update schedule")
	}

	res, err := db.ScheduleGet(ctx, s.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get updated schedule")
	}

	return printJSON(res)
}

func cmdExecutionList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List executions",
		RunE:  runExecutionList,
	}

	flags := cmd.Flags()
	flags.String("schedule-id", "", "Filter by schedule ID (optional)")

	return cmd
}

func runExecutionList(cmd *cobra.Command, args []string) error {
	db, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	filters := map[execution.Field]any{
		execution.FieldDeleted: false,
	}
	if scheduleID := viper.GetString("schedule-id"); scheduleID != "" {
		id := uuid.FromStringOrNil(scheduleID)
		if id == uuid.Nil {
			return fmt.Errorf("invalid format for schedule ID: '%s' is not a valid UUID", scheduleID)
		}
		filters[execution.FieldScheduleID] = id
	}

	token := utilhandler.NewUtilHandler().TimeGetCurTime()
	res, err := db.ExecutionList(context.Background(), listLimit, token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to list executions")
	}

	return printJSON(res)
}
