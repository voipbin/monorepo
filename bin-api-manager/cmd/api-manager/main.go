package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// swagger embed files
	// gin-swagger middleware
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/internal/config"
	"monorepo/bin-api-manager/internal/nethandler"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/lib/service"
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/cachehandler"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/pubsubhandler"
	"monorepo/bin-api-manager/pkg/ratelimithandler"
	"monorepo/bin-api-manager/pkg/servicehandler"
	"monorepo/bin-api-manager/pkg/streamhandler"
	"monorepo/bin-api-manager/pkg/subscribehandler"
	"monorepo/bin-api-manager/pkg/websockhandler"
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
	commondatabasehandler.RegisterDBStatsCollector(sqlDB, "main")
	defer commondatabasehandler.Close(sqlDB)

	// connect to cache
	cache := cachehandler.NewHandler(cfg.RedisAddress, cfg.RedisPassword, cfg.RedisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// Dedicated Redis client for the customer rate limiter (VOIP-1302
	// §4-1). Deliberately separate from the cachehandler's Redis client:
	// pkg/cachehandler exposes only domain cache methods (no raw client
	// getter, matching the monorepo-wide CacheHandler convention), and
	// rate limiting is not a caching concern. See
	// pkg/ratelimithandler for the local wrapper interface.
	rateLimitRedisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDatabase,
	})
	defer func() {
		if errClose := rateLimitRedisClient.Close(); errClose != nil {
			log.Errorf("Could not close rate limiter Redis client. err: %v", errClose)
		}
	}()
	rateLimiter := ratelimithandler.NewHandler(rateLimitRedisClient)

	// connect to rabbitmq
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	run(ctx, sockHandler, db, cache, rateLimiter)

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
	cache cachehandler.CacheHandler,
	rateLimiter ratelimithandler.RateLimiter,
) {
	log := logrus.WithField("func", "run")

	cfg := config.Get()
	addressAdvertiseStream, err := getAddressAdvertiseAudiosock()
	if err != nil {
		// Fail fast: an empty/unresolvable advertise address would be
		// silently handed to Asterisk, which would then be unable to dial
		// back for AudioSocket/ExternalMedia streaming.
		log.Fatalf("Could not resolve the audiosocket advertise address. err: %v", err)
	}

	// Observability: the advertise address is resolved once at startup and
	// never logged afterward, so a misconfigured deployment (e.g. POD_IP
	// pointing at the wrong interface) is otherwise invisible until a
	// dial-back actually fails. addressSource is informational only -- it
	// does not affect resolution, which always goes through
	// nethandler.AdvertiseIP() above.
	addressSource := "auto-detected network interface"
	if os.Getenv(nethandler.EnvPodIP) != "" {
		addressSource = "POD_IP override"
	}
	log.Infof("Resolved the audiosocket advertise address. address: %s, source: %s", addressAdvertiseStream, addressSource)

	// create handlers
	requestHandler := requesthandler.NewRequestHandler(sockHandler, "api_manager")
	pubsubBroker := pubsubhandler.NewBrokerHandler()
	streamHandler := streamhandler.NewStreamHandler(requestHandler, addressAdvertiseStream)

	// per-pod subscribe queue name -- constructed here (before websockHandler) so it can be
	// shared with websockhandler's scopeRefCount for dynamic AMQP bind/unbind (VOIP-1258 §9).
	queueNamePod := fmt.Sprintf("%s-%s", commonoutline.QueueNameAPISubscribe, uuid.Must(uuid.NewV4()))

	websockHandler := websockhandler.NewWebsockHandler(requestHandler, streamHandler, sockHandler, pubsubBroker, queueNamePod)
	serviceHandler, err := servicehandler.NewServiceHandler(requestHandler, db, cache, websockHandler, cfg.GCPProjectID, cfg.GCPBucketName, cfg.JWTKey, cfg.PublicBaseURL)
	if err != nil {
		log.Fatalf("Could not create service handler. err: %v", err)
	}

	go runSubscribe(sockHandler, requestHandler, pubsubBroker, queueNamePod)
	go runListenHTTP(serviceHandler, rateLimiter)
	go runListenStreamsock(ctx, streamHandler)

}

