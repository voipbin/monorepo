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
	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
	"monorepo/bin-webhook-manager/pkg/dbhandler"
	"monorepo/bin-webhook-manager/pkg/listenhandler"
	"monorepo/bin-webhook-manager/pkg/webhookhandler"
)

const serviceName = commonoutline.ServiceNameWebhookManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// args for database
var dbDSN = flag.String("dbDSN", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for webhook-manager.")

// args for redis
var redisAddr = flag.String("redis_addr", "127.0.0.1:6379", "redis address.")
var redisPassword = flag.String("redis_password", "", "redis password")
var redisDB = flag.Int("redis_db", 1, "redis database.")

func main() {

	logrus.Info("Starting webhook-manager.")

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

	if errRun := run(sqlDB, cache); errRun != nil {
		logrus.Errorf("Could not run correctly. err: %v", errRun)
		return
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

	logrus.Info("Init finished.")
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

// run runs the webhook-manager
func run(db *sql.DB, cache cachehandler.CacheHandler) error {

	// run listen
	if err := runListen(db, cache); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen handler
func runListen(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, *rabbitAddr)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameWebhookEvent, serviceName)
	accountHandler := accounthandler.NewAccountHandler(db, reqHandler)

	whHandler := webhookhandler.NewWebhookHandler(db, notifyHandler, accountHandler)

	listenHandler := listenhandler.NewListenHandler(sockHandler, whHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameWebhookRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
