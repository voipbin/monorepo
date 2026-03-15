package main

import (
	"database/sql"
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

	"monorepo/bin-conference-manager/internal/config"
	"monorepo/bin-conference-manager/pkg/cachehandler"
	"monorepo/bin-conference-manager/pkg/conferencecallhandler"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
	"monorepo/bin-conference-manager/pkg/dbhandler"
	"monorepo/bin-conference-manager/pkg/listenhandler"
	"monorepo/bin-conference-manager/pkg/subscribehandler"
)

const (
	serviceName = commonoutline.ServiceNameConferenceManager
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var rootCmd = &cobra.Command{
	Use:   "conference-manager",
	Short: "Conference Manager Service",
	Long:  `A microservice for managing audio conferencing with recording, transcription, and media streaming.`,
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
	log.Info("Starting conference-manager.")

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

	log.Info("Finishing conference-manager.")
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

	requestHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, requestHandler, commonoutline.QueueNameConferenceEvent, serviceName)

	conferenceHandler := conferencehandler.NewConferenceHandler(requestHandler, notifyHandler, db)
	conferencecallHandler := conferencecallhandler.NewConferencecallHandler(requestHandler, notifyHandler, db, conferenceHandler)

	// run listen
	if err := runListen(sockHandler, conferenceHandler, conferencecallHandler); err != nil {
		log.Errorf("Could not start runListen. err: %v", err)
		return err
	}

	// run subscribe
	if err := runSubscribe(sockHandler, conferenceHandler, conferencecallHandler); err != nil {
		log.Errorf("Could not start runSubscribe. err: %v", err)
		return err
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(
	sockHandler sockhandler.SockHandler,
	conferenceHandler conferencehandler.ConferenceHandler,
	conferencecallHandler conferencecallhandler.ConferencecallHandler,
) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(
		sockHandler,
		string(commonoutline.QueueNameConferenceSubscribe),
		subscribeTargets,
		conferenceHandler,
		conferencecallHandler,
	)

	// run
	if err := subHandler.Run(); err != nil {
		logrus.Errorf("Could not run the subscribehandler correctly. err: %v", err)
		return err
	}

	return nil
}

// runListen runs the listen handler
func runListen(
	sockHandler sockhandler.SockHandler,
	conferenceHandler conferencehandler.ConferenceHandler,
	conferencecallHandler conferencecallhandler.ConferencecallHandler,
) error {

	listenHandler := listenhandler.NewListenHandler(
		sockHandler,
		string(commonoutline.QueueNameConferenceRequest),
		string(commonoutline.QueueNameDelay),
		conferenceHandler,
		conferencecallHandler,
	)

	// run
	if err := listenHandler.Run(); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
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
