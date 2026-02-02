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
	cmdAccount.AddCommand(cmdAccountCreate())
	cmdAccount.AddCommand(cmdAccountGet())
	cmdAccount.AddCommand(cmdAccountList())
	cmdAccount.AddCommand(cmdAccountUpdate())
	cmdAccount.AddCommand(cmdAccountUpdatePaymentInfo())
	cmdAccount.AddCommand(cmdAccountDelete())
	cmdAccount.AddCommand(cmdAccountAddBalance())
	cmdAccount.AddCommand(cmdAccountSubtractBalance())

	// Billing subcommands
	cmdBilling := &cobra.Command{Use: "billing", Short: "Billing operations"}
	cmdBilling.AddCommand(cmdBillingGet())
	cmdBilling.AddCommand(cmdBillingList())

	cmdRoot.AddCommand(cmdAccount)
	cmdRoot.AddCommand(cmdBilling)
	return cmdRoot
}

// Account commands

func cmdAccountCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new account",
		RunE:  runAccountCreate,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("name", "", "Account name")
	flags.String("detail", "", "Account detail")
	flags.String("payment-type", "prepaid", "Payment type (prepaid)")
	flags.String("payment-method", "", "Payment method (credit card)")

	return cmd
}

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

func cmdAccountUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an account basic info",
		RunE:  runAccountUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Account ID (required)")
	flags.String("name", "", "Account name (required)")
	flags.String("detail", "", "Account detail")

	return cmd
}

func runAccountUpdate(cmd *cobra.Command, args []string) error {
	accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid account ID format")
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}

	res, err := accountHandler.UpdateBasicInfo(context.Background(), targetID, name, viper.GetString("detail"))
	if err != nil {
		return errors.Wrap(err, "failed to update account")
	}

	return printJSON(res)
}

func cmdAccountUpdatePaymentInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-payment-info",
		Short: "Update account payment info",
		RunE:  runAccountUpdatePaymentInfo,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Account ID (required)")
	flags.String("payment-type", "", "Payment type (prepaid) (required)")
	flags.String("payment-method", "", "Payment method (credit card) (required)")

	return cmd
}

func runAccountUpdatePaymentInfo(cmd *cobra.Command, args []string) error {
	accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid account ID format")
	}

	paymentType := viper.GetString("payment-type")
	if paymentType == "" {
		return fmt.Errorf("payment-type is required")
	}

	paymentMethod := viper.GetString("payment-method")
	if paymentMethod == "" {
		return fmt.Errorf("payment-method is required")
	}

	res, err := accountHandler.UpdatePaymentInfo(
		context.Background(),
		targetID,
		account.PaymentType(paymentType),
		account.PaymentMethod(paymentMethod),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update account payment info")
	}

	return printJSON(res)
}

func cmdAccountDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an account",
		RunE:  runAccountDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Account ID (required)")

	return cmd
}

func cmdAccountAddBalance() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-balance",
		Short: "Add balance to an account",
		RunE:  runAccountAddBalance,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Account ID (required)")
	flags.Float64("amount", 0, "Amount to add (required)")

	return cmd
}

func cmdAccountSubtractBalance() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subtract-balance",
		Short: "Subtract balance from an account",
		RunE:  runAccountSubtractBalance,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Account ID (required)")
	flags.Float64("amount", 0, "Amount to subtract (required)")

	return cmd
}

func runAccountCreate(cmd *cobra.Command, args []string) error {
	accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID format")
	}

	name := viper.GetString("name")
	detail := viper.GetString("detail")
	paymentType := account.PaymentType(viper.GetString("payment-type"))
	paymentMethod := account.PaymentMethod(viper.GetString("payment-method"))

	res, err := accountHandler.Create(context.Background(), customerID, name, detail, paymentType, paymentMethod)
	if err != nil {
		return errors.Wrap(err, "failed to create account")
	}

	return printJSON(res)
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

	res, err := accountHandler.Get(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve account")
	}

	return printJSON(res)
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

	res, err := accountHandler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve accounts")
	}

	return printJSON(res)
}

func runAccountDelete(cmd *cobra.Command, args []string) error {
	accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid account ID format")
	}

	res, err := accountHandler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete account")
	}

	return printJSON(res)
}

func runAccountAddBalance(cmd *cobra.Command, args []string) error {
	accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid account ID format")
	}

	amount := viper.GetFloat64("amount")
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	res, err := accountHandler.AddBalance(context.Background(), targetID, float32(amount))
	if err != nil {
		return errors.Wrap(err, "failed to add balance")
	}

	return printJSON(res)
}

func runAccountSubtractBalance(cmd *cobra.Command, args []string) error {
	accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid account ID format")
	}

	amount := viper.GetFloat64("amount")
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	res, err := accountHandler.SubtractBalance(context.Background(), targetID, float32(amount))
	if err != nil {
		return errors.Wrap(err, "failed to subtract balance")
	}

	return printJSON(res)
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

	res, err := billingHandler.Get(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve billing")
	}

	return printJSON(res)
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

	res, err := billingHandler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve billings")
	}

	return printJSON(res)
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
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameBillingEvent, serviceName, "")

	accHandler := accounthandler.NewAccountHandler(reqHandler, db, notifyHandler)
	billHandler := billinghandler.NewBillingHandler(reqHandler, db, notifyHandler, accHandler)

	return accHandler, billHandler, nil
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
