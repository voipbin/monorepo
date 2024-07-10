package service_agents

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	targets := r.Group("/service_agents")

	// calls
	targets.GET("/calls", callsGET)
	targets.GET("/calls/:id", callsIDGET)

	// chatrooms
	targets.GET("/chatrooms", chatroomsGET)
	targets.POST("/chatrooms", chatroomsPOST)
	targets.GET("/chatrooms/:id", chatroomsIDGET)
	targets.PUT("/chatrooms/:id", chatroomsIDPUT)
	targets.DELETE("/chatrooms/:id", chatroomsIDDELETE)

	// chatroom messages
	targets.GET("/chatroommessages", chatroommessagesGET)
	targets.POST("/chatroommessages", chatroommessagesPOST)
	targets.GET("/:id", chatroommessagesIDGET)
	targets.DELETE("/:id", chatroommessagesIDDELETE)

	// conversations
	targets.GET("/conversations", conversationsGET)
	targets.GET("/conversations/:id", conversationsIDGET)
	targets.GET("/conversations/:id/messages", conversationsIDMessagesGet)
	targets.POST("/conversations/:id/messages", conversationsIDMessagesPost)

}
