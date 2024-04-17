package chatroommessages

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	chatroommessages := r.Group("/chatroommessages")

	chatroommessages.GET("", chatroommessagesGET)
	chatroommessages.POST("", chatroommessagesPOST)

	chatroommessages.GET("/:id", chatroommessagesIDGET)
	chatroommessages.DELETE("/:id", chatroommessagesIDDELETE)
}
