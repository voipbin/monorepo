package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-number-manager/internal/config"
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/cachehandler"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/numberhandler"
	"monorepo/bin-number-manager/pkg/numberhandlertelnyx"
	"monorepo/bin-number-manager/pkg/numberhandlertwilio"

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

const serviceName = commonoutline.ServiceNameNumberManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (numberhandler.NumberHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initNumberHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initNumberHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (numberhandler.NumberHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameNumberEvent, serviceName, "")

	nHandlerTelnyx := numberhandlertelnyx.NewNumberHandler(
		reqHandler,
		db,
		config.Get().TelnyxConnectionID,
		config.Get().TelnyxProfileID,
		config.Get().TelnyxToken,
	)
	nHandlerTwilio := numberhandlertwilio.NewNumberHandler(
		reqHandler,
		db,
		config.Get().TwilioSID,
		config.Get().TwilioToken,
	)

	return numberhandler.NewNumberHandler(reqHandler, db, notifyHandler, nHandlerTelnyx, nHandlerTwilio), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "number-control",
		Short: "Voipbin Number Management CLI",
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

	cmdSub := &cobra.Command{Use: "number", Short: "Number operations"}
	cmdSub.AddCommand(cmdCreate())
	cmdSub.AddCommand(cmdRegister())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdGetAvailable())
	cmdSub.AddCommand(cmdList())
	cmdSub.AddCommand(cmdUpdate())
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

func resolveString(flagName string, label string) (string, error) {
	val := viper.GetString(flagName)
	if val == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	return val, nil
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
		Short: "Create a new number",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("number", "", "Phone number (e.g., +15551234567) (required)")
	flags.String("call-flow-id", "", "Call flow ID")
	flags.String("message-flow-id", "", "Message flow ID")
	flags.String("name", "", "Number name")
	flags.String("detail", "", "Description")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID")
	}

	num, err := resolveString("number", "Phone number")
	if err != nil {
		return errors.Wrap(err, "invalid phone number")
	}

	callFlowID := uuid.FromStringOrNil(viper.GetString("call-flow-id"))
	messageFlowID := uuid.FromStringOrNil(viper.GetString("message-flow-id"))

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(
		context.Background(),
		customerID,
		num,
		callFlowID,
		messageFlowID,
		viper.GetString("name"),
		viper.GetString("detail"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create number")
	}

	return printJSON(res)
}

func cmdRegister() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new number",
		RunE:  runRegister,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("number", "", "Phone number (e.g., +15551234567) (required)")
	flags.String("call-flow-id", "", "Call flow ID")
	flags.String("message-flow-id", "", "Message flow ID")
	flags.String("name", "", "Number name")
	flags.String("detail", "", "Description")

	return cmd
}

func runRegister(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID")
	}

	num, err := resolveString("number", "Phone number")
	if err != nil {
		return errors.Wrap(err, "invalid phone number")
	}

	callFlowID := uuid.FromStringOrNil(viper.GetString("call-flow-id"))
	messageFlowID := uuid.FromStringOrNil(viper.GetString("message-flow-id"))

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Register(
		context.Background(),
		customerID,
		num,
		callFlowID,
		messageFlowID,
		viper.GetString("name"),
		viper.GetString("detail"),
		number.ProviderNameNone,
		"",
		number.StatusActive,
		false,
		false,
	)
	if err != nil {
		return errors.Wrap(err, "failed to register number")
	}

	return printJSON(res)
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a number by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Number ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	numberID, err := resolveUUID("id", "Number ID")
	if err != nil {
		return errors.Wrap(err, "invalid number ID")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Get(context.Background(), numberID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve number")
	}

	return printJSON(res)
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get number list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of results to retrieve")
	flags.String("token", "", "Retrieve results before this token (pagination)")
	flags.String("customer-id", "", "Customer ID to filter (required)")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")

	filters := map[number.Field]any{
		number.FieldCustomerID: customerID,
	}

	res, err := handler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve numbers")
	}

	return printJSON(res)
}

func cmdGetAvailable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-available",
		Short: "Get available numbers for purchase",
		RunE:  runGetAvailable,
	}

	flags := cmd.Flags()
	flags.String("country-code", "", "Country code (e.g., US, GB) (required)")
	flags.Uint("limit", 10, "Number of results to return")

	return cmd
}

func runGetAvailable(cmd *cobra.Command, args []string) error {
	countryCode, err := resolveString("country-code", "Country code")
	if err != nil {
		return errors.Wrap(err, "invalid country code")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	limit := viper.GetUint("limit")
	res, err := handler.GetAvailableNumbers(countryCode, limit)
	if err != nil {
		return errors.Wrap(err, "failed to get available numbers")
	}

	return printJSON(res)
}

func cmdUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a number",
		RunE:  runUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Number ID (required)")
	flags.String("name", "", "Number name")
	flags.String("detail", "", "Description")
	flags.String("call-flow-id", "", "Call flow ID")
	flags.String("message-flow-id", "", "Message flow ID")
	flags.String("status", "", "Status (active, inactive)")

	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	numberID, err := resolveUUID("id", "Number ID")
	if err != nil {
		return errors.Wrap(err, "invalid number ID")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	fields := make(map[number.Field]any)

	if viper.IsSet("name") {
		fields[number.FieldName] = viper.GetString("name")
	}
	if viper.IsSet("detail") {
		fields[number.FieldDetail] = viper.GetString("detail")
	}
	if viper.GetString("call-flow-id") != "" {
		fields[number.FieldCallFlowID] = uuid.FromStringOrNil(viper.GetString("call-flow-id"))
	}
	if viper.GetString("message-flow-id") != "" {
		fields[number.FieldMessageFlowID] = uuid.FromStringOrNil(viper.GetString("message-flow-id"))
	}
	if viper.GetString("status") != "" {
		fields[number.FieldStatus] = number.Status(viper.GetString("status"))
	}

	if len(fields) == 0 {
		return fmt.Errorf("at least one field to update is required")
	}

	res, err := handler.Update(context.Background(), numberID, fields)
	if err != nil {
		return errors.Wrap(err, "failed to update number")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a number by ID",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Number ID (required)")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	numberID, err := resolveUUID("id", "Number ID")
	if err != nil {
		return errors.Wrap(err, "invalid number ID")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Delete(context.Background(), numberID)
	if err != nil {
		return errors.Wrap(err, "failed to delete number")
	}

	return printJSON(res)
}
