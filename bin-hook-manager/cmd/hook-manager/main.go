package main

import (
	"encoding/base64"
	"net/http"
	"os"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-hook-manager/api"
	"monorepo/bin-hook-manager/api/models/common"
	"monorepo/bin-hook-manager/internal/config"
	"monorepo/bin-hook-manager/pkg/servicehandler"
)

const serviceName = commonoutline.ServiceNameHookManager

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

var rootCmd = &cobra.Command{
	Use:   "hook-manager",
	Short: "Hook Manager Service",
	Long:  `Hook Manager handles webhook receivers for the VoIPbin platform.`,
	Run:   runService,
}

func init() {
	// Define flags
	rootCmd.Flags().String("database_dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "Data Source Name for database connection (e.g., user:password@tcp(localhost:3306)/dbname)")
	rootCmd.Flags().String("prometheus_endpoint", "/metrics", "URL for the Prometheus metrics endpoint")
	rootCmd.Flags().String("prometheus_listen_address", ":2112", "Address for Prometheus to listen on (e.g., localhost:8080)")
	rootCmd.Flags().String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "Address of the RabbitMQ server (e.g., amqp://guest:guest@localhost:5672)")
	rootCmd.Flags().String("ssl_privkey_base64", "", "Base64-encoded private key")
	rootCmd.Flags().String("ssl_cert_base64", "", "Base64-encoded cert")

	// Initialize configuration
	config.InitConfig(rootCmd)

	// Initialize logging
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("Failed to execute command: %v", err)
		os.Exit(1)
	}
}

func runService(cmd *cobra.Command, args []string) {
	log := logrus.WithField("func", "runService")

	cfg := config.Get()

	// Initialize Prometheus
	initProm(cfg.PrometheusEndpoint, cfg.PrometheusListenAddress)

	// init ssl
	if errWrite := writeBase64(constSSLCertFilename, cfg.SSLCertBase64); errWrite != nil {
		log.Errorf("Could not write the ssl cert file.")
		return
	}

	if errWrite := writeBase64(constSSLPrivFilename, cfg.SSLPrivkeyBase64); errWrite != nil {
		log.Errorf("Could not write the ssl private key file.")
		return
	}

	// connect to database
	sqlDB, err := commondatabasehandler.Connect(cfg.DatabaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer commondatabasehandler.Close(sqlDB)

	// connect to rabbitmq
	sock := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
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

// initProm inits prometheus settings
func initProm(endpoint, listen string) {
	log := logrus.WithField("func", "initProm").WithFields(logrus.Fields{
		"endpoint": endpoint,
		"listen":   listen,
	})

	http.Handle(endpoint, promhttp.Handler())
	go func() {
		for {
			if errListen := http.ListenAndServe(listen, nil); errListen != nil {
				log.Errorf("Could not start prometheus listener. err: %v", errListen)
				time.Sleep(time.Second * 1)
				continue
			}
			log.Infof("Finishing the prometheus listener.")
			break
		}
	}()
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
	defer func() {
		_ = file.Close()
	}()

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
