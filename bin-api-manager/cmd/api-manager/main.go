package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// swagger embed files
	// gin-swagger middleware
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/lib/service"
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/cachehandler"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/servicehandler"
	"monorepo/bin-api-manager/pkg/streamhandler"
	"monorepo/bin-api-manager/pkg/subscribehandler"
	"monorepo/bin-api-manager/pkg/websockhandler"
	"monorepo/bin-api-manager/pkg/zmqpubhandler"
	"monorepo/bin-api-manager/server"
)

// channels
var chSigs = make(chan os.Signal, 1)

const (
	constSSLPrivFilename = "/tmp/ssl_privkey.pem"
	constSSLCertFilename = "/tmp/ssl_cert.pem"
	defaultAudiosockPort = 9000
)

var (
	databaseDSN             = ""
	gcpBucketName           = ""
	gcpCredentialBase64     = ""
	gcpProjectID            = ""
	jwtKey                  = ""
	prometheusEndpoint      = ""
	prometheusListenAddress = ""
	rabbitMQAddress         = ""
	redisAddress            = ""
	redisDatabase           = 0
	redisPassword           = ""
	sslCertBase64           = ""
	sslPrivkeyBase64        = ""
	listenIPAudiosock       = ""
)

//	@title			VoIPBIN project API
//	@version		3.1.0
//	@description	RESTful API documents for VoIPBIN project.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	VoIPBIN Project
//	@contact.email	pchero21@gmail.com

// @host	api.voipbin.net
// @BasePath
func main() {
	ctx := context.Background()

	log := logrus.WithField("func", "main")

	// connect to database
	sqlDB, err := sql.Open("mysql", databaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer sqlDB.Close()

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// connect to rabbitmq
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	run(ctx, sockHandler, db)

	sig := <-chSigs
	log.Infof("Terminating api-manager. sig: %v", sig)
}

func writeBase64(filename string, data string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "writeBase64",
		"filename": filename,
		"data":     data,
	})

	// Create or open the file
	file, err := os.Create(filename)
	if err != nil {
		log.Errorf("Could not create a file. err: %v", err)
		return err
	}
	defer file.Close()

	tmp, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Fatalf("Error decoding Base64 string: %v", err)
	}

	// Write the string data to the file
	_, err = file.Write(tmp)
	if err != nil {
		log.Errorf("Could not write to file. err: %v", err)
		return err
	}

	return nil
}

func run(
	ctx context.Context,
	sockHandler sockhandler.SockHandler,
	db dbhandler.DBHandler,
) {
	addressListenStream := getAddressListenAudiosock()

	// create handlers
	requestHandler := requesthandler.NewRequestHandler(sockHandler, "api_manager")
	zmqPubHandler := zmqpubhandler.NewZMQPubHandler()
	streamHandler := streamhandler.NewStreamHandler(requestHandler, addressListenStream)
	websockHandler := websockhandler.NewWebsockHandler(requestHandler, streamHandler)
	serviceHandler := servicehandler.NewServiceHandler(requestHandler, db, websockHandler, gcpCredentialBase64, gcpProjectID, gcpBucketName, jwtKey)

	go runSubscribe(sockHandler, zmqPubHandler)
	go runListenHTTP(serviceHandler)
	go runListenStreamsock(ctx, streamHandler)

}

