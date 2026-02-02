package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-common-handler/models/address"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-message-manager/internal/config"
	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/pkg/cachehandler"
	"monorepo/bin-message-manager/pkg/dbhandler"
	"monorepo/bin-message-manager/pkg/messagehandler"
	"monorepo/bin-message-manager/pkg/requestexternal"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = commonoutline.ServiceNameMessageManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (messagehandler.MessageHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initMessageHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initMessageHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (messagehandler.MessageHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameMessageEvent, serviceName, "")
	requestExternal := requestexternal.NewRequestExternal(config.Get().AuthtokenTelnyx, config.Get().AuthtokenMessagebird)

	return messagehandler.NewMessageHandler(reqHandler, notifyHandler, db, requestExternal), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "message-control",
		Short: "Voipbin Message Management CLI",
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

	cmdSub := &cobra.Command{Use: "message", Short: "Message operation"}
	cmdSub.AddCommand(cmdSend())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdList())
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

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSON")
	}
	fmt.Println(string(data))
	return nil
}

func cmdSend() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a new message",
		RunE:  runSend,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("source", "", "Source phone number in E.164 format (required)")
	flags.StringSlice("destinations", []string{}, "Destination phone numbers in E.164 format, comma-separated (required)")
	flags.String("text", "", "Message text (required)")

	return cmd
}

func runSend(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	sourceStr := viper.GetString("source")
	if sourceStr == "" {
		return fmt.Errorf("source is required")
	}
	sourceAddr := address.Address{
		Target: sourceStr,
		Type:   address.TypeTel,
	}

	destinationStrs := viper.GetStringSlice("destinations")
	if len(destinationStrs) == 0 {
		return fmt.Errorf("destinations is required")
	}
	var destinations []address.Address
	for _, dest := range destinationStrs {
		destinations = append(destinations, address.Address{
			Target: dest,
			Type:   address.TypeTel,
		})
	}

	text := viper.GetString("text")
	if text == "" {
		return fmt.Errorf("text is required")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	// Generate new message ID
	messageID := uuid.Must(uuid.NewV4())

	res, err := handler.Send(
		context.Background(),
		messageID,
		customerID,
		&sourceAddr,
		destinations,
		text,
	)
	if err != nil {
		return errors.Wrap(err, "failed to send message")
	}

	return printJSON(res)
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a message by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Message ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	messageID, err := resolveUUID("id", "Message ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve message ID")
	}

	res, err := handler.Get(context.Background(), messageID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve message")
	}

	return printJSON(res)
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get message list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of messages to retrieve")
	flags.String("token", "", "Retrieve messages before this token (pagination)")
	flags.String("customer-id", "", "Customer ID to filter (required)")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")

	filters := map[message.Field]any{
		message.FieldCustomerID: customerID,
		message.FieldDeleted:    false,
	}

	res, err := handler.List(context.Background(), token, uint64(limit), filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve messages")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a message",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Message ID (required)")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Message ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve message ID")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete message")
	}

	return printJSON(res)
}
