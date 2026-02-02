package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-sentinel-manager/internal/config"
	"monorepo/bin-sentinel-manager/pkg/monitoringhandler"
)

const serviceName = commonoutline.ServiceNameSentinelManager

// channels
var chSigs = make(chan os.Signal, 1)

var rootCmd = &cobra.Command{
	Use:   "sentinel-manager",
	Short: "Sentinel Manager Service",
	Long:  `Sentinel Manager is a microservice that monitors system health and manages service status.`,
	RunE:  run,
}

func init() {
	// Define flags
	rootCmd.Flags().String("prometheus_endpoint", "/metrics", "URL for the Prometheus metrics endpoint")
	rootCmd.Flags().String("prometheus_listen_address", ":2112", "Address for Prometheus to listen on (e.g., localhost:8080)")
	rootCmd.Flags().String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "Address of the RabbitMQ server (e.g., amqp://guest:guest@localhost:5672)")

	// Initialize logging
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("Failed to execute command: %v", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Printf("Hello world!\n")

	// Initialize configuration
	if err := config.InitConfig(cmd); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	cfg := config.Get()

	// Initialize Prometheus
	initProm(cfg.PrometheusEndpoint, cfg.PrometheusListenAddress)

	// Register signal handler
	ctx, cancel := context.WithCancel(context.Background())
	registerSignal(cancel)

	// run the service
	runService(ctx, cancel)

	<-ctx.Done()
	return nil
}

func runService(ctx context.Context, cancel context.CancelFunc) {
	log := logrus.WithField("func", "runService")
	defer cancel()

	cfg := config.Get()

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameSentinelEvent, serviceName, "")
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
