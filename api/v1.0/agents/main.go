package agents

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	targets := r.Group("/agents")

	targets.POST("", agentsPOST)
	targets.GET("", agentsGET)

	targets.DELETE("/:id", agentsIDDelete)
	targets.GET("/:id", agentsIDGet)
	targets.PUT("/:id", agentsIDPUT)

	targets.PUT("/:id/addresses", agentsIDAddressesPUT)
	targets.PUT("/:id/password", agentsIDPasswordPUT)
	targets.PUT("/:id/permission", agentsIDPermissionPUT)
	targets.PUT("/:id/status", agentsIDStatusPUT)
	targets.PUT("/:id/tag_ids", agentsIDTagIDsPUT)
}
