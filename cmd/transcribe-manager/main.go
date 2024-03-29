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
	"github.com/gofrs/uuid"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	commonoutline "gitlab.com/voipbin/bin-manager/common-handler.git/models/outline"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/listenhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/streaminghandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcribehandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcripthandler"
)

const serviceName = "transcribe-manager"

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")
var rabbitQueueListen = flag.String("rabbit_queue_listen", "bin-manager.transcribe-manager.request", "rabbitmq queue name for request listen")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// args for database
var dbDSN = flag.String("dbDSN", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for transcribe-manager.")

// args for redis
var redisAddr = flag.String("redis_addr", "127.0.0.1:6379", "redis address.")
var redisPassword = flag.String("redis_password", "", "redis password")
var redisDB = flag.Int("redis_db", 1, "redis database.")

// gcp info
var gcpCredential = flag.String("gcp_credential", "./google_service_account.json", "the GCP credential file path")

func main() {
	fmt.Printf("Starting transcribe-manager.\n")

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

// run runs the call-manager
func run(db *sql.DB, cache cachehandler.CacheHandler) error {
	if err := runListen(db, cache); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListen",
	})

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	rabbitSock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	// request handler
	reqHandler := requesthandler.NewRequestHandler(rabbitSock, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(rabbitSock, reqHandler, commonoutline.QueueNameTranscribeEvent, commonoutline.ServiceNameTranscribeManager)

	hostID := uuid.Must(uuid.NewV4())
	transcriptHandler := transcripthandler.NewTranscriptHandler(reqHandler, db, notifyHandler, *gcpCredential)
	streamingHandler := streaminghandler.NewStreamingHandler(reqHandler, db, notifyHandler, transcriptHandler, *gcpCredential)
	transcribeHandler := transcribehandler.NewTranscribeHandler(reqHandler, db, notifyHandler, transcriptHandler, streamingHandler, hostID)
	listenHandler := listenhandler.NewListenHandler(hostID, rabbitSock, reqHandler, transcribeHandler, transcriptHandler)

	// run
	listenVolatile := fmt.Sprintf("bin-manager.transcribe-manager-%s.request", hostID)
	if err := listenHandler.Run(*rabbitQueueListen, listenVolatile, string(commonoutline.QueueNameDelay)); err != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
