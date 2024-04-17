package api

import (
	"github.com/gin-gonic/gin"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/auth"
	apiv1 "gitlab.com/voipbin/bin-manager/api-manager.git/api/v1.0"
)

// ApplyRoutes applies router to gin Router
func ApplyRoutes(r *gin.Engine) {
	api := r.Group("/")

	api.GET("ping", ping)

	auth.ApplyRoutes(api)
	apiv1.ApplyRoutes(api)
}

// ping handler
//	@Summary		Returns message pong
//	@Description	Used to check the server is alive
//	@Produce		json
//	@Router			/ping [get]
//	@Success		200	"{"message": "pong"}"
//	@BasePath
func ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
