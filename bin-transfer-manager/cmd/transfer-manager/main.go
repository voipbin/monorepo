package main

import (
	"database/sql"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transfer-manager/pkg/cachehandler"
	"monorepo/bin-transfer-manager/pkg/dbhandler"
	"monorepo/bin-transfer-manager/pkg/listenhandler"
	"monorepo/bin-transfer-manager/pkg/subscribehandler"
	"monorepo/bin-transfer-manager/pkg/transferhandler"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")

var rabbitListenSubscribes = flag.String("rabbit_exchange_subscribes", "bin-manager.call-manager.event", "comma separated rabbitmq exchange name for subscribe")

var rabbitQueueListen = flag.String("rabbit_queue_listen", "bin-manager.transfer-manager.request", "rabbitmq queue name for request listen")
var rabbitQueueNotify = flag.String("rabbit_queue_notify", "bin-manager.transfer-manager.event", "rabbitmq exchange name for event notify")
var rabbitQueueSubscribe = flag.String("rabbit_queue_susbscribe", "bin-manager.transfer-manager.subscribe", "rabbitmq queue name for message subscribe queue.")

var rabbitExchangeDelay = flag.String("rabbit_exchange_delay", "bin-manager.delay", "rabbitmq exchange name for delayed messaging.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// args for database
var dbDSN = flag.String("dbDSN", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn.")

// args for redis
var redisAddr = flag.String("redis_addr", "127.0.0.1:6379", "redis address.")
var redisPassword = flag.String("redis_password", "", "redis password")
var redisDB = flag.Int("redis_db", 1, "redis database.")

const (
	serviceName = "transfer-manager"
)

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

// run runs the transfer-manager
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	rabbitSock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(rabbitSock, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(rabbitSock, reqHandler, *rabbitExchangeDelay, *rabbitQueueNotify, serviceName)
	transferHandler := transferhandler.NewTransferHandler(reqHandler, notifyHandler, db)

	// run event listener
	if err := runSubscribe(serviceName, rabbitSock, *rabbitQueueSubscribe, *rabbitListenSubscribes, transferHandler); err != nil {
		return err
	}

	// run request listener
	if err := runRequestListen(rabbitSock, transferHandler); err != nil {
		return err
	}

	return nil
}

// runSubscribe runs the ARI event listen service
func runSubscribe(
	serviceName string,
	rabbitSock rabbitmqhandler.Rabbit,
	subscribeQueue string,
	subscribeTargets string,
	transferHandler transferhandler.TransferHandler,
) error {
	subscribeHandler := subscribehandler.NewSubscribeHandler(serviceName, rabbitSock, subscribeQueue, subscribeTargets, transferHandler)

	// run
	if err := subscribeHandler.Run(); err != nil {
		logrus.Errorf("Could not run the ari event listen handler correctly. err: %v", err)
	}

	return nil
}

// runRequestListen runs the request listen service
func runRequestListen(
	rabbitSock rabbitmqhandler.Rabbit,
	transferHandler transferhandler.TransferHandler,
) error {

	listenHandler := listenhandler.NewListenHandler(
		rabbitSock,
		*rabbitQueueListen,
		*rabbitExchangeDelay,
		transferHandler,
	)

	// run
	if err := listenHandler.Run(); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
