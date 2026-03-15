package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-webhook-manager/internal/config"
	"monorepo/bin-webhook-manager/models/webhook"
	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
	"monorepo/bin-webhook-manager/pkg/dbhandler"
	"monorepo/bin-webhook-manager/pkg/webhookhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = commonoutline.ServiceNameWebhookManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (webhookhandler.WebhookHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initWebhookHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initWebhookHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (webhookhandler.WebhookHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameWebhookEvent, serviceName)
	accountHandler := accounthandler.NewAccountHandler(db, reqHandler)

	return webhookhandler.NewWebhookHandler(db, notifyHandler, accountHandler), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "webhook-control",
		Short: "Voipbin Webhook Management CLI",
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

	cmdSub := &cobra.Command{Use: "webhook", Short: "Webhook operation"}
	cmdSub.AddCommand(cmdSendToCustomer())
	cmdSub.AddCommand(cmdSendToURI())

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

func cmdSendToCustomer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-to-customer",
		Short: "Send webhook to customer using their configured webhook settings",
		RunE:  runSendToCustomer,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("data-type", string(webhook.DataTypeJSON), "Data type (default: application/json)")
	flags.String("data", "", "JSON data to send (required)")

	return cmd
}

func runSendToCustomer(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	dataStr := viper.GetString("data")
	if dataStr == "" {
		return fmt.Errorf("data is required")
	}

	// Validate JSON
	var data json.RawMessage
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		return errors.Wrap(err, "invalid JSON data")
	}

	dataType := webhook.DataType(viper.GetString("data-type"))

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	err = handler.SendWebhookToCustomer(context.Background(), customerID, dataType, data)
	if err != nil {
		return errors.Wrap(err, "failed to send webhook to customer")
	}

	response := map[string]any{
		"status":      "success",
		"customer-id": customerID.String(),
		"data-type":   dataType,
		"message":     "Webhook sent to customer successfully",
	}

	return printJSON(response)
}

func cmdSendToURI() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-to-uri",
		Short: "Send webhook to a specific URI with specified method",
		RunE:  runSendToURI,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("uri", "", "Webhook URI (required)")
	flags.String("method", string(webhook.MethodTypePOST), "HTTP method (POST, PUT, GET, DELETE)")
	flags.String("data-type", string(webhook.DataTypeJSON), "Data type (default: application/json)")
	flags.String("data", "", "JSON data to send (required)")

	return cmd
}

func runSendToURI(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	uri := viper.GetString("uri")
	if uri == "" {
		return fmt.Errorf("uri is required")
	}

	dataStr := viper.GetString("data")
	if dataStr == "" {
		return fmt.Errorf("data is required")
	}

	// Validate JSON
	var data json.RawMessage
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		return errors.Wrap(err, "invalid JSON data")
	}

	method := webhook.MethodType(viper.GetString("method"))
	dataType := webhook.DataType(viper.GetString("data-type"))

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	err = handler.SendWebhookToURI(context.Background(), customerID, uri, method, dataType, data)
	if err != nil {
		return errors.Wrap(err, "failed to send webhook to URI")
	}

	response := map[string]any{
		"status":      "success",
		"customer-id": customerID.String(),
		"uri":         uri,
		"method":      method,
		"data-type":   dataType,
		"message":     "Webhook sent to URI successfully",
	}

	return printJSON(response)
}
