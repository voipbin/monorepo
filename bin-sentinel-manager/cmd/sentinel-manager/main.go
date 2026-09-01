package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	dockerclient "github.com/docker/docker/client"
	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
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
	"monorepo/bin-sentinel-manager/pkg/cachehandler"
	"monorepo/bin-sentinel-manager/pkg/dockerwatchhandler"
)

const serviceName = commonoutline.ServiceNameSentinelManager

// channels
var chSigs = make(chan os.Signal, 1)

var rootCmd = &cobra.Command{
	Use:   "sentinel-manager",
	Short: "Sentinel Manager Service",
	Long:  `Sentinel Manager is a microservice that monitors system health and manages service status.`,
	RunE:  run,

	// This is a long-running daemon, not an interactive CLI: every error reaching Execute is a
	// RUNTIME failure, not a usage mistake. Without these, each fail-loud exit dumps the whole
	// flag-usage blob into the crash-loop output, burying the actual error exactly when someone is
	// reading the logs during an incident. main() already logs the error itself, so silencing
	// cobra's own printing avoids duplicating it too.
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	// Define flags
	rootCmd.Flags().String("prometheus_endpoint", "/metrics", "URL for the Prometheus metrics endpoint")
	rootCmd.Flags().String("prometheus_listen_address", ":2112", "Address for Prometheus to listen on (e.g., localhost:8080)")
	rootCmd.Flags().String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "Address of the RabbitMQ server (e.g., amqp://guest:guest@localhost:5672)")
	rootCmd.Flags().String("docker_socket_proxy_address", "tcp://sentinel-docker-socket-proxy:2375", "Address of the read-only docker-socket-proxy (e.g., tcp://sentinel-docker-socket-proxy:2375)")
	rootCmd.Flags().String("redis_address", "localhost:6379", "Address of the Redis server (e.g., localhost:6379)")
	rootCmd.Flags().String("redis_password", "", "Password of the Redis server")
	rootCmd.Flags().Int("redis_database", 1, "Database index of the Redis server")

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

	// runService blocks until the watcher stops: either because the context was cancelled
	// (SIGTERM -> clean exit 0) or because the watcher hit a fatal condition. A fatal condition
	// MUST leave this process with a non-zero exit status: main() turns this error into
	// os.Exit(1), which is what makes Komodo report a crash-loop instead of a container that
	// exited successfully. A sentinel that "looks up" but watches nothing is worse than one that
	// is visibly down (design §3.2).
	if errService := runService(ctx, cancel); errService != nil {
		return errors.Wrap(errService, "the sentinel-manager service stopped with an error")
	}

	return nil
}

func runService(ctx context.Context, cancel context.CancelFunc) error {
	log := logrus.WithField("func", "runService")
	defer cancel()

	cfg := config.Get()

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	// VOIP-1405: every sentinel-manager event is published to the global topic exchange
	// `bin-manager.event` (VOIP-1404 skeleton). cmd/sentinel-manager is the only publisher of
	// this service. Since VOIP-1418 the payload is `container.Event`, whose
	// EventSubscriptionID() returns the RESOLVED asterisk-id -- sentinel is no longer a
	// placeholder-by-design publisher, though an unresolved id still degrades to the `-`
	// placeholder through the standard path.
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameSentinelEvent, serviceName, notifyhandler.WithGlobalTopicPublish())
	utilHandler := utilhandler.NewUtilHandler()

	// docker client: talks ONLY to the read-only docker-socket-proxy, never to
	// /var/run/docker.sock. An unreachable proxy must surface as a crash-loop rather than a
	// sentinel that looks up but watches nothing.
	docker, errDocker := dockerclient.NewClientWithOpts(
		dockerclient.WithHost(cfg.DockerSocketProxyAddress),
		dockerclient.WithAPIVersionNegotiation(),
	)
	if errDocker != nil {
		return errors.Wrap(errDocker, "could not create the docker client")
	}
	defer func() {
		if errClose := docker.Close(); errClose != nil {
			log.Errorf("Could not close the docker client. err: %v", errClose)
		}
	}()

	cacheHandler := cachehandler.NewHandler(cfg.RedisAddress, cfg.RedisPassword, cfg.RedisDatabase)
	if errConnect := cacheHandler.Connect(); errConnect != nil {
		return errors.Wrap(errConnect, "could not connect to the redis")
	}

	dockerWatchHandler := dockerwatchhandler.NewDockerWatchHandler(reqHandler, notifyHandler, utilHandler, docker, cacheHandler)

	// run the container watcher. This blocks until the context is cancelled or the watcher hits a
	// fatal condition; the error is propagated all the way to a non-zero process exit.
	if errRun := dockerWatchHandler.Run(ctx); errRun != nil {
		return errors.Wrap(errRun, "could not run the docker watch handler correctly")
	}

	log.Info("The docker watch handler stopped cleanly.")

	return nil
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
