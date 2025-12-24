package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-customer-manager/internal/config"
	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/cachehandler"
	"monorepo/bin-customer-manager/pkg/customerhandler"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	"github.com/AlecAivazis/survey/v2"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = commonoutline.ServiceNameCustomerManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "customer-control",
		Short: "Voipbin Customer Management CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.LoadGlobalConfig()
			return nil
		},
	}

	if err := config.BindConfig(rootCmd); err != nil {
		cobra.CheckErr(errors.Wrap(err, "failed to bind infrastructure config"))
	}

	customerCmd := &cobra.Command{Use: "customer", Short: "Customer operation"}
	customerCmd.AddCommand(cmdCreate())
	customerCmd.AddCommand(cmdGet())
	customerCmd.AddCommand(cmdGets())

	rootCmd.AddCommand(customerCmd)
	return rootCmd
}

func cmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new customer",
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
	email := viper.GetString("email")
	if email == "" {
		if errAsk := survey.AskOne(&survey.Input{Message: "Email (Required):"}, &email, survey.WithValidator(survey.Required)); errAsk != nil {
			return errors.Wrap(errAsk, "failed to get email")
		}
	}

	customerHandler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	return executeCreate(customerHandler, email)
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

			return executeGet(handler, args[0])
		},
	}
}

func cmdGets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gets",
		Short: "Get customer list",
		RunE: func(cmd *cobra.Command, args []string) error {
			handler, err := initHandler()
			if err != nil {
				return errors.Wrap(err, "failed to initialize handlers")
			}

			limit := viper.GetInt("limit")
			token := viper.GetString("token")

			return executeGets(handler, limit, token)
		},
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of customers to retrieve")
	flags.String("token", "", "Retrieve customers before this token (pagination)")

	if errBind := viper.BindPFlags(flags); errBind != nil {
		cobra.CheckErr(errors.Wrap(errBind, "failed to bind flags"))
	}

	return cmd
}

func executeCreate(customerHandler customerhandler.CustomerHandler, email string) error {
	fmt.Printf("\nCreating Customer: %s\n", email)
	res, err := customerHandler.Create(
		context.Background(),
		viper.GetString("name"),
		viper.GetString("detail"),
		email,
		viper.GetString("phone_number"),
		viper.GetString("address"),
		customer.WebhookMethod(viper.GetString("webhook_method")),
		viper.GetString("webhook_uri"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create customer")
	}

	fmt.Printf("Success! customer: %v\n", res)
	return nil
}

func executeGets(customerHandler customerhandler.CustomerHandler, limit int, token string) error {
	fmt.Printf("\nRetrieving Customers (limit: %d, token: %s)...\n", limit, token)

	res, err := customerHandler.Gets(context.Background(), uint64(limit), token, nil)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve customers")
	}

	fmt.Printf("Success! customers count: %d\n", len(res))
	for _, c := range res {
		fmt.Printf(" - [%s] %s (%s)\n", c.ID, c.Name, c.Email)
	}

	return nil
}

func executeGet(customerHandler customerhandler.CustomerHandler, id string) error {
	targetID, err := uuid.FromString(id)
	if err != nil {
		return errors.Wrap(err, "invalid customer ID format")
	}

	fmt.Printf("\nRetrieving Customer ID: %s...\n", id)
	res, err := customerHandler.Get(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve customer")
	}

	fmt.Println("\n--- Customer Information ---")
	fmt.Printf("ID:      %s\n", res.ID)
	fmt.Printf("Name:    %s\n", res.Name)
	fmt.Printf("Email:   %s\n", res.Email)
	fmt.Printf("Phone:   %s\n", res.PhoneNumber)
	fmt.Printf("Address: %s\n", res.Address)
	fmt.Printf("Webhook: %s [%s]\n", res.WebhookURI, res.WebhookMethod)
	fmt.Printf("Detail:  %s\n", res.Detail)
	fmt.Println("----------------------------")

	tmp, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal customer")
	}
	fmt.Println("\n--- Raw Data (JSON) ---")
	fmt.Println(string(tmp))
	fmt.Println("-----------------------")

	return nil
}

func initHandler() (customerhandler.CustomerHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, err
	}

	return initCustomerHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if err := res.Connect(); err != nil {
		return nil, err
	}
	return res, nil
}

func initCustomerHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (customerhandler.CustomerHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameCustomerEvent, serviceName)

	return customerhandler.NewCustomerHandler(reqHandler, db, notifyHandler), nil
}
