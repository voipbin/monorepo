package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"monorepo/bin-registrar-manager/internal/config"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/sipauth"
	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
	"monorepo/bin-registrar-manager/pkg/extensiondirecthandler"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"
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

const serviceName = "registrar-control"

func main() {
	cmd := initCommand()
	if err := cmd.Execute(); err != nil {
		log.Fatalf("Execution failed: %v", err)
	}
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "registrar-control",
		Short: "VoIPbin Registrar Management CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				return errors.Wrap(err, "failed to bind flags")
			}
			return config.Bootstrap(cmd)
		},
	}

	// Extension subcommands
	cmdExtension := &cobra.Command{Use: "extension", Short: "Extension operations"}
	cmdExtension.AddCommand(cmdExtensionCreate())
	cmdExtension.AddCommand(cmdExtensionGet())
	cmdExtension.AddCommand(cmdExtensionList())
	cmdExtension.AddCommand(cmdExtensionUpdate())
	cmdExtension.AddCommand(cmdExtensionDelete())
	cmdRoot.AddCommand(cmdExtension)

	// Trunk subcommands
	cmdTrunk := &cobra.Command{Use: "trunk", Short: "Trunk operations"}
	cmdTrunk.AddCommand(cmdTrunkCreate())
	cmdTrunk.AddCommand(cmdTrunkGet())
	cmdTrunk.AddCommand(cmdTrunkList())
	cmdTrunk.AddCommand(cmdTrunkUpdate())
	cmdTrunk.AddCommand(cmdTrunkDelete())
	cmdRoot.AddCommand(cmdTrunk)

	return cmdRoot
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if err := res.Connect(); err != nil {
		return nil, errors.Wrap(err, "could not connect to cache")
	}
	return res, nil
}

func initExtensionHandler() (extensionhandler.ExtensionHandler, error) {
	sqlBin, err := commondatabasehandler.Connect(config.Get().DatabaseDSNBin)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to bin database")
	}

	sqlAsterisk, err := commondatabasehandler.Connect(config.Get().DatabaseDSNAsterisk)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to asterisk database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize cache")
	}

	dbAst := dbhandler.NewHandler(sqlAsterisk, cache)
	dbBin := dbhandler.NewHandler(sqlBin, cache)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameRegistrarEvent, serviceName, "")

	extensionDirectHandler := extensiondirecthandler.NewExtensionDirectHandler(dbBin)
	return extensionhandler.NewExtensionHandler(reqHandler, dbAst, dbBin, notifyHandler, extensionDirectHandler), nil
}

func initTrunkHandler() (trunkhandler.TrunkHandler, error) {
	sqlBin, err := commondatabasehandler.Connect(config.Get().DatabaseDSNBin)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to bin database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize cache")
	}

	dbBin := dbhandler.NewHandler(sqlBin, cache)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameRegistrarEvent, serviceName, "")

	return trunkhandler.NewTrunkHandler(reqHandler, dbBin, notifyHandler), nil
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

// ============================================================================
// Extension Commands
// ============================================================================

func cmdExtensionCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new extension",
		RunE:  runExtensionCreate,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("extension-number", "", "Extension number")
	flags.String("username", "", "Username (required)")
	flags.String("password", "", "Password (required)")
	flags.String("domain", "", "Domain name")

	return cmd
}

func runExtensionCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID")
	}

	username, err := resolveString("username", "Username")
	if err != nil {
		return errors.Wrap(err, "invalid username")
	}

	password, err := resolveString("password", "Password")
	if err != nil {
		return errors.Wrap(err, "invalid password")
	}

	extensionNumber := viper.GetString("extension-number")
	domain := viper.GetString("domain")

	handler, err := initExtensionHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	res, err := handler.Create(context.Background(), customerID, extensionNumber, username, password, domain)
	if err != nil {
		return errors.Wrap(err, "failed to create extension")
	}

	return printJSON(res)
}

func cmdExtensionGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an extension by ID",
		RunE:  runExtensionGet,
	}
	flags := cmd.Flags()
	flags.String("id", "", "Extension ID (required)")
	return cmd
}

func runExtensionGet(cmd *cobra.Command, args []string) error {
	id, err := resolveUUID("id", "Extension ID")
	if err != nil {
		return errors.Wrap(err, "invalid extension ID")
	}

	handler, err := initExtensionHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	res, err := handler.Get(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve extension")
	}

	return printJSON(res)
}

func cmdExtensionList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List extensions",
		RunE:  runExtensionList,
	}
	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID filter (required)")
	flags.String("domain", "", "Domain filter")
	flags.String("username", "", "Username filter")
	flags.String("extension-number", "", "Extension number filter")
	flags.Int("limit", 100, "Limit number of results")
	flags.String("token", "", "Pagination token")
	return cmd
}

