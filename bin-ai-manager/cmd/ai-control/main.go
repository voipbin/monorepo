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

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"

	_ "github.com/go-sql-driver/mysql"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = commonoutline.ServiceNameAIManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "ai-control",
		Short: "Voipbin AI Management CLI",
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

	// AI subcommands
	cmdAI := &cobra.Command{Use: "ai", Short: "AI operations"}
	cmdAI.AddCommand(cmdCreate())
	cmdAI.AddCommand(cmdGet())
	cmdAI.AddCommand(cmdList())
	cmdAI.AddCommand(cmdUpdate())
	cmdAI.AddCommand(cmdDelete())

	cmdRoot.AddCommand(cmdAI)
	return cmdRoot
}

// AI commands

func cmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new AI configuration",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("name", "", "AI name")
	flags.String("detail", "", "AI detail")
	flags.String("engine-type", "", "Engine type (deprecated, use engine-model)")
	flags.String("engine-model", "openai.gpt-4o", "Engine model (e.g., openai.gpt-4o, dialogflow.cx)")
	flags.String("engine-key", "", "Engine API key")
	flags.String("init-prompt", "", "Initial system prompt")
	flags.String("tts-type", "", "TTS type (e.g., openai, elevenlabs, cartesia)")
	flags.String("tts-voice-id", "", "TTS voice ID")
	flags.String("stt-type", "", "STT type (e.g., deepgram, elevenlabs)")

	return cmd
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an AI configuration by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "AI ID (required)")

	return cmd
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get AI configuration list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of AI configurations to retrieve")
	flags.String("token", "", "Retrieve AI configurations before this token (pagination)")
	flags.String("customer-id", "", "Filter by customer ID (required)")

	return cmd
}

func cmdUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an AI configuration",
		RunE:  runUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "AI ID (required)")
	flags.String("name", "", "AI name")
	flags.String("detail", "", "AI detail")
	flags.String("engine-type", "", "Engine type (deprecated, use engine-model)")
	flags.String("engine-model", "", "Engine model (e.g., openai.gpt-4o, dialogflow.cx)")
	flags.String("engine-key", "", "Engine API key")
	flags.String("init-prompt", "", "Initial system prompt")
	flags.String("tts-type", "", "TTS type (e.g., openai, elevenlabs, cartesia)")
	flags.String("tts-voice-id", "", "TTS voice ID")
	flags.String("stt-type", "", "STT type (e.g., deepgram, elevenlabs)")

	return cmd
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an AI configuration",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "AI ID (required)")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID format")
	}

	name := viper.GetString("name")
	detail := viper.GetString("detail")
	engineType := ai.EngineType(viper.GetString("engine-type"))
	engineModel := ai.EngineModel(viper.GetString("engine-model"))
	engineKey := viper.GetString("engine-key")
	initPrompt := viper.GetString("init-prompt")
	ttsType := ai.TTSType(viper.GetString("tts-type"))
	ttsVoiceID := viper.GetString("tts-voice-id")
	sttType := ai.STTType(viper.GetString("stt-type"))

	// Validate engine model
	if engineModel != "" && !ai.IsValidEngineModel(engineModel) {
		return fmt.Errorf("invalid engine model: %s", engineModel)
	}

	res, err := handler.Create(
		context.Background(),
		customerID,
		name,
		detail,
		engineType,
		engineModel,
		map[string]any{}, // engineData - empty for now
		engineKey,
		initPrompt,
		ttsType,
		ttsVoiceID,
		sttType,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create AI")
	}

	return printJSON(res)
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "AI ID")
	if err != nil {
		return errors.Wrap(err, "invalid AI ID format")
	}

	res, err := handler.Get(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve AI")
	}

	return printJSON(res)
}

func runList(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID format")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")

	filters := map[ai.Field]any{
		ai.FieldCustomerID: customerID,
		ai.FieldDeleted:    false,
	}

	res, err := handler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve AI configurations")
	}

	return printJSON(res)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "AI ID")
	if err != nil {
		return errors.Wrap(err, "invalid AI ID format")
	}

	name := viper.GetString("name")
	detail := viper.GetString("detail")
	engineType := ai.EngineType(viper.GetString("engine-type"))
	engineModel := ai.EngineModel(viper.GetString("engine-model"))
	engineKey := viper.GetString("engine-key")
	initPrompt := viper.GetString("init-prompt")
	ttsType := ai.TTSType(viper.GetString("tts-type"))
	ttsVoiceID := viper.GetString("tts-voice-id")
	sttType := ai.STTType(viper.GetString("stt-type"))

	// Validate engine model if provided
	if engineModel != "" && !ai.IsValidEngineModel(engineModel) {
		return fmt.Errorf("invalid engine model: %s", engineModel)
	}

	res, err := handler.Update(
		context.Background(),
		targetID,
		name,
		detail,
		engineType,
		engineModel,
		map[string]any{}, // engineData - empty for now
		engineKey,
		initPrompt,
		ttsType,
		ttsVoiceID,
		sttType,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update AI")
	}

	return printJSON(res)
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "AI ID")
	if err != nil {
		return errors.Wrap(err, "invalid AI ID format")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete AI")
	}

	return printJSON(res)
}

// Handler initialization

func initHandler() (aihandler.AIHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, err
	}

	return initAIHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if err := res.Connect(); err != nil {
		return nil, err
	}
	return res, nil
}

func initAIHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (aihandler.AIHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameAIEvent, serviceName)

	return aihandler.NewAIHandler(reqHandler, notifyHandler, db), nil
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
