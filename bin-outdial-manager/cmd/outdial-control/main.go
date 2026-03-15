package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-common-handler/models/sock"
	commonoutline "monorepo/bin-common-handler/models/outline"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-outdial-manager/internal/config"
	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/pkg/cachehandler"
	"monorepo/bin-outdial-manager/pkg/dbhandler"
	"monorepo/bin-outdial-manager/pkg/outdialhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = commonoutline.ServiceNameOutdialManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (outdialhandler.OutdialHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initOutdialHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initOutdialHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (outdialhandler.OutdialHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameOutdialEvent, serviceName)

	return outdialhandler.NewOutdialHandler(db, reqHandler, notifyHandler), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "outdial-control",
		Short: "Voipbin Outdial Management CLI",
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

	cmdSub := &cobra.Command{Use: "outdial", Short: "Outdial operation"}
	cmdSub.AddCommand(cmdCreate())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdList())
	cmdSub.AddCommand(cmdUpdateBasicInfo())
	cmdSub.AddCommand(cmdUpdateCampaignID())
	cmdSub.AddCommand(cmdUpdateData())
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

func cmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new outdial",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("campaign-id", "", "Campaign ID (required)")
	flags.String("name", "", "Outdial name (required)")
	flags.String("detail", "", "Description")
	flags.String("data", "", "Custom JSON data")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	campaignID, err := resolveUUID("campaign-id", "Campaign ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve campaign ID")
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(
		context.Background(),
		customerID,
		campaignID,
		name,
		viper.GetString("detail"),
		viper.GetString("data"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create outdial")
	}

	return printJSON(res)
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an outdial by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Outdial ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	outdialID, err := resolveUUID("id", "Outdial ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve outdial ID")
	}

	res, err := handler.Get(context.Background(), outdialID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve outdial")
	}

	return printJSON(res)
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get outdial list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of outdials to retrieve")
	flags.String("token", "", "Retrieve outdials before this token (pagination)")
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

	filters := map[outdial.Field]any{
		outdial.FieldCustomerID: customerID,
		outdial.FieldDeleted:    false,
	}

	res, err := handler.List(context.Background(), token, uint64(limit), filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve outdials")
	}

	return printJSON(res)
}

func cmdUpdateBasicInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-basic-info",
		Short: "Update outdial name and detail",
		RunE:  runUpdateBasicInfo,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Outdial ID (required)")
	flags.String("name", "", "New name (required)")
	flags.String("detail", "", "New description")

	return cmd
}

func runUpdateBasicInfo(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Outdial ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve outdial ID")
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}

	res, err := handler.UpdateBasicInfo(context.Background(), id, name, viper.GetString("detail"))
	if err != nil {
		return errors.Wrap(err, "failed to update outdial basic info")
	}

	return printJSON(res)
}

func cmdUpdateCampaignID() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-campaign-id",
		Short: "Update outdial campaign ID",
		RunE:  runUpdateCampaignID,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Outdial ID (required)")
	flags.String("campaign-id", "", "New campaign ID (required)")

	return cmd
}

func runUpdateCampaignID(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Outdial ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve outdial ID")
	}

	campaignID, err := resolveUUID("campaign-id", "Campaign ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve campaign ID")
	}

	res, err := handler.UpdateCampaignID(context.Background(), id, campaignID)
	if err != nil {
		return errors.Wrap(err, "failed to update outdial campaign ID")
	}

	return printJSON(res)
}

func cmdUpdateData() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-data",
		Short: "Update outdial custom data",
		RunE:  runUpdateData,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Outdial ID (required)")
	flags.String("data", "", "Custom JSON data (required)")

	return cmd
}

func runUpdateData(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Outdial ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve outdial ID")
	}

	data := viper.GetString("data")
	if data == "" {
		return fmt.Errorf("data is required")
	}

	res, err := handler.UpdateData(context.Background(), id, data)
	if err != nil {
		return errors.Wrap(err, "failed to update outdial data")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an outdial",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Outdial ID (required)")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Outdial ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve outdial ID")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete outdial")
	}

	return printJSON(res)
}
