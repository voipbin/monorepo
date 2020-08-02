package main

import (
	"flag"

	"github.com/gin-gonic/gin"
	joonix "github.com/joonix/log"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/api-manager/api"
	"gitlab.com/voipbin/bin-manager/api-manager/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/database"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler"
)

var dsn = flag.String("dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn")

var sslKey = flag.String("ssl_private", "./etc/ssl/prikey.pem", "Private key file for ssl connection.")
var sslCert = flag.String("ssl_cert", "./etc/ssl/cert.pem", "Cert key file for ssl connection.")

var jwtKey = flag.String("jwt_key", "voipbin", "key string for jwt hashing")

var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")
var rabbitQueueFlowRequest = flag.String("rabbit_queue_flow", "bin-manager.flow-manager.request", "rabbitmq queue name for flow request")
var rabbitQueueCallRequest = flag.String("rabbit_queue_call", "bin-manager.call-manager.request", "rabbitmq queue name for request listen")
var rabbitExchangeDelay = flag.String("rabbit_exchange_delay", "bin-manager.delay", "rabbitmq exchange name for delayed messaging.")

func main() {

	db, err := database.Init(*dsn)
	if err != nil {
		logrus.Errorf("Could not initiate database. err: %v", err)
		return
	}

	sock := rabbitmq.NewRabbit(*rabbitAddr)
	sock.Connect()

	app := gin.Default()

	// injects
	app.Use(database.Inject(db))
	app.Use(requesthandler.Inject(sock, *rabbitExchangeDelay, *rabbitQueueCallRequest, *rabbitQueueFlowRequest))

	// set jwt middleware
	app.Use(middleware.JWTMiddleware())

	// apply api router
	api.ApplyRoutes(app)

	logrus.Debug("Starting the api service.")
	app.RunTLS(":443", *sslCert, *sslKey)
}

func init() {
	flag.Parse()

	// init log
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)

	// init middleware
	middleware.Init(*jwtKey)
}