func runExtensionList(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID")
	}

	handler, err := initExtensionHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	// Build filters with typed Field constants
	filters := make(map[extension.Field]any)
	filters[extension.FieldCustomerID] = customerID
	filters[extension.FieldDeleted] = false // Only show active extensions (not deleted)

	if domain := viper.GetString("domain"); domain != "" {
		filters[extension.FieldDomainName] = domain
	}
	if username := viper.GetString("username"); username != "" {
		filters[extension.FieldUsername] = username
	}
	if extNum := viper.GetString("extension-number"); extNum != "" {
		filters[extension.FieldExtension] = extNum
	}

	limit := uint64(viper.GetInt("limit"))
	token := viper.GetString("token")

	res, err := handler.List(context.Background(), token, limit, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve extensions")
	}

	return printJSON(res)
}

func cmdExtensionUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an extension",
		RunE:  runExtensionUpdate,
	}
	flags := cmd.Flags()
	flags.String("id", "", "Extension ID (required)")
	flags.String("password", "", "New password")
	flags.String("username", "", "New username")
	flags.String("extension-number", "", "New extension number")
	flags.String("domain", "", "New domain")
	return cmd
}

func runExtensionUpdate(cmd *cobra.Command, args []string) error {
	id, err := resolveUUID("id", "Extension ID")
	if err != nil {
		return errors.Wrap(err, "invalid extension ID")
	}

	// Check at least one update field is provided
	hasUpdate := false
	updates := make(map[extension.Field]any)

	if password := viper.GetString("password"); password != "" {
		updates[extension.FieldPassword] = password
		hasUpdate = true
	}
	if username := viper.GetString("username"); username != "" {
		updates[extension.FieldUsername] = username
		hasUpdate = true
	}
	if extNum := viper.GetString("extension-number"); extNum != "" {
		updates[extension.FieldExtension] = extNum
		hasUpdate = true
	}
	if domain := viper.GetString("domain"); domain != "" {
		updates[extension.FieldDomainName] = domain
		hasUpdate = true
	}

	if !hasUpdate {
		return fmt.Errorf("at least one field must be provided for update: --password, --username, --extension-number, or --domain")
	}

	handler, err := initExtensionHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	res, err := handler.Update(context.Background(), id, updates)
	if err != nil {
		return errors.Wrap(err, "failed to update extension")
	}

	return printJSON(res)
}

func cmdExtensionDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an extension",
		RunE:  runExtensionDelete,
	}
	flags := cmd.Flags()
	flags.String("id", "", "Extension ID (required)")
	return cmd
}

func runExtensionDelete(cmd *cobra.Command, args []string) error {
	id, err := resolveUUID("id", "Extension ID")
	if err != nil {
		return errors.Wrap(err, "invalid extension ID")
	}

	handler, err := initExtensionHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	res, err := handler.Delete(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to delete extension")
	}

	return printJSON(res)
}

// ============================================================================
// Trunk Commands
// ============================================================================

func cmdTrunkCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new trunk",
		RunE:  runTrunkCreate,
	}

	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID (required)")
	flags.String("domain", "", "Domain name (required)")
	flags.String("name", "", "Trunk name")
	flags.String("username", "", "Username")
	flags.String("password", "", "Password")
	flags.String("allowed-ips", "", "Allowed IPs (comma-separated)")

	return cmd
}

func runTrunkCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID")
	}

	domain, err := resolveString("domain", "Domain")
	if err != nil {
		return errors.Wrap(err, "invalid domain")
	}

	name := viper.GetString("name")
	username := viper.GetString("username")
	password := viper.GetString("password")

	// Parse allowed_ips
	var allowedIPs []string
	if allowedIPsStr := viper.GetString("allowed-ips"); allowedIPsStr != "" {
		for _, ip := range strings.Split(allowedIPsStr, ",") {
			trimmed := strings.TrimSpace(ip)
			if trimmed != "" {
				allowedIPs = append(allowedIPs, trimmed)
			}
		}
	}

	handler, err := initTrunkHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	// Determine auth types based on what was provided
	authTypes := []sipauth.AuthType{}
	if username != "" && password != "" {
		authTypes = append(authTypes, sipauth.AuthTypeBasic)
	}
	if len(allowedIPs) > 0 {
		authTypes = append(authTypes, sipauth.AuthTypeIP)
	}

	res, err := handler.Create(context.Background(), customerID, name, "", domain, authTypes, username, password, allowedIPs)
	if err != nil {
		return errors.Wrap(err, "failed to create trunk")
	}

	return printJSON(res)
}

func cmdTrunkGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a trunk by ID",
		RunE:  runTrunkGet,
	}
	flags := cmd.Flags()
	flags.String("id", "", "Trunk ID (required)")
	return cmd
}

func runTrunkGet(cmd *cobra.Command, args []string) error {
	id, err := resolveUUID("id", "Trunk ID")
	if err != nil {
		return errors.Wrap(err, "invalid trunk ID")
	}

	handler, err := initTrunkHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	res, err := handler.Get(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve trunk")
	}

	return printJSON(res)
}

func cmdTrunkList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List trunks",
		RunE:  runTrunkList,
	}
	flags := cmd.Flags()
	flags.String("customer-id", "", "Customer ID filter (required)")
	flags.String("domain", "", "Domain filter")
	flags.String("username", "", "Username filter")
	flags.String("name", "", "Name filter")
	flags.Int("limit", 100, "Limit number of results")
	flags.String("token", "", "Pagination token")
	return cmd
}

func runTrunkList(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer-id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID")
	}

	handler, err := initTrunkHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	// Build filters with typed Field constants
	filters := make(map[trunk.Field]any)
	filters[trunk.FieldCustomerID] = customerID
	filters[trunk.FieldDeleted] = false // Only show active trunks (not deleted)

	if domain := viper.GetString("domain"); domain != "" {
		filters[trunk.FieldDomainName] = domain
	}
	if username := viper.GetString("username"); username != "" {
		filters[trunk.FieldUsername] = username
	}
	if name := viper.GetString("name"); name != "" {
		filters[trunk.FieldName] = name
	}

	limit := uint64(viper.GetInt("limit"))
	token := viper.GetString("token")

	res, err := handler.List(context.Background(), token, limit, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve trunks")
	}

	return printJSON(res)
}

func cmdTrunkUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a trunk",
		RunE:  runTrunkUpdate,
	}
	flags := cmd.Flags()
	flags.String("id", "", "Trunk ID (required)")
	flags.String("password", "", "New password")
	flags.String("username", "", "New username")
	flags.String("name", "", "New name")
	flags.String("domain", "", "New domain")
	flags.String("allowed-ips", "", "New allowed IPs (comma-separated)")
	return cmd
}

func runTrunkUpdate(cmd *cobra.Command, args []string) error {
	id, err := resolveUUID("id", "Trunk ID")
	if err != nil {
		return errors.Wrap(err, "invalid trunk ID")
	}

	// Check at least one update field is provided
	hasUpdate := false
	updates := make(map[trunk.Field]any)

	if password := viper.GetString("password"); password != "" {
		updates[trunk.FieldPassword] = password
		hasUpdate = true
	}
	if username := viper.GetString("username"); username != "" {
		updates[trunk.FieldUsername] = username
		hasUpdate = true
	}
	if name := viper.GetString("name"); name != "" {
		updates[trunk.FieldName] = name
		hasUpdate = true
	}
	if domain := viper.GetString("domain"); domain != "" {
		updates[trunk.FieldDomainName] = domain
		hasUpdate = true
	}
	if allowedIPsStr := viper.GetString("allowed-ips"); allowedIPsStr != "" {
		var allowedIPs []string
		for _, ip := range strings.Split(allowedIPsStr, ",") {
			trimmed := strings.TrimSpace(ip)
			if trimmed != "" {
				allowedIPs = append(allowedIPs, trimmed)
			}
		}
		updates[trunk.FieldAllowedIPs] = allowedIPs
		hasUpdate = true
	}

	if !hasUpdate {
		return fmt.Errorf("at least one field must be provided for update: --password, --username, --name, --domain, or --allowed-ips")
	}

	handler, err := initTrunkHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	res, err := handler.Update(context.Background(), id, updates)
	if err != nil {
		return errors.Wrap(err, "failed to update trunk")
	}

	return printJSON(res)
}

func cmdTrunkDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a trunk",
		RunE:  runTrunkDelete,
	}
	flags := cmd.Flags()
	flags.String("id", "", "Trunk ID (required)")
	return cmd
}

func runTrunkDelete(cmd *cobra.Command, args []string) error {
	id, err := resolveUUID("id", "Trunk ID")
	if err != nil {
		return errors.Wrap(err, "invalid trunk ID")
	}

	handler, err := initTrunkHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	res, err := handler.Delete(context.Background(), id)
	if err != nil {
		return errors.Wrap(err, "failed to delete trunk")
	}

	return printJSON(res)
}
