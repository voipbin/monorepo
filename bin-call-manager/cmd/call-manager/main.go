package main

import (
	"database/sql"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-call-manager/models/common"
	"monorepo/bin-call-manager/pkg/arieventhandler"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
	"monorepo/bin-call-manager/pkg/listenhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
	"monorepo/bin-call-manager/pkg/subscribehandler"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// args for database
var dbDSN = flag.String("dbDSN", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for call-manager.")

// args for redis
var redisAddr = flag.String("redis_addr", "127.0.0.1:6379", "redis address.")
var redisPassword = flag.String("redis_password", "", "redis password")
var redisDB = flag.Int("redis_db", 1, "redis database.")

func main() {
	log := logrus.WithField("func", "main")

	// connect to database
	sqlDB, err := sql.Open("mysql", *dbDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	} else if err := sqlDB.Ping(); err != nil {
		log.Errorf("Could not set the connection correctly. err: %v", err)
		return
	}
	defer sqlDB.Close()

	// connect to cache
	cache := cachehandler.NewHandler(*redisAddr, *redisPassword, *redisDB)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	// run
	if errRun := run(sqlDB, cache); errRun != nil {
		log.Errorf("Could not run the process correctly. err: %v", errRun)
	}
	<-chDone
}

// proces init
func init() {
	flag.Parse()

	// init logs
	initLog()

	// init signal handler
	initSignal()

	// init prometheus setting
	initProm(*promEndpoint, *promListenAddr)

	logrus.Info("init finished.")
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
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

// run runs the call-manager
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	rabbitSock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(rabbitSock, common.Servicename)
	notifyHandler := notifyhandler.NewNotifyHandler(rabbitSock, reqHandler, commonoutline.QueueNameCallEvent, common.Servicename)
	channelHandler := channelhandler.NewChannelHandler(reqHandler, notifyHandler, db)
	bridgeHandler := bridgehandler.NewBridgeHandler(reqHandler, notifyHandler, db)
	externalMediaHandler := externalmediahandler.NewExternalMediaHandler(reqHandler, notifyHandler, db, channelHandler, bridgeHandler)
	recordingHandler := recordinghandler.NewRecordingHandler(reqHandler, notifyHandler, db, channelHandler, bridgeHandler)
	confbridgeHandler := confbridgehandler.NewConfbridgeHandler(reqHandler, notifyHandler, db, cache, channelHandler, bridgeHandler, recordingHandler, externalMediaHandler)
	groupcallHandler := groupcallhandler.NewGroupcallHandler(reqHandler, notifyHandler, db)
	callHandler := callhandler.NewCallHandler(reqHandler, notifyHandler, db, confbridgeHandler, channelHandler, bridgeHandler, recordingHandler, externalMediaHandler, groupcallHandler)
	ariEventHandler := arieventhandler.NewEventHandler(rabbitSock, db, cache, reqHandler, notifyHandler, callHandler, confbridgeHandler, channelHandler, bridgeHandler, recordingHandler)

	// run subscribe listener
	if err := runSubscribe(rabbitSock, ariEventHandler, callHandler, groupcallHandler, confbridgeHandler); err != nil {
		return err
	}

	// run request listener
	if err := runRequestListen(rabbitSock, callHandler, confbridgeHandler, channelHandler, recordingHandler, externalMediaHandler, groupcallHandler); err != nil {
		return err
	}

	return nil
}

// runSubscribe runs the ARI event listen service
func runSubscribe(
	rabbitSock rabbitmqhandler.Rabbit,
	ariEventHandler arieventhandler.ARIEventHandler,
	callHandler callhandler.CallHandler,
	groupcallHandler groupcallhandler.GroupcallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})

	subscribeTargets := []string{
		string(commonoutline.QueueNameAsteriskEventAll),
		string(commonoutline.QueueNameCustomerEvent),
		string(commonoutline.QueueNameFlowEvent),
	}
	log.WithField("subscribe_targets", subscribeTargets).Debug("Running subscribe handler")

	ariEventListenHandler := subscribehandler.NewSubscribeHandler(rabbitSock, commonoutline.QueueNameCallSubscribe, subscribeTargets, ariEventHandler, callHandler, groupcallHandler, confbridgeHandler)

	// run
	if err := ariEventListenHandler.Run(); err != nil {
		log.Errorf("Could not run the ari event listen handler correctly. err: %v", err)
	}

	return nil
}

// runRequestListen runs the request listen service
func runRequestListen(
	rabbitSock rabbitmqhandler.Rabbit,
	callHandler callhandler.CallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
	channelHandler channelhandler.ChannelHandler,
	recordingHandler recordinghandler.RecordingHandler,
	externalMediaHandler externalmediahandler.ExternalMediaHandler,
	groupcallHandler groupcallhandler.GroupcallHandler,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runRequestListen",
	})

	listenHandler := listenhandler.NewListenHandler(rabbitSock, callHandler, confbridgeHandler, channelHandler, recordingHandler, externalMediaHandler, groupcallHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameCallRequest), string(commonoutline.QueueNameDelay)); err != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
