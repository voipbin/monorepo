package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-transfer-manager/internal/config"
	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/cachehandler"
	"monorepo/bin-transfer-manager/pkg/dbhandler"
	"monorepo/bin-transfer-manager/pkg/transferhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = "transfer-manager"

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (transferhandler.TransferHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initTransferHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initTransferHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (transferhandler.TransferHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameTransferEvent, serviceName)

	return transferhandler.NewTransferHandler(reqHandler, notifyHandler, db), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "transfer-control",
		Short: "Voipbin Transfer Management CLI",
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

	cmdSub := &cobra.Command{Use: "transfer", Short: "Transfer operation"}
	cmdSub.AddCommand(cmdServiceStart())
	cmdSub.AddCommand(cmdGetByGroupcall())
	cmdSub.AddCommand(cmdGetByCall())

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

func resolveString(flagName string, label string) (string, error) {
	val := viper.GetString(flagName)
	if val == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	return val, nil
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSON")
	}
	fmt.Println(string(data))
	return nil
}

func cmdServiceStart() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service-start",
		Short: "Start a transfer service",
		RunE:  runServiceStart,
	}

	flags := cmd.Flags()
	flags.String("transfer-type", "", "Transfer type (attended, blind) (required)")
	flags.String("transferer-call-id", "", "Transferer call ID (required)")
	flags.String("transferee-addresses", "", "Transferee addresses JSON array (required)")

	return cmd
}

func runServiceStart(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	transferType, err := resolveString("transfer-type", "Transfer type")
	if err != nil {
		return errors.Wrap(err, "failed to resolve transfer type")
	}

	transfererCallID, err := resolveUUID("transferer-call-id", "Transferer call ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve transferer call ID")
	}

	addressesStr, err := resolveString("transferee-addresses", "Transferee addresses")
	if err != nil {
		return errors.Wrap(err, "failed to resolve transferee addresses")
	}

	var addresses []commonaddress.Address
	if err := json.Unmarshal([]byte(addressesStr), &addresses); err != nil {
		return errors.Wrap(err, "failed to parse transferee addresses JSON")
	}

	res, err := handler.ServiceStart(context.Background(), transfer.Type(transferType), transfererCallID, addresses)
	if err != nil {
		return errors.Wrap(err, "failed to start transfer service")
	}

	return printJSON(res)
}

func cmdGetByGroupcall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-by-groupcall",
		Short: "Get transfer by groupcall ID",
		RunE:  runGetByGroupcall,
	}

	flags := cmd.Flags()
	flags.String("groupcall-id", "", "Groupcall ID (required)")

	return cmd
}

func runGetByGroupcall(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	groupcallID, err := resolveUUID("groupcall-id", "Groupcall ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve groupcall ID")
	}

	res, err := handler.GetByGroupcallID(context.Background(), groupcallID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve transfer")
	}

	return printJSON(res)
}

func cmdGetByCall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-by-call",
		Short: "Get transfer by transferer call ID",
		RunE:  runGetByCall,
	}

	flags := cmd.Flags()
	flags.String("call-id", "", "Transferer Call ID (required)")

	return cmd
}

func runGetByCall(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	callID, err := resolveUUID("call-id", "Transferer Call ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve call ID")
	}

	res, err := handler.GetByTransfererCallID(context.Background(), callID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve transfer")
	}

	return printJSON(res)
}
