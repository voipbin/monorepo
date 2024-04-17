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
	commonoutline "gitlab.com/voipbin/bin-manager/common-handler.git/models/outline"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencecallhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/listenhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/subscribehandler"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// args for database
var dbDSN = flag.String("dbDSN", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for conference-manager.")

// args for redis
var redisAddr = flag.String("redis_addr", "127.0.0.1:6379", "redis address.")
var redisPassword = flag.String("redis_password", "", "redis password")
var redisDB = flag.Int("redis_db", 1, "redis database.")

const (
	serviceName = commonoutline.ServiceNameConferenceManager
)

func main() {
	log := logrus.WithField("func", "main")
	log.Info("Starting conference-manager.")

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

	_ = run(sqlDB, cache)
	<-chDone

	log.Info("Finishing conference-manager.")
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

// run runs the main thread.
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithField("func", "run")

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	rabbitSock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	requestHandler := requesthandler.NewRequestHandler(rabbitSock, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(rabbitSock, requestHandler, commonoutline.QueueNameConferenceEvent, serviceName)

	conferenceHandler := conferencehandler.NewConferenceHandler(requestHandler, notifyHandler, db)
	conferencecallHandler := conferencecallhandler.NewConferencecallHandler(requestHandler, notifyHandler, db, conferenceHandler)

	// run listen
	if err := runListen(rabbitSock, conferenceHandler, conferencecallHandler); err != nil {
		log.Errorf("Could not start runListen. err: %v", err)
		return err
	}

	// run subscribe
	if err := runSubscribe(rabbitSock, conferenceHandler, conferencecallHandler); err != nil {
		log.Errorf("Could not start runSubscribe. err: %v", err)
		return err
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(
	rabbitSock rabbitmqhandler.Rabbit,
	conferenceHandler conferencehandler.ConferenceHandler,
	conferencecallHandler conferencecallhandler.ConferencecallHandler,
) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(
		rabbitSock,
		string(commonoutline.QueueNameConferenceSubscribe),
		subscribeTargets,
		conferenceHandler,
		conferencecallHandler,
	)

	// run
	if err := subHandler.Run(); err != nil {
		logrus.Errorf("Could not run the subscribehandler correctly. err: %v", err)
		return err
	}

	return nil
}

// runListen runs the listen handler
func runListen(
	rabbitSock rabbitmqhandler.Rabbit,
	conferenceHandler conferencehandler.ConferenceHandler,
	conferencecallHandler conferencecallhandler.ConferencecallHandler,
) error {

	listenHandler := listenhandler.NewListenHandler(
		rabbitSock,
		string(commonoutline.QueueNameConferenceRequest),
		string(commonoutline.QueueNameDelay),
		conferenceHandler,
		conferencecallHandler,
	)

	// run
	if err := listenHandler.Run(); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
		return err
	}

	return nil
}
