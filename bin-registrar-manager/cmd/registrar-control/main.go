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
