package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-tts-manager/internal/config"
	"monorepo/bin-tts-manager/pkg/listenhandler"
	"monorepo/bin-tts-manager/pkg/streaminghandler"
	"monorepo/bin-tts-manager/pkg/ttshandler"
)

const serviceName = commonoutline.ServiceNameTTSManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "tts-manager",
		Short: "Voipbin TTS Manager Daemon",
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

func runDaemon() error {
	initSignal()
	initProm(config.Get().PrometheusEndpoint, config.Get().PrometheusListenAddress)

	log := logrus.WithField("func", "runDaemon")
	log.WithField("config", config.Get()).Info("Starting tts-manager...")

	if errRun := run(); errRun != nil {
		return errors.Wrapf(errRun, "could not run tts-manager")
	}

	<-chDone
	log.Info("TTS-manager stopped safely.")
	return nil
}

// Run the services
func run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	// create listen handler
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameTTSEvent, serviceName, "")

	localAddress := os.Getenv("POD_IP")
	podID := os.Getenv("HOSTNAME")
	listenAddress := fmt.Sprintf("%s:8080", localAddress)

	ttsHandler := ttshandler.NewTTSHandler(config.Get().AWSAccessKey, config.Get().AWSSecretKey, "/shared-data", localAddress, reqHandler, notifyHandler)
	streamingHandler := streaminghandler.NewStreamingHandler(reqHandler, notifyHandler, listenAddress, podID, config.Get().ElevenlabsAPIKey)

	// run listener
	go runListen(sockHandler, ttsHandler, streamingHandler, podID)
	go runStreaming(streamingHandler)

	log.Debug("All handlers started successfully")
	return nil
}

// runListen run the listener
func runListen(sockHandler sockhandler.SockHandler, ttsHandler ttshandler.TTSHandler, streamingHandler streaminghandler.StreamingHandler, podID string) {

	if errRun := runListenNormal(sockHandler, ttsHandler, streamingHandler); errRun != nil {
		panic(errors.Wrapf(errRun, "could not run listen handler in normal mode"))
	}

	if errRun := runListenPod(sockHandler, ttsHandler, streamingHandler, podID); errRun != nil {
		panic(errors.Wrapf(errRun, "could not run listen handler in pod mode"))
	}
}

func runListenNormal(sockHandler sockhandler.SockHandler, ttsHandler ttshandler.TTSHandler, streamingHandler streaminghandler.StreamingHandler) error {

	listenHandler := listenhandler.NewListenHandler(sockHandler, ttsHandler, streamingHandler)

	// run
	if errRun := listenHandler.Run(string(commonoutline.QueueNameTTSRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		return errors.Wrapf(errRun, "could not run listen handler in normal mode")
	}

	return nil
}

// runListen run the listener
func runListenPod(sockHandler sockhandler.SockHandler, ttsHandler ttshandler.TTSHandler, streamingHandler streaminghandler.StreamingHandler, podID string) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, ttsHandler, streamingHandler)

	queueName := fmt.Sprintf("%s.%s", commonoutline.QueueNameTTSRequest, podID)
	if err := listenHandler.Run(queueName, string(commonoutline.QueueNameDelay)); err != nil {
		return errors.Wrapf(err, "could not run listen handler in pod mode")
	}

	return nil
}

func runStreaming(streamingHandler streaminghandler.StreamingHandler) {
	log := logrus.WithFields(logrus.Fields{
		"func": "runStreaming",
	})

	log.Debugf("Starting streaming handler.")
	if errRun := streamingHandler.Run(); errRun != nil {
		panic(errors.Wrapf(errRun, "could not run streaming handler"))
	}
}
