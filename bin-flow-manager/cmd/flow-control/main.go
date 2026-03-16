package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-flow-manager/internal/config"
	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/activeflowhandler"
	"monorepo/bin-flow-manager/pkg/cachehandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/flowhandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = commonoutline.ServiceNameFlowManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (flowhandler.FlowHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initFlowHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initFlowHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (flowhandler.FlowHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameFlowEvent, serviceName)

	actionHandler := actionhandler.NewActionHandler()
	variableHandler := variablehandler.NewVariableHandler(db, reqHandler)
	activeflowHandler := activeflowhandler.NewActiveflowHandler(db, reqHandler, notifyHandler, actionHandler, variableHandler)

	return flowhandler.NewFlowHandler(db, reqHandler, notifyHandler, actionHandler, activeflowHandler), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "flow-control",
		Short: "Voipbin Flow Management CLI",
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

	cmdSub := &cobra.Command{Use: "flow", Short: "Flow operations"}
	cmdSub.AddCommand(cmdCreate())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdList())
	cmdSub.AddCommand(cmdUpdate())
	cmdSub.AddCommand(cmdUpdateActions())
	cmdSub.AddCommand(cmdDelete())
	cmdSub.AddCommand(cmdActionGet())

	cmdRoot.AddCommand(cmdSub)
	return cmdRoot
}

func resolveUUID(flagName string, label string) (uuid.UUID, error) {
	val := viper.GetString(flagName)
	if val == "" {
		return uuid.Nil, fmt.Errorf("%s is required", label)
	}

	res := uuid.FromStringOrNil(val)
	if res == uuid.Nil {
		return uuid.Nil, fmt.Errorf("invalid format for %s: '%s' is not a valid UUID", label, val)
	}

	return res, nil
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSON")
	}
	fmt.Println(string(data))
	return nil
}

func parseActions(actionsStr string) ([]action.Action, error) {
	if actionsStr == "" {
		return []action.Action{}, nil
	}

	var actions []action.Action
	if err := json.Unmarshal([]byte(actionsStr), &actions); err != nil {
		return nil, errors.Wrap(err, "failed to parse actions JSON")
	}
	return actions, nil
}

func cmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new flow",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("type", "flow", "Flow type (flow, conference, queue, campaign, transfer)")
	flags.String("name", "", "Flow name (required)")
	flags.String("detail", "", "Flow detail")
	flags.Bool("persist", false, "Persist flow")
	flags.String("actions", "", "Actions JSON array")
	flags.String("on-complete-flow-id", "", "On complete flow ID")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}

	actions, err := parseActions(viper.GetString("actions"))
	if err != nil {
		return errors.Wrap(err, "failed to parse actions")
	}

	onCompleteFlowID := uuid.FromStringOrNil(viper.GetString("on-complete-flow-id"))

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(
		context.Background(),
		customerID,
		flow.Type(viper.GetString("type")),
		name,
		viper.GetString("detail"),
		viper.GetBool("persist"),
		actions,
		onCompleteFlowID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create flow")
	}

	return printJSON(res)
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a flow by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Flow ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	flowID, err := resolveUUID("id", "Flow ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve flow ID")
	}

	res, err := handler.Get(context.Background(), flowID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve flow")
	}

	return printJSON(res)
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get flow list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of flows to retrieve")
	flags.String("token", "", "Retrieve flows before this token (pagination)")
	flags.String("customer-id", "", "Customer ID to filter (required)")
	flags.String("type", "", "Flow type to filter")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")

	filters := map[flow.Field]any{
		flow.FieldCustomerID: customerID,
		flow.FieldDeleted:    false,
	}

	flowType := viper.GetString("type")
	if flowType != "" {
		filters[flow.FieldType] = flow.Type(flowType)
	}

	res, err := handler.List(context.Background(), token, uint64(limit), filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve flows")
	}

	return printJSON(res)
}

func cmdUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a flow",
		RunE:  runUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Flow ID (required)")
	flags.String("name", "", "New flow name (required)")
	flags.String("detail", "", "New flow detail")
	flags.String("actions", "", "Actions JSON array")
	flags.String("on-complete-flow-id", "", "On complete flow ID")

	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	flowID, err := resolveUUID("id", "Flow ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve flow ID")
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}

	actions, err := parseActions(viper.GetString("actions"))
	if err != nil {
		return errors.Wrap(err, "failed to parse actions")
	}

	onCompleteFlowID := uuid.FromStringOrNil(viper.GetString("on-complete-flow-id"))

	res, err := handler.Update(context.Background(), flowID, name, viper.GetString("detail"), actions, onCompleteFlowID)
	if err != nil {
		return errors.Wrap(err, "failed to update flow")
	}

	return printJSON(res)
}

func cmdUpdateActions() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-actions",
		Short: "Update flow actions",
		RunE:  runUpdateActions,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Flow ID (required)")
	flags.String("actions", "", "Actions JSON array (required)")

	return cmd
}

func runUpdateActions(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	flowID, err := resolveUUID("id", "Flow ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve flow ID")
	}

	actionsStr := viper.GetString("actions")
	if actionsStr == "" {
		return fmt.Errorf("actions is required")
	}

	actions, err := parseActions(actionsStr)
	if err != nil {
		return errors.Wrap(err, "failed to parse actions")
	}

	res, err := handler.UpdateActions(context.Background(), flowID, actions)
	if err != nil {
		return errors.Wrap(err, "failed to update flow actions")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a flow",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Flow ID (required)")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Flow ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve flow ID")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete flow")
	}

	return printJSON(res)
}

func cmdActionGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "action-get",
		Short: "Get a specific action from a flow",
		RunE:  runActionGet,
	}

	flags := cmd.Flags()
	flags.String("flow-id", "", "Flow ID (required)")
	flags.String("action-id", "", "Action ID (required)")

	return cmd
}

func runActionGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	flowID, err := resolveUUID("flow-id", "Flow ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve flow ID")
	}

	actionID, err := resolveUUID("action-id", "Action ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve action ID")
	}

	res, err := handler.ActionGet(context.Background(), flowID, actionID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve action")
	}

	return printJSON(res)
}
