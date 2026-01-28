package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-pipecat-manager/internal/config"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/cachehandler"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = outline.ServiceNamePipecatManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (pipecatcallhandler.PipecatcallHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initPipecatcallHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initPipecatcallHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (pipecatcallhandler.PipecatcallHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, outline.QueueNamePipecatEvent, serviceName)

	return pipecatcallhandler.NewPipecatcallHandler(reqHandler, notifyHandler, db, "localhost:0", "cli-host"), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "pipecat-control",
		Short: "Voipbin Pipecat Management CLI",
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

	cmdSub := &cobra.Command{Use: "pipecatcall", Short: "Pipecatcall operation"}
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdStart())
	cmdSub.AddCommand(cmdTerminate())
	cmdSub.AddCommand(cmdSendMessage())

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

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a pipecatcall by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Pipecatcall ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Pipecatcall ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve pipecatcall ID")
	}

	res, err := handler.Get(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve pipecatcall")
	}

	return printJSON(res)
}

func cmdStart() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new pipecatcall",
		RunE:  runStart,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Pipecatcall ID (required)")
	flags.String("customer_id", "", "Customer ID (required)")
	flags.String("activeflow_id", "", "Activeflow ID (required)")
	flags.String("reference_type", "", "Reference type: 'call' or 'ai_call' (required)")
	flags.String("reference_id", "", "Reference ID (required)")
	flags.String("llm_type", "", "LLM type (e.g., 'openai.gpt-4') (required)")
	flags.String("stt_type", "", "STT type: 'deepgram' or empty")
	flags.String("stt_language", "", "STT language code (e.g., 'en')")
	flags.String("tts_type", "", "TTS type: 'cartesia', 'elevenlabs', or empty")
	flags.String("tts_language", "", "TTS language code (e.g., 'en')")
	flags.String("tts_voice_id", "", "TTS voice ID")

	return cmd
}

func runStart(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Pipecatcall ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve pipecatcall ID")
	}

	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	activeflowID, err := resolveUUID("activeflow_id", "Activeflow ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve activeflow ID")
	}

	referenceTypeStr := viper.GetString("reference_type")
	if referenceTypeStr == "" {
		return fmt.Errorf("reference_type is required")
	}
	referenceType := pipecatcall.ReferenceType(referenceTypeStr)

	referenceID, err := resolveUUID("reference_id", "Reference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve reference ID")
	}

	llmTypeStr := viper.GetString("llm_type")
	if llmTypeStr == "" {
		return fmt.Errorf("llm_type is required")
	}
	llmType := pipecatcall.LLMType(llmTypeStr)

	sttType := pipecatcall.STTType(viper.GetString("stt_type"))
	sttLanguage := viper.GetString("stt_language")
	ttsType := pipecatcall.TTSType(viper.GetString("tts_type"))
	ttsLanguage := viper.GetString("tts_language")
	ttsVoiceID := viper.GetString("tts_voice_id")

	res, err := handler.Start(
		context.Background(),
		id,
		customerID,
		activeflowID,
		referenceType,
		referenceID,
		llmType,
		[]map[string]any{}, // Empty llm_messages for CLI
		sttType,
		sttLanguage,
		ttsType,
		ttsLanguage,
		ttsVoiceID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to start pipecatcall")
	}

	return printJSON(res)
}

func cmdTerminate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "terminate",
		Short: "Terminate a pipecatcall",
		RunE:  runTerminate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Pipecatcall ID (required)")

	return cmd
}

func runTerminate(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Pipecatcall ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve pipecatcall ID")
	}

	res, err := handler.Terminate(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to terminate pipecatcall")
	}

	return printJSON(res)
}

func cmdSendMessage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-message",
		Short: "Send a message to a pipecatcall",
		RunE:  runSendMessage,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Pipecatcall ID (required)")
	flags.String("message_id", "", "Message ID (required)")
	flags.String("message_text", "", "Message text (required)")
	flags.Bool("run_immediately", false, "Run message immediately")
	flags.Bool("audio_response", false, "Request audio response")

	return cmd
}

func runSendMessage(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Pipecatcall ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve pipecatcall ID")
	}

	messageID := viper.GetString("message_id")
	if messageID == "" {
		return fmt.Errorf("message_id is required")
	}

	messageText := viper.GetString("message_text")
	if messageText == "" {
		return fmt.Errorf("message_text is required")
	}

	runImmediately := viper.GetBool("run_immediately")
	audioResponse := viper.GetBool("audio_response")

	res, err := handler.SendMessage(context.Background(), id, messageID, messageText, runImmediately, audioResponse)
	if err != nil {
		return errors.Wrap(err, "failed to send message")
	}

	return printJSON(res)
}
