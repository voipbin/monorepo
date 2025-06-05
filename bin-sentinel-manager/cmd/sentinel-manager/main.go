package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-sentinel-manager/pkg/monitoringhandler"
)

const serviceName = commonoutline.ServiceNameSentinelManager

// channels
var chSigs = make(chan os.Signal, 1)

var (
	prometheusEndpoint      = ""
	prometheusListenAddress = ""
	rabbitMQAddress         = ""
)

func main() {
	fmt.Printf("Hello world!\n")
	ctx, cancel := context.WithCancel(context.Background())

	registerSignal(cancel)

	// run the service
	run(ctx, cancel)

	<-ctx.Done()
}

func run(ctx context.Context, cancel context.CancelFunc) {
	log := logrus.WithField("func", "run")
	defer cancel()

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameFlowEvent, serviceName)
	utilHandler := utilhandler.NewUtilHandler()

	monitoringHandler := monitoringhandler.NewMonitoringHandler(reqHandler, notifyHandler, utilHandler)

	// run monitoring
	if errMonitoring := runMonitoring(ctx, monitoringHandler); errMonitoring != nil {
		log.Errorf("Could not run the monitoring correctly. err: %v", errMonitoring)
		return
	}
}

// registerSignal inits sinal settings.
func registerSignal(cancel context.CancelFunc) {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalHandler(cancel)
}

// signalHandler catches signals and set the done
func signalHandler(cancel context.CancelFunc) {
	defer cancel()

	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
}

func runMonitoring(
	ctx context.Context,
	monitoringHandler monitoringhandler.MonitoringHandler,
) error {
	log := logrus.WithField("func", "runMonitoring")

	mapSelectors := map[string][]string{
		"voip": []string{
			"app=asterisk-call",
			"app=asterisk-conference",
			"app=asterisk-registrar",
		},
	}

	// Start monitoring
	if err := monitoringHandler.Run(ctx, mapSelectors); err != nil {
		log.Errorf("Failed to run monitoring handler: %v", err)
		return fmt.Errorf("failed to run monitoring handler: %w", err)
	}

	log.Info("Monitoring started successfully")
	return nil
}
