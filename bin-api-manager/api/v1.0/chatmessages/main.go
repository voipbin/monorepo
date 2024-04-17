package chatmessages

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	chatmessages := r.Group("/chatmessages")

	chatmessages.GET("", chatmessagesGET)
	chatmessages.POST("", chatmessagesPOST)

	chatmessages.GET("/:id", chatmessagesIDGET)
	chatmessages.DELETE("/:id", chatmessagesIDDELETE)
}
