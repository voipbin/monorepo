package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"monorepo/bin-call-manager/internal/config"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
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

const serviceName = commonoutline.ServiceNameCallManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (callhandler.CallHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initCallHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initCallHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (callhandler.CallHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameCallEvent, serviceName, os.Getenv("CLICKHOUSE_ADDRESS"))

	channelHandler := channelhandler.NewChannelHandler(reqHandler, notifyHandler, db)
	bridgeHandler := bridgehandler.NewBridgeHandler(reqHandler, notifyHandler, db)
	externalMediaHandler := externalmediahandler.NewExternalMediaHandler(reqHandler, notifyHandler, db, channelHandler, bridgeHandler, cache, config.Get().AsteriskWSPort)
	recordingHandlerInst := recordinghandler.NewRecordingHandler(reqHandler, notifyHandler, db, channelHandler, bridgeHandler)
	confbridgeHandler := confbridgehandler.NewConfbridgeHandler(reqHandler, notifyHandler, db, cache, channelHandler, bridgeHandler, recordingHandlerInst, externalMediaHandler)
	groupcallHandler := groupcallhandler.NewGroupcallHandler(reqHandler, notifyHandler, db)
	recoveryHandler := callhandler.NewRecoveryHandler(reqHandler, config.Get().HomerAPIAddress, config.Get().HomerAuthToken, config.Get().HomerWhitelist)

	return callhandler.NewCallHandler(reqHandler, notifyHandler, db, confbridgeHandler, channelHandler, bridgeHandler, recordingHandlerInst, externalMediaHandler, groupcallHandler, recoveryHandler), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "call-control",
		Short: "Voipbin Call Management CLI",
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

	cmdSub := &cobra.Command{Use: "call", Short: "Call operation"}
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdList())
	cmdSub.AddCommand(cmdDelete())
	cmdSub.AddCommand(cmdUpdateStatus())
	cmdSub.AddCommand(cmdHangup())

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
		Short: "Get a call by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Call ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	callID, err := resolveUUID("id", "Call ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve call ID")
	}

	res, err := handler.Get(context.Background(), callID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve call")
	}

	return printJSON(res)
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get call list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of calls to retrieve")
	flags.String("token", "", "Retrieve calls before this token (pagination)")
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

	filters := map[call.Field]any{
		call.FieldCustomerID: customerID,
		call.FieldDeleted:    false,
	}

	res, err := handler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve calls")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a call",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Call ID (required)")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	callID, err := resolveUUID("id", "Call ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve call ID")
	}

	res, err := handler.Delete(context.Background(), callID)
	if err != nil {
		return errors.Wrap(err, "failed to delete call")
	}

	return printJSON(res)
}

func cmdUpdateStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-status",
		Short: "Update call status",
		RunE:  runUpdateStatus,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Call ID (required)")
	flags.String("status", "", "New status (dialing/ringing/progressing/terminating/canceling/hangup) (required)")

	return cmd
}

func runUpdateStatus(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	callID, err := resolveUUID("id", "Call ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve call ID")
	}

	statusStr := viper.GetString("status")
	if statusStr == "" {
		return fmt.Errorf("status is required")
	}

	res, err := handler.UpdateStatus(context.Background(), callID, call.Status(statusStr))
	if err != nil {
		return errors.Wrap(err, "failed to update call status")
	}

	return printJSON(res)
}

func cmdHangup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hangup",
		Short: "Hangup a call",
		RunE:  runHangup,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Call ID (required)")
	flags.String("reason", "normal", "Hangup reason (normal/failed/busy/cancel/timeout/noanswer/dialout/amd)")

	return cmd
}

func runHangup(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	callID, err := resolveUUID("id", "Call ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve call ID")
	}

	reason := viper.GetString("reason")
	if reason == "" {
		reason = "normal"
	}

	res, err := handler.HangingUp(context.Background(), callID, call.HangupReason(reason))
	if err != nil {
		return errors.Wrap(err, "failed to hangup call")
	}

	return printJSON(res)
}
