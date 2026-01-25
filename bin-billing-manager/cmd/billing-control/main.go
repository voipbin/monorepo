package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-billing-manager/internal/config"
	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/billinghandler"
	"monorepo/bin-billing-manager/pkg/cachehandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"

	_ "github.com/go-sql-driver/mysql"

	"github.com/AlecAivazis/survey/v2"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = commonoutline.ServiceNameBillingManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "billing-control",
		Short: "Voipbin Billing Management CLI",
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

	// Account subcommands
	cmdAccount := &cobra.Command{Use: "account", Short: "Account operations"}
	cmdAccount.AddCommand(cmdAccountGet())
	cmdAccount.AddCommand(cmdAccountList())

	// Billing subcommands
	cmdBilling := &cobra.Command{Use: "billing", Short: "Billing operations"}
	cmdBilling.AddCommand(cmdBillingGet())
	cmdBilling.AddCommand(cmdBillingList())

	cmdRoot.AddCommand(cmdAccount)
	cmdRoot.AddCommand(cmdBilling)
	return cmdRoot
}

// Account commands

func cmdAccountGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an account by ID",
		RunE:  runAccountGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Account ID (required)")

	return cmd
}

func cmdAccountList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get account list",
		RunE:  runAccountList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of accounts to retrieve")
	flags.String("token", "", "Retrieve accounts before this token (pagination)")
	flags.String("customer-id", "", "Filter by customer ID")

	return cmd
}

func runAccountGet(cmd *cobra.Command, args []string) error {
	accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid account ID format")
	}

	fmt.Printf("\nRetrieving Account ID: %s...\n", targetID)
	res, err := accountHandler.Get(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve account")
	}

	fmt.Println("\n--- Account Information ---")
	fmt.Printf("ID:             %s\n", res.ID)
	fmt.Printf("Customer ID:    %s\n", res.CustomerID)
	fmt.Printf("Name:           %s\n", res.Name)
	fmt.Printf("Detail:         %s\n", res.Detail)
	fmt.Printf("Type:           %s\n", res.Type)
	fmt.Printf("Balance:        $%.2f\n", res.Balance)
	fmt.Printf("Payment Type:   %s\n", res.PaymentType)
	fmt.Printf("Payment Method: %s\n", res.PaymentMethod)
	fmt.Printf("Created:        %s\n", res.TMCreate)
	fmt.Println("---------------------------")

	tmp, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal account")
	}
	fmt.Println("\n--- Raw Data (JSON) ---")
	fmt.Println(string(tmp))
	fmt.Println("-----------------------")

	return nil
}

func runAccountList(cmd *cobra.Command, args []string) error {
	accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")
	customerID := viper.GetString("customer-id")

	filters := make(map[account.Field]any)
	if customerID != "" {
		id := uuid.FromStringOrNil(customerID)
		if id == uuid.Nil {
			return fmt.Errorf("invalid customer ID format: %s", customerID)
		}
		filters[account.FieldCustomerID] = id
	}

	fmt.Printf("\nRetrieving Accounts (limit: %d, token: %s)...\n", limit, token)

	res, err := accountHandler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve accounts")
	}

	fmt.Printf("Success! accounts count: %d\n", len(res))
	for _, a := range res {
		fmt.Printf(" - [%s] %s | $%.2f | %s\n", a.ID, a.Name, a.Balance, a.Type)
	}

	return nil
}

// Billing commands

func cmdBillingGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a billing record by ID",
		RunE:  runBillingGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Billing ID (required)")

	return cmd
}

func cmdBillingList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get billing records list",
		RunE:  runBillingList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of billing records to retrieve")
	flags.String("token", "", "Retrieve billing records before this token (pagination)")
	flags.String("customer-id", "", "Filter by customer ID")
	flags.String("account-id", "", "Filter by account ID")

	return cmd
}

func runBillingGet(cmd *cobra.Command, args []string) error {
	_, billingHandler, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Billing ID")
	if err != nil {
		return errors.Wrap(err, "invalid billing ID format")
	}

	fmt.Printf("\nRetrieving Billing ID: %s...\n", targetID)
	res, err := billingHandler.Get(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve billing")
	}

	fmt.Println("\n--- Billing Information ---")
	fmt.Printf("ID:             %s\n", res.ID)
	fmt.Printf("Customer ID:    %s\n", res.CustomerID)
	fmt.Printf("Account ID:     %s\n", res.AccountID)
	fmt.Printf("Reference:      %s (%s)\n", res.ReferenceType, res.ReferenceID)
	fmt.Printf("Status:         %s\n", res.Status)
	fmt.Printf("Cost Per Unit:  $%.4f\n", res.CostPerUnit)
	fmt.Printf("Cost Total:     $%.4f\n", res.CostTotal)
	fmt.Printf("Billing Units:  %.2f\n", res.BillingUnitCount)
	fmt.Printf("Billing Start:  %s\n", res.TMBillingStart)
	fmt.Printf("Billing End:    %s\n", res.TMBillingEnd)
	fmt.Printf("Created:        %s\n", res.TMCreate)
	fmt.Println("---------------------------")

	tmp, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal billing")
	}
	fmt.Println("\n--- Raw Data (JSON) ---")
	fmt.Println(string(tmp))
	fmt.Println("-----------------------")

	return nil
}

func runBillingList(cmd *cobra.Command, args []string) error {
	_, billingHandler, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")
	customerID := viper.GetString("customer-id")
	accountID := viper.GetString("account-id")

	filters := make(map[billing.Field]any)
	if customerID != "" {
		id := uuid.FromStringOrNil(customerID)
		if id == uuid.Nil {
			return fmt.Errorf("invalid customer ID format: %s", customerID)
		}
		filters[billing.FieldCustomerID] = id
	}
	if accountID != "" {
		id := uuid.FromStringOrNil(accountID)
		if id == uuid.Nil {
			return fmt.Errorf("invalid account ID format: %s", accountID)
		}
		filters[billing.FieldAccountID] = id
	}

	fmt.Printf("\nRetrieving Billings (limit: %d, token: %s)...\n", limit, token)

	res, err := billingHandler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve billings")
	}

	fmt.Printf("Success! billings count: %d\n", len(res))
	for _, b := range res {
		fmt.Printf(" - [%s] %-12s | $%.4f | %-11s | %s\n",
			b.ID, b.ReferenceType, b.CostTotal, b.Status, b.TMCreate[:10])
	}

	return nil
}

// Handler initialization

func initHandlers() (accounthandler.AccountHandler, billinghandler.BillingHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, nil, err
	}

	return initBillingHandlers(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if err := res.Connect(); err != nil {
		return nil, err
	}
	return res, nil
}

func initBillingHandlers(sqlDB *sql.DB, cache cachehandler.CacheHandler) (accounthandler.AccountHandler, billinghandler.BillingHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameBillingEvent, serviceName)

	accHandler := accounthandler.NewAccountHandler(reqHandler, db, notifyHandler)
	billHandler := billinghandler.NewBillingHandler(reqHandler, db, notifyHandler, accHandler)

	return accHandler, billHandler, nil
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
