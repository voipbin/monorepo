package campaigns

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	campaigns := r.Group("/campaigns")

	campaigns.GET("", campaignsGET)
	campaigns.POST("", campaignsPOST)

	campaigns.DELETE("/:id", campaignsIDDELETE)
	campaigns.GET("/:id", campaignsIDGET)
	campaigns.PUT("/:id", campaignsIDPUT)

	campaigns.PUT("/:id/status", campaignsIDStatusPUT)
	campaigns.PUT("/:id/service_level", campaignsIDServiceLevelPUT)
	campaigns.PUT("/:id/actions", campaignsIDActionsPUT)
	campaigns.PUT("/:id/resource_info", campaignsIDResourceInfoPUT)
	campaigns.PUT("/:id/next_campaign_id", campaignsIDNextCampaignIDPUT)
	campaigns.GET("/:id/campaigncalls", campaignsIDCampaigncallsGET)
}
