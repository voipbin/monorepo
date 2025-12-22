package main

import (
	"database/sql"
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
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"monorepo/bin-customer-manager/internal/config"
	"monorepo/bin-customer-manager/pkg/accesskeyhandler"
	"monorepo/bin-customer-manager/pkg/cachehandler"
	"monorepo/bin-customer-manager/pkg/customerhandler"
	"monorepo/bin-customer-manager/pkg/dbhandler"
	"monorepo/bin-customer-manager/pkg/listenhandler"
)

const serviceName = commonoutline.ServiceNameCustomerManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	if errInit := config.InitAll(); errInit != nil {
		logrus.Fatalf("Could not init config. err: %v", errInit)
	}
	config.ParseFlags()

	initSignal()
	initProm(config.GlobalConfig.PrometheusEndpoint, config.GlobalConfig.PrometheusListenAddr)

	log := logrus.WithField("func", "main")
	log.WithField("config", config.GlobalConfig).Debugf("Hello world. The customer-manager is running...")

	sqlDB, err := initDatabase()
	if err != nil {
		log.Errorf("Could not init the database. err: %v", err)
		return
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	cache, err := initCache()
	if err != nil {
		log.Errorf("Could not init the cache. err: %v", err)
		return
	}

	if err := run(sqlDB, cache); err != nil {
		log.Errorf("Run func has finished. err: %v", err)
	}
	<-chDone
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// run runs the listen
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.GlobalConfig.RabbitMQAddress)
	sockHandler.Connect()

	// create handler
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameCustomerEvent, serviceName)
	customerHandler := customerhandler.NewCustomerHandler(reqHandler, db, notifyHandler)
	accesskeyHandler := accesskeyhandler.NewAccesskeyHandler(reqHandler, db, notifyHandler)

	// run listen
	if err := runListen(sockHandler, reqHandler, customerHandler, accesskeyHandler); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(
	sockHandler sockhandler.SockHandler,
	reqHandler requesthandler.RequestHandler,
	customerHandler customerhandler.CustomerHandler,
	accesskeyHandler accesskeyhandler.AccesskeyHandler,
) error {

	listenHandler := listenhandler.NewListenHandler(sockHandler, reqHandler, customerHandler, accesskeyHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameCustomerRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

func initDatabase() (*sql.DB, error) {
	res, err := sql.Open("mysql", config.GlobalConfig.DatabaseDSN)
	if err != nil {
		return nil, errors.Wrapf(err, "could not access to the database")
	} else if err := res.Ping(); err != nil {
		return nil, errors.Wrapf(err, "could not set the database connection correctly")
	}

	return res, nil
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.GlobalConfig.RedisAddress, config.GlobalConfig.RedisPassword, config.GlobalConfig.RedisDatabase)
	if err := res.Connect(); err != nil {
		return nil, errors.Wrapf(err, "could not connect to cache server")
	}

	return res, nil
}

// initSignal inits signal settings.
func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalHandler()
}

func initProm(endpoint, listen string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		for {
			err := http.ListenAndServe(listen, nil)
			if err != nil {
				logrus.Errorf("Could not start prometheus listener: %v", err)
				time.Sleep(time.Second * 1)
				continue
			}
			break
		}
	}()
}
