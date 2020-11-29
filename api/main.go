package api

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"gitlab.com/voipbin/bin-manager/tts-manager.git/api/models/common"
	apiv1 "gitlab.com/voipbin/bin-manager/tts-manager.git/api/v1.0"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/buckethandler"
)

// ApplyRoutes applies routes
func ApplyRoutes(r *gin.Engine) {
	api := r.Group("/")

	apiv1.ApplyRoutes(api)
}

// Run runs the service
func Run(bucketHandler buckethandler.BucketHandler, listenAddr string) {
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
		c.Set(common.OBJServiceHandler, bucketHandler)
		c.Next()
	})

	// apply api router
	ApplyRoutes(app)

	go func() {
		app.Run(listenAddr)
	}()
}
