package conversations

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	conversations := r.Group("/conversations")

	conversations.GET("", conversationsGet)

	conversations.GET("/:id", conversationsIDGet)
	conversations.PUT("/:id", conversationsIDPut)

	conversations.GET("/:id/messages", conversationsIDMessagesGet)
	conversations.POST("/:id/messages", conversationsIDMessagesPost)
}
