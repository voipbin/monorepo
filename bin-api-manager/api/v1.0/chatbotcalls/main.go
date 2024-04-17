package chatbotcalls

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	chatbotcalls := r.Group("/chatbotcalls")

	chatbotcalls.GET("", chatbotcallsGET)

	chatbotcalls.GET("/:id", chatbotcallsIDGET)
	chatbotcalls.DELETE("/:id", chatbotcallsIDDELETE)
}
