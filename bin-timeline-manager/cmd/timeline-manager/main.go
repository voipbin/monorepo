package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-timeline-manager/internal/config"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
	"monorepo/bin-timeline-manager/pkg/listenhandler"
)

var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "timeline-manager",
		Short: "Voipbin Timeline Manager Daemon",
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

func runDaemon() error {
	initSignal()
	initProm(config.Get().PrometheusEndpoint, config.Get().PrometheusListenAddress)

	log := logrus.WithField("func", "runDaemon")
	log.WithField("config", config.Get()).Info("Starting timeline-manager...")

	if errStart := runServices(); errStart != nil {
		return errors.Wrapf(errStart, "could not start services")
	}

	<-chDone
	log.Info("Timeline-manager stopped safely.")
	return nil
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
	if endpoint == "" || listen == "" {
		logrus.Debug("Prometheus metrics server disabled")
		return
	}

	http.Handle(endpoint, promhttp.Handler())
	go func() {
		logrus.Infof("Prometheus metrics server starting on %s%s", listen, endpoint)
		if err := http.ListenAndServe(listen, nil); err != nil {
			logrus.Errorf("Prometheus server error: %v", err)
		}
	}()
}

func runServices() error {
	db := dbhandler.NewHandler(config.Get().ClickHouseAddress, config.Get().ClickHouseDatabase)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	evtHandler := eventhandler.NewEventHandler(db)

	if errListen := runListen(sockHandler, evtHandler); errListen != nil {
		return errors.Wrapf(errListen, "failed to run service listen")
	}

	return nil
}

func runListen(sockListen sockhandler.SockHandler, evtHandler eventhandler.EventHandler) error {
	log := logrus.WithField("func", "runListen")

	listenHdlr := listenhandler.NewListenHandler(sockListen, evtHandler)

	if errRun := listenHdlr.Run(string(commonoutline.QueueNameTimelineRequest)); errRun != nil {
		log.Errorf("Error occurred in listen handler. err: %v", errRun)
	}

	return nil
}
