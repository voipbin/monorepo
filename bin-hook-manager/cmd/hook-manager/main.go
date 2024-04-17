package main

import (
	"database/sql"
	"flag"
	"time"

	commonoutline "gitlab.com/voipbin/bin-manager/common-handler.git/models/outline"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/hook-manager.git/api"
	"gitlab.com/voipbin/bin-manager/hook-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/hook-manager.git/pkg/servicehandler"
)

const serviceName = commonoutline.ServiceNameHookManager

var dsn = flag.String("dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn")

var sslKey = flag.String("ssl_private", "./etc/ssl/prikey.pem", "Private key file for ssl connection.")
var sslCert = flag.String("ssl_cert", "./etc/ssl/cert.pem", "Cert key file for ssl connection.")

var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")

// @title VoIPBIN project event hook
// @version 1.0
// @description RESTful API documents for VoIPBIN project.
// @termsOfService http://swagger.io/terms/

// @contact.name VoIPBIN Project
// @contact.email pchero21@gmail.com

// @host hook.voipbin.net
// @BasePath
func main() {

	log := logrus.WithField("func", "main")

	// connect to database
	sqlDB, err := sql.Open("mysql", *dsn)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer sqlDB.Close()

	// connect to rabbitmq
	sock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	sock.Connect()

	// create servicehandler
	requestHandler := requesthandler.NewRequestHandler(sock, serviceName)
	serviceHandler := servicehandler.NewServiceHandler(requestHandler)

	app := gin.Default()
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
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// inject servicehandler
	app.Use(func(c *gin.Context) {
		c.Set(common.OBJServiceHandler, serviceHandler)
		c.Next()
	})

	// apply api router
	api.ApplyRoutes(app)

	go func() {
		if err := app.Run(":80"); err != nil {
			log.Errorf("Could not run the app. err: %v", err)
		}
	}()
	logrus.Debug("Starting the api service.")
	if errAppRun := app.RunTLS(":443", *sslCert, *sslKey); errAppRun != nil {
		log.Errorf("The api service ended with error. err: %v", errAppRun)
	}

}

func init() {
	flag.Parse()

	// init log
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}
