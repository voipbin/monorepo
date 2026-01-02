package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-agent-manager/internal/config"
	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/agenthandler"
	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/AlecAivazis/survey/v2"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = commonoutline.ServiceNameAgentManager

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (agenthandler.AgentHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize the cache")
	}

	return initAgentHandler(db, cache)
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initAgentHandler(sqlDB *sql.DB, cache cachehandler.CacheHandler) (agenthandler.AgentHandler, error) {
	db := dbhandler.NewHandler(sqlDB, cache)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameAgentEvent, serviceName)

	return agenthandler.NewAgentHandler(reqHandler, db, notifyHandler), nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "agent-control",
		Short: "Voipbin Agent Management CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.LoadGlobalConfig()
			if errBind := viper.BindPFlags(cmd.Flags()); errBind != nil {
				return errors.Wrap(errBind, "failed to bind flags")
			}

			return nil
		},
	}

	if err := config.Bootstrap(cmdRoot); err != nil {
		cobra.CheckErr(errors.Wrap(err, "failed to bind infrastructure config"))
	}

	cmdSub := &cobra.Command{Use: "agent", Short: "Agent operation"}
	cmdSub.AddCommand(cmdCreate())
	cmdSub.AddCommand(cmdGet())
	cmdSub.AddCommand(cmdList())
	cmdSub.AddCommand(cmdUpdatePermission())
	cmdSub.AddCommand(cmdUpdatePassword())

	cmdRoot.AddCommand(cmdSub)
	return cmdRoot
}

func resolveUUID(flagName string, label string) (uuid.UUID, error) {
	res := uuid.FromStringOrNil(viper.GetString(flagName))
	if res == uuid.Nil {
		tmp := ""
		prompt := &survey.Input{Message: fmt.Sprintf("%s (Required):", label)}
		if errAsk := survey.AskOne(prompt, &tmp, survey.WithValidator(survey.Required)); errAsk != nil {
			return uuid.Nil, errors.Wrap(errAsk, "input canceled")
		}

		res = uuid.FromStringOrNil(tmp)
		if res == uuid.Nil {
			return uuid.Nil, fmt.Errorf("invalid format for %s: '%s' is not a valid UUID", label, tmp)
		}
	}

	return res, nil
}

func cmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent",
		RunE:  runCreate,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID")
	flags.String("username", "", "Username")
	flags.String("password", "", "Password (WARNING: using --password will expose the password in process lists and shell history; prefer entering it interactively)")
	flags.Uint64("permission", 0, "Permission")
	flags.String("name", "", "Agent name")
	flags.String("detail", "", "Description")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	username := viper.GetString("username")
	if username == "" {
		if errAsk := survey.AskOne(&survey.Input{Message: "Username (Required):"}, &username, survey.WithValidator(survey.Required)); errAsk != nil {
			return errors.Wrap(errAsk, "failed to get username")
		}
	}

	password := viper.GetString("password")
	if password == "" {
		if errAsk := survey.AskOne(&survey.Password{Message: "Password (Required):"}, &password, survey.WithValidator(survey.Required)); errAsk != nil {
			return errors.Wrap(errAsk, "failed to get password")
		}
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.Create(
		context.Background(),
		customerID,
		username,
		password,
		viper.GetString("name"),
		viper.GetString("detail"),
		agent.RingMethodRingAll,
		agent.Permission(viper.GetUint64("permission")),
		[]uuid.UUID{},
		[]commonaddress.Address{},
	)
	if err != nil {
		return errors.Wrap(err, "failed to create agent")
	}

	logrus.WithField("res", res).Infof("Created a new agent")
	return nil
}

func cmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an agent by ID",
		RunE:  runGet,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Agent ID")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	agentID, err := resolveUUID("id", "Agent ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve agent ID")
	}

	fmt.Printf("\nRetrieving Agent ID: %s...\n", agentID)
	res, err := handler.Get(context.Background(), agentID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve agent")
	}

	fmt.Println("\n--- Agent Information ---")
	fmt.Printf("ID:    %s\n", res.ID)
	fmt.Printf("Customer ID: %s\n", res.CustomerID)
	fmt.Printf("Name:    %s\n", res.Name)
	fmt.Printf("Detail:    %s\n", res.Detail)
	fmt.Println("----------------------------")

	tmp, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal agent")
	}
	fmt.Println("\n--- Raw Data (JSON) ---")
	fmt.Println(string(tmp))
	fmt.Println("-----------------------")

	return nil
}

func cmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get agent list",
		RunE:  runList,
	}

	flags := cmd.Flags()
	flags.Int("limit", 100, "Limit the number of agents to retrieve")
	flags.String("token", "", "Retrieve agents before this token (pagination)")
	flags.String("customer_id", "", "Customer ID to filter")

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

	filters := map[string]string{
		"customer_id": customerID.String(),
	}

	fmt.Printf("\nRetrieving Agents... limit: %d, token: %s, filters: %v\n", limit, token, filters)
	res, err := handler.Gets(context.Background(), uint64(limit), token, filters)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve agents")
	}

	fmt.Printf("Success! agents count: %d\n", len(res))
	for _, c := range res {
		fmt.Printf(" - [%s] %s (%s)\n", c.ID, c.Name, c.Status)
	}

	return nil
}

func cmdUpdatePermission() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-permission",
		Short: "Update agent permission",
		RunE:  runUpdatePermission,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Agent ID")
	flags.Uint64("permission", uint64(agent.PermissionNone), "New Permission Bitmask")

	return cmd
}

func runUpdatePermission(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Agent ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve agent ID")
	}

	res, err := handler.UpdatePermission(context.Background(), id, agent.Permission(viper.GetUint64("permission")))
	if err != nil {
		return errors.Wrap(err, "failed to update agent permission")
	}

	logrus.WithField("res", res).Infof("Updated agent permission")
	return nil
}

func cmdUpdatePassword() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-password",
		Short: "Update agent password",
		RunE:  runUpdatePassword,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Agent ID")
	flags.String("password", "", "New Password")

	return cmd
}

func runUpdatePassword(cmd *cobra.Command, args []string) error {
	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	id, err := resolveUUID("id", "Agent ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve agent ID")
	}

	password := viper.GetString("password")
	if password == "" {
		prompt := &survey.Password{
			Message: "New Password:",
		}
		if err := survey.AskOne(prompt, &password, survey.WithValidator(survey.Required)); err != nil {
			return errors.Wrap(err, "failed to read password")
		}
	}

	res, err := handler.UpdatePassword(context.Background(), id, password)
	if err != nil {
		return errors.Wrap(err, "failed to update agent password")
	}

	logrus.WithField("res", res).Infof("Updated agent password")
	return nil
}
