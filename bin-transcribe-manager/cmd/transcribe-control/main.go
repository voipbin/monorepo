package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-transcribe-manager/internal/config"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/cachehandler"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
	"monorepo/bin-transcribe-manager/pkg/transcribehandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
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

const serviceName = commonoutline.ServiceNameTranscribeManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (transcribehandler.TranscribeHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initTranscribeHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initTranscribeHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (transcribehandler.TranscribeHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameTranscribeEvent, serviceName)

	transcriptHandler := transcripthandler.NewTranscriptHandler(reqHandler, db, notifyHandler)

	// Note: CLI doesn't need a real listening address for streaming
	// Use a placeholder since we're not actually running the streaming server
	listenAddress := "127.0.0.1:8080"
	streamingHandler := streaminghandler.NewStreamingHandler(reqHandler, notifyHandler, transcriptHandler, listenAddress, config.Get().AWSAccessKey, config.Get().AWSSecretKey)
	if streamingHandler == nil {
		return nil, errors.New("failed to initialize streaming handler: no STT providers available")
	}

	hostID := uuid.Must(uuid.NewV4())
	return transcribehandler.NewTranscribeHandler(reqHandler, db, notifyHandler, transcriptHandler, streamingHandler, hostID), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "transcribe-control",
		Short: "Voipbin Transcribe Management CLI",
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

	cmdSub := &cobra.Command{Use: "transcribe", Short: "Transcribe operation"}
	cmdSub.AddCommand(cmdStart())
	cmdSub.AddCommand(cmdStop())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdGetByReferenceIDAndLanguage())
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

func cmdStart() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new transcription session",
		RunE:  runStart,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID (required)")
	flags.String("activeflow_id", "", "Active flow ID (required)")
	flags.String("on_end_flow_id", "", "On end flow ID (required)")
	flags.String("reference_type", "", "Reference type: call, confbridge, recording (required)")
	flags.String("reference_id", "", "Reference ID (required)")
	flags.String("language", "en-US", "Language code in BCP47 format")
	flags.String("direction", "both", "Direction: in, out, both")

	return cmd
}

func runStart(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	activeflowID, err := resolveUUID("activeflow_id", "Active Flow ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve active flow ID")
	}

	onEndFlowID, err := resolveUUID("on_end_flow_id", "On End Flow ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve on end flow ID")
	}

	referenceID, err := resolveUUID("reference_id", "Reference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve reference ID")
	}

	referenceTypeStr := viper.GetString("reference_type")
	if referenceTypeStr == "" {
		return fmt.Errorf("reference_type is required")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Start(
		context.Background(),
		customerID,
		activeflowID,
		onEndFlowID,
		transcribe.ReferenceType(referenceTypeStr),
		referenceID,
		viper.GetString("language"),
		transcribe.Direction(viper.GetString("direction")),
	)
	if err != nil {
		return errors.Wrap(err, "failed to start transcription")
	}

	return printJSON(res)
}

func cmdStop() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a transcription session",
		RunE:  runStop,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Transcribe ID (required)")

	return cmd
}

func runStop(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	transcribeID, err := resolveUUID("id", "Transcribe ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve transcribe ID")
	}

	res, err := handler.Stop(context.Background(), transcribeID)
	if err != nil {
		return errors.Wrap(err, "failed to stop transcription")
	}

	return printJSON(res)
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a transcription session by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Transcribe ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	transcribeID, err := resolveUUID("id", "Transcribe ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve transcribe ID")
	}

	res, err := handler.Get(context.Background(), transcribeID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve transcription")
	}

	return printJSON(res)
}

func cmdGetByReferenceIDAndLanguage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-by-reference",
		Short: "Get a transcription session by reference ID and language",
		RunE:  runGetByReferenceIDAndLanguage,
	}

	flags := cmd.Flags()
	flags.String("reference_id", "", "Reference ID (required)")
	flags.String("language", "", "Language code in BCP47 format (required)")

	return cmd
}

func runGetByReferenceIDAndLanguage(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	referenceID, err := resolveUUID("reference_id", "Reference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve reference ID")
	}

	language := viper.GetString("language")
	if language == "" {
		return fmt.Errorf("language is required")
	}

	res, err := handler.GetByReferenceIDAndLanguage(context.Background(), referenceID, language)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve transcription")
	}

	return printJSON(res)
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get transcription session list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of transcriptions to retrieve")
	flags.String("token", "", "Retrieve transcriptions before this token (pagination)")
	flags.String("customer_id", "", "Customer ID to filter (required)")
	flags.String("reference_type", "", "Reference type to filter: call, confbridge, recording")
	flags.String("reference_id", "", "Reference ID to filter")
	flags.String("status", "", "Status to filter: progressing, done")

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

	limit := viper.GetInt("limit")
	token := viper.GetString("token")

	filters := map[transcribe.Field]any{
		transcribe.FieldCustomerID: customerID,
		transcribe.FieldDeleted:    false,
	}

	if referenceTypeStr := viper.GetString("reference_type"); referenceTypeStr != "" {
		filters[transcribe.FieldReferenceType] = transcribe.ReferenceType(referenceTypeStr)
	}

	if referenceIDStr := viper.GetString("reference_id"); referenceIDStr != "" {
		referenceID := uuid.FromStringOrNil(referenceIDStr)
		if referenceID != uuid.Nil {
			filters[transcribe.FieldReferenceID] = referenceID
		}
	}

	if statusStr := viper.GetString("status"); statusStr != "" {
		filters[transcribe.FieldStatus] = transcribe.Status(statusStr)
	}

	res, err := handler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve transcriptions")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a transcription session",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Transcribe ID (required)")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Transcribe ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve transcribe ID")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete transcription")
	}

	return printJSON(res)
}
