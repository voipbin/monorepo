package service_agents

import (
	"monorepo/bin-api-manager/lib/middleware"

	"github.com/gin-gonic/gin"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}
