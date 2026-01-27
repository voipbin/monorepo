package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"monorepo/bin-chat-manager/internal/config"
	"monorepo/bin-chat-manager/models/chat"
	"monorepo/bin-chat-manager/pkg/cachehandler"
	"monorepo/bin-chat-manager/pkg/chathandler"
	"monorepo/bin-chat-manager/pkg/chatroomhandler"
	"monorepo/bin-chat-manager/pkg/dbhandler"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = commonoutline.ServiceNameChatManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (chathandler.ChatHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initChatHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initChatHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (chathandler.ChatHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameChatEvent, serviceName)

	chatroomHandler := chatroomhandler.NewChatroomHandler(db, reqHandler, notifyHandler)

	return chathandler.NewChatHandler(db, reqHandler, notifyHandler, chatroomHandler), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "chat-control",
		Short: "Voipbin Chat Management CLI",
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

	cmdSub := &cobra.Command{Use: "chat", Short: "Chat operation"}
	cmdSub.AddCommand(cmdCreate())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdList())
	cmdSub.AddCommand(cmdUpdateBasicInfo())
	cmdSub.AddCommand(cmdUpdateRoomOwner())
	cmdSub.AddCommand(cmdAddParticipant())
	cmdSub.AddCommand(cmdRemoveParticipant())
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

func resolveUUIDs(flagName string, label string) ([]uuid.UUID, error) {
	val := viper.GetString(flagName)
	if val == "" {
		return []uuid.UUID{}, nil
	}

	parts := strings.Split(val, ",")
	result := make([]uuid.UUID, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		id := uuid.FromStringOrNil(trimmed)
		if id == uuid.Nil {
			return nil, fmt.Errorf("invalid UUID in %s: '%s'", label, trimmed)
		}
		result = append(result, id)
	}

	return result, nil
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSON")
	}
	fmt.Println(string(data))
	return nil
}

func cmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new chat",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID (required)")
	flags.String("type", "normal", "Chat type: normal (1:1) or group")
	flags.String("owner_id", "", "Room owner ID (required)")
	flags.String("participant_ids", "", "Comma-separated list of participant IDs (required)")
	flags.String("name", "", "Chat name")
	flags.String("detail", "", "Chat description")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	ownerID, err := resolveUUID("owner_id", "Room Owner ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve room owner ID")
	}

	participantIDs, err := resolveUUIDs("participant_ids", "Participant IDs")
	if err != nil {
		return errors.Wrap(err, "failed to resolve participant IDs")
	}

	if len(participantIDs) == 0 {
		return fmt.Errorf("at least one participant ID is required")
	}

	chatType := chat.Type(viper.GetString("type"))
	if chatType != chat.TypeNormal && chatType != chat.TypeGroup {
		return fmt.Errorf("invalid chat type: must be 'normal' or 'group'")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(
		context.Background(),
		customerID,
		chatType,
		ownerID,
		participantIDs,
		viper.GetString("name"),
		viper.GetString("detail"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create chat")
	}

	return printJSON(res)
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a chat by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Chat ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	chatID, err := resolveUUID("id", "Chat ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve chat ID")
	}

	res, err := handler.Get(context.Background(), chatID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve chat")
	}

	return printJSON(res)
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get chat list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Uint64("limit", 100, "Limit the number of chats to retrieve")
	flags.String("token", "", "Retrieve chats before this token (pagination)")
	flags.String("customer_id", "", "Customer ID to filter (required)")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	limit := viper.GetUint64("limit")
	token := viper.GetString("token")

	filters := map[chat.Field]any{
		chat.FieldCustomerID: customerID,
		chat.FieldDeleted:    false,
	}

	res, err := handler.List(context.Background(), token, limit, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve chats")
	}

	return printJSON(res)
}

func cmdUpdateBasicInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-basic-info",
		Short: "Update chat name and detail",
		RunE:  runUpdateBasicInfo,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Chat ID (required)")
	flags.String("name", "", "New chat name")
	flags.String("detail", "", "New chat description")

	return cmd
}

func runUpdateBasicInfo(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Chat ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve chat ID")
	}

	res, err := handler.UpdateBasicInfo(
		context.Background(),
		id,
		viper.GetString("name"),
		viper.GetString("detail"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update chat basic info")
	}

	return printJSON(res)
}

func cmdUpdateRoomOwner() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-room-owner",
		Short: "Update chat room owner",
		RunE:  runUpdateRoomOwner,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Chat ID (required)")
	flags.String("owner_id", "", "New room owner ID (required)")

	return cmd
}

func runUpdateRoomOwner(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Chat ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve chat ID")
	}

	ownerID, err := resolveUUID("owner_id", "Room Owner ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve room owner ID")
	}

	res, err := handler.UpdateRoomOwnerID(context.Background(), id, ownerID)
	if err != nil {
		return errors.Wrap(err, "failed to update room owner")
	}

	return printJSON(res)
}

func cmdAddParticipant() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-participant",
		Short: "Add a participant to chat",
		RunE:  runAddParticipant,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Chat ID (required)")
	flags.String("participant_id", "", "Participant ID to add (required)")

	return cmd
}

func runAddParticipant(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Chat ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve chat ID")
	}

	participantID, err := resolveUUID("participant_id", "Participant ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve participant ID")
	}

	res, err := handler.AddParticipantID(context.Background(), id, participantID)
	if err != nil {
		return errors.Wrap(err, "failed to add participant")
	}

	return printJSON(res)
}

func cmdRemoveParticipant() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-participant",
		Short: "Remove a participant from chat",
		RunE:  runRemoveParticipant,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Chat ID (required)")
	flags.String("participant_id", "", "Participant ID to remove (required)")

	return cmd
}

func runRemoveParticipant(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Chat ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve chat ID")
	}

	participantID, err := resolveUUID("participant_id", "Participant ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve participant ID")
	}

	res, err := handler.RemoveParticipantID(context.Background(), id, participantID)
	if err != nil {
		return errors.Wrap(err, "failed to remove participant")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a chat",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Chat ID (required)")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Chat ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve chat ID")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete chat")
	}

	return printJSON(res)
}
