package main

import (
	"database/sql"
	"encoding/base64"
	"flag"
	"os"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/sirupsen/logrus"

	"monorepo/bin-hook-manager/api"
	"monorepo/bin-hook-manager/api/models/common"
	"monorepo/bin-hook-manager/pkg/servicehandler"
)

const serviceName = commonoutline.ServiceNameHookManager

var dsn = flag.String("dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn")

var sslPrivkeyBase64 = flag.String("ssl_private_base64", "", "Base64 encoded private key for ssl connection.")
var sslCertBase64 = flag.String("ssl_cert_base64", "", "Base64 encoded cert key for ssl connection.")

var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")

const (
	constSSLPrivFilename = "/tmp/ssl_privkey.pem"
	constSSLCertFilename = "/tmp/ssl_cert.pem"
)

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
	logrus.Debug("Starting the hook service.")
	if errAppRun := app.RunTLS(":443", constSSLCertFilename, constSSLPrivFilename); errAppRun != nil {
		log.Errorf("The hook service ended with error. err: %v", errAppRun)
	}

}

func init() {
	flag.Parse()

	// init log
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)

	// init ssl
	if errWrite := writeBase64(constSSLCertFilename, *sslCertBase64); errWrite != nil {
		logrus.Errorf("Could not write the ssl cert file.")
		return
	}

	if errWrite := writeBase64(constSSLPrivFilename, *sslPrivkeyBase64); errWrite != nil {
		logrus.Errorf("Could not write the ssl private key file.")
		return
	}

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
