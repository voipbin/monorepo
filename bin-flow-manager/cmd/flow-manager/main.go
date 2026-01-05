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
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-flow-manager/internal/config"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/activeflowhandler"
	"monorepo/bin-flow-manager/pkg/cachehandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/flowhandler"
	"monorepo/bin-flow-manager/pkg/listenhandler"
	"monorepo/bin-flow-manager/pkg/subscribehandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

const serviceName = commonoutline.ServiceNameFlowManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "flow-manager",
		Short: "Voipbin Flow Manager Daemon",
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

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// connectDatabase connects to the database and cachehandler
func createDBHandler() (dbhandler.DBHandler, error) {
	// connect to database
	db, err := commondatabasehandler.Connect(databaseDSN)
	if err != nil {
		logrus.Errorf("Could not access to database. err: %v", err)
		return nil, err
	}

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		logrus.Errorf("Could not connect to cache server. err: %v", err)
		return nil, err
	}

	// create dbhandler
	dbHandler := dbhandler.NewHandler(db, cache)

	return dbHandler, nil
}

func run(dbHandler dbhandler.DBHandler) {
	log := logrus.WithField("func", "run")

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameFlowEvent, serviceName)

	actionHandler := actionhandler.NewActionHandler()
	variableHandler := variablehandler.NewVariableHandler(dbHandler, reqHandler)
	activeflowHandler := activeflowhandler.NewActiveflowHandler(dbHandler, reqHandler, notifyHandler, actionHandler, variableHandler)
	flowHandler := flowhandler.NewFlowHandler(dbHandler, reqHandler, notifyHandler, actionHandler, activeflowHandler)

	// run listen
	if errListen := runListen(sockHandler, flowHandler, activeflowHandler, variableHandler); errListen != nil {
		log.Errorf("Could not run the listen correctly. err: %v", errListen)
		return
	}

	// run sbuscriber
	if errSubs := runSubscribe(sockHandler, string(commonoutline.QueueNameFlowSubscribe), flowHandler, activeflowHandler); errSubs != nil {
		log.Errorf("Could not run the subscriber correctly. err: %v", errSubs)
		return
	}
}

// runListen runs the listen service
func runListen(
	sockListen sockhandler.SockHandler,
	flowHandler flowhandler.FlowHandler,
	activeflowHandler activeflowhandler.ActiveflowHandler,
	variableHandler variablehandler.VariableHandler,
) error {
	log := logrus.WithField("func", "runListen")

	listenHandler := listenhandler.NewListenHandler(sockListen, flowHandler, activeflowHandler, variableHandler)

	// run the service
	if errRun := listenHandler.Run(string(commonoutline.QueueNameFlowRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		log.Errorf("Error occurred in listen handler. err: %v", errRun)
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(sockHandler sockhandler.SockHandler, subscribeQueue string, flowHandler flowhandler.FlowHandler, activeflowHandler activeflowhandler.ActiveflowHandler) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCustomerEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, subscribeQueue, subscribeTargets, flowHandler, activeflowHandler)

	// run
	if err := subHandler.Run(); err != nil {
		return err
	}

	return nil
}
