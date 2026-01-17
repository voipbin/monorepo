package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	commondb "monorepo/bin-common-handler/pkg/databasehandler"
	commonnotify "monorepo/bin-common-handler/pkg/notifyhandler"
	commonreq "monorepo/bin-common-handler/pkg/requesthandler"
	commonsock "monorepo/bin-common-handler/pkg/sockhandler"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-talk-manager/internal/config"
	"monorepo/bin-talk-manager/pkg/dbhandler"
	"monorepo/bin-talk-manager/pkg/listenhandler"
	"monorepo/bin-talk-manager/pkg/messagehandler"
	"monorepo/bin-talk-manager/pkg/participanthandler"
	"monorepo/bin-talk-manager/pkg/reactionhandler"
	"monorepo/bin-talk-manager/pkg/talkhandler"
)

var (
	chSigs = make(chan os.Signal, 1)
	chDone = make(chan bool, 1)
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "talk-manager",
		Short: "Voipbin Talk Manager Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.LoadGlobalConfig()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemon()
		},
	}

	if errBind := config.Bootstrap(rootCmd); errBind != nil {
		log.Fatalf("Failed to bootstrap config: %v", errBind)
	}

	if errExecute := rootCmd.Execute(); errExecute != nil {
		log.Errorf("Command execution failed: %v", errExecute)
		os.Exit(1)
	}
}

func runDaemon() error {
	initSignal()
	initProm(config.Get().PrometheusEndpoint, config.Get().PrometheusListenAddress)

	logger := log.WithField("func", "runDaemon")
	cfg := config.Get()
	logger.WithField("config", cfg).Info("Starting talk-manager...")

	// Initialize database
	db, err := commondb.Connect(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("Could not connect to the database: %v", err)
	}
	defer commondb.Close(db)

	// Initialize RabbitMQ
	sockHandler := commonsock.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()
	defer sockHandler.Close()
	logger.Info("RabbitMQ initialized")

	// Initialize request and notify handlers
	reqHandler := commonreq.NewRequestHandler(sockHandler, commonoutline.ServiceNameTalkManager)
	notifyHandler := commonnotify.NewNotifyHandler(
		sockHandler,
		reqHandler,
		commonoutline.QueueNameTalkEvent,
		commonoutline.ServiceNameTalkManager,
	)

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDatabase,
	})
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Errorf("Failed to close Redis client: %v", err)
		}
	}()
	logger.Info("Redis initialized")

	// Initialize database handler
	dbHandler := dbhandler.New(db, redisClient)

	// Initialize business logic handlers
	talkHandler := talkhandler.New(dbHandler, notifyHandler)
	messageHandler := messagehandler.New(dbHandler, sockHandler, notifyHandler)
	participantHandler := participanthandler.New(dbHandler, sockHandler, notifyHandler)
	reactionHandler := reactionhandler.New(dbHandler, sockHandler, notifyHandler)

	// Initialize listen handler
	listenHandler := listenhandler.New(
		sockHandler,
		talkHandler,
		messageHandler,
		participantHandler,
		reactionHandler,
	)

	// Start listening for RabbitMQ messages
	ctx := context.Background()
	go func() {
		if err := listenHandler.Listen(ctx); err != nil {
			logger.Fatalf("Listen failed: %v", err)
		}
	}()
	logger.Infof("Listening for RabbitMQ messages on queue: %s", commonoutline.QueueNameTalkRequest)

	<-chDone
	logger.Info("Talk-manager stopped safely.")
	return nil
}

func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-chSigs
		log.Infof("Received signal: %v", sig)
		chDone <- true
	}()
}

func initProm(endpoint, listen string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		log.Infof("Prometheus metrics server starting on %s%s", listen, endpoint)
		if err := http.ListenAndServe(listen, nil); err != nil {
			// Prometheus server error is logged but not treated as fatal to avoid unsafe exit from a goroutine.
			log.Errorf("Prometheus server error: %v", err)
		}
	}()
}
