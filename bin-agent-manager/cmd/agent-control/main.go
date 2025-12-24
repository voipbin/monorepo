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

	"github.com/AlecAivazis/survey/v2"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

func initCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "agent-control",
		Short: "Voipbin Agent Management CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.LoadGlobalConfig()
			return nil
		},
	}

	if err := config.BindConfig(rootCmd); err != nil {
		cobra.CheckErr(errors.Wrap(err, "failed to bind infrastructure config"))
	}

	cmdSub := &cobra.Command{Use: "agent", Short: "Agent operation"}
	cmdSub.AddCommand(cmdCreate())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdGets())

	rootCmd.AddCommand(cmdSub)
	return rootCmd
}

func cmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("name", "", "Customer name")
	flags.String("detail", "", "Description")
	flags.String("email", "", "Customer email (required)")
	flags.String("phone_number", "", "Phone number")
	flags.String("address", "", "Physical address")
	flags.String("webhook_method", "POST", "Webhook HTTP method")
	flags.String("webhook_uri", "", "Webhook URI")

	if errBind := viper.BindPFlags(flags); errBind != nil {
		cobra.CheckErr(errors.Wrap(errBind, "failed to bind flags"))
	}

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {

	customerID := uuid.FromStringOrNil(viper.GetString("customer_id"))
	if customerID == uuid.Nil {
		tmp := ""
		if errAsk := survey.AskOne(&survey.Input{Message: "Customer ID (Required):"}, &tmp, survey.WithValidator(survey.Required)); errAsk != nil {
			return errors.Wrap(errAsk, "failed to get customer ID")
		}

		customerID = uuid.FromStringOrNil(tmp)
		if customerID == uuid.Nil {
			return errors.New("invalid customer ID format")
		}
	}

	username := viper.GetString("username")
	if username == "" {
		if errAsk := survey.AskOne(&survey.Input{Message: "Username (Required):"}, &username, survey.WithValidator(survey.Required)); errAsk != nil {
			return errors.Wrap(errAsk, "failed to get username")
		}
	}

	password := viper.GetString("password")
	if password == "" {
		if errAsk := survey.AskOne(&survey.Password{Message: "Password (Required):"}, &password, survey.WithValidator(survey.Required)); errAsk != nil {
			return errors.Wrap(errAsk, "failed to get password")
		}
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

	logrus.WithField("res", res).Infof("Created a new agent")
	return nil
}

func cmdGet() *cobra.Command {
	return &cobra.Command{
		Use:   "get [id]",
		Short: "Get a customer by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			handler, err := initHandler()
			if err != nil {
				return errors.Wrap(err, "failed to initialize handlers")
			}

			return runGet(handler, args[0])
		},
	}
}

func cmdGets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gets",
		Short: "Get agent list",
		RunE:  runGets,
		// RunE: func(cmd *cobra.Command, args []string) error {
		// 	handler, err := initHandler()
		// 	if err != nil {
		// 		return errors.Wrap(err, "failed to initialize handlers")
		// 	}

		// 	limit := viper.GetInt("limit")
		// 	token := viper.GetString("token")

		// 	return runGets(handler, limit, token)
		// },
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of customers to retrieve")
	flags.String("token", "", "Retrieve customers before this token (pagination)")

	if errBind := viper.BindPFlags(flags); errBind != nil {
		cobra.CheckErr(errors.Wrap(errBind, "failed to bind flags"))
	}

	return cmd
}

// func executeCreate(handler agenthandler.AgentHandler, email string) error {

// 	// handler.Create(
// 	// 	ctx,
// 	// 	viper.GetString("customer_id")
// 	// )

// 	fmt.Printf("\nCreating Agent: %s\n", email)
// 	res, err := handler.Create(
// 		context.Background(),
// 		viper.GetString("name"),
// 		viper.GetString("detail"),
// 		email,
// 		viper.GetString("phone_number"),
// 		viper.GetString("address"),
// 		customer.WebhookMethod(viper.GetString("webhook_method")),
// 		viper.GetString("webhook_uri"),
// 	)
// 	if err != nil {
// 		return errors.Wrap(err, "failed to create customer")
// 	}

// 	fmt.Printf("Success! customer: %v\n", res)
// 	return nil
// }

func runGets(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID := viper.GetString("customer_id")
	if customerID == "" {
		if errAsk := survey.AskOne(&survey.Input{Message: "Customer ID (Required):"}, &customerID, survey.WithValidator(survey.Required)); errAsk != nil {
			return errors.Wrap(errAsk, "failed to get customer ID")
		}

		if customerID == "" {
			return errors.New("invalid customer ID format")
		}
	}
	limit := viper.GetInt("limit")
	token := viper.GetString("token")

	filters := map[string]string{
		"customer_id": customerID,
	}

	fmt.Printf("\nRetrieving Agents... limit: %d, token: %s, filters: %v\n", limit, token, filters)
	res, err := handler.Gets(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve agents")
	}

	fmt.Printf("Success! agents count: %d\n", len(res))
	for _, c := range res {
		fmt.Printf(" - [%s] %s (%s)\n", c.ID, c.Name, c.Status)
	}

	return nil
}

func runGet(handler agenthandler.AgentHandler, id string) error {
	targetID, err := uuid.FromString(id)
	if err != nil {
		return errors.Wrap(err, "invalid customer ID format")
	}

	fmt.Printf("\nRetrieving Agent ID: %s...\n", id)
	res, err := handler.Get(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve agent")
	}

	fmt.Println("\n--- Agent Information ---")
	fmt.Printf("ID:      %s\n", res.ID)
	fmt.Printf("Customer ID: %s\n", res.CustomerID)
	fmt.Printf("Name:    %s\n", res.Name)
	fmt.Printf("Detail:    %s\n", res.Detail)
	fmt.Println("----------------------------")

	tmp, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal agent")
	}
	fmt.Println("\n--- Raw Data (JSON) ---")
	fmt.Println(string(tmp))
	fmt.Println("-----------------------")

	return nil
}

func initHandler() (agenthandler.AgentHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, err
	}

	cache, err := initCache()
	if err != nil {
		return nil, err
	}

	return initAgentHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if err := res.Connect(); err != nil {
		return nil, err
	}
	return res, nil
}

func initAgentHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (agenthandler.AgentHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameCustomerEvent, serviceName)

	return agenthandler.NewAgentHandler(reqHandler, db, notifyHandler), nil
}
