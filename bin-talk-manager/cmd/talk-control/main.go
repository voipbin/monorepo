package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/internal/config"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/pkg/chathandler"
	"monorepo/bin-talk-manager/pkg/dbhandler"
	"monorepo/bin-talk-manager/pkg/messagehandler"
	"monorepo/bin-talk-manager/pkg/participanthandler"
	"monorepo/bin-talk-manager/pkg/reactionhandler"
)

const serviceName = "talk-manager"

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandlers() (chathandler.ChatHandler, messagehandler.MessageHandler, participanthandler.ParticipantHandler, reactionhandler.ReactionHandler, error) {
	db, err := databasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "could not connect to the database")
	}

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Get().RedisAddress,
		Password: config.Get().RedisPassword,
		DB:       config.Get().RedisDatabase,
	})

	utilHandler := utilhandler.NewUtilHandler()
	dbHandler := dbhandler.New(db, redisClient, utilHandler)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, nil, "", serviceName)

	participantHandler := participanthandler.New(dbHandler, sockHandler, notifyHandler, utilHandler)
	chatHandler := chathandler.New(dbHandler, participantHandler, notifyHandler, utilHandler)
	messageHandler := messagehandler.New(dbHandler, sockHandler, notifyHandler, utilHandler)
	reactionHandler := reactionhandler.New(dbHandler, sockHandler, notifyHandler, utilHandler)

	return chatHandler, messageHandler, participantHandler, reactionHandler, nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "talk-control",
		Short: "Voipbin Talk Management CLI",
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

	cmdRoot.AddCommand(cmdChat())
	cmdRoot.AddCommand(cmdMessage())
	cmdRoot.AddCommand(cmdParticipant())
	cmdRoot.AddCommand(cmdReaction())

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

// ============ Chat Commands ============

func cmdChat() *cobra.Command {
	cmd := &cobra.Command{Use: "chat", Short: "Chat operations"}
	cmd.AddCommand(cmdChatCreate())
	cmd.AddCommand(cmdChatGet())
	cmd.AddCommand(cmdChatList())
	cmd.AddCommand(cmdChatUpdate())
	cmd.AddCommand(cmdChatDelete())
	return cmd
}

func cmdChatCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new chat",
		RunE:  runChatCreate,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID (required)")
	flags.String("type", "", "Chat type: direct, group, talk (required)")
	flags.String("name", "", "Chat name")
	flags.String("detail", "", "Chat description")
	flags.String("creator_type", "", "Creator type (required)")
	flags.String("creator_id", "", "Creator ID (required)")

	return cmd
}

func runChatCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	chatType := viper.GetString("type")
	if chatType == "" {
		return fmt.Errorf("chat type is required")
	}

	creatorType := viper.GetString("creator_type")
	if creatorType == "" {
		return fmt.Errorf("creator type is required")
	}

	creatorID, err := resolveUUID("creator_id", "Creator ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve creator ID")
	}

	chatHandler, _, _, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := chatHandler.ChatCreate(
		context.Background(),
		customerID,
		chat.Type(chatType),
		viper.GetString("name"),
		viper.GetString("detail"),
		creatorType,
		creatorID,
		[]participant.ParticipantInput{},
	)
	if err != nil {
		return errors.Wrap(err, "failed to create chat")
	}

	return printJSON(res)
}

func cmdChatGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a chat by ID",
		RunE:  runChatGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Chat ID (required)")

	return cmd
}

func runChatGet(cmd *cobra.Command, args []string) error {
	chatHandler, _, _, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	chatID, err := resolveUUID("id", "Chat ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve chat ID")
	}

	res, err := chatHandler.ChatGet(context.Background(), chatID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve chat")
	}

	return printJSON(res)
}

func cmdChatList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List chats",
		RunE:  runChatList,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID to filter (required)")
	flags.Uint64("size", 100, "Number of chats to retrieve")
	flags.String("token", "", "Pagination token")

	return cmd
}

func runChatList(cmd *cobra.Command, args []string) error {
	chatHandler, _, _, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	filters := map[chat.Field]any{
		chat.FieldCustomerID: customerID,
	}

	size := viper.GetUint64("size")
	token := viper.GetString("token")

	res, err := chatHandler.ChatList(context.Background(), filters, token, size)
	if err != nil {
		return errors.Wrap(err, "failed to list chats")
	}

	return printJSON(res)
}

func cmdChatUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a chat",
		RunE:  runChatUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Chat ID (required)")
	flags.String("name", "", "New chat name")
	flags.String("detail", "", "New chat description")

	return cmd
}

