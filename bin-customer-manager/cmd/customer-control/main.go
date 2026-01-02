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
	"github.com/sirupsen/logrus"

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
	cmdRoot := &cobra.Command{
		Use:   "customer-control",
		Short: "Voipbin Customer Management CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if errBind := viper.BindPFlags(cmd.Flags()); errBind != nil {
				return errors.Wrap(errBind, "failed to bind flags")
			}

			config.LoadGlobalConfig()
			return nil
		},
	}

	if err := config.Bootstrap(cmdRoot); err != nil {
		cobra.CheckErr(errors.Wrap(err, "failed to bootstrap config"))
	}

	cmdSub := &cobra.Command{Use: "customer", Short: "Customer operation"}
	cmdSub.AddCommand(cmdCreate())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdList())
	cmdSub.AddCommand(cmdDelete())

	cmdRoot.AddCommand(cmdSub)
	return cmdRoot
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
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	email := viper.GetString("email")
	if email == "" {
		if errAsk := survey.AskOne(&survey.Input{Message: "Email (Required):"}, &email, survey.WithValidator(survey.Required)); errAsk != nil {
			return errors.Wrap(errAsk, "failed to get email")
		}
	}

	fmt.Printf("\nCreating Customer: %s\n", email)
	res, err := handler.Create(
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

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a customer by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Customer ID (required)")

	return cmd
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get customer list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of customers to retrieve")
	flags.String("token", "", "Retrieve customers before this token (pagination)")

	return cmd
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a customer",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Customer ID")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")

	fmt.Printf("\nRetrieving Customers (limit: %d, token: %s)...\n", limit, token)

	res, err := handler.Gets(context.Background(), uint64(limit), token, nil)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve customers")
	}

	fmt.Printf("Success! customers count: %d\n", len(res))
	for _, c := range res {
		fmt.Printf(" - [%s] %s (%s)\n", c.ID, c.Name, c.Email)
	}

	return nil
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID format")
	}

	fmt.Printf("\nRetrieving Customer ID: %s...\n", targetID)
	res, err := handler.Get(context.Background(), targetID)
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

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID format")
	}

	c, err := handler.Get(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve customer")
	}

	fmt.Printf("\n--- Customer Information ---\n")
	fmt.Printf("ID:      %s\n", c.ID)
	fmt.Printf("Name:    %s\n", c.Name)
	fmt.Printf("Email:   %s\n", c.Email)
	fmt.Printf("Phone:   %s\n", c.PhoneNumber)
	fmt.Printf("Address: %s\n", c.Address)
	fmt.Printf("Webhook: %s [%s]\n", c.WebhookURI, c.WebhookMethod)
	fmt.Printf("Detail:  %s\n", c.Detail)
	fmt.Println("----------------------------")

	confirm := false
	if err := survey.AskOne(&survey.Confirm{Message: fmt.Sprintf("Are you sure you want to delete customer %s?", targetID)}, &confirm); err != nil {
		return errors.Wrap(err, "failed to get confirmation")
	}

	if !confirm {
		fmt.Println("Deletion canceled")
		return nil
	}

	fmt.Printf("\nDeleting Customer ID: %s...\n", targetID)
	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete customer")
	}

	logrus.WithField("res", res).Infof("Deleted customer")
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
