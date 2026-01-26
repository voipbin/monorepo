package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-agent-manager/internal/config"
	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/agenthandler"
	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"
	commonaddress "monorepo/bin-common-handler/models/address"
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

const serviceName = commonoutline.ServiceNameAgentManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (agenthandler.AgentHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initAgentHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initAgentHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (agenthandler.AgentHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameAgentEvent, serviceName)

	return agenthandler.NewAgentHandler(reqHandler, db, notifyHandler), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "agent-control",
		Short: "Voipbin Agent Management CLI",
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

	cmdSub := &cobra.Command{Use: "agent", Short: "Agent operation"}
	cmdSub.AddCommand(cmdCreate())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdList())
	cmdSub.AddCommand(cmdUpdatePermission())
	cmdSub.AddCommand(cmdUpdatePassword())
	cmdSub.AddCommand(cmdDelete())

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

func cmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID (required)")
	flags.String("username", "", "Username (required)")
	flags.String("password", "", "Password (required)")
	flags.Uint64("permission", 0, "Permission")
	flags.String("name", "", "Agent name")
	flags.String("detail", "", "Description")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	username := viper.GetString("username")
	if username == "" {
		return fmt.Errorf("username is required")
	}

	password := viper.GetString("password")
	if password == "" {
		return fmt.Errorf("password is required")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(
		context.Background(),
		customerID,
		username,
		password,
		viper.GetString("name"),
		viper.GetString("detail"),
		agent.RingMethodRingAll,
		agent.Permission(viper.GetUint64("permission")),
		[]uuid.UUID{},
		[]commonaddress.Address{},
	)
	if err != nil {
		return errors.Wrap(err, "failed to create agent")
	}

	return printJSON(res)
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an agent by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Agent ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	agentID, err := resolveUUID("id", "Agent ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve agent ID")
	}

	res, err := handler.Get(context.Background(), agentID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve agent")
	}

	return printJSON(res)
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get agent list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of agents to retrieve")
	flags.String("token", "", "Retrieve agents before this token (pagination)")
	flags.String("customer_id", "", "Customer ID to filter (required)")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")

	filters := map[agent.Field]any{
		agent.FieldCustomerID: customerID,
		agent.FieldDeleted:    false,
	}

	res, err := handler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve agents")
	}

	return printJSON(res)
}

func cmdUpdatePermission() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-permission",
		Short: "Update agent permission",
		RunE:  runUpdatePermission,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Agent ID (required)")
	flags.Uint64("permission", uint64(agent.PermissionNone), "New Permission Bitmask")

	return cmd
}

func runUpdatePermission(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Agent ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve agent ID")
	}

	res, err := handler.UpdatePermissionRaw(context.Background(), id, agent.Permission(viper.GetUint64("permission")))
	if err != nil {
		return errors.Wrap(err, "failed to update agent permission")
	}

	return printJSON(res)
}

func cmdUpdatePassword() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-password",
		Short: "Update agent password",
		RunE:  runUpdatePassword,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Agent ID (required)")
	flags.String("password", "", "New Password (required)")

	return cmd
}

func runUpdatePassword(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Agent ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve agent ID")
	}

	password := viper.GetString("password")
	if password == "" {
		return fmt.Errorf("password is required")
	}

	res, err := handler.UpdatePassword(context.Background(), id, password)
	if err != nil {
		return errors.Wrap(err, "failed to update agent password")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an agent",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Agent ID (required)")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Agent ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve agent ID")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete agent")
	}

	return printJSON(res)
}