func runChatUpdate(cmd *cobra.Command, args []string) error {
	chatHandler, _, _, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	chatID, err := resolveUUID("id", "Chat ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve chat ID")
	}

	var name *string
	var detail *string

	if viper.IsSet("name") {
		n := viper.GetString("name")
		name = &n
	}

	if viper.IsSet("detail") {
		d := viper.GetString("detail")
		detail = &d
	}

	res, err := chatHandler.ChatUpdate(context.Background(), chatID, name, detail)
	if err != nil {
		return errors.Wrap(err, "failed to update chat")
	}

	return printJSON(res)
}

func cmdChatDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a chat",
		RunE:  runChatDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Chat ID (required)")

	return cmd
}

func runChatDelete(cmd *cobra.Command, args []string) error {
	chatHandler, _, _, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	chatID, err := resolveUUID("id", "Chat ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve chat ID")
	}

	res, err := chatHandler.ChatDelete(context.Background(), chatID)
	if err != nil {
		return errors.Wrap(err, "failed to delete chat")
	}

	return printJSON(res)
}

// ============ Message Commands ============

func cmdMessage() *cobra.Command {
	cmd := &cobra.Command{Use: "message", Short: "Message operations"}
	cmd.AddCommand(cmdMessageCreate())
	cmd.AddCommand(cmdMessageGet())
	cmd.AddCommand(cmdMessageList())
	cmd.AddCommand(cmdMessageDelete())
	return cmd
}

func cmdMessageCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new message",
		RunE:  runMessageCreate,
	}

	flags := cmd.Flags()
	flags.String("chat_id", "", "Chat ID (required)")
	flags.String("parent_id", "", "Parent message ID (optional, for threaded replies)")
	flags.String("owner_type", "", "Owner type (required)")
	flags.String("owner_id", "", "Owner ID (required)")
	flags.String("type", "", "Message type (required)")
	flags.String("text", "", "Message text (required)")

	return cmd
}

func runMessageCreate(cmd *cobra.Command, args []string) error {
	_, messageHandler, _, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	chatID, err := resolveUUID("chat_id", "Chat ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve chat ID")
	}

	ownerType := viper.GetString("owner_type")
	if ownerType == "" {
		return fmt.Errorf("owner type is required")
	}

	ownerID, err := resolveUUID("owner_id", "Owner ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve owner ID")
	}

	msgType := viper.GetString("type")
	if msgType == "" {
		return fmt.Errorf("message type is required")
	}

	text := viper.GetString("text")
	if text == "" {
		return fmt.Errorf("message text is required")
	}

	var parentID *uuid.UUID
	if viper.IsSet("parent_id") {
		pid, err := resolveUUID("parent_id", "Parent ID")
		if err != nil {
			return errors.Wrap(err, "failed to resolve parent ID")
		}
		parentID = &pid
	}

	req := messagehandler.MessageCreateRequest{
		ChatID:    chatID,
		ParentID:  parentID,
		OwnerType: ownerType,
		OwnerID:   ownerID,
		Type:      msgType,
		Text:      text,
		Medias:    []message.Media{},
	}

	res, err := messageHandler.MessageCreate(context.Background(), req)
	if err != nil {
		return errors.Wrap(err, "failed to create message")
	}

	return printJSON(res)
}

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
	_, messageHandler, _, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	messageID, err := resolveUUID("id", "Message ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve message ID")
	}

	res, err := messageHandler.MessageGet(context.Background(), messageID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve message")
	}

	return printJSON(res)
}

func cmdMessageList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messages",
		RunE:  runMessageList,
	}

	flags := cmd.Flags()
	flags.String("chat_id", "", "Chat ID to filter (required)")
	flags.Uint64("size", 100, "Number of messages to retrieve")
	flags.String("token", "", "Pagination token")

	return cmd
}

func runMessageList(cmd *cobra.Command, args []string) error {
	_, messageHandler, _, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	chatID, err := resolveUUID("chat_id", "Chat ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve chat ID")
	}

	filters := map[message.Field]any{
		message.FieldChatID: chatID,
	}

	size := viper.GetUint64("size")
	token := viper.GetString("token")

	res, err := messageHandler.MessageList(context.Background(), filters, token, size)
	if err != nil {
		return errors.Wrap(err, "failed to list messages")
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
	_, messageHandler, _, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	messageID, err := resolveUUID("id", "Message ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve message ID")
	}

	res, err := messageHandler.MessageDelete(context.Background(), messageID)
	if err != nil {
		return errors.Wrap(err, "failed to delete message")
	}

	return printJSON(res)
}

// ============ Participant Commands ============

func cmdParticipant() *cobra.Command {
	cmd := &cobra.Command{Use: "participant", Short: "Participant operations"}
	cmd.AddCommand(cmdParticipantAdd())
	cmd.AddCommand(cmdParticipantList())
	cmd.AddCommand(cmdParticipantRemove())
	return cmd
}

func cmdParticipantAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a participant to a chat",
		RunE:  runParticipantAdd,
	}

	flags := cmd.Flags()
	flags.String("chat_id", "", "Chat ID (required)")
	flags.String("owner_type", "", "Owner type (required)")
	flags.String("owner_id", "", "Owner ID (required)")

	return cmd
}

func runParticipantAdd(cmd *cobra.Command, args []string) error {
	_, _, participantHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	chatID, err := resolveUUID("chat_id", "Chat ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve chat ID")
	}

	ownerType := viper.GetString("owner_type")
	if ownerType == "" {
		return fmt.Errorf("owner type is required")
	}

	ownerID, err := resolveUUID("owner_id", "Owner ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve owner ID")
	}

	res, err := participantHandler.ParticipantAdd(context.Background(), chatID, ownerID, ownerType)
	if err != nil {
		return errors.Wrap(err, "failed to add participant")
	}

	return printJSON(res)
}

func cmdParticipantList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List participants in a chat",
		RunE:  runParticipantList,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID (required)")
	flags.String("chat_id", "", "Chat ID (required)")

	return cmd
}

func runParticipantList(cmd *cobra.Command, args []string) error {
	_, _, participantHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	chatID, err := resolveUUID("chat_id", "Chat ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve chat ID")
	}

	res, err := participantHandler.ParticipantList(context.Background(), customerID, chatID)
	if err != nil {
		return errors.Wrap(err, "failed to list participants")
	}

	return printJSON(res)
}

func cmdParticipantRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a participant from a chat",
		RunE:  runParticipantRemove,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID (required)")
	flags.String("participant_id", "", "Participant ID (required)")

	return cmd
}

func runParticipantRemove(cmd *cobra.Command, args []string) error {
	_, _, participantHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	participantID, err := resolveUUID("participant_id", "Participant ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve participant ID")
	}

	err = participantHandler.ParticipantRemove(context.Background(), customerID, participantID)
	if err != nil {
		return errors.Wrap(err, "failed to remove participant")
	}

	fmt.Println(`{"status": "success", "message": "Participant removed"}`)
	return nil
}

// ============ Reaction Commands ============

func cmdReaction() *cobra.Command {
	cmd := &cobra.Command{Use: "reaction", Short: "Reaction operations"}
	cmd.AddCommand(cmdReactionAdd())
	cmd.AddCommand(cmdReactionRemove())
	return cmd
}

func cmdReactionAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a reaction to a message",
		RunE:  runReactionAdd,
	}

	flags := cmd.Flags()
	flags.String("message_id", "", "Message ID (required)")
	flags.String("emoji", "", "Emoji (required)")
	flags.String("owner_type", "", "Owner type (required)")
	flags.String("owner_id", "", "Owner ID (required)")

	return cmd
}

func runReactionAdd(cmd *cobra.Command, args []string) error {
	_, _, _, reactionHandler, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	messageID, err := resolveUUID("message_id", "Message ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve message ID")
	}

	emoji := viper.GetString("emoji")
	if emoji == "" {
		return fmt.Errorf("emoji is required")
	}

	ownerType := viper.GetString("owner_type")
	if ownerType == "" {
		return fmt.Errorf("owner type is required")
	}

	ownerID, err := resolveUUID("owner_id", "Owner ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve owner ID")
	}

	res, err := reactionHandler.ReactionAdd(context.Background(), messageID, emoji, ownerType, ownerID)
	if err != nil {
		return errors.Wrap(err, "failed to add reaction")
	}

	return printJSON(res)
}

func cmdReactionRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a reaction from a message",
		RunE:  runReactionRemove,
	}

	flags := cmd.Flags()
	flags.String("message_id", "", "Message ID (required)")
	flags.String("emoji", "", "Emoji (required)")
	flags.String("owner_type", "", "Owner type (required)")
	flags.String("owner_id", "", "Owner ID (required)")

	return cmd
}

func runReactionRemove(cmd *cobra.Command, args []string) error {
	_, _, _, reactionHandler, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	messageID, err := resolveUUID("message_id", "Message ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve message ID")
	}

	emoji := viper.GetString("emoji")
	if emoji == "" {
		return fmt.Errorf("emoji is required")
	}

	ownerType := viper.GetString("owner_type")
	if ownerType == "" {
		return fmt.Errorf("owner type is required")
	}

	ownerID, err := resolveUUID("owner_id", "Owner ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve owner ID")
	}

	res, err := reactionHandler.ReactionRemove(context.Background(), messageID, emoji, ownerType, ownerID)
	if err != nil {
		return errors.Wrap(err, "failed to remove reaction")
	}

	return printJSON(res)
}
