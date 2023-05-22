package chatbots

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	chatbots := r.Group("/chatbots")

	chatbots.GET("", chatbotsGET)
	chatbots.POST("", chatbotsPOST)

	chatbots.DELETE("/:id", chatbotsIDDELETE)
	chatbots.GET("/:id", chatbotsIDGET)
	chatbots.PUT("/:id", chatbotsIDPUT)
}