func runSubscribe(
	sockHandler sockhandler.SockHandler,
	reqHandler requesthandler.RequestHandler,
	pubHandler pubsubhandler.PubHandler,
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

		pubHandler,
	)

	// run. NOTE: the VOIP-1258 "#" wildcard binding to the new topic exchange lives INSIDE
	// subscribeHandler.Run(), sequenced before ConsumeMessage starts -- see that function's
	// doc comment for why doing it here (after Run() returns) is unsafe: Run() starts
	// ConsumeMessage on a separate goroutine and returns immediately, so a QueueBind call
	// here would race the in-flight basic.consume RPC on the same AMQP channel and could
	// intermittently 503 the channel closed (reproduced in bin-agent-manager production,
	// 2026-07-14, fixed there in commit ca8c104a9 and here proactively for the same reason).
	if errRun := subHandler.Run(); errRun != nil {
		log.Errorf("Could not run the subscribe handler. err: %v", errRun)
		return
	}
}

func runListenHTTP(serviceHandler servicehandler.ServiceHandler, rateLimiter ratelimithandler.RateLimiter) {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListenHTTP",
	})

	// Equivalent to gin.Default() (Logger + Recovery), except the access
	// logger skips the provisioning paths: their token query parameter is a
	// short-lived SIP credential-fetch secret and must never reach stdout.
	// gin.Default() is Logger() + Recovery(), and Logger() is
	// LoggerWithConfig(LoggerConfig{}), so an empty config plus SkipPaths
	// keeps the exact same log format for every other route.
	// The public handler (lib/service.GetProvisioningExtension) emits its own
	// structured log line to preserve observability for the skipped paths.
	app := gin.New()
	app.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/provisioning/extension", "/v1.0/provisioning/extension"},
	}))
	app.Use(gin.Recovery())
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

	cfg := config.Get()

	// Public (unauthenticated) SIP softphone provisioning. Registered after
	// the global servicehandler-injection app.Use above, which already covers
	// this group — do not add a duplicate injection middleware here.
	provisioning := app.Group("/provisioning")
	provisioning.Use(middleware.RateLimit("provisioning_public", cfg.RateLimitProvisioningPublicRPS, cfg.RateLimitProvisioningPublicBurst))
	provisioning.GET("/extension", service.GetProvisioningExtension)

	// register basic services
	app.GET("/ping", service.GetPing)
	auth := app.Group("/auth")
	auth.Use(middleware.RateLimit("auth_public", cfg.RateLimitAuthPublicRPS, cfg.RateLimitAuthPublicBurst))
	auth.POST("/login", service.PostLogin)
	auth.POST("/password-forgot", service.PostPasswordForgot)
	auth.GET("/password-reset", service.GetPasswordReset)
	auth.POST("/password-reset", service.PostPasswordReset)
	auth.POST("/signup", service.PostCustomerSignup)
	auth.GET("/email-verify", service.GetCustomerEmailVerify)
	auth.POST("/email-verify", service.PostCustomerEmailVerify)
	auth.POST("/boot", service.PostBoot)
	// Authenticated auth routes (require middleware). This group's order
	// (Authenticate -> EnforceAccountStatus) is unchanged by VOIP-1302 and
	// deliberately does NOT include CustomerRateLimit -- customer-tier
	// rate limiting is scoped to the v1.0 group only (design doc §3/§4-14).
	// Keeping this order intact preserves the existing behavior that a
	// frozen account cannot call POST /auth/delegate.
	authProtected := app.Group("/auth")
	authProtected.Use(middleware.RateLimit("auth_protected", cfg.RateLimitAuthProtectedRPS, cfg.RateLimitAuthProtectedBurst))
	authProtected.Use(middleware.Authenticate())
	authProtected.Use(middleware.EnforceAccountStatus())
	authProtected.POST("/unregister", service.PostAuthUnregister)
	authProtected.DELETE("/unregister", service.DeleteAuthUnregister)
	authProtected.POST("/delegate", service.PostDelegate)

	appServer := server.NewServer(serviceHandler)

	customerRateLimitConfig := middleware.CustomerRateLimitConfig{
		CustomerRPS:   cfg.RateLimitCustomerV1RPS,
		CustomerBurst: cfg.RateLimitCustomerV1Burst,
		DirectRPS:     cfg.RateLimitCustomerV1DirectRPS,
		DirectBurst:   cfg.RateLimitCustomerV1DirectBurst,
		DelegateRPS:   cfg.RateLimitCustomerV1DelegateRPS,
		DelegateBurst: cfg.RateLimitCustomerV1DelegateBurst,
		RedisTimeout:  time.Duration(cfg.RateLimitCustomerRedisTimeoutMs) * time.Millisecond,
	}

	v1 := app.Group("v1.0")
	v1.Use(middleware.RateLimit("v1", cfg.RateLimitV1RPS, cfg.RateLimitV1Burst))
	// Order is deliberate (VOIP-1302 §4-6): Authenticate() populates
	// auth_identity, CustomerRateLimit consumes it and can reject before
	// the frozen-account check runs -- saving a CustomerRawSelfGet RPC on
	// requests that are about to be rate-limited anyway. This ordering is
	// unique to the v1.0 group; authProtected keeps the original order.
	v1.Use(middleware.Authenticate())
	v1.Use(middleware.CustomerRateLimit(rateLimiter, customerRateLimitConfig))
	v1.Use(middleware.EnforceAccountStatus())
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

	listenAddress := getAddressListenAudiosock()
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

