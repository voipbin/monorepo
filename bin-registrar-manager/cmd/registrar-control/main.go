package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-registrar-manager/internal/config"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"
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

const serviceName = "registrar-control"

// Suppress unused import errors - these will be used in later tasks
var (
	_ = context.Background
	_ = sql.Open
	_ = json.Marshal
	_ = fmt.Sprintf
	_ = (*survey.Input)(nil)
	_ = uuid.NewV4
	_ = logrus.Info
)

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
	cmdRoot.AddCommand(cmdExtension)

	// Trunk subcommands
	cmdTrunk := &cobra.Command{Use: "trunk", Short: "Trunk operations"}
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
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameRegistrarEvent, serviceName)

	return extensionhandler.NewExtensionHandler(reqHandler, dbAst, dbBin, notifyHandler), nil
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
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameRegistrarEvent, serviceName)

	return trunkhandler.NewTrunkHandler(reqHandler, dbBin, notifyHandler), nil
}

func resolveUUID(flagName string, label string) (uuid.UUID, error) {
	res := uuid.FromStringOrNil(viper.GetString(flagName))
	if res == uuid.Nil {
		tmp := ""
		prompt := &survey.Input{Message: fmt.Sprintf("%s (Required):", label)}
		if err := survey.AskOne(prompt, &tmp, survey.WithValidator(survey.Required)); err != nil {
			return uuid.Nil, errors.Wrap(err, "input canceled")
		}

		res = uuid.FromStringOrNil(tmp)
		if res == uuid.Nil {
			return uuid.Nil, fmt.Errorf("invalid format for %s: '%s' is not a valid UUID", label, tmp)
		}
	}

	return res, nil
}

func resolveString(flagName string, label string, required bool) (string, error) {
	res := viper.GetString(flagName)
	if res == "" && required {
		prompt := &survey.Input{Message: fmt.Sprintf("%s (Required):", label)}
		if err := survey.AskOne(prompt, &res, survey.WithValidator(survey.Required)); err != nil {
			return "", errors.Wrap(err, "input canceled")
		}
	}
	return res, nil
}

func formatOutput(data interface{}, format string) error {
	if format == "json" {
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return errors.Wrap(err, "failed to marshal JSON")
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable format (caller provides specific formatting)
	return nil
}

func confirmDelete(resourceType string, id uuid.UUID, details string) (bool, error) {
	if viper.GetBool("force") {
		return true, nil
	}

	fmt.Printf("\n--- %s Information ---\n", resourceType)
	fmt.Print(details)
	fmt.Println("------------------------")

	confirm := false
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("Are you sure you want to delete %s %s?", resourceType, id),
	}
	if err := survey.AskOne(prompt, &confirm); err != nil {
		return false, errors.Wrap(err, "confirmation canceled")
	}

	return confirm, nil
}

func cmdExtensionCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new extension",
		RunE:  runExtensionCreate,
	}

	flags := cmd.Flags()
	flags.String("customer_id", "", "Customer ID")
	flags.String("extension_number", "", "Extension number")
	flags.String("username", "", "Username")
	flags.String("password", "", "Password")
	flags.String("domain", "", "Domain name")
	flags.String("format", "", "Output format (json)")

	return cmd
}

func runExtensionCreate(cmd *cobra.Command, args []string) error {
	customerID, err := resolveUUID("customer_id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "failed to resolve customer ID")
	}

	username, err := resolveString("username", "Username", true)
	if err != nil {
		return errors.Wrap(err, "failed to resolve username")
	}

	password := viper.GetString("password")
	if password == "" {
		prompt := &survey.Password{Message: "Password (Required):"}
		if err := survey.AskOne(prompt, &password, survey.WithValidator(survey.Required)); err != nil {
			return errors.Wrap(err, "failed to get password")
		}
	}

	extensionNumber := viper.GetString("extension_number")
	domain := viper.GetString("domain")

	handler, err := initExtensionHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handler")
	}

	res, err := handler.Create(context.Background(), customerID, extensionNumber, username, password, domain)
	if err != nil {
		return errors.Wrap(err, "failed to create extension")
	}

	format := viper.GetString("format")
	if format == "json" {
		return formatOutput(res, "json")
	}

	// Human-readable output
	fmt.Println("\n--- Extension Created ---")
	fmt.Printf("ID:                %s\n", res.ID)
	fmt.Printf("Customer ID:       %s\n", res.CustomerID)
	fmt.Printf("Extension Number:  %s\n", res.Extension)
	fmt.Printf("Username:          %s\n", res.Username)
	fmt.Printf("Domain:            %s\n", res.DomainName)
	fmt.Println("-------------------------")

	jsonData, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println("\n--- Raw Data (JSON) ---")
	fmt.Println(string(jsonData))
	fmt.Println("-----------------------")

	logrus.WithFields(logrus.Fields{
		"id":          res.ID,
		"customer_id": res.CustomerID,
		"username":    res.Username,
	}).Infof("Created extension")
	return nil
}
