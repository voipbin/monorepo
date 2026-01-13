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

	"github.com/AlecAivazis/survey/v2"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameNumberEvent, serviceName)

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
	cmdSub.AddCommand(cmdList())
	cmdSub.AddCommand(cmdDelete())

	cmdRoot.AddCommand(cmdSub)
	return cmdRoot
}

func resolveUUID(flagName string, label string) (uuid.UUID, error) {
	res := uuid.FromStringOrNil(viper.GetString(flagName))
	if res == uuid.Nil {
		tmp := ""
		prompt := &survey.Input{Message: fmt.Sprintf("%s (Required):", label)}
		if errAsk := survey.AskOne(prompt, &tmp, survey.WithValidator(survey.Required)); errAsk != nil {
			return uuid.Nil, errors.Wrap(errAsk, "input canceled")
		}

		res = uuid.FromStringOrNil(tmp)
		if res == uuid.Nil {
			return uuid.Nil, fmt.Errorf("invalid format for %s: '%s' is not a valid UUID", label, tmp)
		}
	}

	return res, nil
}

func resolveString(flagName string, label string, required bool) (string, error) {
	res := viper.GetString(flagName)
	if res == "" && required {
		prompt := &survey.Input{Message: fmt.Sprintf("%s (Required):", label)}
		if errAsk := survey.AskOne(prompt, &res, survey.WithValidator(survey.Required)); errAsk != nil {
			return "", errors.Wrap(errAsk, "input canceled")
		}
	}

	return res, nil
}

func cmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new number",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID")
	flags.String("number", "", "Phone number (e.g., +15551234567)")
	flags.String("call_flow_id", "", "Call flow ID")
	flags.String("message_flow_id", "", "Message flow ID")
	flags.String("name", "", "Number name")
	flags.String("detail", "", "Description")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	number, err := resolveString("number", "Phone number", true)
	if err != nil {
		return errors.Wrap(err, "failed to resolve phone number")
	}

	callFlowID := uuid.FromStringOrNil(viper.GetString("call_flow_id"))
	messageFlowID := uuid.FromStringOrNil(viper.GetString("message_flow_id"))

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(
		context.Background(),
		customerID,
		number,
		callFlowID,
		messageFlowID,
		viper.GetString("name"),
		viper.GetString("detail"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create number")
	}

	logrus.WithField("res", res).Infof("Created a new number")
	return nil
}

func cmdRegister() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new number",
		RunE:  runRegister,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID")
	flags.String("number", "", "Phone number (e.g., +15551234567)")
	flags.String("call_flow_id", "", "Call flow ID")
	flags.String("message_flow_id", "", "Message flow ID")
	flags.String("name", "", "Number name")
	flags.String("detail", "", "Description")

	return cmd
}

func runRegister(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	num, err := resolveString("number", "Phone number", true)
	if err != nil {
		return errors.Wrap(err, "failed to resolve phone number")
	}

	callFlowID := uuid.FromStringOrNil(viper.GetString("call_flow_id"))
	messageFlowID := uuid.FromStringOrNil(viper.GetString("message_flow_id"))

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

	logrus.WithField("res", res).Infof("Registered a new number")
	return nil
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a number by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Number ID")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	numberID, err := resolveUUID("id", "Number ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve number ID")
	}

	fmt.Printf("\nRetrieving Number ID: %s...\n", numberID)
	res, err := handler.Get(context.Background(), numberID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve number")
	}

	fmt.Println("\n--- Number Information ---")
	fmt.Printf("ID:             %s\n", res.ID)
	fmt.Printf("Number:         %s\n", res.Number)
	fmt.Printf("Customer ID:    %s\n", res.CustomerID)
	fmt.Printf("Call Flow ID:   %s\n", res.CallFlowID)
	fmt.Printf("Message Flow ID: %s\n", res.MessageFlowID)
	fmt.Printf("Name:           %s\n", res.Name)
	fmt.Printf("Provider:       %s\n", res.ProviderName)
	fmt.Printf("Status:         %s\n", res.Status)
	fmt.Println("----------------------------")

	tmp, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal number")
	}
	fmt.Println("\n--- Raw Data (JSON) ---")
	fmt.Println(string(tmp))
	fmt.Println("-----------------------")

	return nil
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
	flags.String("customer_id", "", "Customer ID to filter")

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

	filters := map[number.Field]any{
		number.FieldCustomerID: customerID,
	}

	fmt.Printf("\nRetrieving Numbers... limit: %d, token: %s, filters: %v\n", limit, token, filters)
	res, err := handler.Gets(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve numbers")
	}

	fmt.Printf("Success! numbers count: %d\n", len(res))
	for _, n := range res {
		fmt.Printf(" - [%s] %s (%s) - %s\n", n.ID, n.Number, n.ProviderName, n.Status)
	}

	return nil
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a number by ID",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Number ID")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	numberID, err := resolveUUID("id", "Number ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve number ID")
	}

	n, err := handler.Get(context.Background(), numberID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve number")
	}

	fmt.Printf("\n--- Number Information ---\n")
	fmt.Printf("ID:             %s\n", n.ID)
	fmt.Printf("Number:         %s\n", n.Number)
	fmt.Printf("Customer ID:    %s\n", n.CustomerID)
	fmt.Printf("Call Flow ID:   %s\n", n.CallFlowID)
	fmt.Printf("Message Flow ID: %s\n", n.MessageFlowID)
	fmt.Printf("Name:           %s\n", n.Name)
	fmt.Printf("Provider:       %s\n", n.ProviderName)
	fmt.Printf("Status:         %s\n", n.Status)
	fmt.Println("----------------------------")

	confirm := false
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("Are you sure you want to delete number %s?", numberID),
	}
	if err := survey.AskOne(prompt, &confirm); err != nil {
		return errors.Wrap(err, "confirmation canceled")
	}

	if !confirm {
		fmt.Println("Deletion canceled")
		return nil
	}

	fmt.Printf("\nDeleting Number ID: %s...\n", numberID)
	res, err := handler.Delete(context.Background(), numberID)
	if err != nil {
		return errors.Wrap(err, "failed to delete number")
	}

	logrus.WithField("res", res).Infof("Deleted number")
	return nil
}
