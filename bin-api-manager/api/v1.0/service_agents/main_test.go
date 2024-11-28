package service_agents

import (
	"github.com/gin-gonic/gin"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0")
	ApplyRoutes(v1)
}
