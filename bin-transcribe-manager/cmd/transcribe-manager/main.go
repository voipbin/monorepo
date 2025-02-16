package main

import (
	"database/sql"
	"fmt"
	"os"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/pkg/cachehandler"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/listenhandler"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
	"monorepo/bin-transcribe-manager/pkg/subscribehandler"
	"monorepo/bin-transcribe-manager/pkg/transcribehandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

const serviceName = "transcribe-manager"

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var (
	databaseDSN             = ""
	prometheusEndpoint      = ""
	prometheusListenAddress = ""
	rabbitMQAddress         = ""
	redisAddress            = ""
	redisDatabase           = 0
	redisPassword           = ""
	gcpCredentialBase64     = ""

	awsAccessKey = ""
	awsSecretKey = ""
)

func main() {
	fmt.Printf("Starting transcribe-manager.\n")

	// connect to database
	sqlDB, err := sql.Open("mysql", databaseDSN)
	if err != nil {
		logrus.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer sqlDB.Close()

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		logrus.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	_ = run(sqlDB, cache)
	<-chDone
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// run runs the call-manager
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	hostID := uuid.Must(uuid.NewV4())
	log.Debugf("Generated host id. host_id: %s", hostID)

	listenIP := os.Getenv("POD_IP")
	if listenIP == "" {
		return fmt.Errorf("could not get the listen ip address")
	}
	listenAddress := fmt.Sprintf("%s:%d", listenIP, 8080)

	// create handlers
	db := dbhandler.NewHandler(sqlDB, cache)
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameTranscribeEvent, commonoutline.ServiceNameTranscribeManager)
	transcriptHandler := transcripthandler.NewTranscriptHandler(reqHandler, db, notifyHandler, gcpCredentialBase64)
	streamingHandler := streaminghandler.NewStreamingHandler(reqHandler, notifyHandler, transcriptHandler, listenAddress, gcpCredentialBase64, awsAccessKey, awsSecretKey)
	transcribeHandler := transcribehandler.NewTranscribeHandler(reqHandler, db, notifyHandler, transcriptHandler, streamingHandler, hostID)

	// run request listener
	if err := runListen(sockHandler, hostID, reqHandler, transcriptHandler, transcribeHandler); err != nil {
		return err
	}

	// run subscribe listener
	if errSubscribe := runSubscribe(sockHandler, transcribeHandler); errSubscribe != nil {
		return errSubscribe
	}

	// run streaming listener
	if errStreaming := runStreaming(streamingHandler); errStreaming != nil {
		return errStreaming
	}

	return nil
}

// runListen runs the listen service
func runListen(
	sockHandler sockhandler.SockHandler,
	hostID uuid.UUID,
	reqHandler requesthandler.RequestHandler,
	transcriptHandler transcripthandler.TranscriptHandler,
	transcribeHandler transcribehandler.TranscribeHandler,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListen",
	})

	listenHandler := listenhandler.NewListenHandler(hostID, sockHandler, reqHandler, transcribeHandler, transcriptHandler)

	// run
	listenQueue := fmt.Sprintf("bin-manager.transcribe-manager-%s.request", hostID)
	if err := listenHandler.Run(string(commonoutline.QueueNameTranscribeRequest), listenQueue, string(commonoutline.QueueNameDelay)); err != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// runSubscribe runs the ARI event listen service
func runSubscribe(
	sockHandler sockhandler.SockHandler,
	transcribeHandler transcribehandler.TranscribeHandler,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameCustomerEvent),
	}
	log.WithField("subscribe_targets", subscribeTargets).Debug("Running subscribe handler")

	ariEventListenHandler := subscribehandler.NewSubscribeHandler(sockHandler, commonoutline.QueueNameTranscribeSubscribe, subscribeTargets, transcribeHandler)

	// run
	if err := ariEventListenHandler.Run(); err != nil {
		log.Errorf("Could not run the ari event listen handler correctly. err: %v", err)
	}

	return nil
}

// runStreaming runs the ARI event listen service
func runStreaming(steramingHandler streaminghandler.StreamingHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runStreaming",
	})

	go func() {
		if errRun := steramingHandler.Run(); errRun != nil {
			log.Errorf("Could not run the streaming handler correctly. err: %v", errRun)
		}
	}()

	return nil
}
