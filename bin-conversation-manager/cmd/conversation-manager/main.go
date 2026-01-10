package main

import (
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-conversation-manager/internal/config"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/listenhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
	"monorepo/bin-conversation-manager/pkg/smshandler"
	"monorepo/bin-conversation-manager/pkg/subscribehandler"
)

const (
	serviceName = "conversation-manager"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var rootCmd = &cobra.Command{
	Use:   "conversation-manager",
	Short: "Conversation Manager Service",
	Long:  `A microservice for managing conversation thread management.`,
	Run:   runMain,
}

func init() {
	// Register configuration flags
	config.RegisterFlags(rootCmd)

	// Init logs
	initLog()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("Failed to execute command: %v", err)
		os.Exit(1)
	}
}

func runMain(cmd *cobra.Command, args []string) {
	log := logrus.WithField("func", "main")
	log.Info("Starting conversation-manager.")

	// Initialize configuration
	config.Init(cmd)
	cfg := config.Get()

	// Init signal handler
	initSignal()

	// Init prometheus setting
	initProm(cfg.PrometheusEndpoint, cfg.PrometheusListenAddress)

	// connect to database
	sqlDB, err := commondatabasehandler.Connect(cfg.DatabaseDSN)
	if err != nil {
		logrus.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer commondatabasehandler.Close(sqlDB)

	// connect to cache
	cache := cachehandler.NewHandler(cfg.RedisAddress, cfg.RedisPassword, cfg.RedisDatabase)
	if err := cache.Connect(); err != nil {
		logrus.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	_ = run(sqlDB, cache, cfg)
	<-chDone

	log.Info("Finishing conversation-manager.")
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// run runs the main thread.
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler, cfg *config.Config) error {
	log := logrus.WithField("func", "run")

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameConversationEvent, serviceName)

	lineHandler := linehandler.NewLineHandler(reqHandler)
	accountHandler := accounthandler.NewAccountHandler(db, reqHandler, notifyHandler, lineHandler)
	smsHandler := smshandler.NewSMSHandler(reqHandler, accountHandler)

	messageHandler := messagehandler.NewMessageHandler(db, notifyHandler, accountHandler, lineHandler, smsHandler)
	conversationHandler := conversationhandler.NewConversationHandler(db, notifyHandler, reqHandler, accountHandler, messageHandler, lineHandler, smsHandler)

	// run listen
	if err := runListen(sockHandler, accountHandler, conversationHandler, messageHandler); err != nil {
		log.Errorf("Could not start runListen. err: %v", err)
		return err
	}

	// run subscribe
	if err := runSubscribe(sockHandler, accountHandler, conversationHandler); err != nil {
		log.Errorf("Could not start runSubscribe. err: %v", err)
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(
	sockListen sockhandler.SockHandler,
	accountHandler accounthandler.AccountHandler,
	conversationHandler conversationhandler.ConversationHandler,
	messageHandler messagehandler.MessageHandler,
) error {
	log := logrus.WithField("func", "runListen")

	listenHandler := listenhandler.NewListenHandler(sockListen, accountHandler, conversationHandler, messageHandler)

	// run the service
	if err := listenHandler.Run(string(commonoutline.QueueNameConversationRequest), string(commonoutline.QueueNameDelay)); err != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", err)
		return err
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(
	sockHandler sockhandler.SockHandler,
	accountHandler accounthandler.AccountHandler,
	conversationHandler conversationhandler.ConversationHandler,
) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameMessageEvent),
	}

	subHandler := subscribehandler.NewSubscribeHandler(
		sockHandler,
		string(commonoutline.QueueNameConversationSubscribe),
		subscribeTargets,
		accountHandler,
		conversationHandler,
	)

	// run
	if err := subHandler.Run(); err != nil {
		logrus.Errorf("Could not run the subscribehandler correctly. err: %v", err)
		return err
	}

	return nil
}

// initLog inits log settings.
func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

// initSignal inits sinal settings.
func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalHandler()
}

// initProm inits prometheus settings
func initProm(endpoint, listen string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		for {
			err := http.ListenAndServe(listen, nil)
			if err != nil {
				logrus.Errorf("Could not start prometheus listener")
				time.Sleep(time.Second * 1)
				continue
			}
			break
		}
	}()
}
