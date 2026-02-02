package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-route-manager/internal/config"
	"monorepo/bin-route-manager/pkg/cachehandler"
	"monorepo/bin-route-manager/pkg/dbhandler"
	"monorepo/bin-route-manager/pkg/routehandler"
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

const serviceName = commonoutline.ServiceNameRouteManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "route-control",
		Short: "Voipbin Route Management CLI",
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

	cmdRoute := &cobra.Command{Use: "route", Short: "Route operations"}
	cmdRoute.AddCommand(cmdRouteCreate())
	cmdRoute.AddCommand(cmdRouteGet())
	cmdRoute.AddCommand(cmdRouteList())
	cmdRoute.AddCommand(cmdRouteListByTarget())
	cmdRoute.AddCommand(cmdRouteUpdate())
	cmdRoute.AddCommand(cmdRouteDelete())
	cmdRoute.AddCommand(cmdDialrouteList())

	cmdRoot.AddCommand(cmdRoute)
	return cmdRoot
}

func initHandler() (routehandler.RouteHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initRouteHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initRouteHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (routehandler.RouteHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameRouteEvent, serviceName, "")

	return routehandler.NewRouteHandler(db, reqHandler, notifyHandler), nil
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

func cmdRouteCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new route",
		RunE:  runRouteCreate,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("name", "", "Route name (required)")
	flags.String("detail", "", "Route description")
	flags.String("provider-id", "", "Provider ID (required)")
	flags.Int("priority", 0, "Route priority (lower = higher priority)")
	flags.String("target", "", "Target destination (country code or 'all') (required)")

	return cmd
}

func runRouteCreate(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}

	providerID, err := resolveUUID("provider-id", "Provider ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve provider ID")
	}

	target := viper.GetString("target")
	if target == "" {
		return fmt.Errorf("target is required")
	}

	res, err := handler.Create(
		context.Background(),
		customerID,
		name,
		viper.GetString("detail"),
		providerID,
		viper.GetInt("priority"),
		target,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create route")
	}

	return printJSON(res)
}

func cmdRouteGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a route by ID",
		RunE:  runRouteGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Route ID (required)")

	return cmd
}

func runRouteGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	routeID, err := resolveUUID("id", "Route ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve route ID")
	}

	res, err := handler.Get(context.Background(), routeID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve route")
	}

	return printJSON(res)
}

func cmdRouteList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get route list",
		RunE:  runRouteList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of routes to retrieve")
	flags.String("token", "", "Retrieve routes before this token (pagination)")
	flags.String("customer-id", "", "Customer ID to filter (required)")

	return cmd
}

func runRouteList(cmd *cobra.Command, args []string) error {
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

	res, err := handler.ListByCustomerID(context.Background(), customerID, token, uint64(limit))
	if err != nil {
		return errors.Wrap(err, "failed to retrieve routes")
	}

	return printJSON(res)
}

func cmdRouteListByTarget() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-by-target",
		Short: "Get routes by target destination",
		RunE:  runRouteListByTarget,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("target", "", "Target destination (country code or 'all') (required)")

	return cmd
}

func runRouteListByTarget(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	target := viper.GetString("target")
	if target == "" {
		return fmt.Errorf("target is required")
	}

	res, err := handler.ListByTarget(context.Background(), customerID, target)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve routes")
	}

	return printJSON(res)
}

func cmdRouteUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a route",
		RunE:  runRouteUpdate,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Route ID (required)")
	flags.String("name", "", "Route name (required)")
	flags.String("detail", "", "Route description")
	flags.String("provider-id", "", "Provider ID (required)")
	flags.Int("priority", 0, "Route priority (lower = higher priority)")
	flags.String("target", "", "Target destination (country code or 'all') (required)")

	return cmd
}

func runRouteUpdate(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	routeID, err := resolveUUID("id", "Route ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve route ID")
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("name is required")
	}

	providerID, err := resolveUUID("provider-id", "Provider ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve provider ID")
	}

	target := viper.GetString("target")
	if target == "" {
		return fmt.Errorf("target is required")
	}

	res, err := handler.Update(
		context.Background(),
		routeID,
		name,
		viper.GetString("detail"),
		providerID,
		viper.GetInt("priority"),
		target,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update route")
	}

	return printJSON(res)
}

func cmdRouteDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a route",
		RunE:  runRouteDelete,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Route ID (required)")

	return cmd
}

func runRouteDelete(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	routeID, err := resolveUUID("id", "Route ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve route ID")
	}

	res, err := handler.Delete(context.Background(), routeID)
	if err != nil {
		return errors.Wrap(err, "failed to delete route")
	}

	return printJSON(res)
}

func cmdDialrouteList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dialroute-list",
		Short: "Get effective routes for dialing (merges customer and default routes)",
		RunE:  runDialrouteList,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("target", "", "Target destination (country code or 'all') (required)")

	return cmd
}

func runDialrouteList(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	target := viper.GetString("target")
	if target == "" {
		return fmt.Errorf("target is required")
	}

	res, err := handler.DialrouteList(context.Background(), customerID, target)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve dialroutes")
	}

	return printJSON(res)
}