// getAddressListenAudiosock returns the address the AudioSocket listener
// binds to. This is deliberately NOT derived from the advertise address
// (see getAddressAdvertiseAudiosock): the socket must accept connections on
// every interface, while the advertise address handed to Asterisk must be a
// concrete, routable host. An empty host in "host:port" binds to all
// interfaces (equivalent to 0.0.0.0:<port>).
func getAddressListenAudiosock() string {
	return fmt.Sprintf(":%d", defaultAudiosockPort)
}

// getAddressAdvertiseAudiosock returns the host:port that this process
// hands to Asterisk (via CallV1ExternalMediaStart) so it can dial back for
// AudioSocket/ExternalMedia streaming. This is intentionally separate from
// the AudioSocket listen address (see runListenStreamsock), which always
// binds to all interfaces.
//
// nethandler.AdvertiseIP() itself does not validate that the POD_IP
// override is a well-formed IP address -- it returns that value verbatim so
// it stays reusable if this package is ever promoted to serve other
// consumers that may legitimately advertise a hostname. AudioSocket
// specifically needs a real, dialable IP for Asterisk's dial-back, so that
// validation belongs here, at the consumer that actually requires it.
func getAddressAdvertiseAudiosock() (string, error) {
	ip, err := nethandler.AdvertiseIP()
	if err != nil {
		return "", errors.Wrapf(err, "could not resolve the audiosocket advertise ip")
	}

	if net.ParseIP(ip) == nil {
		// Fail fast: a non-IP POD_IP value (e.g. a hostname, as some other
		// services set for their own advertise-style env vars) would be
		// silently handed to Asterisk, reproducing the same dial-back
		// failure this fix addresses.
		return "", errors.Errorf("the resolved audiosocket advertise address is not a valid ip. ip: %s", ip)
	}

	// net.JoinHostPort (not fmt.Sprintf("%s:%d", ...)) is mandatory here:
	// net.ParseIP above accepts IPv6 as well as IPv4, and a bare "%s:%d"
	// join produces an unparseable string for an IPv6 host (e.g.
	// "2001:db8::1:9000", which net.SplitHostPort/net.ResolveTCPAddr both
	// reject as a malformed address). JoinHostPort brackets an IPv6 host
	// automatically (e.g. "[2001:db8::1]:9000").
	res := net.JoinHostPort(ip, strconv.Itoa(defaultAudiosockPort))

	return res, nil
}
