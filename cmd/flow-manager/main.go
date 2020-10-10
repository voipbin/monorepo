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
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/listenhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/requesthandler"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// log level
var logLevel = flag.Int("log_level", int(log.DebugLevel), "log level")

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")
var rabbitQueueListen = flag.String("rabbit_queue_listen", "bin-manager.flow-manager.request", "rabbitmq queue name for request listen")
var rabbitQueueEvent = flag.String("rabbit_queue_event", "bin-manager.flow-manager.event", "rabbitmq queue name for event notify")
var rabbitExchangeDelay = flag.String("rabbit_exchange_delay", "bin-manager.delay", "rabbitmq exchange name for delayed messaging.")

var rabbitQueueFlowRequest = flag.String("rabbit_queue_flow", "bin-manager.flow-manager.request", "rabbitmq queue name for flow request")
var rabbitQueueCallRequest = flag.String("rabbit_queue_call", "bin-manager.call-manager.request", "rabbitmq queue name for request listen")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// args for database
var dbDSN = flag.String("dbDSN", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for flow-manager.")

// args for redis
var redisAddr = flag.String("redis_addr", "127.0.0.1:6379", "redis address.")
var redisPassword = flag.String("redis_password", "", "redis password")
var redisDB = flag.Int("redis_db", 1, "redis database.")

func main() {
	fmt.Printf("Hello world!\n")

	// connect to database
	db, err := sql.Open("mysql", *dbDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	}

	// connect to cache
	cache := cachehandler.NewHandler(*redisAddr, *redisPassword, *redisDB)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	run(db, cache)
	<-chDone

	return
}

func init() {
	flag.Parse()

	// init logs
	initLog()

	// init signal handler
	initSignal()

	// init prometheus setting
	initProm(*promEndpoint, *promListenAddr)

	log.Info("The init finished.")
}

// initLog inits log settings.
func initLog() {
	log.SetFormatter(joonix.NewFormatter())
	log.SetLevel(log.DebugLevel)
}

// initSignal inits sinal settings.
func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
	go signalHandler()
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	log.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// initProm inits prometheus settings
func initProm(endpoint, listen string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		for {
			err := http.ListenAndServe(listen, nil)
			if err != nil {
				log.Errorf("Could not start prometheus listener")
				time.Sleep(time.Second * 1)
				continue
			}
			break
		}
	}()
}

func run(db *sql.DB, cache cachehandler.CacheHandler) {
	// rabbitmq handler
	rabbitSock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	// database handler
	dbHandler := dbhandler.NewHandler(db, cache)

	// connect to rabbitmq
	sock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	sock.Connect()

	// create servicehandler
	requestHandler := requesthandler.NewRequestHandler(sock, *rabbitExchangeDelay, *rabbitQueueCallRequest, *rabbitQueueFlowRequest)

	// flow
	flowHandler := flowhandler.NewFlowHandler(dbHandler, requestHandler)

	// create and run the listen handler
	// listen to the request queue
	listenHandler := listenhandler.NewListenHandler(
		rabbitSock,
		dbHandler,
		flowHandler,
	)
	listenHandler.Run(*rabbitQueueListen, *rabbitExchangeDelay)
}
