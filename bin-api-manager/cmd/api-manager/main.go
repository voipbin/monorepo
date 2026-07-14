package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// swagger embed files
	// gin-swagger middleware
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/internal/config"
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

const (
	constSSLPrivFilename = "/tmp/ssl_privkey.pem"
	constSSLCertFilename = "/tmp/ssl_cert.pem"
	defaultAudiosockPort = 9000
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
	rootCmd := &cobra.Command{
		Use:   "api-manager",
		Short: "VoIPBIN API Manager - External REST API gateway",
		Long:  "External REST API gateway with JWT authentication, Swagger UI, and microservice orchestration",
		Run:   runDaemon,
	}

	if errBootstrap := config.Bootstrap(rootCmd); errBootstrap != nil {
		logrus.Fatalf("Could not bootstrap config. err: %v", errBootstrap)
	}

	config.LoadGlobalConfig()

	if errPostBootstrap := config.PostBootstrap(); errPostBootstrap != nil {
		logrus.Fatalf("Could not complete post-bootstrap. err: %v", errPostBootstrap)
	}

	if errExecute := rootCmd.Execute(); errExecute != nil {
		logrus.Fatalf("Could not execute command. err: %v", errExecute)
	}
}

func runDaemon(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	log := logrus.WithField("func", "runDaemon")

	cfg := config.Get()

	// connect to database
	sqlDB, err := commondatabasehandler.Connect(cfg.DatabaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer commondatabasehandler.Close(sqlDB)

	// connect to cache
	cache := cachehandler.NewHandler(cfg.RedisAddress, cfg.RedisPassword, cfg.RedisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// connect to rabbitmq
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	run(ctx, sockHandler, db)

	// Wait for termination signal
	chSigs := make(chan os.Signal, 1)
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	sig := <-chSigs
	log.Infof("Terminating api-manager. sig: %v", sig)
}

func run(
	ctx context.Context,
	sockHandler sockhandler.SockHandler,
	db dbhandler.DBHandler,
) {
	log := logrus.WithField("func", "run")

	cfg := config.Get()
	addressListenStream := getAddressListenAudiosock()

	// create handlers
	requestHandler := requesthandler.NewRequestHandler(sockHandler, "api_manager")
	zmqPubHandler := zmqpubhandler.NewZMQPubHandler()
	streamHandler := streamhandler.NewStreamHandler(requestHandler, addressListenStream)

	// per-pod subscribe queue name -- constructed here (before websockHandler) so it can be
	// shared with websockhandler's scopeRefCount for dynamic AMQP bind/unbind (VOIP-1258 §9).
	queueNamePod := fmt.Sprintf("%s-%s", commonoutline.QueueNameAPISubscribe, uuid.Must(uuid.NewV4()))

	websockHandler := websockhandler.NewWebsockHandler(requestHandler, streamHandler, sockHandler, queueNamePod)
	serviceHandler, err := servicehandler.NewServiceHandler(requestHandler, db, websockHandler, cfg.GCPProjectID, cfg.GCPBucketName, cfg.JWTKey)
	if err != nil {
		log.Fatalf("Could not create service handler. err: %v", err)
	}

	go runSubscribe(sockHandler, requestHandler, zmqPubHandler, queueNamePod)
	go runListenHTTP(serviceHandler)
	go runListenStreamsock(ctx, streamHandler)

}

func runSubscribe(
	sockHandler sockhandler.SockHandler,
	reqHandler requesthandler.RequestHandler,
	zmqHandler zmqpubhandler.ZMQPubHandler,
	queueNamePod string,
) {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})

	subscribeTargets := []string{}
	subHandler := subscribehandler.NewSubscribeHandler(
		sockHandler,
		reqHandler,
		queueNamePod,
		subscribeTargets,

		zmqHandler,
	)

	if errRun := subHandler.Run(); errRun != nil {
		log.Errorf("Could not run the subscribe handler. err: %v", errRun)
		return
	}

	// baseline "#" wildcard binding to the new topic exchange (VOIP-1258 §7 round-2 finding):
	// a topic-kind exchange's empty-key bind (what QueueSubscribe used for the old fanout
	// exchange) only matches messages published with an empty routing key, and every
	// VOIP-1258 publish path uses non-empty scope-first keys, so this pod would receive ZERO
	// events without this explicit bind. MUST run AFTER subHandler.Run() above, since that is
	// what actually declares queueNamePod via QueueCreate -- QueueBind against a queue that
	// does not exist yet fails (see rabbitmqhandler.TestQueueBind_ReturnsErrorForNonExistent).
	if err := sockHandler.QueueBind(queueNamePod, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil); err != nil {
		logrus.Errorf("Could not bind to the topic exchange. err: %v", err)
	}
}

func runListenHTTP(serviceHandler servicehandler.ServiceHandler) {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListenHTTP",
	})

	app := gin.Default()
	app.Use(middleware.RequestID()) // NEW — tag every request with a correlation ID first.
	app.NoRoute(server.NoRoute())   // Emit the canonical error envelope for unrouted paths.

	// Use Cloudflare's CF-Connecting-IP header to get the real client IP.
	// This is required because the service runs behind Cloudflare (L7 proxy) + GKE L4 LB.
	// TrustedPlatform takes priority over TrustedProxies in Gin's ClientIP().
	// SetTrustedProxies(nil) is a safety fallback: if CF header is absent (direct access
	// bypassing Cloudflare), c.ClientIP() returns the connection IP instead of trusting XFF.
	app.TrustedPlatform = "CF-Connecting-IP"
	_ = app.SetTrustedProxies(nil)

	// documents
	app.Static("/docs", "docsdev/build/html")
	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	app.GET("/redoc/*any", func(c *gin.Context) {
		c.File("gens/openapi_redoc/api.html")
	})
	app.GET("/openapi.json", func(c *gin.Context) {
		c.File("gens/openapi_redoc/openapi.json")
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
	auth.Use(middleware.RateLimit(10, 20)) // 10 req/s per IP, burst of 20
	auth.POST("/login", service.PostLogin)
	auth.POST("/password-forgot", service.PostPasswordForgot)
	auth.GET("/password-reset", service.GetPasswordReset)
	auth.POST("/password-reset", service.PostPasswordReset)
	auth.POST("/signup", service.PostCustomerSignup)
	auth.GET("/email-verify", service.GetCustomerEmailVerify)
	auth.POST("/email-verify", service.PostCustomerEmailVerify)
	auth.POST("/boot", service.PostBoot)
	// Authenticated auth routes (require middleware)
	authProtected := app.Group("/auth")
	authProtected.Use(middleware.Authenticate())
	authProtected.POST("/unregister", service.PostAuthUnregister)
	authProtected.DELETE("/unregister", service.DeleteAuthUnregister)
	authProtected.POST("/delegate", service.PostDelegate)

	appServer := server.NewServer(serviceHandler)

	v1 := app.Group("v1.0")
	v1.Use(middleware.Authenticate())
	openapi_server.RegisterHandlersWithOptions(v1, appServer, openapi_server.GinServerOptions{
		ErrorHandler: server.BindingErrorHandler,
	})

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

	cfg := config.Get()
	listenAddress := fmt.Sprintf("%s:%d", cfg.ListenIPAudiosock, defaultAudiosockPort)
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
	defer func() {
		_ = listen.Close()
	}()

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
	cfg := config.Get()
	res := fmt.Sprintf("%s:%d", cfg.ListenIPAudiosock, defaultAudiosockPort)

	return res
}
