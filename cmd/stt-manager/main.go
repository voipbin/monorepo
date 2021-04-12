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
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/listenhandler"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/stthandler"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")
var rabbitQueueListen = flag.String("rabbit_queue_listen", "bin-manager.stt-manager.request", "rabbitmq queue name for request listen")
var rabbitQueueNotify = flag.String("rabbit_queue_notify", "bin-manager.stt-manager.event", "rabbitmq queue name for event notify")

var rabbitQueueCallRequest = flag.String("rabbit_queue_call", "bin-manager.call-manager.request", "rabbitmq queue name for call-manager request")
var rabbitQueueStorageRequest = flag.String("rabbit_queue_storage", "bin-manager.storage-manager.request", "rabbitmq queue name for storage-manager request")
var rabbitQueueWebhookRequest = flag.String("rabbit_queue_webhook", "bin-manager.webhook-manager.request", "rabbitmq queue name for webhook-manager request")

var rabbitExchangeDelay = flag.String("rabbit_exchange_delay", "bin-manager.delay", "rabbitmq exchange name for delayed messaging.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// args for database
var dbDSN = flag.String("dbDSN", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for stt-manager.")

// args for redis
var redisAddr = flag.String("redis_addr", "127.0.0.1:6379", "redis address.")
var redisPassword = flag.String("redis_password", "", "redis password")
var redisDB = flag.Int("redis_db", 1, "redis database.")

// gcp info
var gcpCredential = flag.String("gcp_credential", "./credential.json", "the GCP credential file path")
var gcpProjectID = flag.String("gcp_project_id", "project", "the gcp project id")
var gcpBucketName = flag.String("gcp_bucket_name", "bucket", "the gcp bucket name to use")

func main() {
	fmt.Printf("Starting stt-manager.\n")

	// connect to database
	sqlDB, err := sql.Open("mysql", *dbDSN)
	if err != nil {
		logrus.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer sqlDB.Close()

	// connect to cache
	cache := cachehandler.NewHandler(*redisAddr, *redisPassword, *redisDB)
	if err := cache.Connect(); err != nil {
		logrus.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	run(sqlDB, cache)
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

// run runs the call-manager
func run(db *sql.DB, cache cachehandler.CacheHandler) error {
	if err := runListen(db, cache); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	rabbitSock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	// request handler
	reqHandler := requesthandler.NewRequestHandler(
		rabbitSock,
		*rabbitExchangeDelay,
		*rabbitQueueCallRequest,
		*rabbitQueueStorageRequest,
		*rabbitQueueWebhookRequest,
	)

	sttHandler := stthandler.NewSTTHandler(reqHandler, db, cache, *gcpCredential, *gcpProjectID, *gcpBucketName)
	listenHandler := listenhandler.NewListenHandler(rabbitSock, reqHandler, sttHandler)

	// run
	if err := listenHandler.Run(*rabbitQueueListen, *rabbitExchangeDelay); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
