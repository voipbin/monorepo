package main

import (
	"database/sql"
	"encoding/base64"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

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

const (
	defaultDatabaseDSN             = "testid:testpassword@tcp(127.0.0.1:3306)/test"
	defaultPrometheusEndpoint      = "/metrics"
	defaultPrometheusListenAddress = ":2112"
	defaultRabbitMQAddress         = "amqp://guest:guest@localhost:5672"
	defaultSSLPrivkeyBase64        = ""
	defaultSSLCertBase64           = ""
)

var (
	databaseDSN             = ""
	prometheusEndpoint      = ""
	prometheusListenAddress = ""
	rabbitMQAddress         = ""
	sslPrivkeyBase64        = ""
	sslCertBase64           = ""
)

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
	sqlDB, err := sql.Open("mysql", databaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer sqlDB.Close()

	// connect to rabbitmq
	sock := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
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
	initVariable()

	// init log
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)

	initProm(prometheusEndpoint, prometheusListenAddress)

	// init ssl
	if errWrite := writeBase64(constSSLCertFilename, sslCertBase64); errWrite != nil {
		logrus.Errorf("Could not write the ssl cert file.")
		return
	}

	if errWrite := writeBase64(constSSLPrivFilename, sslPrivkeyBase64); errWrite != nil {
		logrus.Errorf("Could not write the ssl private key file.")
		return
	}

}

func initVariable() {
	log := logrus.WithField("func", "initVariable")
	viper.AutomaticEnv()

	pflag.String("rabbitmq_address", defaultRabbitMQAddress, "Address of the RabbitMQ server (e.g., amqp://guest:guest@localhost:5672)")
	pflag.String("prometheus_endpoint", defaultPrometheusEndpoint, "URL for the Prometheus metrics endpoint")
	pflag.String("prometheus_listen_address", defaultPrometheusListenAddress, "Address for Prometheus to listen on (e.g., localhost:8080)")
	pflag.String("database_dsn", defaultDatabaseDSN, "Data Source Name for database connection (e.g., user:password@tcp(localhost:3306)/dbname)")
	pflag.String("ssl_privkey_base64", defaultSSLPrivkeyBase64, "Base64-encoded private key")
	pflag.String("ssl_cert_base64", defaultSSLCertBase64, "Base64-encoded cert")

	pflag.Parse()

	// rabbitmq_address
	if errFlag := viper.BindPFlag("rabbitmq_address", pflag.Lookup("rabbitmq_address")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("rabbitmq_address", "RABBITMQ_ADDRESS"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	rabbitMQAddress = viper.GetString("rabbitmq_address")

	// prometheus_endpoint
	if errFlag := viper.BindPFlag("prometheus_endpoint", pflag.Lookup("prometheus_endpoint")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("prometheus_endpoint", "PROMETHEUS_ENDPOINT"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	prometheusEndpoint = viper.GetString("prometheus_endpoint")

	// prometheus_listen_address
	if errFlag := viper.BindPFlag("prometheus_listen_address", pflag.Lookup("prometheus_listen_address")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("prometheus_listen_address", "PROMETHEUS_LISTEN_ADDRESS"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	prometheusListenAddress = viper.GetString("prometheus_listen_address")

	// database_dsn
	if errFlag := viper.BindPFlag("database_dsn", pflag.Lookup("database_dsn")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("database_dsn", "DATABASE_DSN"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	databaseDSN = viper.GetString("database_dsn")

	// ssl_privkey_base64
	if errFlag := viper.BindPFlag("ssl_privkey_base64", pflag.Lookup("ssl_privkey_base64")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("ssl_privkey_base64", "SSL_PRIVKEY_BASE64"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	sslPrivkeyBase64 = viper.GetString("ssl_privkey_base64")

	// ssl_cert_base64
	if errFlag := viper.BindPFlag("ssl_cert_base64", pflag.Lookup("ssl_cert_base64")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("ssl_cert_base64", "SSL_CERT_BASE64"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	sslCertBase64 = viper.GetString("ssl_cert_base64")
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
