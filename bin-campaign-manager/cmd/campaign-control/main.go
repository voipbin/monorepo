package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-campaign-manager/internal/config"
	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/campaignhandler"
	"monorepo/bin-campaign-manager/pkg/cachehandler"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = commonoutline.ServiceNameCampaignManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (campaignhandler.CampaignHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initCampaignHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initCampaignHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (campaignhandler.CampaignHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameCampaignEvent, serviceName)

	campaigncallHandler := campaigncallhandler.NewCampaigncallHandler(db, reqHandler, notifyHandler)
	outplanHandler := outplanhandler.NewOutplanHandler(db, reqHandler, notifyHandler)

	return campaignhandler.NewCampaignHandler(db, reqHandler, notifyHandler, campaigncallHandler, outplanHandler), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "campaign-control",
		Short: "Voipbin Campaign Management CLI",
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

	cmdSub := &cobra.Command{Use: "campaign", Short: "Campaign operation"}
	cmdSub.AddCommand(cmdCreate())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdList())
	cmdSub.AddCommand(cmdUpdateBasicInfo())
	cmdSub.AddCommand(cmdUpdateStatus())
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
		Short: "Create a new campaign",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID (required)")
	flags.String("type", string(campaign.TypeCall), "Campaign type: call or flow")
	flags.String("name", "", "Campaign name (required)")
	flags.String("detail", "", "Description")
	flags.Int("service_level", 100, "Service level percentage (0-100)")
	flags.String("end_handle", string(campaign.EndHandleStop), "End handle: stop or continue")
	flags.String("outplan_id", "", "Outplan ID (required)")
	flags.String("outdial_id", "", "Outdial ID (required)")
	flags.String("queue_id", "", "Queue ID")
	flags.String("next_campaign_id", "", "Next campaign ID")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}

	outplanID, err := resolveUUID("outplan_id", "Outplan ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve outplan ID")
	}

	outdialID, err := resolveUUID("outdial_id", "Outdial ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve outdial ID")
	}

	// Optional UUIDs
	queueID := uuid.FromStringOrNil(viper.GetString("queue_id"))
	nextCampaignID := uuid.FromStringOrNil(viper.GetString("next_campaign_id"))

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "failed to generate campaign ID")
	}

	res, err := handler.Create(
		context.Background(),
		id,
		customerID,
		campaign.Type(viper.GetString("type")),
		name,
		viper.GetString("detail"),
		[]fmaction.Action{},
		viper.GetInt("service_level"),
		campaign.EndHandle(viper.GetString("end_handle")),
		outplanID,
		outdialID,
		queueID,
		nextCampaignID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create campaign")
	}

	return printJSON(res)
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a campaign by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Campaign ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	campaignID, err := resolveUUID("id", "Campaign ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve campaign ID")
	}

	res, err := handler.Get(context.Background(), campaignID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve campaign")
	}

	return printJSON(res)
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get campaign list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of campaigns to retrieve")
	flags.String("token", "", "Retrieve campaigns before this token (pagination)")
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

	limit := viper.GetInt("limit")
	token := viper.GetString("token")

	filters := map[campaign.Field]any{
		campaign.FieldCustomerID: customerID,
		campaign.FieldDeleted:    false,
	}

	res, err := handler.List(context.Background(), token, uint64(limit), filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve campaigns")
	}

	return printJSON(res)
}

func cmdUpdateBasicInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-basic-info",
		Short: "Update campaign basic information",
		RunE:  runUpdateBasicInfo,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Campaign ID (required)")
	flags.String("name", "", "Campaign name (required)")
	flags.String("detail", "", "Description")
	flags.String("type", string(campaign.TypeCall), "Campaign type: call or flow")
	flags.Int("service_level", 100, "Service level percentage (0-100)")
	flags.String("end_handle", string(campaign.EndHandleStop), "End handle: stop or continue")

	return cmd
}

func runUpdateBasicInfo(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Campaign ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve campaign ID")
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}

	res, err := handler.UpdateBasicInfo(
		context.Background(),
		id,
		name,
		viper.GetString("detail"),
		campaign.Type(viper.GetString("type")),
		viper.GetInt("service_level"),
		campaign.EndHandle(viper.GetString("end_handle")),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update campaign basic info")
	}

	return printJSON(res)
}

func cmdUpdateStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-status",
		Short: "Update campaign status",
		RunE:  runUpdateStatus,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Campaign ID (required)")
	flags.String("status", string(campaign.StatusStop), "Status: run, stop, or stopping")

	return cmd
}

func runUpdateStatus(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Campaign ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve campaign ID")
	}

	res, err := handler.UpdateStatus(context.Background(), id, campaign.Status(viper.GetString("status")))
	if err != nil {
		return errors.Wrap(err, "failed to update campaign status")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a campaign",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Campaign ID (required)")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Campaign ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve campaign ID")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete campaign")
	}

	return printJSON(res)
}
