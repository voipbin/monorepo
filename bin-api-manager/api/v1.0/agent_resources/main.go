package agentresources

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	targets := r.Group("/agent_resources")

	targets.GET("", agentResourcesGET)

	targets.DELETE("/:id", agentResourcesIDDelete)
	targets.GET("/:id", agentResourcesIDGet)
}
