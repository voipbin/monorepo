package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"monorepo/bin-api-manager/internal/config"
	"monorepo/bin-api-manager/pkg/cachehandler"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = "api_manager"

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initHandler() (dbhandler.DBHandler, requesthandler.RequestHandler, error) {
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not connect to the database")
	}

	cache, err := initCache()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not initialize the cache")
	}

	dbHandler := dbhandler.NewHandler(db, cache)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)

	return dbHandler, reqHandler, nil
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrapf(errConnect, "could not connect to the cache")
	}
	return res, nil
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "api-control",
		Short: "VoIPbin API Manager CLI",
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

	cmdRoot.AddCommand(cmdVersion())
	cmdRoot.AddCommand(cmdHealth())

	return cmdRoot
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSON")
	}
	fmt.Println(string(data))
	return nil
}

func cmdVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		RunE:  runVersion,
	}

	return cmd
}

func runVersion(cmd *cobra.Command, args []string) error {
	version := map[string]string{
		"service": serviceName,
		"version": "3.1.0",
		"type":    "api-gateway",
	}

	return printJSON(version)
}

func cmdHealth() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check health status of API Manager dependencies",
		RunE:  runHealth,
	}

	return cmd
}

func runHealth(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	health := map[string]interface{}{
		"service": serviceName,
		"status":  "unknown",
	}

	// Test database connection
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		health["database"] = map[string]string{
			"status": "failed",
			"error":  err.Error(),
		}
	} else {
		defer commondatabasehandler.Close(db)
		if errPing := db.PingContext(ctx); errPing != nil {
			health["database"] = map[string]string{
				"status": "failed",
				"error":  errPing.Error(),
			}
		} else {
			health["database"] = map[string]string{
				"status": "ok",
			}
		}
	}

	// Test cache connection
	cache := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := cache.Connect(); errConnect != nil {
		health["cache"] = map[string]string{
			"status": "failed",
			"error":  errConnect.Error(),
		}
	} else {
		health["cache"] = map[string]string{
			"status": "ok",
		}
	}

	// Test RabbitMQ connection
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()
	health["rabbitmq"] = map[string]string{
		"status": "ok",
	}

	// Determine overall status
	allOK := true
	if db, ok := health["database"].(map[string]string); ok && db["status"] != "ok" {
		allOK = false
	}
	if cache, ok := health["cache"].(map[string]string); ok && cache["status"] != "ok" {
		allOK = false
	}
	if rmq, ok := health["rabbitmq"].(map[string]string); ok && rmq["status"] != "ok" {
		allOK = false
	}

	if allOK {
		health["status"] = "ok"
	} else {
		health["status"] = "degraded"
	}

	return printJSON(health)
}
