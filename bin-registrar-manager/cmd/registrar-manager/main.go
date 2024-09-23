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

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"monorepo/bin-registrar-manager/pkg/cachehandler"
	"monorepo/bin-registrar-manager/pkg/contacthandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/listenhandler"
	"monorepo/bin-registrar-manager/pkg/subscribehandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"
)

const serviceName = commonoutline.ServiceNameRegistrarManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")

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
	log := logrus.WithFields(logrus.Fields{
		"func": "main",
	})
	fmt.Printf("hello world\n")

	// connect to the database asterisk
	sqlAst, err := sql.Open("mysql", *dbDSNAst)
	if err != nil {
		log.Errorf("Could not access to database asterisk. err: %v", err)
		return
	}
	defer sqlAst.Close()

	// connect to the database bin-manager
	sqlBin, err := sql.Open("mysql", *dbDSNBin)
	if err != nil {
		log.Errorf("Could not access to database bin-manager. err: %v", err)
		return
	}
	defer sqlBin.Close()

	// connect to cache
	cache := cachehandler.NewHandler(*redisAddr, *redisPassword, *redisDB)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	if errRun := run(sqlAst, sqlBin, cache); errRun != nil {
		log.Errorf("Could not run. err: %v", errRun)
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

// NewWorker creates worker interface
func run(sqlAst *sql.DB, sqlBin *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})

	dbAst := dbhandler.NewHandler(sqlAst, cache)
	dbBin := dbhandler.NewHandler(sqlBin, cache)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, *rabbitAddr)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameRegistrarEvent, serviceName)
	extensionHandler := extensionhandler.NewExtensionHandler(reqHandler, dbAst, dbBin, notifyHandler)
	trunkHandler := trunkhandler.NewTrunkHandler(reqHandler, dbBin, notifyHandler)
	contactHandler := contacthandler.NewContactHandler(reqHandler, dbAst, dbBin)

	// run listen
	if errListen := runListen(sockHandler, reqHandler, trunkHandler, extensionHandler, contactHandler); errListen != nil {
		log.Errorf("Could not run the listener. err: %v", errListen)
		return errListen
	}

	// run subscriber
	if errSubscribe := runSubscribe(sockHandler, extensionHandler, trunkHandler); errSubscribe != nil {
		log.Errorf("Could not run the subscriber. err: %v", errSubscribe)
		return errSubscribe
	}

	return nil
}

// runListen runs the listen service
func runListen(sockHandler sockhandler.SockHandler, reqHandler requesthandler.RequestHandler, trunkHandler trunkhandler.TrunkHandler, extensionHandler extensionhandler.ExtensionHandler, contactHandler contacthandler.ContactHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListen",
	})

	listenHandler := listenhandler.NewListenHandler(sockHandler, reqHandler, trunkHandler, extensionHandler, contactHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameRegistrarRequest), string(commonoutline.QueueNameDelay)); err != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(sockHandler sockhandler.SockHandler, extensionHandler extensionhandler.ExtensionHandler, trunkHandler trunkhandler.TrunkHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})

	subscribeTargets := []string{
		string(commonoutline.QueueNameCustomerEvent),
	}
	log.WithField("subscribe_targets", subscribeTargets).Debugf("Subscribe target details. len: %d", len(subscribeTargets))

	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, string(commonoutline.QueueNameRegistrarSubscribe), subscribeTargets, extensionHandler, trunkHandler)

	// run
	if err := subHandler.Run(); err != nil {
		log.Errorf("Could not run the subscribehandler correctly. err: %v", err)
		return err
	}

	return nil
}
