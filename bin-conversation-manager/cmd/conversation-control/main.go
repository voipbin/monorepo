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
	"monorepo/bin-conversation-manager/internal/config"
	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
	"monorepo/bin-conversation-manager/pkg/smshandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = "conversation-manager"

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandlers() (conversationhandler.ConversationHandler, accounthandler.AccountHandler, messagehandler.MessageHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "could not initialize the cache")
	}

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, "bin-manager.conversation-manager.event", serviceName, "")

	return initConversationHandlers(db, cache, reqHandler, notifyHandler)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initConversationHandlers(sqlDB *sql.DB, cache cachehandler.CacheHandler, reqHandler requesthandler.RequestHandler, notifyHandler notifyhandler.NotifyHandler) (conversationhandler.ConversationHandler, accounthandler.AccountHandler, messagehandler.MessageHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)

	lineHandler := linehandler.NewLineHandler(reqHandler)
	accountHandler := accounthandler.NewAccountHandler(db, reqHandler, notifyHandler, lineHandler)
	smsHandler := smshandler.NewSMSHandler(reqHandler, accountHandler)
	messageHandler := messagehandler.NewMessageHandler(db, notifyHandler, accountHandler, lineHandler, smsHandler)
	conversationHandler := conversationhandler.NewConversationHandler(db, notifyHandler, reqHandler, accountHandler, messageHandler, lineHandler, smsHandler)

	return conversationHandler, accountHandler, messageHandler, nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "conversation-control",
		Short: "Voipbin Conversation Management CLI",
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

	cmdConversation := &cobra.Command{Use: "conversation", Short: "Conversation operations"}
	cmdConversation.AddCommand(cmdConversationGet())
	cmdConversation.AddCommand(cmdConversationList())

	cmdAccount := &cobra.Command{Use: "account", Short: "Account operations"}
	cmdAccount.AddCommand(cmdAccountCreate())
	cmdAccount.AddCommand(cmdAccountGet())
	cmdAccount.AddCommand(cmdAccountList())
	cmdAccount.AddCommand(cmdAccountUpdate())
	cmdAccount.AddCommand(cmdAccountDelete())

	cmdMessage := &cobra.Command{Use: "message", Short: "Message operations"}
	cmdMessage.AddCommand(cmdMessageGet())
	cmdMessage.AddCommand(cmdMessageList())
	cmdMessage.AddCommand(cmdMessageDelete())

	cmdRoot.AddCommand(cmdConversation)
	cmdRoot.AddCommand(cmdAccount)
	cmdRoot.AddCommand(cmdMessage)

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

// Conversation commands

func cmdConversationGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a conversation by ID",
		RunE:  runConversationGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Conversation ID (required)")

	return cmd
}

func runConversationGet(cmd *cobra.Command, args []string) error {
	conversationHandler, _, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Conversation ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve conversation ID")
	}

	res, err := conversationHandler.Get(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve conversation")
	}

	return printJSON(res)
}

func cmdConversationList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get conversation list",
		RunE:  runConversationList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of conversations to retrieve")
	flags.String("token", "", "Retrieve conversations before this token (pagination)")
	flags.String("customer-id", "", "Customer ID to filter (required)")
	flags.String("type", "", "Conversation type to filter (message, line)")

	return cmd
}

func runConversationList(cmd *cobra.Command, args []string) error {
	conversationHandler, _, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")
	conversationType := viper.GetString("type")

	filters := map[conversation.Field]any{
		conversation.FieldCustomerID: customerID,
		conversation.FieldDeleted:    false,
	}

	if conversationType != "" {
		filters[conversation.FieldType] = conversation.Type(conversationType)
	}

	res, err := conversationHandler.List(context.Background(), token, uint64(limit), filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve conversations")
	}

	return printJSON(res)
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
	flags.String("type", "", "Account type: line, sms (required)")
	flags.String("name", "", "Account name")
	flags.String("detail", "", "Account description")
	flags.String("secret", "", "Account secret (required)")
	flags.String("token", "", "Account token (required)")

	return cmd
}

