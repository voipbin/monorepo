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
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/pkg/cachehandler"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/listenhandler"
	"monorepo/bin-number-manager/pkg/numberhandler"
	"monorepo/bin-number-manager/pkg/subscribehandler"
)

const serviceName = commonoutline.ServiceNameNumberManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var argRabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")

// args for prometheus
var argPromEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var argPromListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// args for database
var argDBDSN = flag.String("dbDSN", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for number-manager.")

// args for redis
var argRedisAddr = flag.String("redis_addr", "127.0.0.1:6379", "redis address.")
var argRedisPassword = flag.String("redis_password", "", "redis password")
var argRedisDB = flag.Int("redis_db", 1, "redis database.")

func main() {
	log := logrus.WithField("func", "main")
	log.Debugf("Hello world. Starting number-manager.")

	// connect to database
	sqlDB, err := sql.Open("mysql", *argDBDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer sqlDB.Close()

	// connect to cache
	cache := cachehandler.NewHandler(*argRedisAddr, *argRedisPassword, *argRedisDB)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	if errRun := run(sqlDB, cache); errRun != nil {
		log.Errorf("The run returned error. err: %v", errRun)
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
	initProm(*argPromEndpoint, *argPromListenAddr)

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

// run runs the listen
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {

	// rabbitmq sock connect
	rabbitSock := rabbitmqhandler.NewRabbit(*argRabbitAddr)
	rabbitSock.Connect()

	// create handlers
	db := dbhandler.NewHandler(sqlDB, cache)
	reqHandler := requesthandler.NewRequestHandler(rabbitSock, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(rabbitSock, reqHandler, commonoutline.QueueNameNumberEvent, serviceName)
	numberHandler := numberhandler.NewNumberHandler(reqHandler, db, notifyHandler)

	if err := runListen(rabbitSock, numberHandler); err != nil {
		return err
	}

	if err := runSubscribe(rabbitSock, numberHandler); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(rabbitSock rabbitmqhandler.Rabbit, numberHandler numberhandler.NumberHandler) error {
	listenHandler := listenhandler.NewListenHandler(rabbitSock, numberHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameNumberRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(rabbitSock rabbitmqhandler.Rabbit, numberHandler numberhandler.NumberHandler) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameFlowEvent),
		string(commonoutline.QueueNameCustomerEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(rabbitSock, string(commonoutline.QueueNameNumberSubscribe), subscribeTargets, numberHandler)

	// run
	if err := subHandler.Run(); err != nil {
		return err
	}

	return nil
}
