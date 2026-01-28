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
	"monorepo/bin-conference-manager/internal/config"
	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/cachehandler"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
	"monorepo/bin-conference-manager/pkg/dbhandler"

	cmrecording "monorepo/bin-call-manager/models/recording"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = commonoutline.ServiceNameConferenceManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (conferencehandler.ConferenceHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initConferenceHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initConferenceHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (conferencehandler.ConferenceHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameConferenceEvent, serviceName)

	return conferencehandler.NewConferenceHandler(reqHandler, notifyHandler, db), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "conference-control",
		Short: "Voipbin Conference Management CLI",
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

	cmdSub := &cobra.Command{Use: "conference", Short: "Conference operation"}
	cmdSub.AddCommand(cmdCreate())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdGetByConfbridge())
	cmdSub.AddCommand(cmdList())
	cmdSub.AddCommand(cmdUpdate())
	cmdSub.AddCommand(cmdDelete())
	cmdSub.AddCommand(cmdUpdateRecordingID())
	cmdSub.AddCommand(cmdRecordingStart())
	cmdSub.AddCommand(cmdRecordingStop())
	cmdSub.AddCommand(cmdTerminating())
	cmdSub.AddCommand(cmdTranscribeStart())
	cmdSub.AddCommand(cmdTranscribeStop())

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

func cmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new conference",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Conference ID (optional, auto-generated if not provided)")
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("type", "conference", "Conference type: conference, connect, queue (default: conference)")
	flags.String("name", "", "Conference name")
	flags.String("detail", "", "Conference description")
	flags.Int("timeout", 0, "Timeout in seconds")
	flags.String("pre-flow-id", "", "Pre-flow ID (optional)")
	flags.String("post-flow-id", "", "Post-flow ID (optional)")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	var id uuid.UUID
	idStr := viper.GetString("id")
	if idStr != "" {
		id = uuid.FromStringOrNil(idStr)
		if id == uuid.Nil {
			return fmt.Errorf("invalid format for Conference ID: '%s' is not a valid UUID", idStr)
		}
	} else {
		id, _ = uuid.NewV4()
	}

	confType := conference.Type(viper.GetString("type"))
	if confType != conference.TypeConference && confType != conference.TypeConnect && confType != conference.TypeQueue {
		return fmt.Errorf("invalid conference type: must be 'conference', 'connect', or 'queue'")
	}

	var preFlowID uuid.UUID
	preFlowIDStr := viper.GetString("pre-flow-id")
	if preFlowIDStr != "" {
		preFlowID = uuid.FromStringOrNil(preFlowIDStr)
		if preFlowID == uuid.Nil {
			return fmt.Errorf("invalid format for Pre-flow ID: '%s' is not a valid UUID", preFlowIDStr)
		}
	}

	var postFlowID uuid.UUID
	postFlowIDStr := viper.GetString("post-flow-id")
	if postFlowIDStr != "" {
		postFlowID = uuid.FromStringOrNil(postFlowIDStr)
		if postFlowID == uuid.Nil {
			return fmt.Errorf("invalid format for Post-flow ID: '%s' is not a valid UUID", postFlowIDStr)
		}
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(
		context.Background(),
		id,
		customerID,
		confType,
		viper.GetString("name"),
		viper.GetString("detail"),
		map[string]interface{}{},
		viper.GetInt("timeout"),
		preFlowID,
		postFlowID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create conference")
	}

	return printJSON(res)
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a conference by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Conference ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	conferenceID, err := resolveUUID("id", "Conference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve conference ID")
	}

	res, err := handler.Get(context.Background(), conferenceID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve conference")
	}

	return printJSON(res)
}

func cmdGetByConfbridge() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-by-confbridge",
		Short: "Get a conference by confbridge ID",
		RunE:  runGetByConfbridge,
	}

	flags := cmd.Flags()
	flags.String("confbridge-id", "", "Confbridge ID (required)")

	return cmd
}

func runGetByConfbridge(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	confbridgeID, err := resolveUUID("confbridge-id", "Confbridge ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve confbridge ID")
	}

	res, err := handler.GetByConfbridgeID(context.Background(), confbridgeID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve conference")
	}

	return printJSON(res)
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get conference list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of conferences to retrieve")
	flags.String("token", "", "Retrieve conferences before this token (pagination)")
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

	filters := map[conference.Field]any{
		conference.FieldCustomerID: customerID,
	}

	res, err := handler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve conferences")
	}

	return printJSON(res)
}

func cmdUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update conference basic info",
		RunE:  runUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Conference ID (required)")
	flags.String("name", "", "Conference name")
	flags.String("detail", "", "Conference description")
	flags.Int("timeout", 0, "Timeout in seconds")
	flags.String("pre-flow-id", "", "Pre-flow ID (optional)")
	flags.String("post-flow-id", "", "Post-flow ID (optional)")

	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Conference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve conference ID")
	}

	var preFlowID uuid.UUID
	preFlowIDStr := viper.GetString("pre-flow-id")
	if preFlowIDStr != "" {
		preFlowID = uuid.FromStringOrNil(preFlowIDStr)
		if preFlowID == uuid.Nil {
			return fmt.Errorf("invalid format for Pre-flow ID: '%s' is not a valid UUID", preFlowIDStr)
		}
	}

	var postFlowID uuid.UUID
	postFlowIDStr := viper.GetString("post-flow-id")
	if postFlowIDStr != "" {
		postFlowID = uuid.FromStringOrNil(postFlowIDStr)
		if postFlowID == uuid.Nil {
			return fmt.Errorf("invalid format for Post-flow ID: '%s' is not a valid UUID", postFlowIDStr)
		}
	}

	res, err := handler.Update(
		context.Background(),
		id,
		viper.GetString("name"),
		viper.GetString("detail"),
		map[string]interface{}{},
		viper.GetInt("timeout"),
		preFlowID,
		postFlowID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update conference")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a conference",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Conference ID (required)")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Conference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve conference ID")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete conference")
	}

	return printJSON(res)
}

func cmdUpdateRecordingID() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-recording-id",
		Short: "Update conference recording ID",
		RunE:  runUpdateRecordingID,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Conference ID (required)")
	flags.String("recording-id", "", "Recording ID (required)")

	return cmd
}

func runUpdateRecordingID(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Conference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve conference ID")
	}

	recordingID, err := resolveUUID("recording-id", "Recording ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve recording ID")
	}

	res, err := handler.UpdateRecordingID(context.Background(), id, recordingID)
	if err != nil {
		return errors.Wrap(err, "failed to update recording ID")
	}

	return printJSON(res)
}

func cmdRecordingStart() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recording-start",
		Short: "Start conference recording",
		RunE:  runRecordingStart,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Conference ID (required)")
	flags.String("activeflow-id", "", "Active flow ID (optional)")
	flags.String("format", "wav", "Recording format: wav, mp3 (default: wav)")
	flags.Int("duration", 0, "Recording duration in seconds (0 for unlimited)")
	flags.String("on-end-flow-id", "", "On-end flow ID (optional)")

	return cmd
}

func runRecordingStart(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Conference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve conference ID")
	}

	var activeflowID uuid.UUID
	activeflowIDStr := viper.GetString("activeflow-id")
	if activeflowIDStr != "" {
		activeflowID = uuid.FromStringOrNil(activeflowIDStr)
		if activeflowID == uuid.Nil {
			return fmt.Errorf("invalid format for Active flow ID: '%s' is not a valid UUID", activeflowIDStr)
		}
	}

	var onEndFlowID uuid.UUID
	onEndFlowIDStr := viper.GetString("on-end-flow-id")
	if onEndFlowIDStr != "" {
		onEndFlowID = uuid.FromStringOrNil(onEndFlowIDStr)
		if onEndFlowID == uuid.Nil {
			return fmt.Errorf("invalid format for On-end flow ID: '%s' is not a valid UUID", onEndFlowIDStr)
		}
	}

	format := cmrecording.Format(viper.GetString("format"))
	if format != cmrecording.FormatWAV {
		return fmt.Errorf("invalid recording format: must be 'wav'")
	}

	res, err := handler.RecordingStart(
		context.Background(),
		id,
		activeflowID,
		format,
		viper.GetInt("duration"),
		onEndFlowID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to start recording")
	}

	return printJSON(res)
}

func cmdRecordingStop() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recording-stop",
		Short: "Stop conference recording",
		RunE:  runRecordingStop,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Conference ID (required)")

	return cmd
}

func runRecordingStop(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Conference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve conference ID")
	}

	res, err := handler.RecordingStop(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to stop recording")
	}

	return printJSON(res)
}

func cmdTerminating() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "terminating",
		Short: "Terminate a conference",
		RunE:  runTerminating,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Conference ID (required)")

	return cmd
}

func runTerminating(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Conference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve conference ID")
	}

	res, err := handler.Terminating(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to terminate conference")
	}

	return printJSON(res)
}

func cmdTranscribeStart() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transcribe-start",
		Short: "Start conference transcription",
		RunE:  runTranscribeStart,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Conference ID (required)")
	flags.String("lang", "en", "Transcription language (default: en)")

	return cmd
}

func runTranscribeStart(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Conference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve conference ID")
	}

	res, err := handler.TranscribeStart(
		context.Background(),
		id,
		viper.GetString("lang"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to start transcription")
	}

	return printJSON(res)
}

func cmdTranscribeStop() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transcribe-stop",
		Short: "Stop conference transcription",
		RunE:  runTranscribeStop,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Conference ID (required)")

	return cmd
}

func runTranscribeStop(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Conference ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve conference ID")
	}

	res, err := handler.TranscribeStop(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to stop transcription")
	}

	return printJSON(res)
}
