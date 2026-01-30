package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-customer-manager/internal/config"
	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/accesskeyhandler"
	"monorepo/bin-customer-manager/pkg/cachehandler"
	"monorepo/bin-customer-manager/pkg/customerhandler"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	_ "github.com/go-sql-driver/mysql"

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

	cmdCustomer := &cobra.Command{Use: "customer", Short: "Customer operation"}
	cmdCustomer.AddCommand(cmdCreate())
	cmdCustomer.AddCommand(cmdGet())
	cmdCustomer.AddCommand(cmdList())
	cmdCustomer.AddCommand(cmdUpdate())
	cmdCustomer.AddCommand(cmdUpdateBillingAccount())
	cmdCustomer.AddCommand(cmdDelete())

	cmdAccesskey := &cobra.Command{Use: "accesskey", Short: "Accesskey operation"}
	cmdAccesskey.AddCommand(cmdAccesskeyCreate())
	cmdAccesskey.AddCommand(cmdAccesskeyGet())
	cmdAccesskey.AddCommand(cmdAccesskeyList())
	cmdAccesskey.AddCommand(cmdAccesskeyUpdate())
	cmdAccesskey.AddCommand(cmdAccesskeyDelete())

	cmdRoot.AddCommand(cmdCustomer)
	cmdRoot.AddCommand(cmdAccesskey)
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
	flags.String("phone-number", "", "Phone number")
	flags.String("address", "", "Physical address")
	flags.String("webhook-method", "POST", "Webhook HTTP method")
	flags.String("webhook-uri", "", "Webhook URI")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	email, err := resolveString("email", "Email")
	if err != nil {
		return errors.Wrap(err, "invalid email")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(
		context.Background(),
		viper.GetString("name"),
		viper.GetString("detail"),
		email,
		viper.GetString("phone-number"),
		viper.GetString("address"),
		customer.WebhookMethod(viper.GetString("webhook-method")),
		viper.GetString("webhook-uri"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create customer")
	}

	return printJSON(res)
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

func cmdUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update customer basic info",
		RunE:  runUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Customer ID (required)")
	flags.String("name", "", "Customer name")
	flags.String("detail", "", "Description")
	flags.String("email", "", "Customer email (required)")
	flags.String("phone-number", "", "Phone number")
	flags.String("address", "", "Physical address")
	flags.String("webhook-method", "POST", "Webhook HTTP method")
	flags.String("webhook-uri", "", "Webhook URI")

	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	targetID, err := resolveUUID("id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID")
	}

	email, err := resolveString("email", "Email")
	if err != nil {
		return errors.Wrap(err, "invalid email")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.UpdateBasicInfo(
		context.Background(),
		targetID,
		viper.GetString("name"),
		viper.GetString("detail"),
		email,
		viper.GetString("phone-number"),
		viper.GetString("address"),
		customer.WebhookMethod(viper.GetString("webhook-method")),
		viper.GetString("webhook-uri"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update customer")
	}

	return printJSON(res)
}

func cmdUpdateBillingAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-billing-account",
		Short: "Update customer billing account ID",
		RunE:  runUpdateBillingAccount,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Customer ID (required)")
	flags.String("billing-account-id", "", "Billing Account ID (required)")

	return cmd
}

func runUpdateBillingAccount(cmd *cobra.Command, args []string) error {
	targetID, err := resolveUUID("id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID")
	}

	billingAccountID, err := resolveUUID("billing-account-id", "Billing Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid billing account ID")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.UpdateBillingAccountID(context.Background(), targetID, billingAccountID)
	if err != nil {
		return errors.Wrap(err, "failed to update customer billing account")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a customer",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Customer ID (required)")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")

	res, err := handler.List(context.Background(), uint64(limit), token, nil)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve customers")
	}

	return printJSON(res)
}

func runGet(cmd *cobra.Command, args []string) error {
	targetID, err := resolveUUID("id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Get(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve customer")
	}

	return printJSON(res)
}

func runDelete(cmd *cobra.Command, args []string) error {
	targetID, err := resolveUUID("id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete customer")
	}

	return printJSON(res)
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

