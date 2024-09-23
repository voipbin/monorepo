package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"monorepo/bin-campaign-manager/pkg/cachehandler"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/campaignhandler"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
	"monorepo/bin-campaign-manager/pkg/listenhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
	"monorepo/bin-campaign-manager/pkg/subscribehandler"
)

const serviceName = commonoutline.ServiceNameCampaignManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// args for database
var dbDSN = flag.String("dbDSN", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for campaign-manager.")

// args for redis
var redisAddr = flag.String("redis_addr", "127.0.0.1:6379", "redis address.")
var redisPassword = flag.String("redis_password", "", "redis password")
var redisDB = flag.Int("redis_db", 1, "redis database.")

func main() {
	fmt.Printf("Hello world!\n")

	// create dbhandler
	dbHandler, err := createDBHandler()
	if err != nil {
		logrus.Errorf("Could not connect to the database or failed to initiate the cachehandler. err: ")
		return
	}

	// run the service
	run(dbHandler)
	<-chDone
}

func init() {
	flag.Parse()

	// init logs
	initLog()

	// init signal handler
	initSignal()

	// init prometheus setting
	initProm(*promEndpoint, *promListenAddr)

	logrus.Info("The init finished.")
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

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
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

// connectDatabase connects to the database and cachehandler
func createDBHandler() (dbhandler.DBHandler, error) {
	// connect to database
	db, err := sql.Open("mysql", *dbDSN)
	if err != nil {
		logrus.Errorf("Could not access to database. err: %v", err)
		return nil, err
	}

	// connect to cache
	cache := cachehandler.NewHandler(*redisAddr, *redisPassword, *redisDB)
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
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, *rabbitAddr)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameCampaignEvent, serviceName)

	outplanHandler := outplanhandler.NewOutplanHandler(dbHandler, reqHandler, notifyHandler)
	campaigncallHandler := campaigncallhandler.NewCampaigncallHandler(dbHandler, reqHandler, notifyHandler)
	campaignHandler := campaignhandler.NewCampaignHandler(dbHandler, reqHandler, notifyHandler, campaigncallHandler, outplanHandler)

	// run listen
	if errListen := runListen(sockHandler, outplanHandler, campaignHandler, campaigncallHandler); errListen != nil {
		log.Errorf("Could not run the listen correctly. err: %v", errListen)
		return
	}

	// run subscribe
	if errSubscribe := runSubscribe(sockHandler, outplanHandler, campaignHandler, campaigncallHandler); errSubscribe != nil {
		log.Errorf("Could not run subscribe correctly. err: %v", errSubscribe)
		return
	}
}

// runListen runs the listen service
func runListen(
	sockListen sockhandler.SockHandler,
	outplanHandler outplanhandler.OutplanHandler,
	campaignHandler campaignhandler.CampaignHandler,
	campaigncallHandler campaigncallhandler.CampaigncallHandler,
) error {
	log := logrus.WithField("func", "runListen")

	listenHandler := listenhandler.NewListenHandler(sockListen, outplanHandler, campaignHandler, campaigncallHandler)

	// run the service
	if errRun := listenHandler.Run(string(commonoutline.QueueNameCampaignRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		log.Errorf("Error occurred in listen handler. err: %v", errRun)
	}

	return nil
}

// runSubscribe runs the subscribe service
func runSubscribe(
	sockListen sockhandler.SockHandler,
	outplanHandler outplanhandler.OutplanHandler,
	campaignHandler campaignhandler.CampaignHandler,
	campaigncallHandler campaigncallhandler.CampaigncallHandler,
) error {
	log := logrus.WithField("func", "runSubscribe")

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameFlowEvent),
	}
	subscribeHandler := subscribehandler.NewSubscribeHandler(
		sockListen,
		string(commonoutline.QueueNameCampaignSubscribe),
		subscribeTargets,
		campaignHandler,
		campaigncallHandler,
		outplanHandler,
	)

	// run
	if err := subscribeHandler.Run(); err != nil {
		log.Errorf("Could not run the subscribe handler correctly. err: %v", err)
		return err
	}

	return nil
}
