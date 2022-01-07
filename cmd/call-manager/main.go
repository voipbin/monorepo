package main

import (
	"database/sql"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/arihandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/confbridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")
var rabbitQueueARIEvent = flag.String("rabbit_queue_arievent", "asterisk.all.event", "rabbitmq asterisk ari event queue name.")
var rabbitQueueListen = flag.String("rabbit_queue_listen", "bin-manager.call-manager.request", "rabbitmq queue name for request listen")

var rabbitExchangeNotify = flag.String("rabbit_exchange_notify", "bin-manager.call-manager.event", "rabbitmq exchange name for event notify")
var rabbitExchangeDelay = flag.String("rabbit_exchange_delay", "bin-manager.delay", "rabbitmq exchange name for delayed messaging.")

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

	if err := run(sqlDB, cache); err != nil {
		log.Errorf("Run func has finished. err: %v", err)
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
	reqHandler := requesthandler.NewRequestHandler(rabbitSock, "call-manager")
	notifyHandler := notifyhandler.NewNotifyHandler(rabbitSock, reqHandler, *rabbitExchangeDelay, *rabbitExchangeNotify)
	callHandler := callhandler.NewCallHandler(reqHandler, notifyHandler, db, cache)
	confbridgeHandler := confbridgehandler.NewConfbridgeHandler(reqHandler, notifyHandler, db, cache)

	if err := runARI(sqlDB, cache); err != nil {
		return err
	}

	if err := runListen(rabbitSock, callHandler, confbridgeHandler); err != nil {
		return err
	}

	return nil
}

// runARI runs the ARI listen service
func runARI(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	rabbitSock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	reqHandler := requesthandler.NewRequestHandler(rabbitSock, "call-manager-ari")

	notifyHandler := notifyhandler.NewNotifyHandler(rabbitSock, reqHandler, *rabbitExchangeDelay, *rabbitExchangeNotify)
	callHandler := callhandler.NewCallHandler(reqHandler, notifyHandler, db, cache)
	confbridgeHandler := confbridgehandler.NewConfbridgeHandler(reqHandler, notifyHandler, db, cache)
	eventHandler := arihandler.NewEventHandler(rabbitSock, db, cache, reqHandler, notifyHandler, callHandler, confbridgeHandler)

	// run
	if err := eventHandler.Run(*rabbitQueueARIEvent, "call-manager"); err != nil {
		logrus.Errorf("Could not run the arihandler correctly. err: %v", err)
	}

	return nil
}

// runListen runs the listen service
func runListen(rabbitSock rabbitmqhandler.Rabbit, callHandler callhandler.CallHandler, confbridgeHandler confbridgehandler.ConfbridgeHandler) error {
	listenHandler := listenhandler.NewListenHandler(rabbitSock, callHandler, confbridgeHandler)

	// run
	if err := listenHandler.Run(*rabbitQueueListen, *rabbitExchangeDelay); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
