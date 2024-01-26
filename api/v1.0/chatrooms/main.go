package chatrooms

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	chatrooms := r.Group("/chatrooms")

	chatrooms.GET("", chatroomsGET)
	chatrooms.POST("", chatroomsPOST)

	chatrooms.DELETE("/:id", chatroomsIDDELETE)
	chatrooms.GET("/:id", chatroomsIDGET)
	chatrooms.PUT("/:id", chatroomsIDPUT)
}
