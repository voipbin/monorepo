package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
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
	if errInit := config.InitAll(); errInit != nil {
		log.Fatalf("Could not init config. err: %v", errInit)
	}

	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "customer-control",
		Short: "Voipbin Customer Management CLI",
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
		Run:   runCreate,
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
		log.Fatalf("Failed to bind flags: %v", errBind)
	}

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) {
	email := viper.GetString("email")
	if email == "" {
		if errAsk := survey.AskOne(&survey.Input{Message: "Email (Required):"}, &email, survey.WithValidator(survey.Required)); errAsk != nil {
			log.Fatalf("Failed to get email: %v", errAsk)
		}
	}

	customerHandler, err := initHandler()
	if err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}

	executeCreate(customerHandler, email)
}

func cmdGet() *cobra.Command {
	return &cobra.Command{
		Use:   "get [id]",
		Short: "Get a customer by ID",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			handler, err := initHandler()
			if err != nil {
				log.Fatalf("Failed to initialize handlers: %v", err)
			}
			executeGet(handler, args[0])
		},
	}
}

func cmdGets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gets",
		Short: "Get customer list",
		Run: func(cmd *cobra.Command, args []string) {
			handler, err := initHandler()
			if err != nil {
				log.Fatalf("Failed to initialize handlers: %v", err)
			}

			limit := viper.GetInt("limit")
			token := viper.GetString("token")
			executeGets(handler, limit, token)
		},
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of customers to retrieve")
	flags.String("token", "", "Retrieve customers before this token (pagination)")

	if errBind := viper.BindPFlags(flags); errBind != nil {
		log.Fatalf("Failed to bind flags: %v", errBind)
	}

	return cmd
}

func executeCreate(customerHandler customerhandler.CustomerHandler, email string) {
	method := viper.GetString("webhook_method")
	uri := viper.GetString("webhook_uri")

	fmt.Printf("\nCreating Customer: %s (Webhook: %s [%s])\n", email, uri, method)
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
		log.Fatalf("Failed to create customer: %v", err)
	}

	fmt.Printf("Success! customer: %v\n", res)
}

func executeGets(customerHandler customerhandler.CustomerHandler, limit int, token string) {
	fmt.Printf("\nRetrieving Customers (limit: %d, token: %s)...\n", limit, token)

	res, err := customerHandler.Gets(context.Background(), uint64(limit), token, nil)
	if err != nil {
		log.Fatalf("Failed to retrieve customers: %v", err)
	}

	fmt.Printf("Success! customers count: %d\n", len(res))
	for _, c := range res {
		fmt.Printf(" - [%s] %s (%s)\n", c.ID, c.Name, c.Email)
	}
}

func executeGet(customerHandler customerhandler.CustomerHandler, id string) {
	targetID := uuid.FromStringOrNil(id)

	fmt.Printf("\nRetrieving Customer ID: %s...\n", id)
	res, err := customerHandler.Get(context.Background(), targetID)
	if err != nil {
		log.Fatalf("Failed to retrieve customer: %v", err)
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
		log.Fatalf("Failed to marshal customer: %v", err)
	}
	fmt.Println("\n--- Raw Data (JSON) ---")
	fmt.Println(string(tmp))
	fmt.Println("-----------------------")
}

func initHandler() (customerhandler.CustomerHandler, error) {
	db, err := initDatabase()
	if err != nil {
		return nil, err
	}

	cache, err := initCache()
	if err != nil {
		return nil, err
	}

	return initCustomerHandler(db, cache)
}

func initDatabase() (*sql.DB, error) {
	res, err := sql.Open("mysql", config.GlobalConfig.DatabaseDSN)
	if err != nil {
		return nil, errors.Wrap(err, "database open error")
	}
	if err := res.Ping(); err != nil {
		return nil, errors.Wrap(err, "database ping error")
	}
	return res, nil
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.GlobalConfig.RedisAddress, config.GlobalConfig.RedisPassword, config.GlobalConfig.RedisDatabase)
	if err := res.Connect(); err != nil {
		return nil, err
	}
	return res, nil
}

func initCustomerHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (customerhandler.CustomerHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.GlobalConfig.RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameCustomerEvent, serviceName)

	return customerhandler.NewCustomerHandler(reqHandler, db, notifyHandler), nil
}
