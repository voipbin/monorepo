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

	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/listenhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallreferencehandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuehandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/subscribehandler"
)

const serviceName = "queue-manager"

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")

var rabbitListenSubscribes = flag.String("rabbit_exchange_subscribes", "bin-manager.call-manager.event", "comma separated rabbitmq exchange name for subscribe")

var rabbitQueueListen = flag.String("rabbit_queue_listen", "bin-manager.queue-manager.request", "rabbitmq queue name for request listen")
var rabbitExchangeNotify = flag.String("rabbit_exchange_notify", "bin-manager.queue-manager.event", "rabbitmq exchange name for event notify")
var rabbitQueueSubscribe = flag.String("rabbit_queue_susbscribe", "bin-manager.queue-manager.subscribe", "rabbitmq queue name for message subscribe queue.")
var rabbitExchangeDelay = flag.String("rabbit_exchange_delay", "bin-manager.delay", "rabbitmq exchange name for delayed messaging.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// args for database
var dbDSN = flag.String("dbDSN", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for queue-manager.")

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

	db := dbhandler.NewHandler(sqlDB, cache)

	if err := run(db); err != nil {
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

// run runs the listen
func run(db dbhandler.DBHandler) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "run",
		},
	)

	// rabbitmq sock connect
	rabbitSock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	// create request-handler
	reqHandler := requesthandler.NewRequestHandler(rabbitSock, serviceName)

	// create notify-handler
	notifyHandler := notifyhandler.NewNotifyHandler(rabbitSock, reqHandler, *rabbitExchangeDelay, *rabbitExchangeNotify)

	// create required handlers
	queuecallReferenceHandler := queuecallreferencehandler.NewQueuecallReferenceHandler(reqHandler, db, notifyHandler)
	queuecallHandler := queuecallhandler.NewQueuecallHandler(reqHandler, db, notifyHandler, queuecallReferenceHandler)
	queueHandler := queuehandler.NewQueueHandler(reqHandler, db, notifyHandler, queuecallHandler, queuecallReferenceHandler)

	// run listen
	if err := runListen(db, rabbitSock, reqHandler, notifyHandler, queueHandler, queuecallHandler, queuecallReferenceHandler); err != nil {
		log.Errorf("Could not run listen. err: %v", err)
		return err
	}

	// run subscribe
	if err := runSubscribe(db, rabbitSock, reqHandler, notifyHandler, queueHandler, queuecallHandler, queuecallReferenceHandler); err != nil {
		log.Errorf("Could not run subscribe. err: %v", err)
		return err
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(
	db dbhandler.DBHandler,
	rabbitSock rabbitmqhandler.Rabbit,
	requestHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	queueHandler queuehandler.QueueHandler,
	queuecallHandler queuecallhandler.QueuecallHandler,
	queuecallReferenceHandler queuecallreferencehandler.QueuecallReferenceHandler,
) error {

	subHandler := subscribehandler.NewSubscribeHandler(
		rabbitSock,
		db,
		*rabbitQueueSubscribe,
		*rabbitListenSubscribes,
		requestHandler,
		notifyHandler,
		queueHandler,
		queuecallHandler,
	)

	// run
	if err := subHandler.Run(*rabbitQueueSubscribe, *rabbitListenSubscribes); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(
	db dbhandler.DBHandler,
	rabbitSock rabbitmqhandler.Rabbit,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	queueHandler queuehandler.QueueHandler,
	queuecallHandler queuecallhandler.QueuecallHandler,
	queuecallReferenceHandler queuecallreferencehandler.QueuecallReferenceHandler,
) error {
	listenHandler := listenhandler.NewListenHandler(rabbitSock, db, reqHandler, queueHandler, queuecallHandler)

	// run
	if err := listenHandler.Run(*rabbitQueueListen, *rabbitExchangeDelay); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
