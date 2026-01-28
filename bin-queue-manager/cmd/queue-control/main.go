package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-queue-manager/internal/config"
	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/cachehandler"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuecallhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"

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

const serviceName = commonoutline.ServiceNameQueueManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (queuehandler.QueueHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initQueueHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initQueueHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (queuehandler.QueueHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameQueueEvent, serviceName)

	return queuehandler.NewQueueHandler(reqHandler, db, notifyHandler), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "queue-control",
		Short: "Voipbin Queue Management CLI",
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

	cmdSub := &cobra.Command{Use: "queue", Short: "Queue operations"}
	cmdSub.AddCommand(cmdCreate())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdList())
	cmdSub.AddCommand(cmdUpdate())
	cmdSub.AddCommand(cmdUpdateTagIDs())
	cmdSub.AddCommand(cmdUpdateRoutingMethod())
	cmdSub.AddCommand(cmdUpdateExecute())
	cmdSub.AddCommand(cmdDelete())

	// Queuecall subcommands (read-only operations for debugging/monitoring)
	cmdQueuecall := &cobra.Command{Use: "queuecall", Short: "Queuecall operations (read-only)"}
	cmdQueuecall.AddCommand(cmdQueuecallGet())
	cmdQueuecall.AddCommand(cmdQueuecallGetByReferenceID())
	cmdQueuecall.AddCommand(cmdQueuecallList())
	cmdQueuecall.AddCommand(cmdQueuecallDelete())

	cmdRoot.AddCommand(cmdSub)
	cmdRoot.AddCommand(cmdQueuecall)
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

func parseUUIDs(uuidsStr string) ([]uuid.UUID, error) {
	if uuidsStr == "" {
		return []uuid.UUID{}, nil
	}

	var uuids []uuid.UUID
	if err := json.Unmarshal([]byte(uuidsStr), &uuids); err != nil {
		return nil, errors.Wrap(err, "failed to parse UUIDs JSON array")
	}
	return uuids, nil
}

func cmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new queue",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("name", "", "Queue name (required)")
	flags.String("detail", "", "Queue detail")
	flags.String("routing-method", "random", "Routing method (random)")
	flags.String("tag-ids", "", "Tag IDs JSON array")
	flags.String("wait-flow-id", "", "Wait flow ID")
	flags.Int("wait-timeout", 300000, "Wait timeout in ms")
	flags.Int("service-timeout", 600000, "Service timeout in ms")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}

	tagIDs, err := parseUUIDs(viper.GetString("tag-ids"))
	if err != nil {
		return errors.Wrap(err, "failed to parse tag IDs")
	}

	waitFlowID := uuid.FromStringOrNil(viper.GetString("wait-flow-id"))

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(
		context.Background(),
		customerID,
		name,
		viper.GetString("detail"),
		queue.RoutingMethod(viper.GetString("routing-method")),
		tagIDs,
		waitFlowID,
		viper.GetInt("wait-timeout"),
		viper.GetInt("service-timeout"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create queue")
	}

	return printJSON(res)
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a queue by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Queue ID (required)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	queueID, err := resolveUUID("id", "Queue ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve queue ID")
	}

	res, err := handler.Get(context.Background(), queueID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve queue")
	}

	return printJSON(res)
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get queue list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of queues to retrieve")
	flags.String("token", "", "Retrieve queues before this token (pagination)")
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

	filters := map[queue.Field]any{
		queue.FieldCustomerID: customerID,
		queue.FieldDeleted:    false,
	}

	res, err := handler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve queues")
	}

	return printJSON(res)
}

func cmdUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a queue",
		RunE:  runUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Queue ID (required)")
	flags.String("name", "", "Queue name (required)")
	flags.String("detail", "", "Queue detail")
	flags.String("routing-method", "random", "Routing method")
	flags.String("tag-ids", "", "Tag IDs JSON array")
	flags.String("wait-flow-id", "", "Wait flow ID")
	flags.Int("wait-timeout", 300000, "Wait timeout in ms")
	flags.Int("service-timeout", 600000, "Service timeout in ms")

	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	queueID, err := resolveUUID("id", "Queue ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve queue ID")
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}

	tagIDs, err := parseUUIDs(viper.GetString("tag-ids"))
	if err != nil {
		return errors.Wrap(err, "failed to parse tag IDs")
	}

	waitFlowID := uuid.FromStringOrNil(viper.GetString("wait-flow-id"))

	res, err := handler.UpdateBasicInfo(
		context.Background(),
		queueID,
		name,
		viper.GetString("detail"),
		queue.RoutingMethod(viper.GetString("routing-method")),
		tagIDs,
		waitFlowID,
		viper.GetInt("wait-timeout"),
		viper.GetInt("service-timeout"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update queue")
	}

	return printJSON(res)
}

func cmdUpdateTagIDs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-tag-ids",
		Short: "Update queue tag IDs only",
		RunE:  runUpdateTagIDs,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Queue ID (required)")
	flags.String("tag-ids", "", "Tag IDs JSON array (required)")

	return cmd
}

