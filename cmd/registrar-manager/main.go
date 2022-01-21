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

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/contacthandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/domainhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/extensionhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/listenhandler"
)

const serviceName = "registrar-manager"

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")
var rabbitQueueListen = flag.String("rabbit_queue_listen", "bin-manager.registrar-manager.request", "rabbitmq queue name for request listen")
var rabbitQueueNotify = flag.String("rabbit_queue_notify", "bin-manager.registrar-manager.event", "rabbitmq queue name for event notify")

var rabbitExchangeDelay = flag.String("rabbit_exchange_delay", "bin-manager.delay", "rabbitmq exchange name for delayed messaging.")

// args for database
var dbDSNAst = flag.String("dbDSNAst", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for asterisk.")
var dbDSNBin = flag.String("dbDSNBin", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for bin-manager.")

// args for redis
var redisAddr = flag.String("redis_addr", "127.0.0.1:6379", "redis address.")
var redisPassword = flag.String("redis_password", "", "redis password")
var redisDB = flag.Int("redis_db", 1, "redis database.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

func main() {
	fmt.Printf("hello world\n")

	// connect to the database asterisk
	sqlAst, err := sql.Open("mysql", *dbDSNAst)
	if err != nil {
		logrus.Errorf("Could not access to database asterisk. err: %v", err)
		return
	}
	defer sqlAst.Close()

	// connect to the database bin-manager
	sqlBin, err := sql.Open("mysql", *dbDSNBin)
	if err != nil {
		logrus.Errorf("Could not access to database bin-manager. err: %v", err)
		return
	}
	defer sqlBin.Close()

	// connect to cache
	cache := cachehandler.NewHandler(*redisAddr, *redisPassword, *redisDB)
	if err := cache.Connect(); err != nil {
		logrus.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	run(sqlAst, sqlBin, cache)
	<-chDone

	return

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
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
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

// NewWorker creates worker interface
func run(sqlAst *sql.DB, sqlBin *sql.DB, cache cachehandler.CacheHandler) error {

	dbAst := dbhandler.NewHandler(sqlAst, cache)
	dbBin := dbhandler.NewHandler(sqlBin, cache)

	rabbitSock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(rabbitSock, serviceName)
	extensionHandler := extensionhandler.NewExtensionHandler(reqHandler, dbAst, dbBin, cache)
	domainHandler := domainhandler.NewDomainHandler(reqHandler, dbAst, dbBin, cache, extensionHandler)
	contactHandler := contacthandler.NewContactHandler(reqHandler, dbAst, dbBin, cache)
	listenHandler := listenhandler.NewListenHandler(rabbitSock, reqHandler, domainHandler, extensionHandler, contactHandler)

	// run
	if err := listenHandler.Run(*rabbitQueueListen, *rabbitExchangeDelay); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
