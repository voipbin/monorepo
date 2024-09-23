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

	"monorepo/bin-agent-manager/pkg/agenthandler"
	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"
	"monorepo/bin-agent-manager/pkg/listenhandler"
	"monorepo/bin-agent-manager/pkg/subscribehandler"
)

const serviceName = commonoutline.ServiceNameAgentManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// args for database
var dbDSN = flag.String("dbDSN", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for agent-manager.")

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
	log := logrus.WithField("func", "initProm")
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		for {
			if errListen := http.ListenAndServe(listen, nil); errListen != nil {
				log.Errorf("Could not start prometheus listener. err: %v", errListen)
				time.Sleep(time.Second * 1)
				continue
			}
			log.Infof("Finishing the prometheus listener.")
			break
		}
	}()
}

// run runs the listen
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, *rabbitAddr)
	sockHandler.Connect()

	// create handlers
	db := dbhandler.NewHandler(sqlDB, cache)
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameAgentEvent, serviceName)
	agentHandler := agenthandler.NewAgentHandler(reqHandler, db, notifyHandler)

	if err := runListen(sockHandler, agentHandler); err != nil {
		return err
	}

	if err := runSubscribe(sockHandler, agentHandler); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(sockHandler sockhandler.SockHandler, agentHandler agenthandler.AgentHandler) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, agentHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameAgentRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(
	sockHandler sockhandler.SockHandler,
	agentHandler agenthandler.AgentHandler,
) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameCustomerEvent),
		string(commonoutline.QueueNameWebhookEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, string(commonoutline.QueueNameAgentSubscribe), subscribeTargets, agentHandler)

	// run
	if err := subHandler.Run(); err != nil {
		return err
	}

	return nil
}
