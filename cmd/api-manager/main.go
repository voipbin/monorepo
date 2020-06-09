package main

import (
	"flag"

	"github.com/gin-gonic/gin"
	joonix "github.com/joonix/log"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/api-manager/api"
	"gitlab.com/voipbin/bin-manager/api-manager/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/database"
)

var dsn = flag.String("dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn")

var jwtKey = flag.String("jwt_key", "voipbin", "key string for jwt hashing")

func main() {

	db, err := database.Init(*dsn)
	if err != nil {
		logrus.Errorf("Could not initiate database. err: %v", err)
		return
	}

	app := gin.Default()
	app.Use(database.Inject(db))
	app.Use(middleware.JWTMiddleware())

	// apply api router
	api.ApplyRoutes(app)

	app.Run(":" + "8080")
}

func init() {
	flag.Parse()

	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)

	middleware.Init(*jwtKey)
}