func runSubscribe(
	sockHandler sockhandler.SockHandler,
	zmqHandler zmqpubhandler.ZMQPubHandler,
) {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})

	queueNamePod := fmt.Sprintf("%s-%s", commonoutline.QueueNameAPISubscribe, uuid.Must(uuid.NewV4()))

	subscribeTargets := []string{
		string(commonoutline.QueueNameWebhookEvent),
		string(commonoutline.QueueNameAgentEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(
		sockHandler,
		queueNamePod,
		subscribeTargets,

		zmqHandler,
	)

	if errRun := subHandler.Run(); errRun != nil {
		log.Errorf("Could not run the subscribe handler. err: %v", errRun)
		return
	}
}

func runListenHTTP(serviceHandler servicehandler.ServiceHandler) {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListenHTTP",
	})

	app := gin.Default()

	// documents
	app.Static("/docs", "docsdev/build")
	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	app.GET("/redoc/*any", func(c *gin.Context) {
		c.File("gens/openapi_redoc/api.html")
	})

	// CORS setting
	// CORS for https://foo.com and https://github.com origins, allowing:
	// - PUT and PATCH methods
	// - Origin header
	// - Credentials share
	// - Preflight requests cached for 12 hours
	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"POST", "GET", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "X-Requested-With", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// inject servicehandler
	app.Use(func(c *gin.Context) {
		c.Set(common.OBJServiceHandler, serviceHandler)
		c.Next()
	})

	// register basic services
	app.GET("/ping", service.GetPing)
	auth := app.Group("/auth")
	auth.POST("/login", service.PostLogin)

	appServer := server.NewServer(serviceHandler)

	v1 := app.RouterGroup.Group("v1.0")
	v1.Use(middleware.Authenticate())
	openapi_server.RegisterHandlers(v1, appServer)

	// // inject servicehandler
	// app.Use(func(c *gin.Context) {
	// 	c.Set(common.OBJServiceHandler, serviceHandler)
	// 	c.Next()
	// })

	// // apply api router
	// api.ApplyRoutes(app)

	logrus.Debug("Starting the api service.")
	if errAppRun := app.RunTLS(":443", constSSLCertFilename, constSSLPrivFilename); errAppRun != nil {
		log.Errorf("The api service ended with error. err: %v", errAppRun)
	}
}

// func runListenHTTPOld(serviceHandler servicehandler.ServiceHandler) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func": "runListenHTTP",
// 	})

// 	app := gin.Default()

// 	// documents
// 	app.Static("/docs", "docsdev/build")
// 	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
// 	app.GET("/redoc/*any", func(c *gin.Context) {
// 		c.File("gens/openapi_redoc/api.html")
// 	})

// 	// CORS setting
// 	// CORS for https://foo.com and https://github.com origins, allowing:
// 	// - PUT and PATCH methods
// 	// - Origin header
// 	// - Credentials share
// 	// - Preflight requests cached for 12 hours
// 	app.Use(cors.New(cors.Config{
// 		AllowOrigins:     []string{"*"},
// 		AllowMethods:     []string{"POST", "GET", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
// 		AllowHeaders:     []string{"Origin", "X-Requested-With", "Content-Type", "Accept", "Authorization"},
// 		ExposeHeaders:    []string{"Content-Length"},
// 		AllowCredentials: false,
// 		MaxAge:           12 * time.Hour,
// 	}))

// 	// inject servicehandler
// 	app.Use(func(c *gin.Context) {
// 		c.Set(common.OBJServiceHandler, serviceHandler)
// 		c.Next()
// 	})

// 	// apply api router
// 	api.ApplyRoutes(app)

// 	logrus.Debug("Starting the api service.")
// 	if errAppRun := app.RunTLS(":443", constSSLCertFilename, constSSLPrivFilename); errAppRun != nil {
// 		log.Errorf("The api service ended with error. err: %v", errAppRun)
// 	}
// }

func runListenStreamsock(ctx context.Context, streamHandler streamhandler.StreamHandler) {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListenAudiosock",
	})

	listenAddress := fmt.Sprintf("%s:%d", listenIPAudiosock, defaultAudiosockPort)
	log.Debugf("Listening audiosock address. address: %s", listenAddress)

	addr, err := net.ResolveTCPAddr("tcp", listenAddress)
	if err != nil {
		log.Errorf("Could not resovle the address. err: %v", err)
		return
	}

	listen, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Errorf("Could not listen the address. err: %v", err)
		return
	}
	defer listen.Close()

	for {
		if ctx.Err() != nil {
			return
		}

		conn, err := listen.Accept()
		if err != nil {
			log.Errorf("Could not accept the connection. err: %v", err)
			continue
		}

		go streamHandler.Process(conn)
	}
}

func getAddressListenAudiosock() string {
	res := fmt.Sprintf("%s:%d", listenIPAudiosock, defaultAudiosockPort)

	return res
}