// Accesskey commands

func initAccesskeyHandler() (accesskeyhandler.AccesskeyHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, err
	}

	return initAccesskeyHandlerWithDeps(db, cache)
}

func initAccesskeyHandlerWithDeps(sqlDB *sql.DB, cache cachehandler.CacheHandler) (accesskeyhandler.AccesskeyHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameCustomerEvent, serviceName)

	return accesskeyhandler.NewAccesskeyHandler(reqHandler, db, notifyHandler), nil
}

func cmdAccesskeyCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new accesskey",
		RunE:  runAccesskeyCreate,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("name", "", "Accesskey name (required)")
	flags.String("detail", "", "Description")
	flags.String("expire", "", "Expiration duration (e.g., 720h for 30 days)")

	return cmd
}

func runAccesskeyCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID")
	}

	name, err := resolveString("name", "Name")
	if err != nil {
		return errors.Wrap(err, "invalid name")
	}

	detail := viper.GetString("detail")

	var expire time.Duration
	expireStr := viper.GetString("expire")
	if expireStr != "" {
		expire, err = time.ParseDuration(expireStr)
		if err != nil {
			return errors.Wrap(err, "invalid expire duration format")
		}
	}

	handler, err := initAccesskeyHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(context.Background(), customerID, name, detail, expire)
	if err != nil {
		return errors.Wrap(err, "failed to create accesskey")
	}

	return printJSON(res)
}

func cmdAccesskeyGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an accesskey by ID",
		RunE:  runAccesskeyGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Accesskey ID (required)")

	return cmd
}

func runAccesskeyGet(cmd *cobra.Command, args []string) error {
	targetID, err := resolveUUID("id", "Accesskey ID")
	if err != nil {
		return errors.Wrap(err, "invalid accesskey ID")
	}

	handler, err := initAccesskeyHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Get(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve accesskey")
	}

	return printJSON(res)
}

func cmdAccesskeyList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get accesskey list",
		RunE:  runAccesskeyList,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Filter by customer ID")
	flags.Int("size", 10, "Number of results to retrieve")
	flags.String("token", "", "Pagination token")

	return cmd
}

func runAccesskeyList(cmd *cobra.Command, args []string) error {
	handler, err := initAccesskeyHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	size := viper.GetInt("size")
	token := viper.GetString("token")

	customerIDStr := viper.GetString("customer-id")
	if customerIDStr != "" {
		customerID := uuid.FromStringOrNil(customerIDStr)
		if customerID == uuid.Nil {
			return errors.New("invalid customer ID format")
		}
		res, err := handler.GetsByCustomerID(context.Background(), uint64(size), token, customerID)
		if err != nil {
			return errors.Wrap(err, "failed to retrieve accesskeys")
		}
		return printJSON(res)
	}

	filters := map[accesskey.Field]any{}
	res, err := handler.List(context.Background(), uint64(size), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve accesskeys")
	}

	return printJSON(res)
}

func cmdAccesskeyUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update accesskey basic info",
		RunE:  runAccesskeyUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Accesskey ID (required)")
	flags.String("name", "", "Accesskey name")
	flags.String("detail", "", "Description")

	return cmd
}

func runAccesskeyUpdate(cmd *cobra.Command, args []string) error {
	targetID, err := resolveUUID("id", "Accesskey ID")
	if err != nil {
		return errors.Wrap(err, "invalid accesskey ID")
	}

	handler, err := initAccesskeyHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.UpdateBasicInfo(
		context.Background(),
		targetID,
		viper.GetString("name"),
		viper.GetString("detail"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update accesskey")
	}

	return printJSON(res)
}

func cmdAccesskeyDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an accesskey",
		RunE:  runAccesskeyDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Accesskey ID (required)")

	return cmd
}

func runAccesskeyDelete(cmd *cobra.Command, args []string) error {
	targetID, err := resolveUUID("id", "Accesskey ID")
	if err != nil {
		return errors.Wrap(err, "invalid accesskey ID")
	}

	handler, err := initAccesskeyHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete accesskey")
	}

	return printJSON(res)
}