func runUpdateTagIDs(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	queueID, err := resolveUUID("id", "Queue ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve queue ID")
	}

	tagIDsStr := viper.GetString("tag-ids")
	if tagIDsStr == "" {
		return fmt.Errorf("tag-ids is required")
	}

	tagIDs, err := parseUUIDs(tagIDsStr)
	if err != nil {
		return errors.Wrap(err, "failed to parse tag IDs")
	}

	res, err := handler.UpdateTagIDs(context.Background(), queueID, tagIDs)
	if err != nil {
		return errors.Wrap(err, "failed to update queue tag IDs")
	}

	return printJSON(res)
}

func cmdUpdateRoutingMethod() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-routing-method",
		Short: "Update queue routing method only",
		RunE:  runUpdateRoutingMethod,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Queue ID (required)")
	flags.String("routing-method", "", "Routing method (random) (required)")

	return cmd
}

func runUpdateRoutingMethod(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	queueID, err := resolveUUID("id", "Queue ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve queue ID")
	}

	routingMethod := viper.GetString("routing-method")
	if routingMethod == "" {
		return fmt.Errorf("routing-method is required")
	}

	res, err := handler.UpdateRoutingMethod(context.Background(), queueID, queue.RoutingMethod(routingMethod))
	if err != nil {
		return errors.Wrap(err, "failed to update queue routing method")
	}

	return printJSON(res)
}

func cmdUpdateExecute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-execute",
		Short: "Update queue execute state",
		RunE:  runUpdateExecute,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Queue ID (required)")
	flags.String("execute", "", "Execute state (run, stop) (required)")

	return cmd
}

func runUpdateExecute(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	queueID, err := resolveUUID("id", "Queue ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve queue ID")
	}

	execute := viper.GetString("execute")
	if execute == "" {
		return fmt.Errorf("execute is required")
	}

	res, err := handler.UpdateExecute(context.Background(), queueID, queue.Execute(execute))
	if err != nil {
		return errors.Wrap(err, "failed to update queue execute state")
	}

	return printJSON(res)
}

func cmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a queue",
		RunE:  runDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Queue ID (required)")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Queue ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve queue ID")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete queue")
	}

	return printJSON(res)
}

// Queuecall commands (read-only operations for debugging/monitoring)

func initQueuecallHandler() (queuecallhandler.QueuecallHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	dbHandler := dbhandler.NewHandler(db, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameQueueEvent, serviceName)

	// For read-only operations, we pass nil for queueHandler since it's not used by Get/List/Delete
	return queuecallhandler.NewQueuecallHandler(reqHandler, dbHandler, notifyHandler, nil), nil
}

func cmdQueuecallGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a queuecall by ID",
		RunE:  runQueuecallGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Queuecall ID (required)")

	return cmd
}

func cmdQueuecallGetByReferenceID() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-by-reference",
		Short: "Get a queuecall by reference ID (call_id, etc.)",
		RunE:  runQueuecallGetByReferenceID,
	}

	flags := cmd.Flags()
	flags.String("reference-id", "", "Reference ID (required)")

	return cmd
}

func cmdQueuecallList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List queuecalls",
		RunE:  runQueuecallList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of queuecalls to retrieve")
	flags.String("token", "", "Retrieve queuecalls before this token (pagination)")
	flags.String("customer-id", "", "Filter by customer ID (required)")
	flags.String("queue-id", "", "Filter by queue ID")
	flags.String("status", "", "Filter by status (initiating, waiting, connecting, service, done, abandoned)")

	return cmd
}

func cmdQueuecallDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a queuecall",
		RunE:  runQueuecallDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Queuecall ID (required)")

	return cmd
}

func runQueuecallGet(cmd *cobra.Command, args []string) error {
	handler, err := initQueuecallHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Queuecall ID")
	if err != nil {
		return errors.Wrap(err, "invalid Queuecall ID format")
	}

	res, err := handler.Get(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve queuecall")
	}

	return printJSON(res)
}

func runQueuecallGetByReferenceID(cmd *cobra.Command, args []string) error {
	handler, err := initQueuecallHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	referenceID, err := resolveUUID("reference-id", "Reference ID")
	if err != nil {
		return errors.Wrap(err, "invalid reference ID format")
	}

	res, err := handler.GetByReferenceID(context.Background(), referenceID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve queuecall by reference ID")
	}

	return printJSON(res)
}

func runQueuecallList(cmd *cobra.Command, args []string) error {
	handler, err := initQueuecallHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID format")
	}

	limit := viper.GetInt("limit")
	token := viper.GetString("token")

	filters := map[queuecall.Field]any{
		queuecall.FieldCustomerID: customerID,
	}

	// Optional filters
	if queueID := viper.GetString("queue-id"); queueID != "" {
		if id := uuid.FromStringOrNil(queueID); id != uuid.Nil {
			filters[queuecall.FieldQueueID] = id
		}
	}
	if status := viper.GetString("status"); status != "" {
		filters[queuecall.FieldStatus] = queuecall.Status(status)
	}

	res, err := handler.List(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve queuecalls")
	}

	return printJSON(res)
}

func runQueuecallDelete(cmd *cobra.Command, args []string) error {
	handler, err := initQueuecallHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Queuecall ID")
	if err != nil {
		return errors.Wrap(err, "invalid Queuecall ID format")
	}

	res, err := handler.Delete(context.Background(), targetID)
	if err != nil {
		return errors.Wrap(err, "failed to delete queuecall")
	}

	return printJSON(res)
}
