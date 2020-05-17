package main

import (
	"database/sql"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	joonix "github.com/joonix/log"
	log "github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/listenhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")
var rabbitQueueARIEvent = flag.String("rabbit_queue_arievent", "asterisk_ari_event", "rabbitmq asterisk ari event queue name.")
var rabbitQueueARIRequest = flag.String("rabbit_queue_arirequest", "asterisk_ari_request", "rabbitmq asterisk ari request queue prefix.")
var rabbitQueueFlowRequest = flag.String("rabbit_queue_flow", "bin-manager.flow-manager.request", "rabbitmq queue name for flow request")
var rabbitQueueListen = flag.String("rabbit_queue_listen", "bin-manager.call-manager.request", "rabbitmq queue name for request listen")
var rabbitQueueNotify = flag.String("rabbit_queue_notify", "bin-manager.call-manager.event", "rabbitmq queue name for event notify")

var rabbitExchangeDelay = flag.String("rabbit_exchange_delay", "bin-manager.delay", "rabbitmq exchange name for delayed messaging.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// args for database
var dbDSN = flag.String("dbDSN", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for call-manager.")

// workerCount
var workerCount = flag.Int("worker_count", 3, "counts of workers")

type worker struct {
	rabbitSock rabbitmq.Rabbit

	ariHandler    arihandler.ARIHandler
	reqHandler    requesthandler.RequestHandler
	callHandler   callhandler.CallHandler
	listenHandler listenhandler.ListenHandler

	db dbhandler.DBHandler
}

func main() {
	// connect to database
	sqlDB, err := sql.Open("mysql", *dbDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer sqlDB.Close()

	for i := 0; i < *workerCount; i++ {
		run(sqlDB)
	}
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

	log.Info("init finished.")
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	log.Debugf("Received signal. sig: %v", sig)
	chDone <- true
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

// NewWorker creates worker interface
func run(db *sql.DB) error {
	if err := runARI(db); err != nil {
		return err
	}

	if err := runListen(db); err != nil {
		return err
	}

	return nil
}

func runARI(sqlDB *sql.DB) error {
	// dbhandler
	db := dbhandler.NewHandler(sqlDB)

	// rabbitmq sock connect
	rabbitSock := rabbitmq.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	reqHandler := requesthandler.NewRequestHandler(
		rabbitSock,
		*rabbitExchangeDelay,
		*rabbitQueueListen,
		*rabbitQueueFlowRequest,
	)

	callHandler := callhandler.NewSvcHandler(reqHandler, db)
	ariHandler := arihandler.NewARIHandler(rabbitSock, db, reqHandler, callHandler)

	// run
	if err := ariHandler.Run(*rabbitQueueARIEvent, "call-manager"); err != nil {
		log.Errorf("Could not run the arihandler correctly. err: %v", err)
	}

	return nil
}

func runListen(sqlDB *sql.DB) error {
	// dbhandler
	db := dbhandler.NewHandler(sqlDB)

	// rabbitmq sock connect
	rabbitSock := rabbitmq.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	// request handler
	reqHandler := requesthandler.NewRequestHandler(
		rabbitSock,
		*rabbitExchangeDelay,
		*rabbitQueueListen,
		*rabbitQueueFlowRequest,
	)

	// callHandler := callhandler.NewSvcHandler(reqHandler, db)
	listenHandler := listenhandler.NewListenHandler(rabbitSock, db, reqHandler)
	if err := listenHandler.Run(*rabbitQueueListen, *rabbitExchangeDelay); err != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
