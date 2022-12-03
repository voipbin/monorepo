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
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/listenhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandler"
)

const serviceName = "number-manager"

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var argRabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")
var argRabbitQueueListen = flag.String("rabbit_queue_listen", "bin-manager.number-manager.request", "rabbitmq queue name for request listen")
var argRabbitExchangeNotify = flag.String("rabbit_queue_event", "bin-manager.number-manager.event", "rabbitmq queue name for event notify")
var argRabbitExchangeDelay = flag.String("rabbit_exchange_delay", "bin-manager.delay", "rabbitmq exchange name for delayed messaging.")

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

	// create db handler
	db, err := createDBHandler(sqlDB, *argRedisAddr, *argRedisPassword, *argRedisDB)
	if err != nil {
		log.Errorf("Could not create dbhandler. err: %v", err)
		return
	}

	// create rabbit sock
	sock := createRabbitSock(*argRabbitAddr)

	if errRun := run(db, sock); errRun != nil {
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

// createDBHandler create the dbhandler and returns created dbhandler.
// it connects to the database and cache.
func createDBHandler(sqlDB *sql.DB, redisAddr, redisPassword string, redisDB int) (dbhandler.DBHandler, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "createDBHandler",
	})

	// connect to cache
	cache := cachehandler.NewHandler(redisAddr, redisPassword, redisDB)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return nil, err
	}

	// create dbhandler
	res := dbhandler.NewHandler(sqlDB, cache)
	return res, nil
}

// createRabbitSock create rabbitmq socket
func createRabbitSock(rabbitAddr string) rabbitmqhandler.Rabbit {
	res := rabbitmqhandler.NewRabbit(rabbitAddr)
	res.Connect()

	return res
}

// run runs the service
func run(db dbhandler.DBHandler, sock rabbitmqhandler.Rabbit) error {
	if err := runListen(db, sock); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(db dbhandler.DBHandler, sock rabbitmqhandler.Rabbit) error {

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sock, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sock, reqHandler, *argRabbitExchangeDelay, *argRabbitExchangeNotify, serviceName)
	numberHandler := numberhandler.NewNumberHandler(reqHandler, db, notifyHandler)
	listenHandler := listenhandler.NewListenHandler(sock, numberHandler)

	// run
	if err := listenHandler.Run(*argRabbitQueueListen, *argRabbitExchangeDelay); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
