package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-transcribe-manager/internal/config"
	"monorepo/bin-transcribe-manager/pkg/cachehandler"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/listenhandler"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
	"monorepo/bin-transcribe-manager/pkg/subscribehandler"
	"monorepo/bin-transcribe-manager/pkg/transcribehandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

const serviceName = "transcribe-manager"

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "transcribe-manager",
		Short: "Voipbin Transcribe Manager Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.LoadGlobalConfig()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemon()
		},
	}

	if errBind := config.Bootstrap(rootCmd); errBind != nil {
		logrus.Fatalf("Failed to bootstrap config: %v", errBind)
	}

	if errExecute := rootCmd.Execute(); errExecute != nil {
		logrus.Errorf("Command execution failed: %v", errExecute)
		os.Exit(1)
	}
}

func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-chSigs
		logrus.Infof("Received signal: %v", sig)
		chDone <- true
	}()
}

func initProm(endpoint, listen string) {
	// Skip Prometheus initialization if endpoint or listen address is not configured
	if endpoint == "" || listen == "" {
		logrus.Debug("Prometheus metrics disabled (endpoint or listen address not configured)")
		return
	}

	http.Handle(endpoint, promhttp.Handler())
	go func() {
		logrus.Infof("Prometheus metrics server starting on %s%s", listen, endpoint)
		if err := http.ListenAndServe(listen, nil); err != nil {
			// Prometheus server error is logged but not treated as fatal to avoid unsafe exit from a goroutine.
			logrus.Errorf("Prometheus server error: %v", err)
		}
	}()
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrap(errConnect, "cache connect error")
	}
	return res, nil
}

func runDaemon() error {
	initSignal()
	initProm(config.Get().PrometheusEndpoint, config.Get().PrometheusListenAddress)

	log := logrus.WithField("func", "runDaemon")
	log.WithField("config", config.Get()).Info("Starting transcribe-manager...")

	sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return errors.Wrapf(err, "could not connect to the database")
	}
	defer commondatabasehandler.Close(sqlDB)

	cache, err := initCache()
	if err != nil {
		return errors.Wrapf(err, "could not initialize the cache")
	}

	if errRun := run(sqlDB, cache); errRun != nil {
		return errors.Wrapf(errRun, "could not run transcribe-manager")
	}

	<-chDone
	log.Info("Transcribe-manager stopped safely.")
	return nil
}

// run runs the call-manager
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	hostID := uuid.Must(uuid.NewV4())
	log.Debugf("Generated host id. host_id: %s", hostID)

	if config.Get().PodIP == "" {
		return fmt.Errorf("could not get the listen ip address: POD_IP not configured")
	}
	listenAddress := fmt.Sprintf("%s:%d", config.Get().PodIP, config.Get().StreamingListenPort)
	log.Debugf("Listening address... listen_address: %s", listenAddress)

	// create handlers
	db := dbhandler.NewHandler(sqlDB, cache)
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameTranscribeEvent, commonoutline.ServiceNameTranscribeManager)
	transcriptHandler := transcripthandler.NewTranscriptHandler(reqHandler, db, notifyHandler)
	streamingHandler := streaminghandler.NewStreamingHandler(reqHandler, notifyHandler, transcriptHandler, listenAddress, config.Get().AWSAccessKey, config.Get().AWSSecretKey)
	if streamingHandler == nil {
		return fmt.Errorf("failed to initialize streaming handler: no STT providers available")
	}
	transcribeHandler := transcribehandler.NewTranscribeHandler(reqHandler, db, notifyHandler, transcriptHandler, streamingHandler, hostID)

	// run request listener
	if errListen := runListen(sockHandler, hostID, reqHandler, transcriptHandler, transcribeHandler); errListen != nil {
		return errors.Wrapf(errListen, "could not run the listen handler")
	}

	// run subscribe listener
	if errSubscribe := runSubscribe(sockHandler, transcribeHandler); errSubscribe != nil {
		return errors.Wrapf(errSubscribe, "could not run the subscribe handler")
	}

	// run streaming listener
	if errStreaming := runStreaming(streamingHandler); errStreaming != nil {
		return errors.Wrapf(errStreaming, "could not run the streaming handler")
	}

	return nil
}

// runListen runs the listen service
func runListen(
	sockHandler sockhandler.SockHandler,
	hostID uuid.UUID,
	reqHandler requesthandler.RequestHandler,
	transcriptHandler transcripthandler.TranscriptHandler,
	transcribeHandler transcribehandler.TranscribeHandler,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListen",
	})
	log.Debugf("Running listen handler")

	listenHandler := listenhandler.NewListenHandler(hostID, sockHandler, reqHandler, transcribeHandler, transcriptHandler)

	// run
	listenQueue := fmt.Sprintf("bin-manager.transcribe-manager-%s.request", hostID)
	if errRun := listenHandler.Run(string(commonoutline.QueueNameTranscribeRequest), listenQueue, string(commonoutline.QueueNameDelay)); errRun != nil {
		return errors.Wrapf(errRun, "could not run the listenhandler correctly.")
	}

	return nil
}

// runSubscribe runs the ARI event listen service
func runSubscribe(
	sockHandler sockhandler.SockHandler,
	transcribeHandler transcribehandler.TranscribeHandler,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})
	log.Debugf("Running subscribe handler")

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameCustomerEvent),
	}
	log.WithField("subscribe_targets", subscribeTargets).Debug("Running subscribe handler")

	ariEventListenHandler := subscribehandler.NewSubscribeHandler(sockHandler, commonoutline.QueueNameTranscribeSubscribe, subscribeTargets, transcribeHandler)

	// run
	if errRun := ariEventListenHandler.Run(); errRun != nil {
		return errors.Wrapf(errRun, "could not run the subscribehandler correctly.")
	}

	return nil
}

// runStreaming runs the ARI event listen service
func runStreaming(streamhingHandler streaminghandler.StreamingHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runStreaming",
	})
	log.Debugf("Running streaming handler")

	if errRun := streamhingHandler.Run(); errRun != nil {
		return errors.Wrapf(errRun, "could not run the streaminghandler correctly.")
	}

	return nil
}
