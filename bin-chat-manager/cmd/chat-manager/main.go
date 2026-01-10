package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-chat-manager/internal/config"
	"monorepo/bin-chat-manager/pkg/cachehandler"
	"monorepo/bin-chat-manager/pkg/chathandler"
	"monorepo/bin-chat-manager/pkg/chatroomhandler"
	"monorepo/bin-chat-manager/pkg/dbhandler"
	"monorepo/bin-chat-manager/pkg/listenhandler"
	"monorepo/bin-chat-manager/pkg/messagechathandler"
	"monorepo/bin-chat-manager/pkg/messagechatroomhandler"
)

const serviceName = commonoutline.ServiceNameChatManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var rootCmd = &cobra.Command{
	Use:   "chat-manager",
	Short: "Chat Manager Service",
	Long:  "Chat Manager handles chat rooms and messaging functionality in the VoIPbin platform",
	Run:   runCommand,
}

func init() {
	// Register flags
	config.RegisterFlags(rootCmd)

	// Initialize logging
	initLog()

	// Initialize signal handling
	initSignal()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("Failed to execute command: %v", err)
		os.Exit(1)
	}
}

func runCommand(cmd *cobra.Command, args []string) {
	// Initialize configuration
	config.Init(cmd)
	cfg := config.Get()

	// Initialize Prometheus
	initProm(cfg.PrometheusEndpoint, cfg.PrometheusListenAddress)

	logrus.Info("Starting chat-manager service")

	// create dbhandler
	dbHandler, err := createDBHandler(cfg)
	if err != nil {
		logrus.Errorf("Could not connect to the database or failed to initiate the cachehandler. err: %v", err)
		return
	}

	// run the service
	run(dbHandler, cfg)
	<-chDone
}

// initLog initializes log settings
func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

// initSignal initializes signal settings
func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalHandler()
}

// initProm initializes prometheus settings
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

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// createDBHandler connects to the database and cachehandler
func createDBHandler(cfg *config.Config) (dbhandler.DBHandler, error) {
	// connect to database
	db, err := commondatabasehandler.Connect(cfg.DatabaseDSN)
	if err != nil {
		logrus.Errorf("Could not access to database. err: %v", err)
		return nil, err
	}

	// connect to cache
	cache := cachehandler.NewHandler(cfg.RedisAddress, cfg.RedisPassword, cfg.RedisDatabase)
	if err := cache.Connect(); err != nil {
		logrus.Errorf("Could not connect to cache server. err: %v", err)
		return nil, err
	}

	// create dbhandler
	dbHandler := dbhandler.NewHandler(db, cache)

	return dbHandler, nil
}

func run(dbHandler dbhandler.DBHandler, cfg *config.Config) {
	log := logrus.WithField("func", "run")

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameChatEvent, serviceName)

	chatroomHandler := chatroomhandler.NewChatroomHandler(dbHandler, reqHandler, notifyHandler)
	chatHandler := chathandler.NewChatHandler(dbHandler, reqHandler, notifyHandler, chatroomHandler)

	messagechatroomHandler := messagechatroomhandler.NewMessagechatroomHandler(dbHandler, reqHandler, notifyHandler, chatroomHandler)
	messagechatHandler := messagechathandler.NewMessagechatHandler(dbHandler, reqHandler, notifyHandler, chatroomHandler, messagechatroomHandler)

	// run listen
	if errListen := runListen(sockHandler, chatHandler, chatroomHandler, messagechatHandler, messagechatroomHandler); errListen != nil {
		log.Errorf("Could not run the listen correctly. err: %v", errListen)
		return
	}
}

// runListen runs the listen service
func runListen(
	sockListen sockhandler.SockHandler,

	chatHandler chathandler.ChatHandler,
	chatroomHandler chatroomhandler.ChatroomHandler,
	messagechatHandler messagechathandler.MessagechatHandler,
	messagechatroomHandler messagechatroomhandler.MessagechatroomHandler,
) error {
	log := logrus.WithField("func", "runListen")

	listenHandler := listenhandler.NewListenHandler(sockListen, chatHandler, chatroomHandler, messagechatHandler, messagechatroomHandler)

	// run the service
	if errRun := listenHandler.Run(string(commonoutline.QueueNameChatRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		log.Errorf("Error occurred in listen handler. err: %v", errRun)
	}

	return nil
}
