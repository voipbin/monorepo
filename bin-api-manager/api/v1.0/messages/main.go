package messages

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	messages := r.Group("/messages")

	messages.GET("", messagesGET)
	messages.POST("", messagesPOST)

	messages.GET("/:id", messagesIDGET)
	messages.DELETE("/:id", messagesIDDELETE)
}
