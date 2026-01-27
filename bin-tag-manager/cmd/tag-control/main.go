package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-tag-manager/internal/config"
	"monorepo/bin-tag-manager/models/tag"
	"monorepo/bin-tag-manager/pkg/cachehandler"
	"monorepo/bin-tag-manager/pkg/dbhandler"
	"monorepo/bin-tag-manager/pkg/taghandler"

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

const serviceName = commonoutline.ServiceNameTagManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (taghandler.TagHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initTagHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initTagHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (taghandler.TagHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameTagEvent, serviceName)

	return taghandler.NewTagHandler(reqHandler, db, notifyHandler), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "tag-control",
		Short: "Voipbin Tag Management CLI",
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

	cmdSub := &cobra.Command{Use: "tag", Short: "Tag operations"}
	cmdSub.AddCommand(cmdCreate())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdList())
	cmdSub.AddCommand(cmdUpdate())
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
		Short: "Create a new tag",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID (required)")
	flags.String("name", "", "Tag name (required)")
	flags.String("detail", "", "Tag detail")

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

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(
		context.Background(),
		customerID,
		name,
		viper.GetString("detail"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create tag")
	}

	return printJSON(res)
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a tag by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Tag ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	tagID, err := resolveUUID("id", "Tag ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve tag ID")
	}

	res, err := handler.Get(context.Background(), tagID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve tag")
	}

	return printJSON(res)
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get tag list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of tags to retrieve")
	flags.String("token", "", "Retrieve tags before this token (pagination)")
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

	filters := map[tag.Field]any{
		tag.FieldCustomerID: customerID,
		tag.FieldDeleted:    false,
	}

	res, err := handler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve tags")
	}

	return printJSON(res)
}

func cmdUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a tag",
		RunE:  runUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Tag ID (required)")
	flags.String("name", "", "New tag name (required)")
	flags.String("detail", "", "New tag detail")

	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	tagID, err := resolveUUID("id", "Tag ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve tag ID")
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}

	res, err := handler.UpdateBasicInfo(context.Background(), tagID, name, viper.GetString("detail"))
	if err != nil {
		return errors.Wrap(err, "failed to update tag")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a tag",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Tag ID (required)")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Tag ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve tag ID")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete tag")
	}

	return printJSON(res)
}