func runAccountCreate(cmd *cobra.Command, args []string) error {
	_, accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	accountType := viper.GetString("type")
	if accountType == "" {
		return fmt.Errorf("type is required")
	}

	secret := viper.GetString("secret")
	if secret == "" {
		return fmt.Errorf("secret is required")
	}

	token := viper.GetString("token")
	if token == "" {
		return fmt.Errorf("token is required")
	}

	res, err := accountHandler.Create(
		context.Background(),
		customerID,
		account.Type(accountType),
		viper.GetString("name"),
		viper.GetString("detail"),
		secret,
		token,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create account")
	}

	return printJSON(res)
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

func runAccountGet(cmd *cobra.Command, args []string) error {
	_, accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve account ID")
	}

	res, err := accountHandler.Get(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve account")
	}

	return printJSON(res)
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
	flags.String("customer-id", "", "Customer ID to filter (required)")
	flags.String("type", "", "Account type to filter (line, sms)")

	return cmd
}

func runAccountList(cmd *cobra.Command, args []string) error {
	_, accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")
	accountType := viper.GetString("type")

	filters := map[account.Field]any{
		account.FieldCustomerID: customerID,
		account.FieldDeleted:    false,
	}

	if accountType != "" {
		filters[account.FieldType] = account.Type(accountType)
	}

	res, err := accountHandler.List(context.Background(), token, uint64(limit), filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve accounts")
	}

	return printJSON(res)
}

func cmdAccountUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an account",
		RunE:  runAccountUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Account ID (required)")
	flags.String("name", "", "Account name")
	flags.String("detail", "", "Account description")
	flags.String("secret", "", "Account secret")
	flags.String("token", "", "Account token")

	return cmd
}

func runAccountUpdate(cmd *cobra.Command, args []string) error {
	_, accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve account ID")
	}

	fields := make(map[account.Field]any)

	if viper.IsSet("name") {
		fields[account.FieldName] = viper.GetString("name")
	}
	if viper.IsSet("detail") {
		fields[account.FieldDetail] = viper.GetString("detail")
	}
	if viper.IsSet("secret") {
		fields[account.FieldSecret] = viper.GetString("secret")
	}
	if viper.IsSet("token") {
		fields[account.FieldToken] = viper.GetString("token")
	}

	if len(fields) == 0 {
		return fmt.Errorf("at least one field to update is required (name, detail, secret, token)")
	}

	res, err := accountHandler.Update(context.Background(), id, fields)
	if err != nil {
		return errors.Wrap(err, "failed to update account")
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

func runAccountDelete(cmd *cobra.Command, args []string) error {
	_, accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve account ID")
	}

	res, err := accountHandler.Delete(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to delete account")
	}

	return printJSON(res)
}

// Message commands

func cmdMessageGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a message by ID",
		RunE:  runMessageGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Message ID (required)")

	return cmd
}

func runMessageGet(cmd *cobra.Command, args []string) error {
	_, _, messageHandler, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Message ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve message ID")
	}

	res, err := messageHandler.Get(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve message")
	}

	return printJSON(res)
}

func cmdMessageList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get message list",
		RunE:  runMessageList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of messages to retrieve")
	flags.String("token", "", "Retrieve messages before this token (pagination)")
	flags.String("customer-id", "", "Customer ID to filter (required)")
	flags.String("conversation-id", "", "Conversation ID to filter")
	flags.String("direction", "", "Message direction to filter (incoming, outgoing)")
	flags.String("status", "", "Message status to filter (failed, progressing, done)")

	return cmd
}

func runMessageList(cmd *cobra.Command, args []string) error {
	_, _, messageHandler, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")
	conversationIDStr := viper.GetString("conversation-id")
	direction := viper.GetString("direction")
	status := viper.GetString("status")

	filters := map[message.Field]any{
		message.FieldCustomerID: customerID,
		message.FieldDeleted:    false,
	}

	if conversationIDStr != "" {
		conversationID := uuid.FromStringOrNil(conversationIDStr)
		if conversationID == uuid.Nil {
			return fmt.Errorf("invalid conversation-id format")
		}
		filters[message.FieldConversationID] = conversationID
	}

	if direction != "" {
		filters[message.FieldDirection] = message.Direction(direction)
	}

	if status != "" {
		filters[message.FieldStatus] = message.Status(status)
	}

	res, err := messageHandler.List(context.Background(), token, uint64(limit), filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve messages")
	}

	return printJSON(res)
}

func cmdMessageDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a message",
		RunE:  runMessageDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Message ID (required)")

	return cmd
}

func runMessageDelete(cmd *cobra.Command, args []string) error {
	_, _, messageHandler, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Message ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve message ID")
	}

	res, err := messageHandler.Delete(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to delete message")
	}

	return printJSON(res)
}
