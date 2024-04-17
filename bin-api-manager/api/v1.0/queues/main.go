package queues

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	targets := r.Group("/queues")

	targets.GET("", queuesGET)
	targets.POST("", queuesPOST)
	targets.GET("/:id", queuesIDGet)
	targets.DELETE("/:id", queuesIDDelete)
	targets.PUT("/:id", queuesIDPUT)
	targets.PUT("/:id/tag_ids", queuesIDTagIDsPUT)
	targets.PUT("/:id/routing_method", queuesIDRoutingMethodPUT)
	targets.PUT("/:id/actions", queuesIDActionsPUT)
}
