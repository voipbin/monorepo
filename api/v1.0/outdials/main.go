package outdials

import (
	"github.com/gin-gonic/gin"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	outdials := r.Group("/outdials")

	outdials.GET("", outdialsGET)
	outdials.POST("", outdialsPOST)

	outdials.DELETE("/:id", outdialsIDDELETE)
	outdials.GET("/:id", outdialsIDGET)
	outdials.PUT("/:id", outdialsIDPUT)

	outdials.PUT("/:id/data", outdialsIDDataPUT)
	outdials.PUT("/:id/campaign_id", outdialsIDCampaignIDPUT)

	outdials.POST("/:id/targets", outdialsIDTargetsPOST)
	outdials.GET("/:id/targets", outdialsIDTargetsGET)
	outdials.GET("/:id/targets/:target_id", outdialsIDTargetsIDGET)
	outdials.DELETE("/:id/targets/:target_id", outdialsIDTargetsIDDELETE)
}
