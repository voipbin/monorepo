package main

import (
	"database/sql"
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
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-agent-manager/internal/config"
	"monorepo/bin-agent-manager/pkg/agenthandler"
	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"
	"monorepo/bin-agent-manager/pkg/listenhandler"
	"monorepo/bin-agent-manager/pkg/subscribehandler"
)

const serviceName = commonoutline.ServiceNameAgentManager

var (
	chSigs = make(chan os.Signal, 1)
	chDone = make(chan bool, 1)
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "agent-manager",
		Short: "Voipbin Agent Manager Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.LoadGlobalConfig()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemon()
		},
	}

	if errBind := config.BindConfig(rootCmd); errBind != nil {
		logrus.Fatalf("Failed to bind config: %v", errBind)
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
	log.WithField("config", config.Get()).Info("Starting agent-manager...")

	sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return errors.Wrapf(err, "could not connect to the database")
	}
	defer commondatabasehandler.Close(sqlDB)

	cache, err := initCache()
	if err != nil {
		return err
	}

	if errStart := startServices(sqlDB, cache); errStart != nil {
		return errStart
	}

	<-chDone
	log.Info("Agent-manager stopped safely.")
	return nil
}

func startServices(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	db := dbhandler.NewHandler(sqlDB, cache)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameAgentEvent, serviceName)
	agentHandler := agenthandler.NewAgentHandler(reqHandler, db, notifyHandler)

	if errListen := startServiceListen(sockHandler, agentHandler); errListen != nil {
		return errors.Wrapf(errListen, "failed to start service listen")
	}

	if errSubscribe := startServiceSubscribe(sockHandler, agentHandler); errSubscribe != nil {
		return errors.Wrapf(errSubscribe, "failed to start service subscribe")
	}

	return nil
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if err := res.Connect(); err != nil {
		return nil, errors.Wrap(err, "cache connect error")
	}
	return res, nil
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
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		logrus.Infof("Prometheus metrics server starting on %s%s", listen, endpoint)
		if err := http.ListenAndServe(listen, nil); err != nil {
			logrus.Errorf("Prometheus server error: %v", err)
		}
	}()
}

// startServiceListen runs the listen service
func startServiceListen(sockHandler sockhandler.SockHandler, agentHandler agenthandler.AgentHandler) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, agentHandler)

	// run
	if errRun := listenHandler.Run(string(commonoutline.QueueNameAgentRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", errRun)
		return errRun
	}

	return nil
}

// startServiceSubscribe runs the subscribed event handler
func startServiceSubscribe(
	sockHandler sockhandler.SockHandler,
	agentHandler agenthandler.AgentHandler,
) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameCustomerEvent),
		string(commonoutline.QueueNameWebhookEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, string(commonoutline.QueueNameAgentSubscribe), subscribeTargets, agentHandler)

	// run
	if errRun := subHandler.Run(); errRun != nil {
		return errRun
	}

	return nil
}
